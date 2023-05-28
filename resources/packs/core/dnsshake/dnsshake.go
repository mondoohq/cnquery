package dnsshake

import (
	"net"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/hashicorp/go-multierror"
	"github.com/miekg/dns"
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

func New(fqdn string) (*DnsClient, error) {
	// use Google DNS for now
	config := &dns.ClientConfig{}
	config.Servers = []string{"8.8.8.8", "8.8.4.4"}
	config.Search = []string{}
	config.Port = "53"
	config.Ndots = 1
	config.Timeout = 5
	config.Attempts = 2

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
	var errs error

	res := map[string]DnsRecord{}
	for i := range dnsTypes {
		dnsType := dnsTypes[i]

		workers.Add(1)
		go func() {
			defer workers.Done()

			records, err := d.queryDnsType(d.fqdn, dnsType)
			if err != nil {
				d.sync.Lock()
				errs = multierror.Append(errs, err)
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
	return res, errs
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
