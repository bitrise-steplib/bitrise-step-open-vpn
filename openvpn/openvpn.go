// Package openvpn is a small reusable bundle that establishes an OpenVPN
// client connection. It encapsulates the platform-specific details (Linux vs
// macOS) behind a single Connector so Steps and other tooling can reuse it.
//
// The platform-specific connectors live in linux.go and darwin.go. These file
// names intentionally avoid the `_linux.go`/`_darwin.go` suffixes, which would
// apply implicit GOOS build constraints; all connectors must compile on every
// platform so NewConnector can dispatch at runtime.
package openvpn

import (
	"fmt"
	"os"
	"time"

	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/fileutil"
	"github.com/bitrise-io/go-utils/v2/log"
)

const (
	// OSLinux and OSDarwin are the supported runtime.GOOS values.
	OSLinux  = "linux"
	OSDarwin = "darwin"

	// statusCheckDelay is how long we wait before checking that the connection
	// came up, matching the original Step's `sleep 5`.
	statusCheckDelay = 5 * time.Second
)

// Connection holds everything needed to dial an OpenVPN server. The
// certificate and key fields hold the already base64-decoded PEM contents.
type Connection struct {
	Host       string
	Port       int
	Proto      string
	CACert     []byte
	ClientCert []byte
	ClientKey  []byte
}

// Connector establishes an OpenVPN connection, writing the underlying OpenVPN
// process output to the given log path. Each supported operating system has
// its own implementation.
type Connector interface {
	Connect(conn Connection, logPath string) error
}

// NewConnector returns the Connector implementation for the given operating
// system (a runtime.GOOS value such as "linux" or "darwin"), or an error if
// the operating system is not supported.
func NewConnector(os string, cmdFactory command.Factory, fileManager fileutil.FileManager, logger log.Logger) (Connector, error) {
	switch os {
	case OSLinux:
		return linuxConnector{cmdFactory: cmdFactory, fileManager: fileManager, logger: logger}, nil
	case OSDarwin:
		return darwinConnector{cmdFactory: cmdFactory, fileManager: fileManager, logger: logger}, nil
	default:
		return nil, fmt.Errorf("unsupported operating system: %s", os)
	}
}

// writeFiles writes each path/content pair, failing on the first error.
func writeFiles(fileManager fileutil.FileManager, files map[string][]byte) error {
	for path, content := range files {
		if err := fileManager.WriteBytes(path, content); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
	}
	return nil
}

// createLogFile creates (or truncates) the file the OpenVPN output is written to.
func createLogFile(logPath string) (*os.File, error) {
	logFile, err := os.Create(logPath)
	if err != nil {
		return nil, fmt.Errorf("create log file: %w", err)
	}
	return logFile, nil
}

// failWithLog prints the captured OpenVPN log and returns the given error.
func failWithLog(logger log.Logger, logPath string, cause error) error {
	if content, err := os.ReadFile(logPath); err == nil {
		logger.Printf("OpenVPN log:")
		logger.Printf("%s", string(content))
	}
	return cause
}
