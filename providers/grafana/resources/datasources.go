// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
)

// grafanaDatasourceJSON mirrors one element of the /api/datasources response.
type grafanaDatasourceJSON struct {
	ID        int             `json:"id"`
	UID       string          `json:"uid"`
	OrgID     int             `json:"orgId"`
	Name      string          `json:"name"`
	Type      string          `json:"type"`
	Access    string          `json:"access"`
	URL       string          `json:"url"`
	IsDefault bool            `json:"isDefault"`
	ReadOnly  bool            `json:"readOnly"`
	BasicAuth bool            `json:"basicAuth"`
	JSONData  json.RawMessage `json:"jsonData"`
}

func (g *mqlGrafana) datasources() ([]interface{}, error) {
	conn, err := grafanaConnection(g.MqlRuntime)
	if err != nil {
		return nil, err
	}
	resp, err := conn.Get(context.Background(), "/api/datasources")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("grafana: GET /api/datasources returned status %d", resp.StatusCode)
	}

	var raw []grafanaDatasourceJSON
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("grafana: decoding /api/datasources response: %w", err)
	}

	list := make([]interface{}, 0, len(raw))
	for _, ds := range raw {
		var jsonDataDict interface{}
		if len(ds.JSONData) > 0 && string(ds.JSONData) != "null" {
			var parsed interface{}
			if err := json.Unmarshal(ds.JSONData, &parsed); err != nil {
				return nil, fmt.Errorf("grafana: parsing jsonData for datasource %s: %w", ds.UID, err)
			}
			jsonDataDict, err = convert.JsonToDict(parsed)
			if err != nil {
				return nil, fmt.Errorf("grafana: converting jsonData for datasource %s: %w", ds.UID, err)
			}
		}

		res, err := CreateResource(g.MqlRuntime, "grafana.datasource", map[string]*llx.RawData{
			"id":        llx.IntData(int64(ds.ID)),
			"uid":       llx.StringData(ds.UID),
			"orgId":     llx.IntData(int64(ds.OrgID)),
			"name":      llx.StringData(ds.Name),
			"type":      llx.StringData(ds.Type),
			"access":    llx.StringData(ds.Access),
			"url":       llx.StringData(ds.URL),
			"isDefault": llx.BoolData(ds.IsDefault),
			"readOnly":  llx.BoolData(ds.ReadOnly),
			"basicAuth": llx.BoolData(ds.BasicAuth),
			"jsonData":  llx.DictData(jsonDataDict),
		})
		if err != nil {
			return nil, err
		}
		list = append(list, res)
	}
	return list, nil
}

func (d *mqlGrafanaDatasource) id() (string, error) {
	return "grafana-ds/" + d.Uid.Data, nil
}
