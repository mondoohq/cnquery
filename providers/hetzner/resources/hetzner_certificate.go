// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

func (r *mqlHetznerCertificate) id() (string, error) {
	return fmt.Sprintf("hetzner.certificate/%d", r.Id.Data), nil
}

func (h *mqlHetzner) certificates() ([]any, error) {
	c := conn(h.MqlRuntime)
	items, err := paginate(func(opts hcloud.ListOpts) ([]*hcloud.Certificate, *hcloud.Response, error) {
		return c.Client().Certificate.List(ctx(), hcloud.CertificateListOpts{ListOpts: opts})
	})
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(items))
	for _, cert := range items {
		res, err := newMqlHetznerCertificate(h.MqlRuntime, cert)
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func newMqlHetznerCertificate(runtime *plugin.Runtime, cert *hcloud.Certificate) (*mqlHetznerCertificate, error) {
	status := map[string]any{}
	if cert.Status != nil {
		status["issuance"] = string(cert.Status.Issuance)
		status["renewal"] = string(cert.Status.Renewal)
		if cert.Status.Error != nil {
			status["error"] = cert.Status.Error.Error()
		}
	}
	usedBy := make([]any, 0, len(cert.UsedBy))
	for _, ref := range cert.UsedBy {
		usedBy = append(usedBy, map[string]any{
			"type": string(ref.Type),
			"id":   ref.ID,
		})
	}

	res, err := CreateResource(runtime, "hetzner.certificate", map[string]*llx.RawData{
		"__id":           llx.StringData(fmt.Sprintf("hetzner.certificate/%d", cert.ID)),
		"id":             llx.IntData(cert.ID),
		"name":           llx.StringData(cert.Name),
		"type":           llx.StringData(string(cert.Type)),
		"fingerprint":    llx.StringData(cert.Fingerprint),
		"notValidBefore": llx.TimeDataPtr(timePtr(cert.NotValidBefore)),
		"notValidAfter":  llx.TimeDataPtr(timePtr(cert.NotValidAfter)),
		"domainNames":    stringArrayData(cert.DomainNames),
		"status":         llx.DictData(status),
		"created":        llx.TimeDataPtr(timePtr(cert.Created)),
		"usedBy":         dictArrayData(usedBy),
		"labels":         labelData(cert.Labels),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlHetznerCertificate), nil
}

func initHetznerCertificate(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := idArg(args, "id")
	if !ok {
		return nil, nil, errIDRequired("certificate")
	}
	cert, _, err := conn(runtime).Client().Certificate.GetByID(ctx(), id)
	if err != nil {
		return nil, nil, err
	}
	if cert == nil {
		return nil, nil, notFoundErr("certificate", id)
	}
	res, err := newMqlHetznerCertificate(runtime, cert)
	return args, res, err
}
