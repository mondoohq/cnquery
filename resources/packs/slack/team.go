package slack

import "go.mondoo.com/cnquery/resources"

func (o *mqlSlackTeam) id() (string, error) {
	id, err := o.Id()
	if err != nil {
		return "", err
	}
	return "slack.team/" + id, nil
}

// init method for team
func (s *mqlSlackTeam) init(args *resources.Args) (*resources.Args, SlackTeam, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	op, err := slackProvider(s.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}

	client := op.Client()
	teamInfo, err := client.GetTeamInfo()
	if err != nil {
		return nil, nil, err
	}

	(*args)["id"] = teamInfo.ID
	(*args)["name"] = teamInfo.Name
	(*args)["domain"] = teamInfo.Domain
	(*args)["emailDomain"] = teamInfo.EmailDomain

	return args, nil, nil
}
