// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"sync"

	vaultapi "github.com/hashicorp/vault/api"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/vault/connection"
)

// mqlVaultInternal caches the health payload. Version, cluster name and cluster
// ID all come from the same call, so a query touching more than one of them
// costs a single request rather than one per field.
type mqlVaultInternal struct {
	healthOnce sync.Once
	health     *vaultapi.HealthResponse
	healthErr  error
}

func (r *mqlVault) id() (string, error) {
	conn, err := vaultConn(r.MqlRuntime)
	if err != nil {
		return "", err
	}
	return connection.NewVaultServerIdentifier(conn.Host(), conn.Namespace()), nil
}

// vaultConn pulls the connection off the runtime.
func vaultConn(runtime *plugin.Runtime) (*connection.VaultConnection, error) {
	conn, ok := runtime.Connection.(*connection.VaultConnection)
	if !ok {
		return nil, errors.New("no Vault connection on the runtime")
	}
	return conn, nil
}

// vaultClient returns the authenticated API client.
func vaultClient(runtime *plugin.Runtime) (*vaultapi.Client, error) {
	conn, err := vaultConn(runtime)
	if err != nil {
		return nil, err
	}
	return conn.Client(), nil
}

// fetchHealth reads sys/health once per resource instance.
func (r *mqlVault) fetchHealth() (*vaultapi.HealthResponse, error) {
	r.healthOnce.Do(func() {
		client, err := vaultClient(r.MqlRuntime)
		if err != nil {
			r.healthErr = err
			return
		}
		r.health, r.healthErr = client.Sys().Health()
	})
	return r.health, r.healthErr
}

func (r *mqlVault) version() (string, error) {
	health, err := r.fetchHealth()
	if err != nil {
		return "", err
	}
	return health.Version, nil
}

func (r *mqlVault) clusterName() (string, error) {
	health, err := r.fetchHealth()
	if err != nil {
		return "", err
	}
	return health.ClusterName, nil
}

func (r *mqlVault) clusterId() (string, error) {
	health, err := r.fetchHealth()
	if err != nil {
		return "", err
	}
	return health.ClusterID, nil
}

func (r *mqlVault) initialized() (bool, error) {
	health, err := r.fetchHealth()
	if err != nil {
		return false, err
	}
	return health.Initialized, nil
}

func (r *mqlVault) namespacePath() (string, error) {
	conn, err := vaultConn(r.MqlRuntime)
	if err != nil {
		return "", err
	}
	return conn.Namespace(), nil
}

func (r *mqlVault) seal() (*mqlVaultSealStatus, error) {
	client, err := vaultClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	status, err := client.Sys().SealStatus()
	if err != nil {
		return nil, err
	}
	if status == nil {
		// A nil status with no error means the server answered but told us
		// nothing. Report the field as null rather than inventing a seal type,
		// which would read as an unsealed Shamir server.
		r.Seal.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	res, err := CreateResource(r.MqlRuntime, "vault.sealStatus", map[string]*llx.RawData{
		"__id":         llx.StringData("sealStatus"),
		"type":         llx.StringData(status.Type),
		"sealed":       llx.BoolData(status.Sealed),
		"initialized":  llx.BoolData(status.Initialized),
		"autoUnseal":   llx.BoolData(isAutoUnseal(status.Type)),
		"threshold":    llx.IntData(int64(status.T)),
		"shares":       llx.IntData(int64(status.N)),
		"progress":     llx.IntData(int64(status.Progress)),
		"migration":    llx.BoolData(status.Migration),
		"recoverySeal": llx.BoolData(status.RecoverySeal),
		"storageType":  llx.StringData(status.StorageType),
		"version":      llx.StringData(status.Version),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlVaultSealStatus), nil
}

// isAutoUnseal reports whether the seal delegates the root key to an external
// key management service. Shamir is the only seal that requires operators to
// supply key shares, so everything else unseals on its own. An empty type is
// reported as Shamir rather than auto-unseal, because guessing the safer answer
// here would hide a server that needs manual unsealing.
func isAutoUnseal(sealType string) bool {
	return sealType != "" && sealType != "shamir"
}
