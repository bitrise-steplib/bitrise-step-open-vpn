package step

import (
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-steputils/v2/export"
	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/pathutil"

	"github.com/bitrise-steplib/bitrise-step-open-vpn/openvpn"
)

const logPathOutputKey = "OPENVPN_LOG_PATH"

// Input is the raw user configuration parsed from the Step inputs.
type Input struct {
	Host      string          `env:"host,required"`
	Port      int             `env:"port,required"`
	Proto     string          `env:"proto,opt[udp,tcp]"`
	CACrt     stepconf.Secret `env:"ca_crt,required"`
	ClientCrt stepconf.Secret `env:"client_crt,required"`
	ClientKey stepconf.Secret `env:"client_key,required"`
}

// Config is the processed and validated configuration the Step runs with.
type Config struct {
	Connection openvpn.Connection
}

// Result holds the outcome of a Step run.
type Result struct {
	LogPath string
}

// OpenVPNStep connects to an OpenVPN server. Dependencies are injected at
// creation time to keep the logic testable.
type OpenVPNStep struct {
	inputParser  stepconf.InputParser
	logger       log.Logger
	pathProvider pathutil.PathProvider
	exporter     export.Exporter
	connector    openvpn.Connector
}

// NewOpenVPNStep creates an OpenVPNStep with its dependencies injected.
func NewOpenVPNStep(
	inputParser stepconf.InputParser,
	logger log.Logger,
	pathProvider pathutil.PathProvider,
	exporter export.Exporter,
	connector openvpn.Connector,
) OpenVPNStep {
	return OpenVPNStep{
		inputParser:  inputParser,
		logger:       logger,
		pathProvider: pathProvider,
		exporter:     exporter,
		connector:    connector,
	}
}

// ProcessConfig parses the inputs, base64-decodes the certificates and key,
// and returns the validated run configuration.
func (s OpenVPNStep) ProcessConfig() (Config, error) {
	var input Input
	if err := s.inputParser.Parse(&input); err != nil {
		return Config{}, fmt.Errorf("parse inputs: %w", err)
	}

	stepconf.Print(input)
	s.logger.Println()

	caCert, err := decodeBase64(input.CACrt)
	if err != nil {
		return Config{}, fmt.Errorf("decode CA certificate: %w", err)
	}
	clientCert, err := decodeBase64(input.ClientCrt)
	if err != nil {
		return Config{}, fmt.Errorf("decode client certificate: %w", err)
	}
	clientKey, err := decodeBase64(input.ClientKey)
	if err != nil {
		return Config{}, fmt.Errorf("decode client key: %w", err)
	}

	return Config{
		Connection: openvpn.Connection{
			Host:       input.Host,
			Port:       input.Port,
			Proto:      input.Proto,
			CACert:     caCert,
			ClientCert: clientCert,
			ClientKey:  clientKey,
		},
	}, nil
}

// InstallDependencies is a no-op: openvpn and net-tools are declared as
// brew/apt_get dependencies in step.yml and installed by the Bitrise CLI.
func (s OpenVPNStep) InstallDependencies() error {
	return nil
}

// Run establishes the VPN connection and returns the path to the OpenVPN log.
// The Result (with the log path) is returned even on failure so the caller can
// still export it.
func (s OpenVPNStep) Run(config Config) (Result, error) {
	logPath, err := s.createLogPath()
	if err != nil {
		return Result{}, err
	}
	result := Result{LogPath: logPath}

	s.logger.Infof("Connecting to OpenVPN server")
	if err := s.connector.Connect(config.Connection, logPath); err != nil {
		return result, err
	}
	s.logger.Donef("Connected")

	return result, nil
}

// ExportOutputs exports the OpenVPN log path so later Steps can read it.
func (s OpenVPNStep) ExportOutputs(result Result) error {
	if result.LogPath == "" {
		return nil
	}
	if err := s.exporter.ExportOutput(logPathOutputKey, result.LogPath); err != nil {
		return fmt.Errorf("export %s: %w", logPathOutputKey, err)
	}
	s.logger.Printf("Log path exported ($%s=%s)", logPathOutputKey, result.LogPath)
	return nil
}

func (s OpenVPNStep) createLogPath() (string, error) {
	tmpDir, err := s.pathProvider.CreateTempDir("open-vpn")
	if err != nil {
		return "", fmt.Errorf("create temp dir for log: %w", err)
	}
	return filepath.Join(tmpDir, "openvpn.log"), nil
}

// decodeBase64 decodes a base64-encoded secret, tolerating surrounding
// whitespace and newlines the same way the `base64 -d` CLI does.
func decodeBase64(value stepconf.Secret) ([]byte, error) {
	cleaned := strings.Join(strings.Fields(string(value)), "")
	return base64.StdEncoding.DecodeString(cleaned)
}
