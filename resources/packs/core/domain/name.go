package domain

import (
	"net/url"
	"strings"

	"golang.org/x/net/publicsuffix"
)

// Domain Names https://datatracker.ietf.org/doc/html/rfc1035
// Public Suffix List https://publicsuffix.org
// Reserved Top-Level Domains https://datatracker.ietf.org/doc/html/rfc2606

// Domain embeds net/url and adds extra fields ontop
type Name struct {
	Host                string
	Port                string
	EffectiveTLDPlusOne string
	TLD                 string
	IcannManagedTLD     bool
	Labels              []string
}

// Parse mirrors net/url.Parse except instead it returns
// a tld.URL, which contains extra fields.
func Parse(fqdn string) (*Name, error) {
	// check if fqdn has a scheme otherwise go does not parse the host properly
	if !strings.Contains(fqdn, "//") {
		fqdn = "//" + fqdn
	}

	url, err := url.Parse(fqdn)
	if err != nil {
		return nil, err
	}
	if url.Host == "" {
		return &Name{}, nil
	}
	host, port := splitHostPort(url.Host)

	// effective top-level domain + one label
	etldplusone, err := publicsuffix.EffectiveTLDPlusOne(host)
	if err != nil {
		return nil, err
	}

	suffix, icann := publicsuffix.PublicSuffix(strings.ToLower(host))
	if err != nil {
		return nil, err
	}

	return &Name{
		Host:                host,
		Port:                port,
		EffectiveTLDPlusOne: etldplusone,
		TLD:                 suffix,
		IcannManagedTLD:     icann,
		Labels:              strings.Split(host, "."),
	}, nil
}

// splitHostPort separates host and port. If the port is not valid, it returns
// the entire input as host, and it doesn't check the validity of the host.
// Unlike net.SplitHostPort, but per RFC 3986, it requires ports to be numeric.
// NOTE: method is copied from go package url under BSD license
func splitHostPort(hostPort string) (host, port string) {
	host = hostPort

	colon := strings.LastIndexByte(host, ':')
	if colon != -1 && validOptionalPort(host[colon:]) {
		host, port = host[:colon], host[colon+1:]
	}

	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		host = host[1 : len(host)-1]
	}

	return
}

// validOptionalPort reports whether port is either an empty string
// or matches /^:\d*$/
// NOTE: method is copied from go package url under BSD license
func validOptionalPort(port string) bool {
	if port == "" {
		return true
	}
	if port[0] != ':' {
		return false
	}
	for _, b := range port[1:] {
		if b < '0' || b > '9' {
			return false
		}
	}
	return true
}
