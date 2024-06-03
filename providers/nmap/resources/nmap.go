// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/Ullaakut/nmap/v3"
	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/types"
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

func newMqlNmapHost(runtime *plugin.Runtime, host nmap.Host) (*mqlNmapHost, error) {
	distance, _ := convert.JsonToDict(host.Distance)
	os, _ := convert.JsonToDict(host.OS)
	trace, _ := convert.JsonToDict(host.Trace)
	addresses, _ := convert.JsonToDictSlice(host.Addresses)
	hostnames, _ := convert.JsonToDictSlice(host.Hostnames)

	id := uuid.New().String()

	ports := make([]interface{}, 0)
	for _, port := range host.Ports {
		r, err := newMqlNmapPort(runtime, id, port)
		if err != nil {
			return nil, err
		}
		ports = append(ports, r)
	}

	name := ""
	if len(host.Addresses) == 1 {
		name = host.Addresses[0].Addr
	} else {
		entries := []string{}
		for _, addr := range host.Addresses {
			if addr.Addr != "" {
				entries = append(entries, addr.Addr)
			}
		}
		name = strings.Join(entries, ", ")
	}

	mqlNmapHostResource, err := CreateResource(runtime, "nmap.host", map[string]*llx.RawData{
		"__id":      llx.StringData("nmap.host/" + id),
		"name":      llx.StringData(name),
		"distance":  llx.DictData(distance),
		"os":        llx.DictData(os),
		"endTime":   llx.TimeData(time.Time(host.EndTime)),
		"comment":   llx.StringData(host.Comment),
		"trace":     llx.DictData(trace),
		"addresses": llx.ArrayData(addresses, types.Dict),
		"hostnames": llx.ArrayData(hostnames, types.Dict),
		"ports":     llx.ArrayData(ports, types.Resource("nmap.port")),
		"state":     llx.StringData(host.Status.State),
	})
	return mqlNmapHostResource.(*mqlNmapHost), err
}

func newMqlNmapPort(runtime *plugin.Runtime, id string, port nmap.Port) (*mqlNmapPort, error) {

	mqlPort, err := CreateResource(runtime, "nmap.port", map[string]*llx.RawData{
		"__id":     llx.StringData("nmap.port/" + id + "/" + strconv.Itoa(int(port.ID))),
		"port":     llx.IntData(int64(port.ID)),
		"service":  llx.StringData(port.Service.Name),
		"method":   llx.StringData(port.Service.Method),
		"protocol": llx.StringData(port.Protocol),
		"product":  llx.StringData(port.Service.Product),
		"version":  llx.StringData(port.Service.Version),
		"state":    llx.StringData(port.State.State),
	})
	return mqlPort.(*mqlNmapPort), err
}

func (r *mqlNmapTarget) hosts() ([]interface{}, error) {
	return nil, r.scan()
}

func (r *mqlNmapTarget) warnings() ([]interface{}, error) {
	return nil, r.scan()
}
