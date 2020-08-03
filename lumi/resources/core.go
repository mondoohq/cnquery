package resources

import "time"

func (p *lumiTime) id() (string, error) {
	return "time", nil
}

func (p *lumiTime) GetNow() (time.Time, error) {
	// TODO: needs a ticking event where the time gets updated
	return time.Now(), nil
}
