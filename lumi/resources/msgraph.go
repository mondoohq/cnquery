package resources

import (
	"context"
	"errors"

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

func (m *lumiMsgraphBetaSecurity) GetLatestSecureScores() (interface{}, error) {
	return nil, errors.New("not implemented")
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
