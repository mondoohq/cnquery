package resources

import (
	"github.com/miekg/dns"
	"go.mondoo.io/mondoo/lumi/resources/dnsshake"
)

func (d *lumiDns) id() (string, error) {
	id, _ := d.Fqdn()
	return "dns/" + id, nil
}

func (d *lumiDns) GetParams() (interface{}, error) {
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

	return jsonToDict(records)
}

// GetRecords returns successful dns records
func (d *lumiDns) GetRecords(params map[string]interface{}) ([]interface{}, error) {
	// convert responses to dns types
	dnsEntries := []interface{}{}
	for k := range params {
		r := params[k].(map[string]interface{})

		// filter by successful dns records
		if r["rCode"] != dns.RcodeToString[dns.RcodeSuccess] {
			continue
		}

		lumiDnsRecord, err := d.Runtime.CreateResource("dns.record",
			"name", r["name"],
			"ttl", r["TTL"],
			"class", r["class"],
			"type", r["type"],
			"rdata", r["rData"],
		)
		if err != nil {
			return nil, err
		}

		dnsEntries = append(dnsEntries, lumiDnsRecord.(DnsRecord))
	}

	return dnsEntries, nil
}

func (d *lumiDnsRecord) id() (string, error) {
	name, _ := d.Name()
	t, _ := d.Type()
	c, _ := d.Class()
	return "dns.record/" + name + "/" + c + "/" + t, nil
}
