package resources

import (
	"time"

	"go.mondoo.io/mondoo/llx"
)

func (p *lumiTime) id() (string, error) {
	return "time", nil
}

func (p *lumiTime) GetNow() (*time.Time, error) {
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

func (p *lumiTime) GetSecond() (*time.Time, error) {
	return &second, nil
}

func (p *lumiTime) GetMinute() (*time.Time, error) {
	return &minute, nil
}

func (p *lumiTime) GetHour() (*time.Time, error) {
	return &hour, nil
}

func (p *lumiTime) GetDay() (*time.Time, error) {
	return &day, nil
}
