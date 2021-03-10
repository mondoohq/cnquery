package k8s

import (
	"errors"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/cosmo/resources"
	api "go.mondoo.io/mondoo/cosmo/resources"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/fsutil"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func New(tc *transports.TransportConfig) (*Transport, error) {
	if tc.Backend != transports.TransportBackend_CONNECTION_K8S {
		return nil, errors.New("backend is not supported for k8s transport")
	}

	// TODO: this cf dependency must go
	var cf *genericclioptions.ConfigFlags
	cf = genericclioptions.NewConfigFlags(true)

	// initialize api client
	d, err := api.NewDiscovery(cf)
	if err != nil {
		return nil, err
	}
	log.Debug().Msg("loaded kubeconfig successfully")

	return &Transport{
		d:    d,
		opts: tc.Options,
	}, nil
}

type Transport struct {
	d    *resources.Discovery
	opts map[string]string
}

func (t *Transport) RunCommand(command string) (*transports.Command, error) {
	return nil, errors.New("k8s does not implement RunCommand")
}

func (t *Transport) FileInfo(path string) (transports.FileInfoDetails, error) {
	return transports.FileInfoDetails{}, errors.New("k8s does not implement FileInfo")
}

func (t *Transport) FS() afero.Fs {
	return &fsutil.NoFs{}
}

func (t *Transport) Close() {}

func (t *Transport) Capabilities() transports.Capabilities {
	return transports.Capabilities{
		transports.Capability_Gcp,
	}
}

func (t *Transport) Options() map[string]string {
	return t.opts
}

func (t *Transport) Kind() transports.Kind {
	return transports.Kind_KIND_API
}

func (t *Transport) Runtime() string {
	return transports.RUNTIME_AWS
}
