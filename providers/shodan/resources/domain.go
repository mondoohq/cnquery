// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"time"

	"github.com/shadowscatcher/shodan/models"
	"github.com/shadowscatcher/shodan/search"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/shodan/connection"
)

func initShodanDomain(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if _, ok := args["name"]; !ok {
		// try to get the ip from the connection
		conn := runtime.Connection.(*connection.ShodanConnection)
		if conn.Conf.Options != nil && conn.Conf.Options["search"] == "domain" {
			args["name"] = llx.StringData(conn.Conf.Host)
		}
	}

	if _, ok := args["name"]; !ok {
		return nil, nil, errors.New("missing required argument 'name'")
	}

	return args, nil, nil
}

func (r *mqlShodanDomain) id() (string, error) {
	return "shodan.domain/" + r.Name.Data, nil
}

func (r *mqlShodanDomain) fetchBaseInformation() error {
	conn := r.MqlRuntime.Connection.(*connection.ShodanConnection)
	client := conn.Client()
	if client == nil {
		return errors.New("cannot retrieve new data while using a mock connection")
	}

	// set default information
	r.Tags = plugin.TValue[[]interface{}]{Data: nil, Error: nil, State: plugin.StateIsSet | plugin.StateIsNull}
	r.Subdomains = plugin.TValue[[]interface{}]{Data: nil, Error: nil, State: plugin.StateIsSet | plugin.StateIsNull}
	r.Nsrecords = plugin.TValue[[]interface{}]{Data: nil, Error: nil, State: plugin.StateIsSet | plugin.StateIsNull}

	ctx := context.Background()
	domain := r.Name.Data
	domainSearch := search.DomainQuery{
		Domain: domain,
		Page:   1,
	}

	var tags []string
	subdomains := make([]string, 0)
	nsrecords := make([]models.NsRecord, 0)
	for {
		results, err := client.DnsDomain(ctx, domainSearch)
		if err != nil {
			return err
		}

		tags = results.Tags
		subdomains = append(subdomains, results.Subdomains...)
		nsrecords = append(nsrecords, results.Data...)

		if results.More == false {
			break
		}
		domainSearch.Page++
	}

	r.Tags = plugin.TValue[[]interface{}]{Data: convert.SliceAnyToInterface(tags), Error: nil, State: plugin.StateIsSet}
	r.Subdomains = plugin.TValue[[]interface{}]{Data: convert.SliceAnyToInterface(subdomains), Error: nil, State: plugin.StateIsSet}

	var mqlNsRecords []interface{}
	for _, nsrecord := range nsrecords {
		lastSeen := llx.NilData

		t, err := time.Parse(time.RFC3339, nsrecord.LastSeen)
		if err == nil {
			lastSeen = llx.TimeData(t)
		}

		recordResource, err := CreateResource(r.MqlRuntime, "shodan.nsrecord", map[string]*llx.RawData{
			"domain":    llx.StringData(domain),
			"subdomain": llx.StringData(nsrecord.Subdomain),
			"type":      llx.StringData(nsrecord.Type),
			"value":     llx.StringData(nsrecord.Value),
			"lastSeen":  lastSeen,
		})
		if err != nil {
			return err
		}
		mqlNsRecords = append(mqlNsRecords, recordResource)
	}
	r.Nsrecords = plugin.TValue[[]interface{}]{Data: mqlNsRecords, Error: nil, State: plugin.StateIsSet}

	return nil
}

func (r *mqlShodanDomain) tags() ([]interface{}, error) {
	return nil, r.fetchBaseInformation()
}

func (r *mqlShodanDomain) subdomains() ([]interface{}, error) {
	return nil, r.fetchBaseInformation()
}

func (r *mqlShodanDomain) nsrecords() ([]interface{}, error) {
	return nil, r.fetchBaseInformation()
}

func (r *mqlShodanNsrecord) id() (string, error) {
	id := "shodan.nsrecord/" + r.Domain.Data
	if r.Subdomain.Data != "" {
		id += "/" + r.Subdomain.Data
	}
	if r.Type.Data != "" {
		id += "/" + r.Type.Data
	}
	if r.Value.Data != "" {
		id += "/" + r.Value.Data
	}
	return id, nil
}
