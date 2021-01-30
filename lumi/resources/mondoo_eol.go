package resources

import (
	"errors"
	"go.mondoo.io/mondoo/llx"
	"go.mondoo.io/mondoo/vadvisor"
	"go.mondoo.io/mondoo/vadvisor/sources/eol"
	"time"
)

func (s *lumiMondooEol) id() (string, error) {
	name, _ := s.Product()
	version, _ := s.Version()

	return "product:" + name + ":" + version, nil
}

func (p *lumiMondooEol) GetDate() (*time.Time, error) {

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
