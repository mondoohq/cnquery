// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/likexian/whois"
	whoisparser "github.com/likexian/whois-parser"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/network/connection"
	"go.mondoo.com/cnquery/v11/types"
)

func initWhois(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	conn := runtime.Connection.(*connection.HostConnection)
	host := conn.Conf.Host
	if target, ok := args["target"]; ok {
		host = target.Value.(string)
		delete(args, "target")
	}

	args["host"] = llx.StringData(host)
	return args, nil, nil
}

func (d *mqlWhois) id() (string, error) {
	return "whois/" + d.Host.Data, nil
}

func (d *mqlWhois) fetch() error {
	host := d.Host.Data

	// set default values
	d.Domain = plugin.TValue[*mqlWhoisDomainInfo]{Data: nil, Error: nil, State: plugin.StateIsSet | plugin.StateIsNull}
	d.Registrar = plugin.TValue[*mqlWhoisContact]{Data: nil, Error: nil, State: plugin.StateIsSet | plugin.StateIsNull}
	d.Registrant = plugin.TValue[*mqlWhoisContact]{Data: nil, Error: nil, State: plugin.StateIsSet | plugin.StateIsNull}
	d.Administrative = plugin.TValue[*mqlWhoisContact]{Data: nil, Error: nil, State: plugin.StateIsSet | plugin.StateIsNull}
	d.Technical = plugin.TValue[*mqlWhoisContact]{Data: nil, Error: nil, State: plugin.StateIsSet | plugin.StateIsNull}
	d.Billing = plugin.TValue[*mqlWhoisContact]{Data: nil, Error: nil, State: plugin.StateIsSet | plugin.StateIsNull}

	// fetch whois data
	rawWhoisResult, err := whois.Whois(host)
	if err != nil {
		return err
	}

	result, err := whoisparser.Parse(rawWhoisResult)
	if err != nil {
		return err
	}

	domainInfo, err := newMqlWhoisDomain(d.MqlRuntime, result.Domain)
	if domainInfo != nil {
		d.Domain = plugin.TValue[*mqlWhoisDomainInfo]{Data: domainInfo, Error: err, State: plugin.StateIsSet}
	}

	registrar, err := newMqlWhoisContact(d.MqlRuntime, result.Registrar)
	if registrar != nil {
		d.Registrar = plugin.TValue[*mqlWhoisContact]{Data: registrar, Error: err, State: plugin.StateIsSet}
	}

	registrant, err := newMqlWhoisContact(d.MqlRuntime, result.Registrant)
	if registrant != nil {
		d.Registrant = plugin.TValue[*mqlWhoisContact]{Data: registrant, Error: err, State: plugin.StateIsSet}
	}

	administrative, err := newMqlWhoisContact(d.MqlRuntime, result.Administrative)
	if administrative != nil {
		d.Administrative = plugin.TValue[*mqlWhoisContact]{Data: administrative, Error: err, State: plugin.StateIsSet}
	}

	technical, err := newMqlWhoisContact(d.MqlRuntime, result.Technical)
	if technical != nil {
		d.Technical = plugin.TValue[*mqlWhoisContact]{Data: technical, Error: err, State: plugin.StateIsSet}
	}

	billing, err := newMqlWhoisContact(d.MqlRuntime, result.Billing)
	if billing != nil {
		d.Billing = plugin.TValue[*mqlWhoisContact]{Data: billing, Error: err, State: plugin.StateIsSet}
	}
	return nil
}

func (d *mqlWhois) domain() (*mqlWhoisDomainInfo, error) {
	return nil, d.fetch()
}

func (d *mqlWhois) registrar() (*mqlWhoisContact, error) {
	return nil, d.fetch()
}

func (d *mqlWhois) registrant() (*mqlWhoisContact, error) {
	return nil, d.fetch()
}

func (d *mqlWhois) administrative() (*mqlWhoisContact, error) {
	return nil, d.fetch()
}

func (d *mqlWhois) technical() (*mqlWhoisContact, error) {
	return nil, d.fetch()
}

func (d *mqlWhois) billing() (*mqlWhoisContact, error) {
	return nil, d.fetch()
}

func newMqlWhoisDomain(runtime *plugin.Runtime, domain *whoisparser.Domain) (*mqlWhoisDomainInfo, error) {
	if domain == nil {
		return nil, nil
	}
	mqlDomainInfo, err := CreateResource(runtime, "whois.domainInfo", map[string]*llx.RawData{
		"__id":        llx.StringData(domain.ID),
		"domain":      llx.StringData(domain.Domain),
		"name":        llx.StringData(domain.Name),
		"punyCode":    llx.StringData(domain.Punycode),
		"extension":   llx.StringData(domain.Extension),
		"whoisServer": llx.StringData(domain.WhoisServer),
		"status":      llx.ArrayData(convert.SliceAnyToInterface(domain.Status), types.String),
		"nameServers": llx.ArrayData(convert.SliceAnyToInterface(domain.NameServers), types.String),
		"dnssec":      llx.BoolData(domain.DNSSec),
		"createdAt":   llx.TimeDataPtr(domain.CreatedDateInTime),
		"updatedAt":   llx.TimeDataPtr(domain.UpdatedDateInTime),
		"expiresAt":   llx.TimeDataPtr(domain.ExpirationDateInTime),
	})
	return mqlDomainInfo.(*mqlWhoisDomainInfo), err
}

func newMqlWhoisContact(runtime *plugin.Runtime, contact *whoisparser.Contact) (*mqlWhoisContact, error) {
	if contact == nil {
		return nil, nil
	}
	mqlContact, err := CreateResource(runtime, "whois.contact", map[string]*llx.RawData{
		"__id":         llx.StringData(contact.ID),
		"name":         llx.StringData(contact.Name),
		"organization": llx.StringData(contact.Organization),
		"street":       llx.StringData(contact.Street),
		"city":         llx.StringData(contact.City),
		"province":     llx.StringData(contact.Province),
		"postalCode":   llx.StringData(contact.PostalCode),
		"country":      llx.StringData(contact.Country),
		"phone":        llx.StringData(contact.Phone),
		"email":        llx.StringData(contact.Email),
		"registrarUrl": llx.StringData(contact.ReferralURL),
	})
	return mqlContact.(*mqlWhoisContact), err
}
