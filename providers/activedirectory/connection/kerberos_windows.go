// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build windows

package connection

import (
	"fmt"

	"github.com/go-ldap/ldap/v3"
	"github.com/go-ldap/ldap/v3/gssapi"
)

func newImplicitKerberosClient() (ldap.GSSAPIClient, func() error, error) {
	gssClient, err := gssapi.NewSSPIClient()
	if err != nil {
		return nil, nil, fmt.Errorf("windows current-session Kerberos client: %w", err)
	}
	return gssClient, gssClient.Close, nil
}
