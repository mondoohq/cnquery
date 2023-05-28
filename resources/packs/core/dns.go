package core

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"strconv"
	"strings"

	"github.com/miekg/dns"
	"go.mondoo.com/cnquery/motor/providers/network"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core/dnsshake"
	"go.mondoo.com/cnquery/resources/packs/core/domain"
)

func (d *mqlDomainName) id() (string, error) {
	id, _ := d.Fqdn()
	return "domainName/" + id, nil
}

func (d *mqlDomainName) init(args *resources.Args) (*resources.Args, DomainName, error) {
	fqdn, ok := (*args)["fqdn"]
	if !ok {
		if transport, ok := d.MotorRuntime.Motor.Provider.(*network.Provider); ok {
			fqdn = transport.FQDN
		}

		(*args)["fqdn"] = fqdn
	}

	if fqdn == nil {
		return nil, nil, errors.New("domainName resource requires fqdn argument")
	}

	dn, err := domain.Parse(fqdn.(string))
	if err != nil {
		return nil, nil, err
	}

	(*args)["effectiveTLDPlusOne"] = dn.EffectiveTLDPlusOne
	(*args)["tld"] = dn.TLD
	(*args)["tldIcannManaged"] = dn.IcannManagedTLD
	(*args)["labels"] = StrSliceToInterface(dn.Labels)

	return args, nil, nil
}

func (d *mqlDns) id() (string, error) {
	id, _ := d.Fqdn()
	return "dns/" + id, nil
}

func (d *mqlDns) init(args *resources.Args) (*resources.Args, Dns, error) {
	_, ok := (*args)["fqdn"]
	if !ok {
		var fqdn string

		if transport, ok := d.MotorRuntime.Motor.Provider.(*network.Provider); ok {
			fqdn = transport.FQDN
		}

		(*args)["fqdn"] = fqdn
	}

	return args, nil, nil
}

func (d *mqlDns) GetParams() (interface{}, error) {
	fqdn, err := d.Fqdn()
	if err != nil {
		return nil, err
	}

	dnsShaker, err := dnsshake.New(fqdn)
	if err != nil {
		return nil, err
	}

	records, err := dnsShaker.Query()
	if err != nil {
		return nil, err
	}

	return JsonToDict(records)
}

// GetRecords returns successful dns records
func (d *mqlDns) GetRecords(params interface{}) ([]interface{}, error) {
	// NOTE: mql does not cache the results of GetRecords since it has an input argument
	// Iterations over map keys are not deterministic and therefore we need to sort the keys

	paramsM, ok := params.(map[string]interface{})
	if !ok {
		return []interface{}{}, nil
	}

	// convert responses to dns types
	keys := []string{}
	resultMap := make(map[string]interface{})
	for k := range paramsM {
		r := paramsM[k].(map[string]interface{})

		// filter by successful dns records
		if r["rCode"] != dns.RcodeToString[dns.RcodeSuccess] {
			continue
		}

		mqlDnsRecord, err := d.MotorRuntime.CreateResource("dns.record",
			"name", r["name"],
			"ttl", r["TTL"],
			"class", r["class"],
			"type", r["type"],
			"rdata", r["rData"],
		)
		if err != nil {
			return nil, err
		}

		id := mqlDnsRecord.MqlResource().Id
		keys = append(keys, id)
		resultMap[id] = mqlDnsRecord
	}

	// sort keys
	sort.Strings(keys)
	dnsEntries := []interface{}{}
	for i := range keys {
		mqlDnsRecord := resultMap[keys[i]]
		dnsEntries = append(dnsEntries, mqlDnsRecord.(DnsRecord))
	}

	return dnsEntries, nil
}

func (d *mqlDnsRecord) id() (string, error) {
	name, _ := d.Name()
	t, _ := d.Type()
	c, _ := d.Class()
	return "dns.record/" + name + "/" + c + "/" + t, nil
}

func (d *mqlDns) GetMx(params interface{}) ([]interface{}, error) {
	paramsM, ok := params.(map[string]interface{})
	if !ok {
		return []interface{}{}, nil
	}

	mxEntries := []interface{}{}
	record, ok := paramsM["MX"]
	if !ok {
		return mxEntries, nil
	}

	r := record.(map[string]interface{})

	var name, c, t string
	var ttl int64
	var rdata []interface{}

	if r["name"] != nil {
		name = r["name"].(string)
	}

	if r["class"] != nil {
		c = r["class"].(string)
	}

	if r["type"] != nil {
		t = r["type"].(string)
	}

	if r["TTL"] != nil {
		ttl = r["TTL"].(int64)
	}

	if r["rData"] != nil {
		rdata = r["rData"].([]interface{})
	}

	for j := range rdata {
		entry := rdata[j].(string)

		// use dns package to parse mx entry
		s := name + "\t" + strconv.FormatInt(ttl, 10) + "\t" + c + "\t" + t + "\t" + entry
		got, err := dns.NewRR(s)
		if err != nil {
			return nil, err
		}

		switch v := got.(type) {
		case *dns.MX:
			mxEntry, err := d.MotorRuntime.CreateResource("dns.mxRecord",
				"name", name,
				"preference", int64(v.Preference),
				"domainName", v.Mx,
			)
			if err != nil {
				return nil, err
			}
			mxEntries = append(mxEntries, mxEntry)
		}
	}

	return mxEntries, nil
}

func (d *mqlDnsMxRecord) id() (string, error) {
	name, err := d.Name()
	domainName, _ := d.DomainName()
	return "dns.mx/" + name + "+" + domainName, err
}

func (d *mqlDns) GetDkim(params interface{}) ([]interface{}, error) {
	paramsM, ok := params.(map[string]interface{})
	if !ok {
		return []interface{}{}, nil
	}

	dkimEntries := []interface{}{}

	record, ok := paramsM["TXT"]
	if !ok {
		return dkimEntries, nil
	}

	r := record.(map[string]interface{})

	var name string
	var rdata []interface{}

	if r["name"] != nil {
		name = r["name"].(string)
	}

	if r["rData"] != nil {
		rdata = r["rData"].([]interface{})
	}

	for j := range rdata {
		entry := rdata[j].(string)
		entry = strings.TrimSpace(entry)

		if !strings.HasPrefix(entry, "v=DKIM1;") {
			continue
		}

		dkimRepr, err := dnsshake.NewDkimPublicKeyRepresentation(entry)
		if err != nil {
			return nil, err
		}

		dkimRecord, err := d.MotorRuntime.CreateResource("dns.dkimRecord",
			"domain", name,
			"dnsTxt", entry,
			"version", dkimRepr.Version,
			"hashAlgorithms", StrSliceToInterface(dkimRepr.HashAlgorithms),
			"keyType", dkimRepr.KeyType,
			"notes", dkimRepr.Notes,
			"publicKeyData", dkimRepr.PublicKeyData,
			"serviceTypes", StrSliceToInterface(dkimRepr.ServiceType),
			"flags", StrSliceToInterface(dkimRepr.Flags),
		)
		if err != nil {
			return nil, err
		}
		dkimRecord.MqlResource().Cache.Store("_dkim", &resources.CacheEntry{Data: dkimRepr})
		dkimEntries = append(dkimEntries, dkimRecord)
	}

	return dkimEntries, nil
}

func (d *mqlDnsDkimRecord) id() (string, error) {
	name, err := d.Domain()
	if err != nil {
		return "", err
	}
	dnsTxt, err := d.DnsTxt()
	if err != nil {
		return "", err
	}
	hasher := sha256.New()
	hasher.Write([]byte(dnsTxt))
	sha256 := hex.EncodeToString(hasher.Sum(nil))
	return "dns.dkim/" + name + "/" + sha256, err
}

func (d *mqlDnsDkimRecord) GetValid() (bool, error) {
	entry, ok := d.MqlResource().Cache.Load("_dkim")
	if !ok {
		return false, errors.New("could not load dkim data")
	}

	rep, ok := entry.Data.(*dnsshake.DkimPublicKeyRepresentation)
	if !ok {
		return false, errors.New("could not load dkim data")
	}

	ok, _, _ = rep.Valid()
	return ok, nil
}
