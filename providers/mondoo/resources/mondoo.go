// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"time"

	"go.mondoo.com/cnquery/v11/explorer/resources"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers/mondoo/connection"
	"go.mondoo.com/cnquery/v11/types"
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
					CreatedAt *string
					UpdatedAt *string
					// Annotations map[string]string
					Annotations []keyValue
					Labels      []keyValue
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
			"name":        llx.StringData(e.Node.Name),
			"mrn":         llx.StringData(e.Node.Mrn),
			"platform":    llx.StringData(e.Node.AssetType),
			"annotations": llx.MapData(keyvals2map(e.Node.Annotations), types.Map(types.String, types.String)),
			"labels":      llx.MapData(keyvals2map(e.Node.Labels), types.Map(types.String, types.String)),
			"createdAt":   llx.TimeDataPtr(string2time(e.Node.CreatedAt)),
			"updatedAt":   llx.TimeDataPtr(string2time(e.Node.UpdatedAt)),
		})
		if err != nil {
			return nil, err
		}
		res[i] = raw
	}

	return res, nil
}

type keyValue struct {
	Key   string
	Value *string
}

func keyvals2map(keyvals []keyValue) map[string]any {
	if len(keyvals) == 0 {
		return nil
	}
	res := make(map[string]any, len(keyvals))
	for i := range keyvals {
		cur := keyvals[i]
		if cur.Value == nil {
			res[cur.Key] = ""
		} else {
			res[cur.Key] = *cur.Value
		}
	}
	return res
}

func string2time(s *string) *time.Time {
	if s == nil {
		return nil
	}
	res, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return nil
	}
	return &res
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
