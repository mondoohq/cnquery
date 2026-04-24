// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"

	bqreservation "cloud.google.com/go/bigquery/reservation/apiv1"
	"cloud.google.com/go/bigquery/reservation/apiv1/reservationpb"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/gcp/connection"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func (g *mqlGcpProjectBigqueryService) reservations() ([]any, error) {
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
	creds, err := conn.Credentials(bqreservation.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := bqreservation.NewClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	var res []any
	for _, location := range locations {
		it := client.ListReservations(ctx, &reservationpb.ListReservationsRequest{
			Parent: fmt.Sprintf("projects/%s/locations/%s", projectId, location),
		})
		for {
			r, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				if isGRPCSkippable(err) {
					log.Warn().Err(err).Str("location", location).Msg("could not list BigQuery reservations")
					break
				}
				return nil, err
			}

			autoscale, err := protoToDict(r.Autoscale)
			if err != nil {
				return nil, err
			}

			var created, updated *llx.RawData
			if r.CreationTime != nil {
				created = llx.TimeData(r.CreationTime.AsTime())
			} else {
				created = llx.NilData
			}
			if r.UpdateTime != nil {
				updated = llx.TimeData(r.UpdateTime.AsTime())
			} else {
				updated = llx.NilData
			}

			mqlRes, err := CreateResource(g.MqlRuntime, "gcp.project.bigqueryService.reservation", map[string]*llx.RawData{
				"name":                    llx.StringData(r.Name),
				"projectId":               llx.StringData(projectId),
				"location":                llx.StringData(location),
				"slotCapacity":            llx.IntData(r.SlotCapacity),
				"ignoreIdleSlots":         llx.BoolData(r.IgnoreIdleSlots),
				"autoscale":               llx.DictData(autoscale),
				"concurrency":             llx.IntData(r.Concurrency),
				"edition":                 llx.StringData(r.Edition.String()),
				"primaryLocation":         llx.StringData(r.PrimaryLocation),
				"secondaryLocation":       llx.StringData(r.SecondaryLocation),
				"originalPrimaryLocation": llx.StringData(r.OriginalPrimaryLocation),
				"created":                 created,
				"updated":                 updated,
			})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlRes)
		}
	}
	return res, nil
}

func (g *mqlGcpProjectBigqueryServiceReservation) id() (string, error) {
	return g.Name.Data, g.Name.Error
}
