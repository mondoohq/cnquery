// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"go.mondoo.com/cnquery/v11/explorer/resources"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers/mondoo/connection"
	"go.mondoo.com/cnquery/v11/utils/multierr"
	"go.mondoo.com/mondoo-go"
)

func (m *mqlMondooSpace) assets() ([]any, error) {
	conn := m.MqlRuntime.Connection.(*connection.Connection)

	var q struct {
		Assets struct {
			TotalCount int `graphql:"totalCount"`
			Edges      []struct {
				Node struct {
					ID        string
					Mrn       string
					State     string
					Name      string
					AssetType string `graphql:"asset_type"`
				}
			}
		} `graphql:"assets(spaceMrn: $spaceMrn)"`
	}
	vars := map[string]any{
		"spaceMrn": mondoogql.String(conn.Upstream.SpaceMrn),
	}

	err := conn.Client.Query(context.Background(), &q, vars)
	if err != nil {
		return nil, err
	}

	res := make([]any, len(q.Assets.Edges))
	for i := range q.Assets.Edges {
		e := q.Assets.Edges[i]
		raw, err := CreateResource(m.MqlRuntime, "mondoo.asset", map[string]*llx.RawData{
			"name":     llx.StringData(e.Node.Name),
			"mrn":      llx.StringData(e.Node.Mrn),
			"platform": llx.StringData(e.Node.AssetType),
		})
		if err != nil {
			return nil, err
		}
		res[i] = raw
	}

	return res, nil
}

func (m *mqlMondooAsset) id() (string, error) {
	return m.Mrn.Data, nil
}

func (m *mqlMondooAsset) resources() ([]any, error) {
	conn := m.MqlRuntime.Connection.(*connection.Connection)
	upstream := conn.Upstream

	explorer, err := resources.NewRemoteServices(upstream.ApiEndpoint, upstream.Plugins, upstream.HttpClient)
	if err != nil {
		return nil, err
	}

	// urecording, err := recording.NewUpstreamRecording(context.Background(), explorer, m.Mrn.Data)
	// if err != nil {
	// 	return nil, err
	// }

	list, err := explorer.ListResources(context.Background(), &resources.ListResourcesReq{
		EntityMrn: m.Mrn.Data,
	})
	if err != nil {
		return nil, multierr.Wrap(err, "failed to list resources for asset "+m.Mrn.Data)
	}

	res := make([]any, len(list.Resources))
	for i := range list.Resources {
		resource := list.Resources[i]
		raw, err := CreateResource(m.MqlRuntime, "mondoo.resource", map[string]*llx.RawData{
			"name": llx.StringData(resource.Resource),
			"id":   llx.StringData(resource.Id),
		})
		if err != nil {
			return nil, multierr.Wrap(err, "failed to initialize resource on Mondoo asset "+m.Mrn.Data)
		}
		res[i] = raw
	}

	return res, nil
}

func (m *mqlMondooResource) id() (string, error) {
	return m.Name.Data + "\x00" + m.Id.Data, nil
}
