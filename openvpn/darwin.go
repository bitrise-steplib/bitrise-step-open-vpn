package openvpn

import (
	"fmt"
	"strconv"
	"time"

	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/fileutil"
	"github.com/bitrise-io/go-utils/v2/log"
)

// darwinConnector writes the keys next to the working directory and starts
// openvpn as a background process via sudo.
type darwinConnector struct {
	cmdFactory  command.Factory
	fileManager fileutil.FileManager
	logger      log.Logger
}

func (c darwinConnector) Connect(conn Connection, logPath string) error {
	c.logger.Printf("Configuring for macOS")

	if err := writeFiles(c.fileManager, map[string][]byte{
		"ca.crt":     conn.CACert,
		"client.crt": conn.ClientCert,
		"client.key": conn.ClientKey,
	}); err != nil {
		return err
	}

	logFile, err := createLogFile(logPath)
	if err != nil {
		return err
	}
	defer func() { _ = logFile.Close() }()

	c.logger.Printf("Starting openvpn")
	cmd := c.cmdFactory.Create("sudo", darwinArgs(conn), &command.Opts{
		Stdout: logFile,
		Stderr: logFile,
	})
	if err := cmd.Start(); err != nil {
		return failWithLog(c.logger, logPath, fmt.Errorf("start openvpn: %w", err))
	}

	c.logger.Printf("Checking connection status")
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()

	select {
	case err := <-exited:
		return failWithLog(c.logger, logPath, fmt.Errorf("openvpn process exited: %w", err))
	case <-time.After(statusCheckDelay):
		// Still running after the wait window: connection is considered up.
		// The process is intentionally left running for subsequent Steps.
		return nil
	}
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
