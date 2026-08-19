// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
)

func (r *mqlVault) namespaces() ([]any, error) {
	client, err := vaultClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	resp, err := client.Logical().List("sys/namespaces")
	if err != nil {
		// Namespaces are an Enterprise feature. Community edition answers 404
		// on the endpoint, which is an absence of the feature rather than a
		// failure to read it, so report an empty list. A permission failure is
		// not absence and is left to surface as an error.
		if isFeatureAbsent(err) {
			return []any{}, nil
		}
		return nil, err
	}
	// A List with no entries returns a nil response rather than an empty one.
	if resp == nil || resp.Data == nil {
		return []any{}, nil
	}

	keys, _ := resp.Data["keys"].([]any)
	infos, _ := resp.Data["key_info"].(map[string]any)

	res := make([]any, 0, len(keys))
	for _, rawKey := range keys {
		path, ok := rawKey.(string)
		if !ok {
			continue
		}

		var id string
		var metadata map[string]string
		if info, ok := infos[path].(map[string]any); ok {
			id, _ = info["id"].(string)
			metadata = stringMap(info["custom_metadata"])
		}

		mqlNamespace, err := CreateResource(r.MqlRuntime, "vault.namespace", map[string]*llx.RawData{
			"__id":           llx.StringData(path),
			"path":           llx.StringData(path),
			"id":             llx.StringData(id),
			"customMetadata": llx.MapData(convert.MapToInterfaceMap(metadata), "string"),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlNamespace)
	}
	return res, nil
}

// stringMap narrows a decoded JSON object to string values, dropping entries
// that are not strings. Vault renders custom metadata as strings, so a
// non-string value means the payload was not metadata.
func stringMap(raw any) map[string]string {
	values, ok := raw.(map[string]any)
	if !ok {
		return nil
	}

	out := make(map[string]string, len(values))
	for key, value := range values {
		if text, ok := value.(string); ok {
			out[key] = text
		}
	}
	return out
}
