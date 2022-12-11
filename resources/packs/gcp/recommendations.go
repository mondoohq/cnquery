package gcp

import (
	"context"
	"fmt"

	recommender "cloud.google.com/go/recommender/apiv1"
	"cloud.google.com/go/recommender/apiv1/recommenderpb"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/resources/packs/core"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func (g *mqlGcpProject) GetRecommendations() ([]interface{}, error) {
	projectId, err := g.Id()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	credentials, err := provider.Credentials(recommender.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	c, err := recommender.NewClient(ctx, option.WithCredentials(credentials))
	if err != nil {
		log.Info().Err(err).Msg("could not create client")
		return nil, err
	}

	parent := fmt.Sprintf("projects/%s/locations/%s/recommenders/%s", projectId, "us-central1-a", "google.compute.instance.IdleResourceRecommender")

	it := c.ListRecommendations(ctx, &recommenderpb.ListRecommendationsRequest{
		Parent: parent,
	})

	res := []interface{}{}
	for {
		item, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		category := ""
		if item.PrimaryImpact != nil {
			category = item.PrimaryImpact.Category.String()
		}

		primaryImpact, _ := core.JsonToDict(item.PrimaryImpact)
		additionalImpact, _ := core.JsonToDictSlice(item.AdditionalImpact)
		content, _ := core.JsonToDict(item.Content)
		lastRefreshTime := item.LastRefreshTime.AsTime()
		priority := item.Priority.String()
		state, _ := core.JsonToDict(item.StateInfo)

		mqlService, err := g.MotorRuntime.CreateResource("gcp.recommendation",
			"id", item.Name,
			"name", item.Description,
			"primaryImpact", primaryImpact,
			"additionalImpact", additionalImpact,
			"content", content,
			"category", category,
			"priority", priority,
			"lastRefreshTime", &lastRefreshTime,
			"state", state,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlService)
	}

	return res, nil
}

func (g *mqlGcpRecommendation) id() (string, error) {
	name, err := g.Id()
	if err != nil {
		return "", err
	}

	return "gcp.recommendation/" + name, nil
}
