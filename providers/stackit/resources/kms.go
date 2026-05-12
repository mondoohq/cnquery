// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/stackitcloud/stackit-sdk-go/services/kms"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

func (r *mqlStackitKms) keyRings() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.KMS()
	if err != nil {
		return nil, err
	}
	resp, err := client.ListKeyRingsExecute(bgctx(), c.ProjectID(), c.Region())
	if err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, err
	}
	items, _ := resp.GetKeyRingsOk()
	out := make([]any, 0, len(items))
	for i := range items {
		res, err := buildKmsKeyRing(r.MqlRuntime, &items[i])
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

// keys() is a convenience that flattens key rings → keys so callers can
// audit every key in the project without an explicit two-level traversal.
func (r *mqlStackitKms) keys() ([]any, error) {
	rings, err := r.keyRings()
	if err != nil {
		return nil, err
	}
	out := []any{}
	for _, ring := range rings {
		kr, ok := ring.(*mqlStackitKmsKeyRing)
		if !ok {
			continue
		}
		keys, err := kr.keys()
		if err != nil {
			return nil, err
		}
		out = append(out, keys...)
	}
	return out, nil
}

func buildKmsKeyRing(runtime *plugin.Runtime, kr *kms.KeyRing) (plugin.Resource, error) {
	createdAt, ok1 := kr.GetCreatedAtOk()
	args := map[string]*llx.RawData{
		"id":          llx.StringData(kr.GetId()),
		"displayName": llx.StringData(kr.GetDisplayName()),
		"description": llx.StringData(kr.GetDescription()),
		"state":       llx.StringData(string(kr.GetState())),
		"createdAt":   llx.TimeDataPtr(timeOrNil(createdAt, ok1)),
	}
	return CreateResource(runtime, "stackit.kms.keyRing", args)
}

func (r *mqlStackitKmsKeyRing) id() (string, error) {
	return "stackit.kms.keyRing/" + r.Id.Data, nil
}

func (r *mqlStackitKmsKeyRing) keys() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.KMS()
	if err != nil {
		return nil, err
	}
	resp, err := client.ListKeysExecute(bgctx(), c.ProjectID(), c.Region(), r.Id.Data)
	if err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, err
	}
	items, _ := resp.GetKeysOk()
	out := make([]any, 0, len(items))
	for i := range items {
		res, err := buildKmsKey(r.MqlRuntime, &items[i])
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func buildKmsKey(runtime *plugin.Runtime, k *kms.Key) (plugin.Resource, error) {
	createdAt, ok1 := k.GetCreatedAtOk()
	deletionDate, ok2 := k.GetDeletionDateOk()
	args := map[string]*llx.RawData{
		"id":           llx.StringData(k.GetId()),
		"keyRingId":    llx.StringData(k.GetKeyRingId()),
		"displayName":  llx.StringData(k.GetDisplayName()),
		"description":  llx.StringData(k.GetDescription()),
		"purpose":      llx.StringData(string(k.GetPurpose())),
		"algorithm":    llx.StringData(string(k.GetAlgorithm())),
		"protection":   llx.StringData(string(k.GetProtection())),
		"accessScope":  llx.StringData(string(k.GetAccessScope())),
		"importOnly":   llx.BoolData(k.GetImportOnly()),
		"state":        llx.StringData(string(k.GetState())),
		"createdAt":    llx.TimeDataPtr(timeOrNil(createdAt, ok1)),
		"deletionDate": llx.TimeDataPtr(timeOrNil(deletionDate, ok2)),
	}
	return CreateResource(runtime, "stackit.kms.key", args)
}

func (r *mqlStackitKmsKey) id() (string, error) {
	return "stackit.kms.key/" + r.KeyRingId.Data + "/" + r.Id.Data, nil
}
