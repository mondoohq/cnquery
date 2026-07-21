// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"
	"time"

	bqconnection "cloud.google.com/go/bigquery/connection/apiv1"
	"cloud.google.com/go/bigquery/connection/apiv1/connectionpb"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/gcp/connection"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// bigqueryLocations returns the unique set of locations discovered from the
// project's datasets. BigQuery connections and reservations are location-scoped
// and there is no project-wide list endpoint, so we use the dataset locations
// as the set of locations to query. A project with no datasets in a region will
// also have no connections/reservations there in practice.
func (g *mqlGcpProjectBigqueryService) bigqueryLocations() ([]string, error) {
	datasets := g.GetDatasets()
	if datasets.Error != nil {
		return nil, datasets.Error
	}

	seen := make(map[string]struct{}, len(datasets.Data))
	locations := make([]string, 0, len(datasets.Data))
	for _, d := range datasets.Data {
		ds := d.(*mqlGcpProjectBigqueryServiceDataset)
		if ds.Location.Error != nil || ds.Location.Data == "" {
			continue
		}
		// BigQuery connections/reservations APIs expect uppercase for
		// multi-regions ("US", "EU") but the REST dataset metadata returns
		// them uppercase already. Regional locations are lowercase in both.
		loc := ds.Location.Data
		if _, ok := seen[loc]; ok {
			continue
		}
		seen[loc] = struct{}{}
		locations = append(locations, loc)
	}
	return locations, nil
}

func (g *mqlGcpProjectBigqueryService) connections() ([]any, error) {
	if !g.serviceEnabled {
		return nil, nil
	}
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	locations, err := g.bigqueryLocations()
	if err != nil {
		return nil, err
	}
	if len(locations) == 0 {
		return nil, nil
	}

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(bqconnection.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := bqconnection.NewClient(ctx, option.WithCredentials(creds), connection.GRPCClientTraceOption())
	if err != nil {
		return nil, err
	}
	defer client.Close()

	var res []any
	for _, location := range locations {
		it := client.ListConnections(ctx, &connectionpb.ListConnectionsRequest{
			Parent: fmt.Sprintf("projects/%s/locations/%s", projectId, location),
		})
		for {
			c, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				if isGRPCSkippable(err) {
					log.Warn().Err(err).Str("location", location).Msg("could not list BigQuery connections")
					break
				}
				return nil, err
			}

			connectionType, properties, err := connectionProperties(c)
			if err != nil {
				return nil, err
			}

			var created, modified *llx.RawData
			if c.CreationTime != 0 {
				created = llx.TimeData(time.UnixMilli(c.CreationTime))
			} else {
				created = llx.NilData
			}
			if c.LastModifiedTime != 0 {
				modified = llx.TimeData(time.UnixMilli(c.LastModifiedTime))
			} else {
				modified = llx.NilData
			}

			mqlConn, err := CreateResource(g.MqlRuntime, "gcp.project.bigqueryService.connection", map[string]*llx.RawData{
				"name":          llx.StringData(c.Name),
				"projectId":     llx.StringData(projectId),
				"location":      llx.StringData(location),
				"friendlyName":  llx.StringData(c.FriendlyName),
				"description":   llx.StringData(c.Description),
				"type":          llx.StringData(connectionType),
				"properties":    llx.DictData(properties),
				"created":       created,
				"modified":      modified,
				"hasCredential": llx.BoolData(c.HasCredential),
			})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlConn)
		}
	}
	return res, nil
}

// connectionProperties returns the connection type and a dict of the
// properties payload for the configured oneof variant.
func connectionProperties(c *connectionpb.Connection) (string, map[string]any, error) {
	switch p := c.Properties.(type) {
	case *connectionpb.Connection_CloudSql:
		d, err := protoToDict(p.CloudSql)
		return "CLOUD_SQL", d, err
	case *connectionpb.Connection_Aws:
		d, err := protoToDict(p.Aws)
		return "AWS", d, err
	case *connectionpb.Connection_Azure:
		d, err := protoToDict(p.Azure)
		return "AZURE", d, err
	case *connectionpb.Connection_CloudSpanner:
		d, err := protoToDict(p.CloudSpanner)
		return "CLOUD_SPANNER", d, err
	case *connectionpb.Connection_CloudResource:
		d, err := protoToDict(p.CloudResource)
		return "CLOUD_RESOURCE", d, err
	case *connectionpb.Connection_Spark:
		d, err := protoToDict(p.Spark)
		return "SPARK", d, err
	case *connectionpb.Connection_SalesforceDataCloud:
		d, err := protoToDict(p.SalesforceDataCloud)
		return "SALESFORCE_DATA_CLOUD", d, err
	case nil:
		return "UNKNOWN", nil, nil
	default:
		return "UNKNOWN", nil, nil
	}
}

func (g *mqlGcpProjectBigqueryServiceConnection) id() (string, error) {
	return g.Name.Data, g.Name.Error
}
