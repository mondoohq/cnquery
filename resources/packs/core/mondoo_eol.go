package core

import (
	"errors"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/vadvisor"
	"go.mondoo.com/cnquery/vadvisor/sources/eol"
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

	platformEolInfo := eol.EolInfo(&vadvisor.Platform{
		Name:    name,
		Release: version,
	})

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
