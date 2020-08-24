package resources

import (
	"context"
	"errors"

	msgraphbeta "github.com/yaegashi/msgraph.go/beta"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/motor/transports"
	ms365_transport "go.mondoo.io/mondoo/motor/transports/ms365"
)

func ms365transport(t transports.Transport) (*ms365_transport.Transport, error) {
	at, ok := t.(*ms365_transport.Transport)
	if !ok {
		return nil, errors.New("ms365 resource is not supported on this transport")
	}
	return at, nil
}

func (m *lumiMsgraphBetaSecurity) id() (string, error) {
	return "msgraph.beta.security", nil
}

func msSecureScoreToLumi(runtime *lumi.Runtime, score msgraphbeta.SecureScore) (interface{}, error) {
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

	lumiResource, err := runtime.CreateResource("msgraph.beta.security.securityscore",
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
	return lumiResource, nil
}

func (m *lumiMsgraphBetaSecurity) GetLatestSecureScores() (interface{}, error) {
	mt, err := ms365transport(m.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	graphBetaClient, err := mt.GraphBetaClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	scores, err := graphBetaClient.Security().SecureScores().Request().Get(ctx)
	if err != nil {
		return nil, err
	}

	if len(scores) == 0 {
		return nil, errors.New("could not retrieve any score")
	}

	latestScore := scores[0]
	for i := range scores {
		score := scores[i]
		if score.CreatedDateTime != nil && (latestScore.CreatedDateTime == nil || score.CreatedDateTime.Before(*latestScore.CreatedDateTime)) {
			latestScore = score
		}
	}

	return msSecureScoreToLumi(m.Runtime, latestScore)
}

func (m *lumiMsgraphBetaSecurity) GetSecureScores() ([]interface{}, error) {
	mt, err := ms365transport(m.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	graphBetaClient, err := mt.GraphBetaClient()
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
		lumiResource, err := msSecureScoreToLumi(m.Runtime, score)
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
