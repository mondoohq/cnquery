package shadow

import (
	"encoding/csv"
	"io"
	"strconv"
	"strings"
	"time"
)

type ShadowEntry struct {
	User         string
	Password     string
	LastChanged  *time.Time
	MinDays      string
	MaxDays      string
	WarnDays     string
	InactiveDays string
	ExpiryDates  string
	Reserved     string
}

func ParseShadow(r io.Reader) ([]ShadowEntry, error) {
	res := []ShadowEntry{}

	csvReader := csv.NewReader(r)
	csvReader.Comma = ':'
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		// the /etc/shadow file gives the count of days since jan 1, 1970
		start := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
		var lastChangedTime *time.Time
		if record[2] == "" {
			// if the last_changes field is an empty string, nothing was ever changed, return nil
			lastChangedTime = nil
		} else {
			i, err := strconv.Atoi(record[2])
			if err != nil {
				return nil, err
			}
			date := start.Add(time.Hour * 24 * time.Duration(i))
			lastChangedTime = &date
		}
		res = append(res, ShadowEntry{
			User:         strings.TrimSpace(record[0]),
			Password:     strings.TrimSpace(record[1]),
			LastChanged:  lastChangedTime,
			MinDays:      strings.TrimSpace(record[3]),
			MaxDays:      strings.TrimSpace(record[4]),
			WarnDays:     strings.TrimSpace(record[5]),
			InactiveDays: strings.TrimSpace(record[6]),
			ExpiryDates:  strings.TrimSpace(record[7]),
			Reserved:     strings.TrimSpace(record[8]),
		})
	}

	return res, nil
}
