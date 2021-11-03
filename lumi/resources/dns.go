package resources

import (
	"go.mondoo.io/mondoo/lumi/resources/dnsshake"
)

func (d *lumiDns) id() (string, error) {
	id, _ := d.Fqdn()
	return "dns/" + id, nil
}

func (d *lumiDns) GetRecords() ([]interface{}, error) {
	fqdn, err := d.Fqdn()
	if err != nil {
		return nil, err
	}

	dnsShaker, err := dnsshake.New(fqdn)
	if err != nil {
		return nil, err
	}

	records, err := dnsShaker.Test()
	if err != nil {
		return nil, err
	}

	// convert responses to dns types
	dnsEntries := make([]interface{}, len(records))
	i := 0
	for k := range records {
		r := records[k]

		lumiDnsRecord, err := d.Runtime.CreateResource("dns.record",
			"name", r.Name,
			"ttl", r.TTL,
			"class", r.Class,
			"type", r.Type,
			"rdata", strSliceToInterface(r.RData),
		)
		if err != nil {
			return nil, err
		}

		dnsEntries[i] = lumiDnsRecord.(DnsRecord)
		i++
	}

	return dnsEntries, nil
}

func (d *lumiDnsRecord) id() (string, error) {
	name, _ := d.Name()
	t, _ := d.Type()
	c, _ := d.Class()
	return "dns.record/" + name + "/" + c + "/" + t, nil
}
