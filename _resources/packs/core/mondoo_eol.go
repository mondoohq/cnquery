package core

import (
	"context"
	"errors"
	"time"

	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/upstream/mvd"
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
	if mcc == nil || mcc.ApiEndpoint == "" {
		return nil, errors.New(MissingUpstreamErr)
	}

	// get new advisory report
	// start scanner client
	scannerClient, err := newAdvisoryScannerHttpClient(mcc.ApiEndpoint, mcc.Plugins, mcc.HttpClient)
	if err != nil {
		return nil, err
	}

	platformEolInfo, err := scannerClient.IsEol(context.Background(), &mvd.Platform{
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
