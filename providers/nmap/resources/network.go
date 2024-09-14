package resources

import (
	"context"
	"time"

	"github.com/Ullaakut/nmap/v3"
	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
)

func (r *mqlNmapNetwork) id() (string, error) {
	return "nmap.target/" + r.Target.Data, nil
}

func initNmapNetwork(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return args, nil, nil
}

func (r *mqlNmapNetwork) scan() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// set default values
	r.Hosts = plugin.TValue[[]interface{}]{Data: nil, Error: nil, State: plugin.StateIsSet | plugin.StateIsNull}
	r.Warnings = plugin.TValue[[]interface{}]{Data: nil, Error: nil, State: plugin.StateIsSet | plugin.StateIsNull}

	if r.Target.Data == "" {
		return errors.New("target is required")
	}

	scanner, err := nmap.NewScanner(
		ctx,
		nmap.WithConnectScan(),
		nmap.WithTimingTemplate(nmap.TimingAggressive),
		nmap.WithServiceInfo(),
		nmap.WithDisabledDNSResolution(), // -n
		nmap.WithTargets(r.Target.Data),
	)
	if err != nil {
		return errors.Wrap(err, "unable to create nmap scanner")
	}

	result, warnings, err := scanner.Run()

	if warnings != nil && len(*warnings) > 0 {
		r.Warnings = plugin.TValue[[]interface{}]{Data: convert.SliceAnyToInterface(*warnings), Error: nil, State: plugin.StateIsSet}
	}

	var hosts []interface{}
	for _, host := range result.Hosts {
		r, err := newMqlNmapHost(r.MqlRuntime, host)
		if err != nil {
			return err
		}
		hosts = append(hosts, r)
	}

	r.Hosts = plugin.TValue[[]interface{}]{Data: hosts, Error: nil, State: plugin.StateIsSet}

	return nil
}

func (r *mqlNmapNetwork) hosts() ([]interface{}, error) {
	return nil, r.scan()
}

func (r *mqlNmapNetwork) warnings() ([]interface{}, error) {
	return nil, r.scan()
}
