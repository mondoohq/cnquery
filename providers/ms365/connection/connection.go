// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"runtime"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/pkg/errors"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/vault"
	"go.mondoo.com/cnquery/v9/providers/os/connection"
	"go.mondoo.com/cnquery/v9/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v9/providers/os/resources/powershell"
)

const (
	OptionTenantID     = "tenant-id"
	OptionClientID     = "client-id"
	OptionOrganization = "organization"
)

type Ms365Connection struct {
	id           uint32
	Conf         *inventory.Config
	asset        *inventory.Asset
	token        azcore.TokenCredential
	tenantId     string
	clientId     string
	organization string
	// TODO: move those to MQL resources caching once it makes sense to do so
	exchangeReport     *ExchangeOnlineReport
	exchangeReportLock sync.Mutex
	teamsReport        *MsTeamsReport
	teamsReportLock    sync.Mutex
}

func NewMs365Connection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*Ms365Connection, error) {
	tenantId := conf.Options[OptionTenantID]
	clientId := conf.Options[OptionClientID]
	organization := conf.Options[OptionOrganization]
	var cred *vault.Credential
	if len(conf.Credentials) != 0 {
		cred = conf.Credentials[0]
	}

	if len(tenantId) == 0 {
		return nil, errors.New("ms365 backend requires a tenant-id")
	}
	token, err := getTokenCredential(cred, tenantId, clientId)
	if err != nil {
		return nil, errors.Wrap(err, "cannot fetch credentials for microsoft provider")
	}
	return &Ms365Connection{
		Conf:         conf,
		id:           id,
		asset:        asset,
		token:        token,
		tenantId:     tenantId,
		clientId:     clientId,
		organization: organization,
	}, nil
}

func (h *Ms365Connection) Name() string {
	return "ms365"
}

func (h *Ms365Connection) ID() uint32 {
	return h.id
}

func (p *Ms365Connection) Asset() *inventory.Asset {
	return p.asset
}

func (p *Ms365Connection) Token() azcore.TokenCredential {
	return p.token
}

func (p *Ms365Connection) TenantId() string {
	return p.tenantId
}

func (p *Ms365Connection) PlatformId() string {
	return "//platformid.api.mondoo.app/runtime/ms365/tenant/" + p.tenantId
}

// TODO: use LocalConnection here for running cmds?
func (p *Ms365Connection) runPowershellScript(script string) (*shared.Command, error) {
	cmd := connection.CommandRunner{}
	var encodedCmd string
	if runtime.GOOS == "windows" {
		cmd.Shell = []string{"powershell", "-c"}
		encodedCmd = powershell.Encode(script)
	} else {
		cmd.Shell = []string{"sh", "-c"}
		encodedCmd = powershell.EncodeUnix(script)
	}
	return cmd.Exec(encodedCmd, []string{})
}
