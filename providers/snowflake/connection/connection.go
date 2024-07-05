// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"crypto/rsa"
	"encoding/pem"
	"errors"

	"github.com/Snowflake-Labs/terraform-provider-snowflake/pkg/sdk"
	"github.com/snowflakedb/gosnowflake"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/vault"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
	"golang.org/x/crypto/ssh"
)

type SnowflakeConnection struct {
	plugin.Connection
	Conf  *inventory.Config
	asset *inventory.Asset
	// Custom connection fields
	client *sdk.Client
}

func NewSnowflakeConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*SnowflakeConnection, error) {
	conn := &SnowflakeConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
	}

	// initialize your connection here

	if len(conf.Credentials) == 0 {
		return nil, status.Error(codes.InvalidArgument, "missing credentials for snowflake connection")
	}

	if conf.Options == nil {
		conf.Options = make(map[string]string)
	}

	cfg := &gosnowflake.Config{
		Account: conf.Options["account"],
		Region:  conf.Options["region"],
		Role:    conf.Options["role"],
	}

	for i := range conf.Credentials {
		cred := conf.Credentials[i]
		switch cred.Type {
		case vault.CredentialType_password:
			cfg.User = cred.User
			cfg.Password = string(cred.Secret)
			cfg.Authenticator = gosnowflake.AuthTypeSnowflake
		case vault.CredentialType_private_key:
			cfg.User = cred.User

			// snowflake requires a RSA private key in PEM format
			key, err := parsePrivateKey(cred.Secret, []byte(cred.Password))
			if err != nil {
				return nil, err
			}
			cfg.PrivateKey = key
			cfg.Authenticator = gosnowflake.AuthTypeJwt
		}
	}

	client, err := sdk.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	conn.client = client

	return conn, nil
}

func parsePrivateKey(privateKeyBytes []byte, passphrase []byte) (*rsa.PrivateKey, error) {
	privateKeyBlock, _ := pem.Decode(privateKeyBytes)
	if privateKeyBlock == nil {
		return nil, errors.New("could not decode private key")
	}

	var privateKey interface{}
	var err error
	if privateKeyBlock.Type == "ENCRYPTED PRIVATE KEY" {
		if len(passphrase) == 0 {
			return nil, errors.New("private key is encrypted, but no passphrase provided")
		}

		privateKey, err = ssh.ParseRawPrivateKeyWithPassphrase(privateKeyBlock.Bytes, passphrase)
		if err != nil {
			return nil, errors.New("could not parse encrypted private key " + err.Error())
		}
	} else {
		privateKey, err = ssh.ParseRawPrivateKey(privateKeyBytes)
		if err != nil {
			return nil, errors.New("could not parse private key err " + err.Error())
		}
	}

	rsaPrivateKey, ok := privateKey.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("snowflake requires a RSA private key in PEM format")
	}
	return rsaPrivateKey, nil
}

func (c *SnowflakeConnection) Name() string {
	return "snowflake"
}

func (c *SnowflakeConnection) Asset() *inventory.Asset {
	return c.asset
}

func (c *SnowflakeConnection) Client() *sdk.Client {
	return c.client
}
