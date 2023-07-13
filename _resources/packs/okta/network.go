package okta

import (
	"context"

	"go.mondoo.com/cnquery/resources/packs/core"

	"go.mondoo.com/cnquery/resources"

	"github.com/okta/okta-sdk-golang/v2/okta"

	"github.com/okta/okta-sdk-golang/v2/okta/query"
)

func (o *mqlOkta) GetNetworks() ([]interface{}, error) {
	op, err := oktaProvider(o.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	client := op.Client()

	networkSlice, resp, err := client.NetworkZone.ListNetworkZones(
		ctx,
		query.NewQueryParams(
			query.WithLimit(queryLimit),
		),
	)
	if err != nil {
		return nil, err
	}

	if len(networkSlice) == 0 {
		return nil, nil
	}

	list := []interface{}{}
	appendEntry := func(datalist []*okta.NetworkZone) error {
		for i := range datalist {
			entry := datalist[i]

			r, err := newMqlOktaNetworkZone(o.MotorRuntime, entry)
			if err != nil {
				return err
			}
			list = append(list, r)
		}
		return nil
	}

	err = appendEntry(networkSlice)
	if err != nil {
		return nil, err
	}

	for resp != nil && resp.HasNextPage() {
		var networkSlice []*okta.NetworkZone
		resp, err = resp.Next(ctx, &networkSlice)
		if err != nil {
			return nil, err
		}
		err = appendEntry(networkSlice)
		if err != nil {
			return nil, err
		}
	}
	return list, nil
}

func newMqlOktaNetworkZone(runtime *resources.Runtime, entry *okta.NetworkZone) (interface{}, error) {
	proxies, err := core.JsonToDictSlice(entry.Proxies)
	if err != nil {
		return nil, err
	}

	locations, err := core.JsonToDictSlice(entry.Locations)
	if err != nil {
		return nil, err
	}

	gateways, err := core.JsonToDictSlice(entry.Gateways)
	if err != nil {
		return nil, err
	}

	return runtime.CreateResource("okta.network",
		"id", entry.Id,
		"name", entry.Name,
		"type", entry.Type,
		"created", entry.Created,
		"lastUpdated", entry.LastUpdated,
		"status", entry.Status,
		"system", core.ToBool(entry.System),
		"asns", core.StrSliceToInterface(entry.Asns),
		"usage", entry.Usage,
		"proxyType", entry.ProxyType,
		"proxies", proxies,
		"locations", locations,
		"gateways", gateways,
	)
}

func (o *mqlOktaNetwork) id() (string, error) {
	id, err := o.Id()
	if err != nil {
		return "", err
	}
	return "okta.network/" + id, nil
}
