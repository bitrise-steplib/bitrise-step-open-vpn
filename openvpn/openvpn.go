// Package openvpn is a small reusable bundle that establishes an OpenVPN
// client connection. It encapsulates the platform-specific details (Linux vs
// macOS) behind a single Connector so Steps and other tooling can reuse it.
package openvpn

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/fileutil"
	"github.com/bitrise-io/go-utils/v2/log"
)

const (
	// OSLinux and OSDarwin are the supported runtime.GOOS values.
	OSLinux  = "linux"
	OSDarwin = "darwin"

	// linuxConfigDir is where the OpenVPN service expects its config and keys.
	linuxConfigDir = "/etc/openvpn"

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

// Connector establishes an OpenVPN connection for a given operating system.
// All side-effecting dependencies are injected so the logic stays testable.
type Connector struct {
	os          string
	cmdFactory  command.Factory
	fileManager fileutil.FileManager
	logger      log.Logger
}

// NewConnector creates a Connector for the given operating system (a
// runtime.GOOS value such as "linux" or "darwin").
func NewConnector(os string, cmdFactory command.Factory, fileManager fileutil.FileManager, logger log.Logger) Connector {
	return Connector{
		os:          os,
		cmdFactory:  cmdFactory,
		fileManager: fileManager,
		logger:      logger,
	}
}

// Connect establishes the VPN connection, writing the underlying OpenVPN
// process output to logPath.
func (c Connector) Connect(conn Connection, logPath string) error {
	switch c.os {
	case OSLinux:
		return c.connectLinux(conn, logPath)
	case OSDarwin:
		return c.connectDarwin(conn, logPath)
	default:
		return fmt.Errorf("unsupported operating system: %s", c.os)
	}
}

// connectLinux configures /etc/openvpn and starts the openvpn service.
func (c Connector) connectLinux(conn Connection, logPath string) error {
	c.logger.Printf("Configuring for Linux")

	files := map[string][]byte{
		filepath.Join(linuxConfigDir, "ca.crt"):     conn.CACert,
		filepath.Join(linuxConfigDir, "client.crt"): conn.ClientCert,
		filepath.Join(linuxConfigDir, "client.key"): conn.ClientKey,
	}
	for path, content := range files {
		if err := c.fileManager.WriteBytes(path, content); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
	}

	configPath := filepath.Join(linuxConfigDir, "client.conf")
	if err := c.fileManager.Write(configPath, RenderClientConfig(conn), 0644); err != nil {
		return fmt.Errorf("write %s: %w", configPath, err)
	}

	logFile, err := os.Create(logPath)
	if err != nil {
		return fmt.Errorf("create log file: %w", err)
	}
	defer func() { _ = logFile.Close() }()

	c.logger.Printf("Starting openvpn service")
	cmd := c.cmdFactory.Create("service", []string{"openvpn", "start", "client"}, &command.Opts{
		Stdout: logFile,
		Stderr: logFile,
	})
	if err := cmd.Run(); err != nil {
		return c.failWithLog(logPath, fmt.Errorf("start openvpn service: %w", err))
	}

	c.logger.Printf("Checking connection status")
	time.Sleep(statusCheckDelay)
	if !c.linuxTunnelIsUp() {
		return c.failWithLog(logPath, fmt.Errorf("no OpenVPN tunnel (tun0) found"))
	}

	return nil
}

// connectDarwin writes the keys next to the working directory and starts
// openvpn as a background process via sudo.
func (c Connector) connectDarwin(conn Connection, logPath string) error {
	c.logger.Printf("Configuring for macOS")

	files := map[string][]byte{
		"ca.crt":     conn.CACert,
		"client.crt": conn.ClientCert,
		"client.key": conn.ClientKey,
	}
	for path, content := range files {
		if err := c.fileManager.WriteBytes(path, content); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
	}

	logFile, err := os.Create(logPath)
	if err != nil {
		return fmt.Errorf("create log file: %w", err)
	}
	defer func() { _ = logFile.Close() }()

	c.logger.Printf("Starting openvpn")
	cmd := c.cmdFactory.Create("sudo", darwinArgs(conn), &command.Opts{
		Stdout: logFile,
		Stderr: logFile,
	})
	if err := cmd.Start(); err != nil {
		return c.failWithLog(logPath, fmt.Errorf("start openvpn: %w", err))
	}

	c.logger.Printf("Checking connection status")
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()

	select {
	case err := <-exited:
		return c.failWithLog(logPath, fmt.Errorf("openvpn process exited: %w", err))
	case <-time.After(statusCheckDelay):
		// Still running after the wait window: connection is considered up.
		// The process is intentionally left running for subsequent Steps.
		return nil
	}
}

// linuxTunnelIsUp reports whether a tun0 interface exists.
func (c Connector) linuxTunnelIsUp() bool {
	out, err := c.cmdFactory.Create("ifconfig", nil, nil).RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(out, "tun0")
}

// failWithLog prints the captured OpenVPN log and returns the given error.
func (c Connector) failWithLog(logPath string, cause error) error {
	if content, err := os.ReadFile(logPath); err == nil {
		c.logger.Printf("OpenVPN log:")
		c.logger.Printf("%s", string(content))
	}
	return cause
}

// RenderClientConfig renders the /etc/openvpn/client.conf used on Linux.
func RenderClientConfig(conn Connection) string {
	return fmt.Sprintf(`client
dev tun
proto %s
remote %s %d
resolv-retry infinite
nobind
persist-key
persist-tun
comp-lzo
verb 3
ca ca.crt
cert client.crt
key client.key
`, conn.Proto, conn.Host, conn.Port)
}

// darwinArgs builds the argument list passed to `sudo` to run openvpn on macOS.
func darwinArgs(conn Connection) []string {
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
		"--ca", "ca.crt",
		"--cert", "client.crt",
		"--key", "client.key",
	}
}
