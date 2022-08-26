package core

import (
	"time"

	"go.mondoo.com/cnquery/llx"
)

func (p *mqlTime) id() (string, error) {
	return "time", nil
}

func (p *mqlTime) GetNow() (*time.Time, error) {
	// TODO: needs a ticking event where the time gets updated
	res := time.Now()
	return &res, nil
}

var (
	second = llx.DurationToTime(1)
	minute = llx.DurationToTime(60)
	hour   = llx.DurationToTime(60 * 60)
	day    = llx.DurationToTime(24 * 60 * 60)
)

func (p *mqlTime) GetSecond() (*time.Time, error) {
	return &second, nil
}

func (p *mqlTime) GetMinute() (*time.Time, error) {
	return &minute, nil
}

func (p *mqlTime) GetHour() (*time.Time, error) {
	return &hour, nil
}

func (p *mqlTime) GetDay() (*time.Time, error) {
	return &day, nil
}

func (p *mqlTime) GetToday() (*time.Time, error) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	return &today, nil
}

func (p *mqlTime) GetTomorrow() (*time.Time, error) {
	cur, _ := p.GetToday()
	res := cur.Add(24 * time.Hour)

	return &res, nil
}
