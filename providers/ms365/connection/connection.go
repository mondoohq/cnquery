// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"fmt"
	"io"
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
	var encodedCmd string
	if runtime.GOOS == "windows" {
		encodedCmd = powershell.Encode(script)
	} else {
		encodedCmd = powershell.EncodeUnix(script)
	}
	return p.runCmd(encodedCmd)
}

func (p *Ms365Connection) runCmd(cmd string) (*shared.Command, error) {
	cmdR := connection.CommandRunner{}
	if runtime.GOOS == "windows" {
		cmdR.Shell = []string{"powershell", "-c"}
	} else {
		cmdR.Shell = []string{"sh", "-c"}
	}
	return cmdR.Exec(cmd, []string{})
}

func (p *Ms365Connection) checkPowershellAvailable() (bool, error) {
	if runtime.GOOS == "windows" {
		// assume powershell is always present on windows
		return true, nil
	}
	// for unix, we need to check if pwsh is available
	cmd := "which pwsh"
	res, err := p.runCmd(cmd)
	if err != nil {
		return false, err
	}
	if res.ExitStatus != 0 {
		data, err := io.ReadAll(res.Stderr)
		if err != nil {
			return false, err
		}
		return false, fmt.Errorf("failed to determine if powershell is available: %s", string(data))
	}

	return res.ExitStatus == 0, nil
}

func (p *Ms365Connection) checkAndRunPowershellScript(script string) (*shared.Command, error) {
	pwshAvailable, err := p.checkPowershellAvailable()
	if err != nil {
		return nil, err
	}
	if !pwshAvailable {
		return nil, fmt.Errorf("powershell is not available")
	}
	return p.runPowershellScript(script)
}
