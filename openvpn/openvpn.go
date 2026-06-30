// Package openvpn is a small reusable bundle that establishes an OpenVPN
// client connection. It encapsulates the connection details behind a single
// Connector so Steps and other tooling can reuse it.
//
// OpenVPN needs elevated privileges to create the tun device and modify the
// routing table (CAP_NET_ADMIN). Rather than implicitly assuming the Step runs
// as root, the connection is started explicitly via `sudo`, and all writable
// state lives in a temporary directory.
package openvpn

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/fileutil"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/pathutil"
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
// process output to the given log path.
type Connector interface {
	Connect(conn Connection, logPath string) error
}

// NewConnector returns a Connector for the given operating system (a
// runtime.GOOS value such as "linux" or "darwin"), or an error if the
// operating system is not supported.
func NewConnector(os string, cmdFactory command.Factory, fileManager fileutil.FileManager, pathProvider pathutil.PathProvider, logger log.Logger) (Connector, error) {
	switch os {
	case OSLinux, OSDarwin:
		return connector{
			os:           os,
			cmdFactory:   cmdFactory,
			fileManager:  fileManager,
			pathProvider: pathProvider,
			logger:       logger,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported operating system: %s", os)
	}
}

type connector struct {
	os           string
	cmdFactory   command.Factory
	fileManager  fileutil.FileManager
	pathProvider pathutil.PathProvider
	logger       log.Logger
}

// Connect writes the certificates to a writable temp directory and starts
// openvpn as a background process via sudo, then confirms the connection came
// up. The process is intentionally left running for subsequent Steps.
func (c connector) Connect(conn Connection, logPath string) error {
	c.logger.Printf("Configuring OpenVPN client")

	caPath, certPath, keyPath, err := writeCerts(c.pathProvider, c.fileManager, conn)
	if err != nil {
		return err
	}

	logFile, err := createLogFile(logPath)
	if err != nil {
		return err
	}
	defer func() { _ = logFile.Close() }()

	c.logger.Printf("Starting openvpn")
	cmd := c.cmdFactory.Create("sudo", openvpnArgs(conn, caPath, certPath, keyPath), &command.Opts{
		Stdout: logFile,
		Stderr: logFile,
	})
	if err := cmd.Start(); err != nil {
		return failWithLog(c.logger, logPath, fmt.Errorf("start openvpn: %w", err))
	}

	c.logger.Printf("Checking connection status")
	return c.checkConnection(cmd, logPath)
}

// checkConnection verifies the connection came up. This is the only
// OS-specific part of connecting: on Linux we confirm the tun0 interface
// exists; on macOS (where the tun device is a less predictable utunN) we
// confirm the openvpn process is still running.
func (c connector) checkConnection(cmd command.Command, logPath string) error {
	if c.os == OSDarwin {
		exited := make(chan error, 1)
		go func() { exited <- cmd.Wait() }()
		select {
		case err := <-exited:
			return failWithLog(c.logger, logPath, fmt.Errorf("openvpn process exited: %w", err))
		case <-time.After(statusCheckDelay):
			return nil
		}
	}

	// Linux
	time.Sleep(statusCheckDelay)
	_, err := c.cmdFactory.Create("ip", []string{"link", "show", "tun0"}, nil).RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return failWithLog(c.logger, logPath, fmt.Errorf("no OpenVPN tunnel (tun0) found"))
	}
	return nil
}

// writeCerts writes the connection's certificates and key into a fresh
// temporary directory and returns their absolute paths. Keeping them in a
// writable temp dir (rather than /etc/openvpn) is what lets the Step run
// without root.
func writeCerts(pathProvider pathutil.PathProvider, fileManager fileutil.FileManager, conn Connection) (caPath, certPath, keyPath string, err error) {
	certDir, err := pathProvider.CreateTempDir("open-vpn-certs")
	if err != nil {
		return "", "", "", fmt.Errorf("create certificate directory: %w", err)
	}

	caPath = certDir + "/ca.crt"
	certPath = certDir + "/client.crt"
	keyPath = certDir + "/client.key"
	for path, content := range map[string][]byte{
		caPath:   conn.CACert,
		certPath: conn.ClientCert,
		keyPath:  conn.ClientKey,
	} {
		if err := fileManager.WriteBytes(path, content); err != nil {
			return "", "", "", fmt.Errorf("write %s: %w", path, err)
		}
	}
	return caPath, certPath, keyPath, nil
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

// openvpnArgs builds the argument list passed to `sudo` to run openvpn.
func openvpnArgs(conn Connection, caPath, certPath, keyPath string) []string {
	return []string{
		"openvpn",
		"--client",
		"--dev", "tun",
		"--proto", conn.Proto,
		"--remote", conn.Host, strconv.Itoa(conn.Port),
		"--resolv-retry", "infinite",
		"--nobind",
		"--persist-key",
		"--persist-tun",
		"--comp-lzo",
		"--verb", "3",
		"--ca", caPath,
		"--cert", certPath,
		"--key", keyPath,
	}
}
