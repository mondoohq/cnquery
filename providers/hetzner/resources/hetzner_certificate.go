// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

type mqlHetznerCertificateInternal struct {
	cacheServerIDs       []int64
	cacheLoadBalancerIDs []int64
}

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
	var serverIDs, lbIDs []int64
	for _, ref := range cert.UsedBy {
		switch string(ref.Type) {
		case "server":
			serverIDs = append(serverIDs, ref.ID)
		case "load_balancer":
			lbIDs = append(lbIDs, ref.ID)
		}
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
		"labels":         labelData(cert.Labels),
	})
	if err != nil {
		return nil, err
	}
	m := res.(*mqlHetznerCertificate)
	m.cacheServerIDs = serverIDs
	m.cacheLoadBalancerIDs = lbIDs
	return m, nil
}

func (m *mqlHetznerCertificate) servers() ([]any, error) {
	out := make([]any, 0, len(m.cacheServerIDs))
	for _, id := range m.cacheServerIDs {
		ref, err := NewResource(m.MqlRuntime, "hetzner.server", map[string]*llx.RawData{
			"id": llx.IntData(id),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, ref)
	}
	return out, nil
}

func (m *mqlHetznerCertificate) loadBalancers() ([]any, error) {
	out := make([]any, 0, len(m.cacheLoadBalancerIDs))
	for _, id := range m.cacheLoadBalancerIDs {
		ref, err := NewResource(m.MqlRuntime, "hetzner.loadBalancer", map[string]*llx.RawData{
			"id": llx.IntData(id),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, ref)
	}
	return out, nil
}

func initHetznerCertificate(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := idArg(args, "id")
	if !ok {
		return args, nil, nil
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
