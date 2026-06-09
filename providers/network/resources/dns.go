// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"slices"
	"strconv"
	"strings"

	"github.com/miekg/dns"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/network/connection"
	"go.mondoo.com/mql/v13/providers/network/resources/dnsshake"
	"go.mondoo.com/mql/v13/providers/network/resources/domain"
	"go.mondoo.com/mql/v13/types"
	"go.mondoo.com/mql/v13/utils/sortx"
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

func (d *mqlDns) params(fqdn string) (any, error) {
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

func (d *mqlDns) records(params any) ([]any, error) {
	// NOTE: mql does not cache the results of GetRecords since it has an input argument
	// Iterations over map keys are not deterministic and therefore we need to sort the keys

	paramsM, ok := params.(map[string]any)
	if !ok {
		return nil, errors.New("incorrect structure of params received")
	}

	// convert responses to dns types
	resultMap := make(map[string]*mqlDnsRecord)
	for k := range paramsM {
		r, ok := paramsM[k].(map[string]any)
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
			"rdata": llx.ArrayData(llx.TArr2Raw(r["rData"].([]any)), types.String),
		})
		if err != nil {
			return nil, err
		}

		record := o.(*mqlDnsRecord)
		resultMap[record.__id] = record
	}

	keys := sortx.Keys(resultMap)
	res := []any{}
	for i := range keys {
		res = append(res, resultMap[keys[i]])
	}

	return res, nil
}

func (d *mqlDnsRecord) id() (string, error) {
	return "dns.record/" + d.Name.Data + "/" + d.Class.Data + "/" + d.Type.Data, nil
}

func (d *mqlDns) mx(params any) ([]any, error) {
	paramsM, ok := params.(map[string]any)
	if !ok {
		return []any{}, nil
	}

	mxEntries := []any{}
	record, ok := paramsM["MX"]
	if !ok {
		return mxEntries, nil
	}

	r := record.(map[string]any)

	var name, c, t string
	var ttl int64
	var rdata []any

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
		rdata = r["rData"].([]any)
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

func (d *mqlDns) dkim(params any) ([]any, error) {
	paramsM, ok := params.(map[string]any)
	if !ok {
		return []any{}, nil
	}

	dkimEntries := []any{}

	record, ok := paramsM["TXT"]
	if !ok {
		return dkimEntries, nil
	}

	r := record.(map[string]any)

	var name string
	var rdata []any

	if r["name"] != nil {
		name = r["name"].(string)
	}

	if r["rData"] != nil {
		rdata = r["rData"].([]any)
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

func (d *mqlDns) dnssec(params any) (*mqlDnsDnssecConfig, error) {
	keys := []any{}
	algoSet := map[int64]struct{}{}

	if paramsM, ok := params.(map[string]any); ok {
		if record, ok := paramsM["DNSKEY"].(map[string]any); ok && record["rCode"] == dns.RcodeToString[dns.RcodeSuccess] {
			var name, class string
			var ttl int64
			var rdata []any
			if record["name"] != nil {
				name = record["name"].(string)
			}
			if record["class"] != nil {
				class = record["class"].(string)
			}
			if record["TTL"] != nil {
				ttl = record["TTL"].(int64)
			}
			if record["rData"] != nil {
				rdata = record["rData"].([]any)
			}

			for j := range rdata {
				entry, ok := rdata[j].(string)
				if !ok {
					continue
				}

				// reuse the dns package to parse the DNSKEY rdata into its fields
				s := name + "\t" + strconv.FormatInt(ttl, 10) + "\t" + class + "\tDNSKEY\t" + entry
				rr, err := dns.NewRR(s)
				if err != nil {
					return nil, err
				}
				key, ok := rr.(*dns.DNSKEY)
				if !ok {
					continue
				}

				// the SEP flag (least-significant bit of the flags field)
				// marks a key-signing key; flags 257 is a KSK, 256 a ZSK
				keySigningKey := key.Flags&1 == 1
				keyResource, err := CreateResource(d.MqlRuntime, "dns.dnssecKey", map[string]*llx.RawData{
					"__id":          llx.StringData(fmt.Sprintf("dns.dnssecKey/%d/%d/%s", key.Algorithm, key.Flags, key.PublicKey)),
					"flags":         llx.IntData(int64(key.Flags)),
					"protocol":      llx.IntData(int64(key.Protocol)),
					"algorithm":     llx.IntData(int64(key.Algorithm)),
					"publicKey":     llx.StringData(key.PublicKey),
					"keySigningKey": llx.BoolData(keySigningKey),
				})
				if err != nil {
					return nil, err
				}
				keys = append(keys, keyResource)
				algoSet[int64(key.Algorithm)] = struct{}{}
			}
		}
	}

	algorithms := make([]any, 0, len(algoSet))
	for a := range algoSet {
		algorithms = append(algorithms, a)
	}
	slices.SortFunc(algorithms, func(a, b any) int {
		return int(a.(int64) - b.(int64))
	})

	res, err := CreateResource(d.MqlRuntime, "dns.dnssecConfig", map[string]*llx.RawData{
		"__id":       llx.StringData("dns.dnssecConfig/" + d.Fqdn.Data),
		"enabled":    llx.BoolData(len(keys) > 0),
		"keys":       llx.ArrayData(keys, types.Resource("dns.dnssecKey")),
		"algorithms": llx.ArrayData(algorithms, types.Int),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlDnsDnssecConfig), nil
}

// parseSPF extracts the version, ordered mechanisms, and the qualifier of the
// operative `all` mechanism from an SPF TXT record. SPF mechanisms are
// evaluated left to right and `all` always matches, so the first `all` term is
// the operative one and any later all-like terms are unreachable.
func parseSPF(txt string) (version string, mechanisms []string, allQualifier string) {
	mechanisms = []string{}
	for i, f := range strings.Fields(txt) {
		if i == 0 {
			version = strings.TrimPrefix(f, "v=")
			continue
		}
		mechanisms = append(mechanisms, f)

		if allQualifier != "" {
			continue // first `all` wins; ignore any later all-like terms
		}
		lower := strings.ToLower(f)
		switch {
		case lower == "all":
			allQualifier = "+" // a bare `all` uses the default pass qualifier
		case len(lower) == 4 && lower[1:] == "all" && strings.ContainsRune("+-~?", rune(lower[0])):
			allQualifier = lower[0:1]
		}
	}
	return version, mechanisms, allQualifier
}

func (d *mqlDns) spf(params any) ([]any, error) {
	entries := []any{}

	paramsM, ok := params.(map[string]any)
	if !ok {
		return entries, nil
	}
	record, ok := paramsM["TXT"].(map[string]any)
	if !ok || record["rCode"] != dns.RcodeToString[dns.RcodeSuccess] {
		return entries, nil
	}

	name, _ := record["name"].(string)
	rdata, _ := record["rData"].([]any)
	for j := range rdata {
		entry, ok := rdata[j].(string)
		if !ok {
			continue
		}
		entry = strings.TrimSpace(entry)
		if !strings.HasPrefix(strings.ToLower(entry), "v=spf1") {
			continue
		}

		version, mechanisms, allQualifier := parseSPF(entry)
		res, err := CreateResource(d.MqlRuntime, "dns.spfRecord", map[string]*llx.RawData{
			"__id":         llx.StringData("dns.spf/" + name + "/" + entry),
			"dnsTxt":       llx.StringData(entry),
			"version":      llx.StringData(version),
			"mechanisms":   llx.ArrayData(llx.TArr2Raw(mechanisms), types.String),
			"allQualifier": llx.StringData(allQualifier),
		})
		if err != nil {
			return nil, err
		}
		entries = append(entries, res)
	}
	return entries, nil
}

// parseDMARC splits a DMARC TXT record into its lowercased tag map.
func parseDMARC(txt string) map[string]string {
	tags := map[string]string{}
	for _, part := range strings.Split(txt, ";") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		tags[strings.ToLower(strings.TrimSpace(kv[0]))] = strings.TrimSpace(kv[1])
	}
	return tags
}

func dmarcUris(raw string) []any {
	uris := []any{}
	if raw == "" {
		return uris
	}
	for _, u := range strings.Split(raw, ",") {
		if u = strings.TrimSpace(u); u != "" {
			uris = append(uris, u)
		}
	}
	return uris
}

func (d *mqlDns) dmarc() (*mqlDnsDmarcRecord, error) {
	// DMARC policy is published at the _dmarc subdomain, not on the base name.
	dmarcFqdn := "_dmarc." + d.Fqdn.Data
	shaker, err := dnsshake.New(dmarcFqdn)
	if err != nil {
		return nil, err
	}
	records, err := shaker.Query("TXT")
	if err != nil {
		return nil, err
	}

	var dmarcTxt string
	if rec, ok := records["TXT"]; ok && rec.RCode == dns.RcodeToString[dns.RcodeSuccess] {
		for _, entry := range rec.RData {
			entry = strings.TrimSpace(entry)
			if strings.HasPrefix(strings.ToLower(entry), "v=dmarc1") {
				dmarcTxt = entry
				break
			}
		}
	}

	if dmarcTxt == "" {
		d.Dmarc.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	tags := parseDMARC(dmarcTxt)

	percentage := int64(100) // DMARC defaults pct to 100 when the tag is absent
	if pct, ok := tags["pct"]; ok {
		if n, err := strconv.ParseInt(pct, 10, 64); err == nil {
			percentage = n
		}
	}

	res, err := CreateResource(d.MqlRuntime, "dns.dmarcRecord", map[string]*llx.RawData{
		"__id":                llx.StringData("dns.dmarc/" + dmarcFqdn),
		"dnsTxt":              llx.StringData(dmarcTxt),
		"version":             llx.StringData(tags["v"]),
		"policy":              llx.StringData(tags["p"]),
		"subdomainPolicy":     llx.StringData(tags["sp"]),
		"aggregateReportUris": llx.ArrayData(dmarcUris(tags["rua"]), types.String),
		"forensicReportUris":  llx.ArrayData(dmarcUris(tags["ruf"]), types.String),
		"percentage":          llx.IntData(percentage),
		"spfAlignment":        llx.StringData(tags["aspf"]),
		"dkimAlignment":       llx.StringData(tags["adkim"]),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlDnsDmarcRecord), nil
}

// addressesFromParams extracts the resolved IPv4 (A) and IPv6 (AAAA) addresses
// from a dns params dict.
func addressesFromParams(params any) ([]string, error) {
	paramsM, ok := params.(map[string]any)
	if !ok {
		return nil, errors.New("incorrect structure of params received")
	}

	addrs := []string{}
	for _, key := range []string{"A", "AAAA"} {
		entry, ok := paramsM[key].(map[string]any)
		if !ok {
			continue
		}
		rdata, ok := entry["rData"].([]any)
		if !ok {
			continue
		}
		for _, v := range rdata {
			if s, ok := v.(string); ok && s != "" {
				addrs = append(addrs, s)
			}
		}
	}
	return addrs, nil
}

func (d *mqlDns) reverse(params any) ([]any, error) {
	addrs, err := addressesFromParams(params)
	if err != nil {
		return nil, err
	}

	resultMap := make(map[string]*mqlDnsRecord)
	for _, addr := range addrs {
		// dns.ReverseAddr builds the in-addr.arpa (IPv4) or ip6.arpa (IPv6)
		// name; it returns an error for malformed addresses, which we skip.
		arpa, err := dns.ReverseAddr(addr)
		if err != nil {
			continue
		}

		shaker, err := dnsshake.New(arpa)
		if err != nil {
			return nil, err
		}

		records, err := shaker.Query("PTR")
		if err != nil {
			return nil, err
		}

		ptr, ok := records["PTR"]
		if !ok || ptr.RCode != dns.RcodeToString[dns.RcodeSuccess] {
			continue
		}

		o, err := CreateResource(d.MqlRuntime, "dns.record", map[string]*llx.RawData{
			"name":  llx.StringData(ptr.Name),
			"ttl":   llx.IntData(ptr.TTL),
			"class": llx.StringData(ptr.Class),
			"type":  llx.StringData(ptr.Type),
			"rdata": llx.ArrayData(llx.TArr2Raw(ptr.RData), types.String),
		})
		if err != nil {
			return nil, err
		}

		record := o.(*mqlDnsRecord)
		resultMap[record.__id] = record
	}

	keys := sortx.Keys(resultMap)
	res := []any{}
	for i := range keys {
		res = append(res, resultMap[keys[i]])
	}
	return res, nil
}
