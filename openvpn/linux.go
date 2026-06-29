package openvpn

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/fileutil"
	"github.com/bitrise-io/go-utils/v2/log"
)

// linuxConfigDir is where the OpenVPN service expects its config and keys.
const linuxConfigDir = "/etc/openvpn"

// linuxConnector configures /etc/openvpn and starts the openvpn service.
type linuxConnector struct {
	cmdFactory  command.Factory
	fileManager fileutil.FileManager
	logger      log.Logger
}

func (c linuxConnector) Connect(conn Connection, logPath string) error {
	c.logger.Printf("Configuring for Linux")

	if err := writeFiles(c.fileManager, map[string][]byte{
		filepath.Join(linuxConfigDir, "ca.crt"):     conn.CACert,
		filepath.Join(linuxConfigDir, "client.crt"): conn.ClientCert,
		filepath.Join(linuxConfigDir, "client.key"): conn.ClientKey,
	}); err != nil {
		return err
	}

	configPath := filepath.Join(linuxConfigDir, "client.conf")
	if err := c.fileManager.Write(configPath, RenderClientConfig(conn), 0644); err != nil {
		return fmt.Errorf("write %s: %w", configPath, err)
	}

	logFile, err := createLogFile(logPath)
	if err != nil {
		return err
	}
	defer func() { _ = logFile.Close() }()

	c.logger.Printf("Starting openvpn service")
	cmd := c.cmdFactory.Create("service", []string{"openvpn", "start", "client"}, &command.Opts{
		Stdout: logFile,
		Stderr: logFile,
	})
	if err := cmd.Run(); err != nil {
		return failWithLog(c.logger, logPath, fmt.Errorf("start openvpn service: %w", err))
	}

	c.logger.Printf("Checking connection status")
	time.Sleep(statusCheckDelay)
	if !c.tunnelIsUp() {
		return failWithLog(c.logger, logPath, fmt.Errorf("no OpenVPN tunnel (tun0) found"))
	}

	return nil
}

// tunnelIsUp reports whether a tun0 interface exists.
func (c linuxConnector) tunnelIsUp() bool {
	out, err := c.cmdFactory.Create("ifconfig", nil, nil).RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(out, "tun0")
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
