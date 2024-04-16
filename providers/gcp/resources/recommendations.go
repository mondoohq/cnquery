// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/gcp/connection"

	"google.golang.org/api/cloudresourcemanager/v1"

	"google.golang.org/api/compute/v1"

	recommender "cloud.google.com/go/recommender/apiv1"
	"cloud.google.com/go/recommender/apiv1/recommenderpb"
	"github.com/rs/zerolog/log"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func newMqlRecommendation(runtime *plugin.Runtime, item *recommenderpb.Recommendation) (*mqlGcpRecommendation, error) {
	category := ""
	if item.PrimaryImpact != nil {
		category = item.PrimaryImpact.Category.String()
	}

	primaryImpact, _ := convert.JsonToDict(item.PrimaryImpact)
	additionalImpact, _ := convert.JsonToDictSlice(item.AdditionalImpact)
	content, _ := convert.JsonToDict(item.Content)
	lastRefreshTime := item.LastRefreshTime.AsTime()
	priority := item.Priority.String()
	state, _ := convert.JsonToDict(item.StateInfo)

	// /projects/{projectid}/locations/{zone}/recommenders/{recommender}/recommendations/{id}
	values := strings.Split(item.Name, "/")

	res, err := CreateResource(runtime, "gcp.recommendation", map[string]*llx.RawData{
		"id":               llx.StringData(values[7]),
		"projectId":        llx.StringData(values[1]),
		"zoneName":         llx.StringData(values[3]),
		"name":             llx.StringData(item.Description),
		"recommender":      llx.StringData(values[5]),
		"primaryImpact":    llx.DictData(primaryImpact),
		"additionalImpact": llx.DictData(additionalImpact),
		"content":          llx.DictData(content),
		"category":         llx.StringData(category),
		"priority":         llx.StringData(priority),
		"lastRefreshTime":  llx.TimeData(lastRefreshTime),
		"state":            llx.DictData(state),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpRecommendation), nil
}

// https://cloud.google.com/recommender/docs/recommenders#recommenders
var recommenders = []string{
	"google.bigquery.capacityCommitments.Recommender",
	"google.cloudsql.instance.IdleRecommender",
	"google.cloudsql.instance.OverprovisionedRecommender",
	"google.compute.commitment.UsageCommitmentRecommender",
	//"google.cloudbilling.commitment.SpendBasedCommitmentRecommender", // API returns errors with that recommender on project level
	"google.compute.image.IdleResourceRecommender",
	"google.compute.address.IdleResourceRecommender",
	"google.compute.disk.IdleResourceRecommender",
	"google.compute.instance.IdleResourceRecommender",
	//"google.accounts.security.SecurityKeyRecommender", // API returns errors with that recommender on project level
	"google.iam.policy.Recommender",
	"google.gmp.project.ManagementRecommender",
	"google.run.service.IdentityRecommender",
	"google.run.service.SecurityRecommender",
	"google.cloudsql.instance.OutOfDiskRecommender",
	"google.compute.instanceGroupManager.MachineTypeRecommender",
	"google.compute.instance.MachineTypeRecommender",
	"google.clouderrorreporting.Recommender",
	"google.logging.productSuggestion.ContainerRecommender",
	"google.container.DiagnosisRecommender",
	"google.resourcemanager.projectUtilization.Recommender",
}

// GetRecommendations returns recommendations from Google Cloud
func (g *mqlGcpProject) recommendations() ([]interface{}, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	projectId := g.Id.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	client, err := conn.Client(cloudresourcemanager.CloudPlatformReadOnlyScope)
	if err != nil {
		return nil, err
	}

	// get all zones
	ctx := context.Background()
	computeSvc, err := compute.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	zones := []*compute.Zone{}
	req := computeSvc.Zones.List(projectId)
	if err := req.Pages(ctx, func(page *compute.ZoneList) error {
		for _, zone := range page.Items {
			zones = append(zones, zone)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	// gather all recommendations
	credentials, err := conn.Credentials(recommender.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	c, err := recommender.NewClient(ctx, option.WithCredentials(credentials))
	if err != nil {
		log.Info().Err(err).Msg("could not create client")
		return nil, err
	}

	res := []interface{}{}
	var wg sync.WaitGroup
	wg.Add(len(zones))
	mux := &sync.Mutex{}

	for i := range zones {
		zoneName := zones[i].Name
		// we run a worker routine per zone
		go func(zoneNameValue string) {
			for j := range recommenders {
				recommender := recommenders[j]

				parent := fmt.Sprintf("projects/%s/locations/%s/recommenders/%s", projectId, zoneNameValue, recommender)
				it := c.ListRecommendations(ctx, &recommenderpb.ListRecommendationsRequest{
					Parent: parent,
				})

				for {
					item, err := it.Next()
					if err == iterator.Done {
						break
					}
					if err != nil {
						log.Error().Str("parent", parent).Err(err).Msg("could not request recommendations")
						break
					}

					mqlRecommendation, err := newMqlRecommendation(g.MqlRuntime, item)
					if err != nil {
						log.Error().Str("parent", parent).Err(err).Msg("could not create mql recommendation")
						break
					}
					mux.Lock()
					res = append(res, mqlRecommendation)
					mux.Unlock()
				}
			}
			wg.Done()
		}(zoneName)
	}
	wg.Wait()
	return res, nil
}

func (g *mqlGcpRecommendation) id() (string, error) {
	if g.Id.Error != nil {
		return "", g.Id.Error
	}

	return "gcp.recommendation/" + g.Id.Data, nil
}
