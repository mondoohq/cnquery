// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"fmt"
	"runtime"

	azcore "github.com/Azure/azure-sdk-for-go/sdk/azcore"
	errors "github.com/cockroachdb/errors"
	msgrapgh_org "github.com/microsoftgraph/msgraph-sdk-go/organization"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/azauth"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/vault"
	"go.mondoo.com/cnquery/v11/providers/os/connection/local"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v11/providers/os/resources/powershell"
)

const (
	OptionTenantID      = "tenant-id"
	OptionClientID      = "client-id"
	OptionOIDCToken     = "oidc-token"
	OptionOrganization  = "organization"
	OptionSharepointUrl = "sharepoint-url"
)

type Ms365Connection struct {
	plugin.Connection
	Conf          *inventory.Config
	asset         *inventory.Asset
	token         azcore.TokenCredential
	tenantId      string
	clientId      string
	organization  string
	sharepointUrl string
}

func NewMs365Connection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*Ms365Connection, error) {
	tenantId := conf.Options[OptionTenantID]
	clientId := conf.Options[OptionClientID]
	oidcToken := conf.Options[OptionOIDCToken]

	organization := conf.Options[OptionOrganization]
	sharepointUrl := conf.Options[OptionSharepointUrl]
	var cred *vault.Credential
	if len(conf.Credentials) != 0 {
		cred = conf.Credentials[0]
	}

	if len(tenantId) == 0 {
		return nil, errors.New("ms365 backend requires a tenant-id")
	}
	token, err := azauth.GetTokenFromCredential(cred, tenantId, clientId, oidcToken)
	if err != nil {
		return nil, errors.Wrap(err, "cannot fetch credentials for microsoft provider")
	}

	// test connection
	client, err := graphClient(token)
	if err != nil {
		return nil, errors.Wrap(err, "authentication failed")
	}
	_, err = client.Organization().Get(context.Background(), &msgrapgh_org.OrganizationRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, errors.Wrap(err, "authentication failed")
	}
	return &Ms365Connection{
		Connection:    plugin.NewConnection(id, asset),
		Conf:          conf,
		asset:         asset,
		token:         token,
		tenantId:      tenantId,
		clientId:      clientId,
		organization:  organization,
		sharepointUrl: sharepointUrl,
	}, nil
}

func (h *Ms365Connection) Name() string {
	return "ms365"
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

func (p *Ms365Connection) ClientId() string {
	return p.clientId
}

func (p *Ms365Connection) PlatformId() string {
	return "//platformid.api.mondoo.app/runtime/ms365/tenant/" + p.tenantId
}

func (p *Ms365Connection) SharepointUrl() string {
	return p.sharepointUrl
}

func (p *Ms365Connection) Organization() string {
	return p.organization
}

// indicates if a certificate credential is provided
func (p *Ms365Connection) IsCertProvided() bool {
	return len(p.Conf.Credentials) > 0 && p.Conf.Credentials[0].Type == vault.CredentialType_pkcs12
}

func (p *Ms365Connection) RunPowershellScript(script string) (*shared.Command, error) {
	var encodedCmd string
	if runtime.GOOS == "windows" {
		encodedCmd = powershell.Encode(script)
	} else {
		encodedCmd = powershell.EncodeUnix(script)
	}
	return p.RunCmd(encodedCmd)
}

func (p *Ms365Connection) RunCmd(cmd string) (*shared.Command, error) {
	cmdR := local.CommandRunner{}
	if runtime.GOOS == "windows" {
		cmdR.Shell = []string{"powershell", "-c"}
	} else {
		cmdR.Shell = []string{"sh", "-c"}
	}
	return cmdR.Exec(cmd, []string{})
}

func (p *Ms365Connection) CheckPowershellAvailable() (bool, error) {
	if runtime.GOOS == "windows" {
		// assume powershell is always present on windows
		return true, nil
	}
	// for unix, we need to check if pwsh is available
	cmd := "which pwsh"
	res, err := p.RunCmd(cmd)
	if err != nil {
		return false, err
	}

	return res.ExitStatus == 0, nil
}

func (p *Ms365Connection) CheckAndRunPowershellScript(script string) (*shared.Command, error) {
	pwshAvailable, err := p.CheckPowershellAvailable()
	if err != nil {
		return nil, err
	}
	if !pwshAvailable {
		return nil, fmt.Errorf("powershell is not available")
	}
	return p.RunPowershellScript(script)
}
