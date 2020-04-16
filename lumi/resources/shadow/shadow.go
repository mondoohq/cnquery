package shadow

import (
	"encoding/csv"
	"io"
	"strings"
)

type ShadowEntry struct {
	User         string
	Password     string
	LastChanges  string
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
		res = append(res, ShadowEntry{
			User:         strings.TrimSpace(record[0]),
			Password:     strings.TrimSpace(record[1]),
			LastChanges:  strings.TrimSpace(record[2]),
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
