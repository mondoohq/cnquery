package resources

import (
	"context"
	"errors"

	msgraph "github.com/yaegashi/msgraph.go/beta"
	msgraphbeta "github.com/yaegashi/msgraph.go/beta"
	"github.com/yaegashi/msgraph.go/msauth"
	"golang.org/x/oauth2"
)

func graphClient() (*msgraph.GraphServiceRequestBuilder, *msgraphbeta.GraphServiceRequestBuilder, error) {
	// mondoo inc
	tenantID := "<tenant_id>"
	clientID := "<application_id>"
	clientSecret := "<application_secret>"

	var scopes = []string{msauth.DefaultMSGraphScope}

	ctx := context.Background()
	m := msauth.NewManager()
	ts, err := m.ClientCredentialsGrant(ctx, tenantID, clientID, clientSecret, scopes)
	if err != nil {
		return nil, nil, err
	}

	httpClient := oauth2.NewClient(ctx, ts)
	graphClient := msgraph.NewClient(httpClient)
	graphBetaClient := msgraphbeta.NewClient(httpClient)

	return graphClient, graphBetaClient, nil
}

func (m *lumiMsgraphBetaSecurity) id() (string, error) {
	return "msgraph.beta.security", nil
}

func (m *lumiMsgraphBetaSecurity) GetLatestSecureScores() (interface{}, error) {
	return nil, errors.New("not implemented")
}

func (m *lumiMsgraphBetaSecurity) GetSecureScores() ([]interface{}, error) {

	_, graphBetaClient, err := graphClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	scores, err := graphBetaClient.Security().SecureScores().Request().Get(ctx)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range scores {
		score := scores[i]

		averageComparativeScores := []interface{}{}
		for j := range score.AverageComparativeScores {
			entry, err := jsonToDict(score.AverageComparativeScores[j])
			if err != nil {
				return nil, err
			}
			averageComparativeScores = append(averageComparativeScores, entry)
		}

		controlScores := []interface{}{}
		for j := range score.ControlScores {
			entry, err := jsonToDict(score.ControlScores[j])
			if err != nil {
				return nil, err
			}
			controlScores = append(controlScores, entry)
		}

		vendorInformation, err := jsonToDict(score.VendorInformation)
		if err != nil {
			return nil, err
		}

		lumiResource, err := m.Runtime.CreateResource("msgraph.beta.security.securityscore",
			"id", toString(score.ID),
			"activeUserCount", toInt(score.ActiveUserCount),
			"averageComparativeScores", averageComparativeScores,
			"azureTenantId", toString(score.AzureTenantID),
			"controlScores", controlScores,
			"createdDateTime", score.CreatedDateTime,
			"currentScore", toFloat64(score.CurrentScore),
			"enabledServices", strSliceToInterface(score.EnabledServices),
			"licensedUserCount", toInt(score.LicensedUserCount),
			"maxScore", toFloat64(score.MaxScore),
			"vendorInformation", vendorInformation,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiResource)
	}
	return res, nil
}

func (s *lumiMsgraphBetaSecuritySecurityscore) id() (string, error) {
	return s.Id()
}
