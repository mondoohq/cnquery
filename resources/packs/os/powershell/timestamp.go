package powershell

import (
	"regexp"
	"strconv"
	"time"
)

var powershellTimestamp = regexp.MustCompile(`Date\((\d+)\)`)

func PSJsonTimestamp(date string) *time.Time {
	// extract unix seconds
	m := powershellTimestamp.FindStringSubmatch(date)
	if len(m) > 0 {
		i, err := strconv.ParseInt(m[1], 10, 64)
		if err != nil {
			return nil
		}

		tm := time.Unix(0, i*int64(time.Millisecond))
		return &tm
	}
	return nil
}
