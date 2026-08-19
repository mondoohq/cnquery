// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/minio/connection"
)

func (a *mqlMinioKmsKey) id() (string, error) {
	if a.Name.Data == "" {
		return "", errors.New("minio.kmsKey requires a name")
	}
	return "kmsKey/" + a.Name.Data, nil
}

// initMinioKmsKey resolves a key by name.
//
// The key management service answers with HTTP 200 whatever the outcome and
// reports what went wrong in the per-operation error fields, so a key that does
// not exist arrives as a successful response carrying an encryption error
// rather than as a failed request. `healthy` is therefore derived from those
// fields rather than from the request having succeeded.
func initMinioKmsKey(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}
	nameArg, ok := args["name"]
	if !ok {
		return args, nil, nil
	}
	name, ok := nameArg.Value.(string)
	if !ok || name == "" {
		return nil, nil, errors.New("minio.kmsKey requires a name")
	}

	conn := runtime.Connection.(*connection.MinioConnection)
	status, err := conn.Admin().GetKeyStatus(context.Background(), name)
	if err != nil {
		return nil, nil, err
	}
	if status == nil {
		return nil, nil, errors.New("minio.kmsKey with name " + name + " reported no status")
	}

	args["name"] = llx.StringData(status.KeyID)
	args["encryptionError"] = llx.StringData(status.EncryptionErr)
	args["decryptionError"] = llx.StringData(status.DecryptionErr)
	args["version"] = llx.StringData(status.KeyVersion)
	args["healthy"] = llx.BoolData(status.EncryptionErr == "" && status.DecryptionErr == "")
	return args, nil, nil
}
