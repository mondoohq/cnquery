// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package dnsshake

import (
	"net"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/miekg/dns"
	"go.mondoo.com/mql/v13/utils/multierr"
)

type DnsClient struct {
	config *dns.ClientConfig
	fqdn   string
	sync   sync.Mutex
}

type DnsRecord struct {
	// DNS name
	Name string `json:"name"`
	// Time-To-Live (TTL) in seconds
	TTL int64 `json:"ttl"`
	// DNS class
	Class string `json:"class"`
	// DNS type
	Type string `json:"type"`
	// Resource Data
	RData []string `json:"rData"`
	// DNS Response Code
	RCode string `json:"rCode"`
	// Error during dns request
	Error error `json:"error"`
}

type DnsServer struct {
	IP    string
	Owner string
}

var CommonDnsServers = []DnsServer{
	// Google
	{IP: "8.8.8.8", Owner: "Google"},
	{IP: "8.8.4.4", Owner: "Google"},
	// Cloudflare
	{IP: "1.1.1.1", Owner: "Cloudflare"},
	{IP: "1.0.0.1", Owner: "Cloudflare"},
	{IP: "1.1.1.2", Owner: "Cloudflare (malware blocking)"},
	{IP: "1.0.0.2", Owner: "Cloudflare (malware blocking)"},
	{IP: "1.1.1.3", Owner: "Cloudflare (malware + adult content blocking)"},
	{IP: "1.0.0.3", Owner: "Cloudflare (malware + adult content blocking)"},
	// Quad9
	{IP: "9.9.9.9", Owner: "Quad9"},
	{IP: "149.112.112.112", Owner: "Quad9"},
	// OpenDNS (Cisco)
	{IP: "208.67.222.222", Owner: "OpenDNS (Cisco)"},
	{IP: "208.67.220.220", Owner: "OpenDNS (Cisco)"},
	{IP: "208.67.222.123", Owner: "OpenDNS FamilyShield (Cisco)"},
	{IP: "208.67.220.123", Owner: "OpenDNS FamilyShield (Cisco)"},
	// Comodo Secure DNS
	{IP: "8.26.56.26", Owner: "Comodo Secure DNS"},
	{IP: "8.20.247.20", Owner: "Comodo Secure DNS"},
	// Verisign
	{IP: "64.6.64.6", Owner: "Verisign"},
	{IP: "64.6.65.6", Owner: "Verisign"},
	// AdGuard
	{IP: "94.140.14.14", Owner: "AdGuard"},
	{IP: "94.140.15.15", Owner: "AdGuard"},
	{IP: "94.140.14.15", Owner: "AdGuard (family protection)"},
	{IP: "94.140.15.16", Owner: "AdGuard (family protection)"},
	// CleanBrowsing
	{IP: "185.228.168.9", Owner: "CleanBrowsing (security)"},
	{IP: "185.228.169.9", Owner: "CleanBrowsing (security)"},
	{IP: "185.228.168.168", Owner: "CleanBrowsing (adult filter)"},
	{IP: "185.228.169.168", Owner: "CleanBrowsing (adult filter)"},
	{IP: "185.228.168.10", Owner: "CleanBrowsing (family)"},
	{IP: "185.228.169.11", Owner: "CleanBrowsing (family)"},
	// DNS.WATCH
	{IP: "84.200.69.80", Owner: "DNS.WATCH"},
	{IP: "84.200.70.40", Owner: "DNS.WATCH"},
	// Neustar UltraDNS (Vercara)
	{IP: "156.154.70.1", Owner: "Neustar UltraDNS (Vercara)"},
	{IP: "156.154.71.1", Owner: "Neustar UltraDNS (Vercara)"},
	// Level3 / Lumen
	{IP: "4.2.2.1", Owner: "Level3 (Lumen)"},
	{IP: "4.2.2.2", Owner: "Level3 (Lumen)"},
	// Yandex
	{IP: "77.88.8.8", Owner: "Yandex"},
	{IP: "77.88.8.1", Owner: "Yandex"},
	// AliDNS (Alibaba)
	{IP: "223.5.5.5", Owner: "AliDNS (Alibaba)"},
	{IP: "223.6.6.6", Owner: "AliDNS (Alibaba)"},
	// 114DNS (China)
	{IP: "114.114.114.114", Owner: "114DNS"},
	{IP: "114.114.115.115", Owner: "114DNS"},
	// Freenom World
	{IP: "80.80.80.80", Owner: "Freenom World"},
	{IP: "80.80.81.81", Owner: "Freenom World"},
	// Alternate DNS
	{IP: "76.76.19.19", Owner: "Alternate DNS"},
	{IP: "76.223.122.150", Owner: "Alternate DNS"},
	// Control D
	{IP: "76.76.2.0", Owner: "Control D"},
	{IP: "76.76.10.0", Owner: "Control D"},
}

type Config struct {
	Servers []string
}

func New(fqdn string, conf *Config) (*DnsClient, error) {
	// use Google DNS for now
	config := &dns.ClientConfig{
		Search:   []string{},
		Port:     "53",
		Ndots:    1,
		Timeout:  5,
		Attempts: 2,
	}
	if conf != nil && len(conf.Servers) > 0 {
		config.Servers = conf.Servers
	} else {
		for i := 0; i < 3 && i < len(CommonDnsServers); i++ {
			config.Servers = append(config.Servers, CommonDnsServers[i].IP)
		}
	}

	// try to load unix dns server
	// TODO: this does not work on windows https://github.com/go-acme/lego/issues/1015
	resolveFile := "/etc/resolv.conf"
	_, err := os.Stat(resolveFile)
	if err == nil {
		rConfig, err := dns.ClientConfigFromFile(resolveFile)
		if err == nil {
			config = rConfig
		}
	}

	return &DnsClient{
		fqdn:   fqdn,
		config: config,
	}, nil
}

// stringToType is a map of strings to each RR type.
var stringToType = map[string]uint16{
	"A":          dns.TypeA,
	"AAAA":       dns.TypeAAAA,
	"AFSDB":      dns.TypeAFSDB,
	"ANY":        dns.TypeANY,
	"APL":        dns.TypeAPL,
	"ATMA":       dns.TypeATMA,
	"AVC":        dns.TypeAVC,
	"AXFR":       dns.TypeAXFR,
	"CAA":        dns.TypeCAA,
	"CDNSKEY":    dns.TypeCDNSKEY,
	"CDS":        dns.TypeCDS,
	"CERT":       dns.TypeCERT,
	"CNAME":      dns.TypeCNAME,
	"CSYNC":      dns.TypeCSYNC,
	"DHCID":      dns.TypeDHCID,
	"DLV":        dns.TypeDLV,
	"DNAME":      dns.TypeDNAME,
	"DNSKEY":     dns.TypeDNSKEY,
	"DS":         dns.TypeDS,
	"EID":        dns.TypeEID,
	"EUI48":      dns.TypeEUI48,
	"EUI64":      dns.TypeEUI64,
	"GID":        dns.TypeGID,
	"GPOS":       dns.TypeGPOS,
	"HINFO":      dns.TypeHINFO,
	"HIP":        dns.TypeHIP,
	"HTTPS":      dns.TypeHTTPS,
	"ISDN":       dns.TypeISDN,
	"IXFR":       dns.TypeIXFR,
	"KEY":        dns.TypeKEY,
	"KX":         dns.TypeKX,
	"L32":        dns.TypeL32,
	"L64":        dns.TypeL64,
	"LOC":        dns.TypeLOC,
	"LP":         dns.TypeLP,
	"MAILA":      dns.TypeMAILA,
	"MAILB":      dns.TypeMAILB,
	"MB":         dns.TypeMB,
	"MD":         dns.TypeMD,
	"MF":         dns.TypeMF,
	"MG":         dns.TypeMG,
	"MINFO":      dns.TypeMINFO,
	"MR":         dns.TypeMR,
	"MX":         dns.TypeMX,
	"NAPTR":      dns.TypeNAPTR,
	"NID":        dns.TypeNID,
	"NIMLOC":     dns.TypeNIMLOC,
	"NINFO":      dns.TypeNINFO,
	"NS":         dns.TypeNS,
	"NSEC":       dns.TypeNSEC,
	"NSEC3":      dns.TypeNSEC3,
	"NSEC3PARAM": dns.TypeNSEC3PARAM,
	"NULL":       dns.TypeNULL,
	"NXT":        dns.TypeNXT,
	"None":       dns.TypeNone,
	"OPENPGPKEY": dns.TypeOPENPGPKEY,
	"OPT":        dns.TypeOPT,
	"PTR":        dns.TypePTR,
	"PX":         dns.TypePX,
	"RKEY":       dns.TypeRKEY,
	"RP":         dns.TypeRP,
	"RRSIG":      dns.TypeRRSIG,
	"RT":         dns.TypeRT,
	"Reserved":   dns.TypeReserved,
	"SIG":        dns.TypeSIG,
	"SMIMEA":     dns.TypeSMIMEA,
	"SOA":        dns.TypeSOA,
	"SPF":        dns.TypeSPF,
	"SRV":        dns.TypeSRV,
	"SSHFP":      dns.TypeSSHFP,
	"SVCB":       dns.TypeSVCB,
	"TA":         dns.TypeTA,
	"TALINK":     dns.TypeTALINK,
	"TKEY":       dns.TypeTKEY,
	"TLSA":       dns.TypeTLSA,
	"TSIG":       dns.TypeTSIG,
	"TXT":        dns.TypeTXT,
	"UID":        dns.TypeUID,
	"UINFO":      dns.TypeUINFO,
	"UNSPEC":     dns.TypeUNSPEC,
	"URI":        dns.TypeURI,
	"X25":        dns.TypeX25,
	"ZONEMD":     dns.TypeZONEMD,
	"NSAP-PTR":   dns.TypeNSAPPTR,
}

func (d *DnsClient) Query(dnsTypes ...string) (map[string]DnsRecord, error) {
	if len(dnsTypes) == 0 {
		for k := range stringToType {
			dnsTypes = append(dnsTypes, k)
		}
	}

	workers := sync.WaitGroup{}
	var errs multierr.Errors

	res := map[string]DnsRecord{}
	for i := range dnsTypes {
		dnsType := dnsTypes[i]

		workers.Add(1)
		go func() {
			defer workers.Done()

			records, err := d.queryDnsType(d.fqdn, dnsType)
			if err != nil {
				d.sync.Lock()
				errs.Add(err)
				d.sync.Unlock()
				return
			}

			d.sync.Lock()
			for k := range records {
				res[k] = records[k]
			}
			d.sync.Unlock()
		}()
	}

	workers.Wait()
	return res, errs.Deduplicate()
}

func (d *DnsClient) queryDnsType(fqdn string, t string) (map[string]DnsRecord, error) {
	dnsType, ok := stringToType[t]
	if !ok {
		return nil, errors.New("unknown dns type")
	}
	dnsTypText := dns.Type(dnsType).String()

	res := map[string]DnsRecord{}

	c := &dns.Client{}
	m := &dns.Msg{}
	m.SetEdns0(4096, false)
	m.SetQuestion(dns.Fqdn(fqdn), dnsType)
	m.RecursionDesired = true

	r, _, err := c.Exchange(m, net.JoinHostPort(d.config.Servers[0], d.config.Port))
	if err != nil {
		res[dnsTypText] = DnsRecord{
			Type:  dnsTypText,
			Error: err,
		}
		return res, nil
	}

	if r.Rcode != dns.RcodeSuccess {
		res[dnsTypText] = DnsRecord{
			Type:  dnsTypText,
			RCode: dns.RcodeToString[r.Rcode],
		}
		return res, nil
	}

	// extract more information if dns request was successful
	for i := range r.Answer {
		a := r.Answer[i]

		typ := dns.Type(a.Header().Rrtype).String()

		var rec DnsRecord

		rec, ok := res[typ]
		if !ok {
			rec = DnsRecord{
				Name:  a.Header().Name,
				Type:  typ,
				Class: dns.Class(a.Header().Class).String(),
				TTL:   int64(a.Header().Ttl),
				RData: []string{},
				RCode: dns.RcodeToString[r.Rcode],
			}
		}

		switch v := a.(type) {
		case *dns.A:
			rec.RData = append(rec.RData, v.A.String())
		case *dns.NS:
			rec.RData = append(rec.RData, v.Ns)
		case *dns.MD:
			rec.RData = append(rec.RData, v.Md)
		case *dns.MF:
			rec.RData = append(rec.RData, v.Mf)
		case *dns.CNAME:
			rec.RData = append(rec.RData, v.Target)
		case *dns.MB:
			rec.RData = append(rec.RData, v.Mb)
		case *dns.MG:
			rec.RData = append(rec.RData, v.Mg)
		case *dns.MR:
			rec.RData = append(rec.RData, v.Mr)
		case *dns.NULL:
			rec.RData = append(rec.RData, v.Data)
		case *dns.PTR:
			rec.RData = append(rec.RData, v.Ptr)
		case *dns.TXT:
			rec.RData = append(rec.RData, strings.Join(v.Txt, ""))
		case *dns.EID:
			rec.RData = append(rec.RData, v.Endpoint)
		case *dns.NIMLOC:
			rec.RData = append(rec.RData, v.Locator)
		case *dns.SPF:
			rec.RData = append(rec.RData, strings.Join(v.Txt, ""))
		case *dns.UINFO:
			rec.RData = append(rec.RData, v.Uinfo)
		case *dns.UID:
			rec.RData = append(rec.RData, strconv.FormatInt(int64(v.Uid), 10))
		case *dns.GID:
			rec.RData = append(rec.RData, strconv.FormatInt(int64(v.Gid), 10))
		case *dns.EUI48:
			strconv.FormatInt(int64(v.Address), 10)
		case *dns.EUI64:
			strconv.FormatInt(int64(v.Address), 10)
		case *dns.AVC:
			rec.RData = append(rec.RData, strings.Join(v.Txt, ""))
		default:
			rec.RData = append(rec.RData, strings.TrimPrefix(v.String(), v.Header().String()))
		}

		res[typ] = rec
	}
	return res, nil
}
