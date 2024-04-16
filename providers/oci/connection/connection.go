// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"errors"

	"github.com/oracle/oci-go-sdk/v65/common"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/vault"
)

type OciConnection struct {
	plugin.Connection
	Conf        *inventory.Config
	asset       *inventory.Asset
	config      common.ConfigurationProvider
	tenancyOcid string
}

func NewOciConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*OciConnection, error) {
	conn := &OciConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
	}

	// initialize your connection here
	var configProvider common.ConfigurationProvider
	// if we have passed in credentials, assume we want to pass in all values explicitly.
	if len(conf.Credentials) > 0 {
		fingerprint := conf.Options["fingerprint"]
		if fingerprint == "" {
			return nil, errors.New("OCI provider fingerprint value cannot be empty")
		}
		tenancyOcid := conf.Options["tenancy"]
		if tenancyOcid == "" {
			return nil, errors.New("OCI provider tenancy value cannot be empty")
		}
		userOcid := conf.Options["user"]
		if userOcid == "" {
			return nil, errors.New("OCI provider user value cannot be empty")
		}
		region := conf.Options["region"]
		if region == "" {
			return nil, errors.New("OCI provider region value cannot be empty")
		}

		pkey := conf.Credentials[0]
		if pkey.Type != vault.CredentialType_private_key {
			return nil, errors.New("OCI provider does not support credential type: " + pkey.Type.String())
		}
		configProvider = common.NewRawConfigurationProvider(tenancyOcid, userOcid, region, fingerprint, string(pkey.Secret), nil)
	} else {
		configProvider = common.DefaultConfigProvider()
	}
	tenancyOcid, err := configProvider.TenancyOCID()
	if err != nil {
		return nil, err
	}

	conn.config = configProvider
	conn.tenancyOcid = tenancyOcid

	return conn, nil
}

func (s *OciConnection) Name() string {
	return "oci"
}

func (s *OciConnection) Asset() *inventory.Asset {
	return s.asset
}
