// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/pkg/errors"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/vault"
)

const (
	OptionTenantID   = "tenant-id"
	OptionClientID   = "client-id"
	OptionDataReport = "mondoo-ms365-datareport"
)

type Ms365Connection struct {
	id                          uint32
	Conf                        *inventory.Config
	asset                       *inventory.Asset
	token                       azcore.TokenCredential
	tenantId                    string
	powershellDataReportFile    string
	ms365PowershellReport       *Microsoft365Report
	ms365PowershellReportLoader sync.Mutex
}

func NewMs365Connection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*Ms365Connection, error) {
	tenantId := conf.Options[OptionTenantID]
	clientId := conf.Options[OptionClientID]
	// we need credentials for ms365. for azure these are optional, we fallback to the AZ cli (if installed)
	if len(conf.Credentials) == 0 || conf.Credentials[0] == nil {
		return nil, errors.New("ms365 provider requires a credentials file, pass path via --certificate-path option")
	}

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
		Conf:                     conf,
		id:                       id,
		asset:                    asset,
		token:                    token,
		tenantId:                 tenantId,
		powershellDataReportFile: conf.Options[OptionDataReport],
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

// NOTE: this is a temporary solution and will be replaced with logic that calls powershell directly and
// hopefully provides more flexibility in the future
func (p *Ms365Connection) GetMs365DataReport() (*Microsoft365Report, error) {
	p.ms365PowershellReportLoader.Lock()
	defer p.ms365PowershellReportLoader.Unlock()

	if p.ms365PowershellReport != nil {
		return p.ms365PowershellReport, nil
	}

	if p.powershellDataReportFile == "" {
		return nil, errors.New("powershell data report file not not provided")
	}

	if _, err := os.Stat(p.powershellDataReportFile); os.IsNotExist(err) {
		return nil, errors.New("could not load powershell data report from: " + p.powershellDataReportFile)
	}

	// get path from transport option
	data, err := os.ReadFile(p.powershellDataReportFile)
	if err != nil {
		return nil, err
	}

	p.ms365PowershellReport = &Microsoft365Report{}
	err = json.Unmarshal(data, p.ms365PowershellReport)
	if err != nil {
		return nil, err
	}
	return p.ms365PowershellReport, nil
}
