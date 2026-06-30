package main

import (
	"os"
	"runtime"

	"github.com/bitrise-io/go-steputils/v2/export"
	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/fileutil"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/pathutil"

	"github.com/bitrise-steplib/bitrise-step-open-vpn/openvpn"
	"github.com/bitrise-steplib/bitrise-step-open-vpn/step"
)

func main() {
	os.Exit(run())
}

func run() int {
	logger := log.NewLogger()
	openVPNStep, err := createStep(logger)
	if err != nil {
		logger.Errorf(err.Error())
		return 1
	}

	config, err := openVPNStep.ProcessConfig()
	if err != nil {
		logger.Errorf(err.Error())
		return 1
	}

	if err := openVPNStep.InstallDependencies(); err != nil {
		logger.Errorf(err.Error())
		return 1
	}

	result, runErr := openVPNStep.Run(config)

	// The log path output is exported even when the run fails, so the log
	// file stays discoverable for troubleshooting later Steps.
	if err := openVPNStep.ExportOutputs(result); err != nil {
		logger.Errorf(err.Error())
		return 1
	}

	if runErr != nil {
		logger.Errorf(runErr.Error())
		return 1
	}

	return 0
}

func createStep(logger log.Logger) (step.OpenVPNStep, error) {
	envRepository := env.NewRepository()
	inputParser := stepconf.NewInputParser(envRepository)
	cmdFactory := command.NewFactory(envRepository)
	fileManager := fileutil.NewFileManager()
	pathProvider := pathutil.NewPathProvider()
	exporter := export.NewExporter(cmdFactory, fileManager)
	connector, err := openvpn.NewConnector(runtime.GOOS, cmdFactory, fileManager, pathProvider, logger)
	if err != nil {
		return step.OpenVPNStep{}, err
	}

	return step.NewOpenVPNStep(inputParser, logger, pathProvider, exporter, connector), nil
}
