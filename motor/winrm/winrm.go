package winrm

import (
	"bytes"
	"errors"
	"time"

	"github.com/masterzen/winrm"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/types"
)

func VerifyConfig(endpoint *types.Endpoint) (*winrm.Endpoint, error) {
	if endpoint.Backend != "winrm" {
		return nil, errors.New("only winrm backend for winrm transport supported")
	}

	p, err := endpoint.IntPort()
	if err != nil {
		return nil, errors.New("port is not a valid number " + endpoint.Port)
	}

	winrmEndpoint := &winrm.Endpoint{
		Host:     endpoint.Host,
		Port:     p,
		Insecure: false,
		HTTPS:    false,
		Timeout:  time.Duration(0),
	}

	return winrmEndpoint, nil
}

func DefaultConfig(endpoint *winrm.Endpoint) *winrm.Endpoint {
	// use default port if port is 0
	if endpoint.Port <= 0 {
		endpoint.Port = 5985
	}
	return endpoint
}

// New creates a winrm client and establishes a connection to verify the connection
func New(endpoint *types.Endpoint) (*WinrmTransport, error) {

	// ensure all required configs are set
	winrmEndpoint, err := VerifyConfig(endpoint)
	if err != nil {
		return nil, err
	}

	// set default config if required
	winrmEndpoint = DefaultConfig(winrmEndpoint)

	params := winrm.DefaultParameters
	params.TransportDecorator = func() winrm.Transporter { return &winrm.ClientNTLM{} }

	client, err := winrm.NewClientWithParameters(winrmEndpoint, endpoint.User, endpoint.Password, params)
	if err != nil {
		return nil, err
	}

	// test connection
	log.Debug().Msg("winrm> connecting to remote shell via WinRM")
	shell, err := client.CreateShell()
	if err != nil {
		return nil, err
	}

	err = shell.Close()
	if err != nil {
		return nil, err
	}

	log.Info().Msg("winrm> connection established")
	return &WinrmTransport{Endpoint: winrmEndpoint, Client: client}, nil
}

type WinrmTransport struct {
	Endpoint *winrm.Endpoint
	Client   *winrm.Client
}

func (t *WinrmTransport) RunCommand(command string) (*types.Command, error) {
	log.Debug().Str("command", command).Str("transport", "ssh").Msg("winrm> run command")

	stdoutBuffer := &bytes.Buffer{}
	stderrBuffer := &bytes.Buffer{}

	exitCode, err := t.Client.Run(command, stdoutBuffer, stderrBuffer)
	if err != nil {
		log.Error().Err(err).Str("command", command).Msg("could not execute winrm command")
		return nil, err
	}

	log.Debug().Str("stdout", stdoutBuffer.String()).Str("stderr", stderrBuffer.String()).Int("exitcode", exitCode).Msg("winrm command executed")

	mcmd := &types.Command{
		Command:    command,
		Stdout:     stdoutBuffer,
		Stderr:     stderrBuffer,
		ExitStatus: exitCode,
	}

	return mcmd, nil
}

func (t *WinrmTransport) File(path string) (types.File, error) {
	log.Debug().Str("path", path).Str("transport", "ssh").Msg("winrm> fetch file")
	return nil, errors.New("not implemented")
}

func (t *WinrmTransport) Close() {

}
