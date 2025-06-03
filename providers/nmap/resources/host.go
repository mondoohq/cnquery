// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"github.com/rs/zerolog/log"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"github.com/Ullaakut/nmap/v3"
	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/nmap/connection"
)

type mqlNmapHostInternal struct {
	lock sync.Mutex
}

func initNmapHost(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if _, ok := args["name"]; !ok {
		// try to get the ip from the connection
		conn := runtime.Connection.(*connection.NmapConnection)
		if conn.Conf.Options != nil && conn.Conf.Options["search"] == "host" {
			args["name"] = llx.StringData(conn.Conf.Host)
		}
	}

	if _, ok := args["name"]; !ok {
		return nil, nil, errors.New("missing required argument 'name'")
	}

	return args, nil, nil
}

func newMqlNmapHost(runtime *plugin.Runtime, host nmap.Host) (*mqlNmapHost, error) {
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
		"__id": llx.StringData("nmap.host/" + name),
		"name": llx.StringData(name),
	})
	return mqlNmapHostResource.(*mqlNmapHost), err
}

func (r *mqlNmapHost) id() (string, error) {
	return "nmap.host/" + r.Name.Data, nil
}

func (r *mqlNmapHost) scan() error {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.Distance = plugin.TValue[interface{}]{Data: nil, Error: nil, State: plugin.StateIsSet | plugin.StateIsNull}
	r.Os = plugin.TValue[interface{}]{Data: nil, Error: nil, State: plugin.StateIsSet | plugin.StateIsNull}
	r.EndTime = plugin.TValue[*time.Time]{Data: nil, Error: nil, State: plugin.StateIsSet | plugin.StateIsNull}
	r.Comment = plugin.TValue[string]{Data: "", Error: nil, State: plugin.StateIsSet | plugin.StateIsNull}
	r.Trace = plugin.TValue[interface{}]{Data: nil, Error: nil, State: plugin.StateIsSet | plugin.StateIsNull}
	r.Addresses = plugin.TValue[[]interface{}]{Data: nil, Error: nil, State: plugin.StateIsSet | plugin.StateIsNull}
	r.Hostnames = plugin.TValue[[]interface{}]{Data: nil, Error: nil, State: plugin.StateIsSet | plugin.StateIsNull}
	r.State = plugin.TValue[string]{Data: "", Error: nil, State: plugin.StateIsSet | plugin.StateIsNull}
	r.Ports = plugin.TValue[[]interface{}]{Data: nil, Error: nil, State: plugin.StateIsSet | plugin.StateIsNull}

	setError := func(err error) {
		r.Distance = plugin.TValue[interface{}]{Data: nil, Error: err, State: plugin.StateIsSet | plugin.StateIsNull}
		r.Os = plugin.TValue[interface{}]{Data: nil, Error: err, State: plugin.StateIsSet | plugin.StateIsNull}
		r.EndTime = plugin.TValue[*time.Time]{Data: nil, Error: err, State: plugin.StateIsSet | plugin.StateIsNull}
		r.Comment = plugin.TValue[string]{Data: "", Error: err, State: plugin.StateIsSet | plugin.StateIsNull}
		r.Trace = plugin.TValue[interface{}]{Data: nil, Error: err, State: plugin.StateIsSet | plugin.StateIsNull}
		r.Addresses = plugin.TValue[[]interface{}]{Data: nil, Error: err, State: plugin.StateIsSet | plugin.StateIsNull}
		r.Hostnames = plugin.TValue[[]interface{}]{Data: nil, Error: err, State: plugin.StateIsSet | plugin.StateIsNull}
		r.State = plugin.TValue[string]{Data: "", Error: err, State: plugin.StateIsSet | plugin.StateIsNull}
		r.Ports = plugin.TValue[[]interface{}]{Data: nil, Error: err, State: plugin.StateIsSet | plugin.StateIsNull}

	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	log.Info().Str("host", r.Name.Data).Str("id", r.MqlID()).Msg("Scanning host")

	// nmap -sT -sV  -n --min-parallelism 100 -T4 192.168.1.0/24
	scanner, err := nmap.NewScanner(
		ctx,
		nmap.WithConnectScan(),           // -sT
		nmap.WithServiceInfo(),           // -sV
		nmap.WithDisabledDNSResolution(), // -n
		nmap.WithMinParallelism(100),
		nmap.WithTimingTemplate(nmap.TimingAggressive),
		nmap.WithTargets(r.Name.Data),
	)
	if err != nil {
		setError(err)
		return errors.Wrap(err, "unable to create nmap scanner")
	}

	result, _, err := scanner.Run()
	if err != nil {
		setError(err)
		return errors.Wrap(err, "unable to create nmap scanner")
	}

	if len(result.Hosts) == 0 {
		return nil
	} else if len(result.Hosts) > 1 {
		setError(err)
		return errors.New("nmap scan returned more than one host")
	}

	host := result.Hosts[0]
	id := r.Name.Data
	t := time.Time(host.EndTime)

	distance, err := convert.JsonToDict(host.Distance)
	if err == nil {
		r.Distance = plugin.TValue[interface{}]{Data: distance, Error: nil, State: plugin.StateIsSet}
	}

	os, err := convert.JsonToDict(host.OS)
	if err == nil {
		r.Os = plugin.TValue[interface{}]{Data: os, Error: nil, State: plugin.StateIsSet}
	}

	r.EndTime = plugin.TValue[*time.Time]{Data: &t, Error: nil, State: plugin.StateIsSet}
	r.Comment = plugin.TValue[string]{Data: host.Comment, Error: nil, State: plugin.StateIsSet}

	trace, err := convert.JsonToDict(host.Trace)
	if err == nil {
		r.Trace = plugin.TValue[interface{}]{Data: trace, Error: nil, State: plugin.StateIsSet}
	}

	addresses, err := convert.JsonToDictSlice(host.Addresses)
	if err == nil {
		r.Addresses = plugin.TValue[[]interface{}]{Data: addresses, Error: nil, State: plugin.StateIsSet}
	}

	hostnames, err := convert.JsonToDictSlice(host.Hostnames)
	if err == nil {
		r.Hostnames = plugin.TValue[[]interface{}]{Data: hostnames, Error: nil, State: plugin.StateIsSet}
	}

	ports := make([]interface{}, 0)
	for _, port := range host.Ports {
		r, err := newMqlNmapPort(r.MqlRuntime, id, port)
		if err != nil {
			return err
		}
		ports = append(ports, r)
	}

	if len(ports) > 0 {
		r.Ports = plugin.TValue[[]interface{}]{Data: ports, Error: nil, State: plugin.StateIsSet}
	}

	r.State = plugin.TValue[string]{Data: host.Status.State, Error: nil, State: plugin.StateIsSet}

	return nil
}

func (r *mqlNmapHost) distance() (interface{}, error) {
	return nil, r.scan()
}

func (r *mqlNmapHost) os() (interface{}, error) {
	return nil, r.scan()
}

func (r *mqlNmapHost) endTime() (*time.Time, error) {
	return nil, r.scan()
}

func (r *mqlNmapHost) comment() (string, error) {
	return "", r.scan()
}

func (r *mqlNmapHost) trace() (interface{}, error) {
	return nil, r.scan()
}

func (r *mqlNmapHost) addresses() ([]interface{}, error) {
	return nil, r.scan()
}

func (r *mqlNmapHost) hostnames() ([]interface{}, error) {
	return nil, r.scan()
}

func (r *mqlNmapHost) ports() ([]interface{}, error) {
	return nil, r.scan()
}

func (r *mqlNmapHost) state() (string, error) {
	return "", r.scan()
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
