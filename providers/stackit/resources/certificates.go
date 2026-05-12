// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/stackitcloud/stackit-sdk-go/services/certificates"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

func (r *mqlStackit) certificates() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.Certificates()
	if err != nil {
		return nil, err
	}
	out := []any{}
	pageId := ""
	for {
		req := client.ListCertificates(bgctx(), c.ProjectID(), c.Region())
		if pageId != "" {
			req = req.PageId(pageId)
		}
		resp, err := req.Execute()
		if err != nil {
			if isAccessDenied(err) {
				return []any{}, nil
			}
			return nil, err
		}
		items, _ := resp.GetItemsOk()
		for i := range items {
			res, err := buildCertificate(r.MqlRuntime, &items[i])
			if err != nil {
				return nil, err
			}
			out = append(out, res)
		}
		next, ok := resp.GetNextPageIdOk()
		if !ok || next == "" {
			break
		}
		pageId = next
	}
	return out, nil
}

func buildCertificate(runtime *plugin.Runtime, cert *certificates.GetCertificateResponse) (plugin.Resource, error) {
	args := map[string]*llx.RawData{
		"id":        llx.StringData(cert.GetId()),
		"name":      llx.StringData(cert.GetName()),
		"region":    llx.StringData(cert.GetRegion()),
		"publicKey": llx.StringData(cert.GetPublicKey()),
	}
	return CreateResource(runtime, "stackit.certificate", args)
}

func (r *mqlStackitCertificate) id() (string, error) {
	return "stackit.certificate/" + r.Id.Data, nil
}

func initStackitCertificate(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := idArg(args, "id")
	if !ok {
		return args, nil, nil
	}
	c := conn(runtime)
	client, err := c.Certificates()
	if err != nil {
		return nil, nil, err
	}
	cert, err := client.GetCertificateExecute(bgctx(), c.ProjectID(), c.Region(), id)
	if err != nil {
		return nil, nil, err
	}
	res, err := buildCertificate(runtime, cert)
	if err != nil {
		return nil, nil, err
	}
	return nil, res, nil
}
