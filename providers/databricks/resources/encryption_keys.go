// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/types"
)

func (r *mqlDatabricks) customerManagedKeys() ([]any, error) {
	acc, err := accountClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	keys, err := acc.EncryptionKeys.List(context.Background())
	if err != nil {
		return nil, err
	}

	out := []any{}
	for i := range keys {
		k := keys[i]

		useCases := make([]any, 0, len(k.UseCases))
		for _, uc := range k.UseCases {
			useCases = append(useCases, string(uc))
		}

		var keyArn, keyAlias, keyRegion, kmsKeyId string
		if k.AwsKeyInfo != nil {
			keyArn = k.AwsKeyInfo.KeyArn
			keyAlias = k.AwsKeyInfo.KeyAlias
			keyRegion = k.AwsKeyInfo.KeyRegion
		}
		if k.GcpKeyInfo != nil {
			kmsKeyId = k.GcpKeyInfo.KmsKeyId
		}

		res, err := CreateResource(r.MqlRuntime, "databricks.customerManagedKey", map[string]*llx.RawData{
			"__id":         llx.StringData("databricks.customerManagedKey/" + k.CustomerManagedKeyId),
			"id":           llx.StringData(k.CustomerManagedKeyId),
			"useCases":     llx.ArrayData(useCases, types.String),
			"creationTime": llx.TimeDataPtr(epochMsTime(k.CreationTime)),
			"keyArn":       llx.StringData(keyArn),
			"keyAlias":     llx.StringData(keyAlias),
			"keyRegion":    llx.StringData(keyRegion),
			"kmsKeyId":     llx.StringData(kmsKeyId),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}
