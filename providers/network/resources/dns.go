// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net"
	"strconv"
	"strings"

	"github.com/miekg/dns"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/network/connection"
	"go.mondoo.com/cnquery/v11/providers/network/resources/dnsshake"
	"go.mondoo.com/cnquery/v11/providers/network/resources/domain"
	"go.mondoo.com/cnquery/v11/types"
	"go.mondoo.com/cnquery/v11/utils/sortx"
)

func (d *mqlDomainName) id() (string, error) {
	return "domainName/" + d.Fqdn.Data, nil
}

func initDomainName(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	fqdn, ok := args["fqdn"]
	if !ok {
		conn := runtime.Connection.(*connection.HostConnection)
		fqdn = llx.StringData(conn.FQDN())
		args["fqdn"] = fqdn
	}

	if fqdn == nil {
		return nil, nil, errors.New("domainName resource requires fqdn argument")
	}

	dn, err := domain.Parse(fqdn.Value.(string))
	if err != nil {
		return nil, nil, err
	}

	args["effectiveTLDPlusOne"] = llx.StringData(dn.EffectiveTLDPlusOne)
	args["tld"] = llx.StringData(dn.TLD)
	args["tldIcannManaged"] = llx.BoolData(dn.IcannManagedTLD)
	args["labels"] = llx.ArrayData(llx.TArr2Raw[string](dn.Labels), types.String)

	return args, nil, nil
}

func (d *mqlDns) id() (string, error) {
	return "dns/" + d.Fqdn.Data, nil
}

func initDns(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	_, ok := args["fqdn"]
	if !ok {
		conn := runtime.Connection.(*connection.HostConnection)
		fqdn := llx.StringData(conn.FQDN())

		// Check whether the fqdn is valid
		// In case of ssh connections, this could also be an ip address
		ip := net.ParseIP(fqdn.Value.(string))
		if ip == nil {
			args["fqdn"] = fqdn
		} else {
			args["fqdn"] = llx.StringData("")
		}
	}

	return args, nil, nil
}

func (d *mqlDns) params(fqdn string) (interface{}, error) {
	dnsShaker, err := dnsshake.New(fqdn)
	if err != nil {
		return nil, err
	}

	records, err := dnsShaker.Query()
	if err != nil {
		return nil, err
	}

	return convert.JsonToDict(records)
}

func (d *mqlDns) records(params interface{}) ([]interface{}, error) {
	// NOTE: mql does not cache the results of GetRecords since it has an input argument
	// Iterations over map keys are not deterministic and therefore we need to sort the keys

	paramsM, ok := params.(map[string]interface{})
	if !ok {
		return nil, errors.New("incorrect structure of params received")
	}

	// convert responses to dns types
	resultMap := make(map[string]*mqlDnsRecord)
	for k := range paramsM {
		r, ok := paramsM[k].(map[string]interface{})
		if !ok {
			return nil, errors.New("incorrect structure of params entries received")
		}

		// filter by successful dns records
		if r["rCode"] != dns.RcodeToString[dns.RcodeSuccess] {
			continue
		}

		var ttl *llx.RawData
		if r["TTL"] == nil {
			ttl = llx.NilData
		} else {
			ttl = llx.IntData(r["TTL"].(int64))
		}
		o, err := CreateResource(d.MqlRuntime, "dns.record", map[string]*llx.RawData{
			"name":  llx.StringData(r["name"].(string)),
			"ttl":   ttl,
			"class": llx.StringData(r["class"].(string)),
			"type":  llx.StringData(r["type"].(string)),
			"rdata": llx.ArrayData(llx.TArr2Raw(r["rData"].([]interface{})), types.String),
		})
		if err != nil {
			return nil, err
		}

		record := o.(*mqlDnsRecord)
		resultMap[record.__id] = record
	}

	keys := sortx.Keys(resultMap)
	res := []interface{}{}
	for i := range keys {
		res = append(res, resultMap[keys[i]])
	}

	return res, nil
}

func (d *mqlDnsRecord) id() (string, error) {
	return "dns.record/" + d.Name.Data + "/" + d.Class.Data + "/" + d.Type.Data, nil
}

func (d *mqlDns) mx(params interface{}) ([]interface{}, error) {
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
			mxEntry, err := CreateResource(d.MqlRuntime, "dns.mxRecord", map[string]*llx.RawData{
				"name":       llx.StringData(name),
				"preference": llx.IntData(int64(v.Preference)),
				"domainName": llx.StringData(v.Mx),
			})
			if err != nil {
				return nil, err
			}
			mxEntries = append(mxEntries, mxEntry)
		}
	}

	return mxEntries, nil
}

func (d *mqlDnsMxRecord) id() (string, error) {
	return "dns.mx/" + d.Name.Data + "+" + d.DomainName.Data, nil
}

func (d *mqlDns) dkim(params interface{}) ([]interface{}, error) {
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

		o, err := CreateResource(d.MqlRuntime, "dns.dkimRecord", map[string]*llx.RawData{
			"domain":         llx.StringData(name),
			"dnsTxt":         llx.StringData(entry),
			"version":        llx.StringData(dkimRepr.Version),
			"hashAlgorithms": llx.ArrayData(llx.TArr2Raw(dkimRepr.HashAlgorithms), types.String),
			"keyType":        llx.StringData(dkimRepr.KeyType),
			"notes":          llx.StringData(dkimRepr.Notes),
			"publicKeyData":  llx.StringData(dkimRepr.PublicKeyData),
			"serviceTypes":   llx.ArrayData(llx.TArr2Raw(dkimRepr.ServiceType), types.String),
			"flags":          llx.ArrayData(llx.TArr2Raw(dkimRepr.Flags), types.String),
		})
		if err != nil {
			return nil, err
		}
		record := o.(*mqlDnsDkimRecord)
		record.dkim = dkimRepr
		dkimEntries = append(dkimEntries, record)
	}

	return dkimEntries, nil
}

type mqlDnsDkimRecordInternal struct {
	dkim *dnsshake.DkimPublicKeyRepresentation
}

func (d *mqlDnsDkimRecord) id() (string, error) {
	hasher := sha256.New()
	hasher.Write([]byte(d.DnsTxt.Data))
	sha256 := hex.EncodeToString(hasher.Sum(nil))
	return "dns.dkim/" + d.Domain.Data + "/" + sha256, nil
}

func (d *mqlDnsDkimRecord) valid() (bool, error) {
	if d.dkim == nil {
		return false, errors.New("could not load dkim data")
	}

	ok, _, _ := d.dkim.Valid()
	return ok, nil
}
