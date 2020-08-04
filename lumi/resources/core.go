package resources

import (
	"time"

	"go.mondoo.io/mondoo/llx"
)

func (p *lumiTime) id() (string, error) {
	return "time", nil
}

func (p *lumiTime) GetNow() (time.Time, error) {
	// TODO: needs a ticking event where the time gets updated
	return time.Now(), nil
}

var (
	second = time.Unix(1+llx.ZeroTimeOffset, 0)
	minute = time.Unix(60+llx.ZeroTimeOffset, 0)
	hour   = time.Unix(60*60+llx.ZeroTimeOffset, 0)
	day    = time.Unix(24*60*60+llx.ZeroTimeOffset, 0)
)

func (p *lumiTime) GetSecond() (time.Time, error) {
	return second, nil
}

func (p *lumiTime) GetMinute() (time.Time, error) {
	return minute, nil
}

func (p *lumiTime) GetHour() (time.Time, error) {
	return hour, nil
}

func (p *lumiTime) GetDay() (time.Time, error) {
	return day, nil
}
