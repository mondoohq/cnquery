package okta

import (
	"context"

	"github.com/okta/okta-sdk-golang/v2/okta"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (o *mqlOkta) GetDomains() ([]interface{}, error) {
	op, err := oktaProvider(o.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	client := op.Client()
	domainSlice, _, err := client.Domain.ListDomains(
		ctx,
	)
	if err != nil {
		return nil, err
	}

	if len(domainSlice.Domains) == 0 {
		return nil, nil
	}

	list := []interface{}{}
	for i := range domainSlice.Domains {
		entry := domainSlice.Domains[i]
		r, err := newMqlOktaDomain(o.MotorRuntime, entry)
		if err != nil {
			return nil, err
		}
		list = append(list, r)

	}

	return list, nil
}

func newMqlOktaDomain(runtime *resources.Runtime, entry *okta.Domain) (interface{}, error) {
	publicCertificate, err := core.JsonToDict(entry.PublicCertificate)
	if err != nil {
		return nil, err
	}

	dnsRecords, err := core.JsonToDictSlice(entry.DnsRecords)

	return runtime.CreateResource("okta.domain",
		"id", entry.Id,
		"domain", entry.Domain,
		"validationStatus", entry.ValidationStatus,
		"publicCertificate", publicCertificate,
		"dnsRecords", dnsRecords,
	)
}

func (o *mqlOktaDomain) id() (string, error) {
	id, err := o.Id()
	if err != nil {
		return "", err
	}
	return "okta.domain/" + id, nil
}
