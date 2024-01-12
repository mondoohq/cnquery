// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"

	"github.com/pkg/errors"

	"go.mondoo.com/cnquery/v10/providers-sdk/v1/vault"
)

func getTokenCredential(credential *vault.Credential, tenantId, clientId string) (azcore.TokenCredential, error) {
	var azCred azcore.TokenCredential
	var err error
	if credential == nil {
		return nil, errors.New("no credentials provided")
	} else {
		// we only support private key authentication for ms 365
		switch credential.Type {
		case vault.CredentialType_pkcs12:
			certs, privateKey, err := azidentity.ParseCertificates(credential.Secret, []byte(credential.Password))
			if err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("could not parse provided certificate at %s", credential.PrivateKeyPath))
			}
			azCred, err = azidentity.NewClientCertificateCredential(tenantId, clientId, certs, privateKey, &azidentity.ClientCertificateCredentialOptions{})
			if err != nil {
				return nil, errors.Wrap(err, "error creating credentials from a certificate")
			}
		case vault.CredentialType_password:
			azCred, err = azidentity.NewClientSecretCredential(tenantId, clientId, string(credential.Secret), &azidentity.ClientSecretCredentialOptions{})
			if err != nil {
				return nil, errors.Wrap(err, "error creating credentials from a secret")
			}
		default:
			return nil, errors.New("invalid secret configuration for microsoft transport: " + credential.Type.String())
		}
	}
	return azCred, nil
}
