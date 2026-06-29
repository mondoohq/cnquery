// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package vsimulator

import (
	"net/http/httptest"
	"net/url"

	"github.com/vmware/govmomi/simulator"
	// Registers the vAPI/REST simulator endpoints (tags, content library,
	// certificate management) so connections exercising the REST client work
	// against the simulator, not just the SOAP/vim25 API.
	_ "github.com/vmware/govmomi/vapi/simulator"
)

const (
	Username = "my-username"
	Password = "my-password"
)

// start vsphere simulator
// see https://pkg.go.dev/github.com/vmware/govmomi/simulator#pkg-overview
func New() (*VsphereSimulator, error) {
	model := simulator.VPX()
	defer model.Remove()
	err := model.Create()
	if err != nil {
		return nil, err
	}

	model.Service.Listen = &url.URL{
		User: url.UserPassword(Username, Password),
	}

	// use the httptest tls generation instead of writing our own
	tlsSrv := httptest.NewTLSServer(nil)
	tls := tlsSrv.TLS

	model.Service.TLS = tls
	// Wire up the registered vAPI/REST endpoints (see the blank import above).
	model.Service.RegisterEndpoints = true
	s := model.Service.NewServer()

	return &VsphereSimulator{
		TlsSrv: tlsSrv,
		Server: s,
	}, nil
}

type VsphereSimulator struct {
	TlsSrv *httptest.Server
	Server *simulator.Server
}

func (vs *VsphereSimulator) Close() {
	vs.TlsSrv.Close()
	vs.Server.Close()
}
