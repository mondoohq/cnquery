package core

import (
	"context"
	"errors"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/resources/packs/core/vadvisor"
	"go.mondoo.com/ranger-rpc"
	"time"
)

func (s *mqlMondooEol) id() (string, error) {
	name, _ := s.Product()
	version, _ := s.Version()

	return "product:" + name + ":" + version, nil
}

func (p *mqlMondooEol) GetDate() (*time.Time, error) {

	name, _ := p.Product()
	version, _ := p.Version()

	r := p.MotorRuntime
	mcc := r.UpstreamConfig
	if mcc == nil {
		return nil, errors.New("mondoo upstream configuration is missing")
	}

	// get new advisory report
	// start scanner client
	scannerClient, err := newAdvisoryScannerHttpClient(mcc.ApiEndpoint, mcc.Plugins, ranger.DefaultHttpClient())
	if err != nil {
		return nil, err
	}

	platformEolInfo, err := scannerClient.IsEol(context.Background(), &vadvisor.Platform{
		Name:    name,
		Release: version,
	})
	if err != nil {
		return nil, err
	}

	if platformEolInfo == nil {
		return nil, errors.New("no platform eol information available")
	}

	var eolDate *time.Time

	if platformEolInfo.EolDate != "" {
		parsedEolDate, err := time.Parse(time.RFC3339, platformEolInfo.EolDate)
		if err != nil {
			return nil, errors.New("could not parse eol date: " + platformEolInfo.EolDate)
		}
		eolDate = &parsedEolDate
	} else {
		eolDate = &llx.NeverFutureTime
	}

	return eolDate, nil
}
