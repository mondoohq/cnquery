package winrm

import (
	"bytes"
	"errors"
	"os"
	"time"

	"github.com/masterzen/winrm"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/winrm/cat"
)

func VerifyConfig(endpoint *transports.TransportConfig) (*winrm.Endpoint, error) {
	if endpoint.Backend != transports.TransportBackend_CONNECTION_WINRM {
		return nil, errors.New("only winrm backend for winrm transport supported")
	}

	p, err := endpoint.IntPort()
	if err != nil {
		return nil, errors.New("port is not a valid number " + endpoint.Port)
	}

	winrmEndpoint := &winrm.Endpoint{
		Host:     endpoint.Host,
		Port:     p,
		Insecure: endpoint.Insecure,
		HTTPS:    true,
		Timeout:  time.Duration(0),
	}

	return winrmEndpoint, nil
}

func DefaultConfig(endpoint *winrm.Endpoint) *winrm.Endpoint {
	// use default port if port is 0
	if endpoint.Port <= 0 {
		endpoint.Port = 5986
	}

	if endpoint.Port == 5985 {
		log.Warn().Msg("winrm port 5985 is using http communication instead of https, passwords are not encrypted")
		endpoint.HTTPS = false
	}

	if os.Getenv("WINRM_DISABLE_HTTPS") == "true" {
		log.Warn().Msg("WINRM_DISABLE_HTTPS is set, winrm is using http communication instead of https, passwords are not encrypted")
		endpoint.HTTPS = false
	}

	return endpoint
}

// New creates a winrm client and establishes a connection to verify the connection
func New(endpoint *transports.TransportConfig) (*WinrmTransport, error) {

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

	log.Debug().Msg("winrm> connection established")
	return &WinrmTransport{
		Endpoint: winrmEndpoint,
		Client:   client,
		kind:     endpoint.Kind,
		runtime:  endpoint.Runtime,
	}, nil
}

type WinrmTransport struct {
	Endpoint *winrm.Endpoint
	Client   *winrm.Client
	kind     transports.Kind
	runtime  string
	fs       afero.Fs
}

func (t *WinrmTransport) RunCommand(command string) (*transports.Command, error) {
	log.Debug().Str("command", command).Str("transport", "winrm").Msg("winrm> run command")

	stdoutBuffer := &bytes.Buffer{}
	stderrBuffer := &bytes.Buffer{}

	exitCode, err := t.Client.Run(command, stdoutBuffer, stderrBuffer)
	if err != nil {
		log.Error().Err(err).Str("command", command).Msg("could not execute winrm command")
		return nil, err
	}

	// log.Debug().Int("exitcode", exitCode).Msg("winrm command executed")
	// fmt.Println(stdoutBuffer.String())
	// fmt.Println(stderrBuffer.String())

	mcmd := &transports.Command{
		Command:    command,
		Stdout:     stdoutBuffer,
		Stderr:     stderrBuffer,
		ExitStatus: exitCode,
	}

	return mcmd, nil
}

func (t *WinrmTransport) FileInfo(path string) (transports.FileInfoDetails, error) {
	fs := t.FS()
	afs := &afero.Afero{Fs: fs}
	stat, err := afs.Stat(path)
	if err != nil {
		return transports.FileInfoDetails{}, err
	}

	uid := int64(-1)
	gid := int64(-1)
	mode := stat.Mode()

	return transports.FileInfoDetails{
		Mode: transports.FileModeDetails{mode},
		Size: stat.Size(),
		Uid:  uid,
		Gid:  gid,
	}, nil
}

func (t *WinrmTransport) FS() afero.Fs {
	if t.fs == nil {
		t.fs = cat.New(t)
	}
	return t.fs
}

func (t *WinrmTransport) Close() {
	// nothing to do yet
}

func (t *WinrmTransport) Capabilities() transports.Capabilities {
	return transports.Capabilities{
		transports.Cabability_RunCommand,
		transports.Cabability_File,
	}
}

func (t *WinrmTransport) Kind() transports.Kind {
	return t.kind
}

func (t *WinrmTransport) Runtime() string {
	return t.runtime
}
