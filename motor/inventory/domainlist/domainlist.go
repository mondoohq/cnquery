package domainlist

import (
	"bufio"
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/cockroachdb/errors"
	"go.mondoo.io/mondoo/motor/asset"
	v1 "go.mondoo.io/mondoo/motor/inventory/v1"
	"go.mondoo.io/mondoo/motor/providers"
)

func Parse(input io.Reader) (*Inventory, error) {
	inventory := &Inventory{}
	scanner := bufio.NewScanner(input)

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		inventory.Hosts = append(inventory.Hosts, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return inventory, nil
}

type Inventory struct {
	Hosts []string
}

func (in *Inventory) ToV1Inventory() *v1.Inventory {
	out := v1.New()

	r := &networkResolver{}

	for i := range in.Hosts {
		host := in.Hosts[i]
		name := host

		// prefix with host to ensure the connection parsing works as expected
		if !strings.Contains(host, "//") {
			host = "host://" + host
		}

		tc, err := r.ParseConnectionURL(host, "", "")
		if err != nil {
			log.Warn().Err(err).Str("hostname", host).Msg("could not parse hostname")
		}

		out.Spec.Assets = append(out.Spec.Assets, &asset.Asset{
			Name:        name,
			Connections: []*providers.TransportConfig{tc},
		})
	}

	return out
}

type networkResolver struct{}

func (r *networkResolver) ParseConnectionURL(fullUrl string, identityFile string, password string) (*providers.TransportConfig, error) {
	url, err := url.Parse(fullUrl)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse target URL")
	}

	// TODO: Processing the family here needs a bit more work. It is unclear
	// where this will evolve for now, so let's keep watching it.
	// So far we know:
	// - all of them are in the `api` family (also their kind is set this way)
	// - multiple families on one service are possible (eg: http, tls, tcp)
	res := providers.TransportConfig{
		Backend: providers.ProviderType_HOST,
		Options: map[string]string{"scheme": url.Scheme},
	}

	schemeBits := strings.Split(url.Scheme, "+")
	for i := range schemeBits {
		x := strings.ToLower(schemeBits[i])
		switch x {
		case "tls", "tcp", "udp":
			// FIXME: properly check for schema bits
			res.Options[x] = ""
		}
	}

	hostBits := strings.Split(url.Host, ":")
	switch len(hostBits) {
	case 1:
		res.Host = hostBits[0]
	case 2:
		res.Host = hostBits[0]
		port, err := strconv.Atoi(hostBits[1])
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse port in target URL")
		}
		res.Port = int32(port)
	default:
		return nil, errors.New("malformed target URL, host cannot be parsed")
	}

	return &res, nil
}
