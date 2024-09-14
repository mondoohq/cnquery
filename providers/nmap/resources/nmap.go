// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

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

// standard nmap scan
// nmap -sT -T4 192.168.178.0/24
//
// include service and version Detection
// nmap -sT -T4 -sV 192.168.178.0/24
//
// fast discovery
// nmap -sn -n -T4 192.168.178.0/24
func (r *mqlNmap) id() (string, error) {
	return "nmap", nil
}

func (r *mqlNmapTarget) id() (string, error) {
	return "nmap.target/" + r.Target.Data, nil
}

func initNmapTarget(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return args, nil, nil
}

func (r *mqlNmapTarget) scan() error {
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

func (r *mqlNmapTarget) hosts() ([]interface{}, error) {
	return nil, r.scan()
}

func (r *mqlNmapTarget) warnings() ([]interface{}, error) {
	return nil, r.scan()
}
