// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"go.mondoo.com/cnquery/v11/llx"
	"strings"

	"github.com/shadowscatcher/shodan/search"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/shodan/connection"
)

func initShodanHost(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if _, ok := args["ip"]; !ok {
		// try to get the ip from the connection
		conn := runtime.Connection.(*connection.ShodanConnection)
		if conn.Conf.Options != nil && conn.Conf.Options["search"] == "host" {
			args["ip"] = llx.StringData(conn.Conf.Host)
		}
	}

	if _, ok := args["ip"]; !ok {
		return nil, nil, errors.New("missing required argument 'host'")
	}

	return args, nil, nil
}

func (r *mqlShodanHost) id() (string, error) {
	return "shodan.host/" + r.Ip.Data, nil
}

func (r *mqlShodanHost) fetchBaseInformation() error {
	conn := r.MqlRuntime.Connection.(*connection.ShodanConnection)
	client := conn.Client()
	if client == nil {
		return errors.New("cannot retrieve new data while using a mock connection")
	}

	// set default information
	r.Os = plugin.TValue[string]{Data: "", Error: nil, State: plugin.StateIsSet | plugin.StateIsNull}
	r.Org = plugin.TValue[string]{Data: "", Error: nil, State: plugin.StateIsSet | plugin.StateIsNull}
	r.Isp = plugin.TValue[string]{Data: "", Error: nil, State: plugin.StateIsSet | plugin.StateIsNull}
	r.Asn = plugin.TValue[string]{Data: "", Error: nil, State: plugin.StateIsSet | plugin.StateIsNull}
	r.Tags = plugin.TValue[[]interface{}]{Data: nil, Error: nil, State: plugin.StateIsSet | plugin.StateIsNull}
	r.Hostnames = plugin.TValue[[]interface{}]{Data: nil, Error: nil, State: plugin.StateIsSet | plugin.StateIsNull}
	r.Ports = plugin.TValue[[]interface{}]{Data: nil, Error: nil, State: plugin.StateIsSet | plugin.StateIsNull}
	r.Vulnerabilities = plugin.TValue[[]interface{}]{Data: nil, Error: nil, State: plugin.StateIsSet | plugin.StateIsNull}

	ctx := context.Background()
	host, err := client.Host(ctx, search.HostParams{
		IP: r.Ip.Data,
	})

	// ignore no information available error since it is not an error
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "no information available") {
		return nil
	} else if err != nil {
		return err
	}

	if host.OS != nil {
		r.Os = plugin.TValue[string]{Data: *host.OS, Error: nil, State: plugin.StateIsSet}
	}

	if host.Org != nil {
		r.Org = plugin.TValue[string]{Data: *host.Org, Error: nil, State: plugin.StateIsSet}
	}

	if host.ISP != nil {
		r.Isp = plugin.TValue[string]{Data: *host.ISP, Error: nil, State: plugin.StateIsSet}
	}

	if host.ASN != nil {
		r.Asn = plugin.TValue[string]{Data: *host.ASN, Error: nil, State: plugin.StateIsSet}
	}

	if host.Tags != nil {
		r.Tags = plugin.TValue[[]interface{}]{Data: convert.SliceAnyToInterface(host.Tags), Error: nil, State: plugin.StateIsSet}
	}

	if host.Hostnames != nil {
		r.Hostnames = plugin.TValue[[]interface{}]{Data: convert.SliceAnyToInterface(host.Hostnames), Error: nil, State: plugin.StateIsSet}
	}

	if host.Ports != nil {
		// we cannot use convert.SliceIntToInterface since the ports need to be int64
		ports := make([]interface{}, len(host.Ports))
		for i := range host.Ports {
			ports[i] = int64(host.Ports[i])
		}
		r.Ports = plugin.TValue[[]interface{}]{Data: ports, Error: nil, State: plugin.StateIsSet}
	}

	if host.Vulns != nil {
		r.Vulnerabilities = plugin.TValue[[]interface{}]{Data: convert.SliceAnyToInterface(host.Vulns), Error: nil, State: plugin.StateIsSet}
	}

	return nil
}

func (r *mqlShodanHost) os() (string, error) {
	return "", r.fetchBaseInformation()
}

func (r *mqlShodanHost) org() (string, error) {
	return "", r.fetchBaseInformation()
}

func (r *mqlShodanHost) isp() (string, error) {
	return "", r.fetchBaseInformation()
}

func (r *mqlShodanHost) asn() (string, error) {
	return "", r.fetchBaseInformation()
}

func (r *mqlShodanHost) tags() ([]interface{}, error) {
	return nil, r.fetchBaseInformation()
}

func (r *mqlShodanHost) hostnames() ([]interface{}, error) {
	return nil, r.fetchBaseInformation()
}

func (r *mqlShodanHost) ports() ([]interface{}, error) {
	return nil, r.fetchBaseInformation()
}

func (r *mqlShodanHost) vulnerabilities() ([]interface{}, error) {
	return nil, r.fetchBaseInformation()
}
