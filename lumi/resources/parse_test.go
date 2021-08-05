package resources_test

import (
	"testing"
	"time"
)

func TestParse_Date(t *testing.T) {
	simpleDate, err := time.Parse("2006-01-02", "2023-12-23")
	if err != nil {
		panic("cannot parse time needed for testing")
	}

	runSimpleTests(t, []simpleTest{
		{
			"parse.date('2023-12-23T00:00:00Z')",
			0, &simpleDate,
		},
		{
			"parse.date('2023/12/23', '2006/01/02')",
			0, &simpleDate,
		},
		{
			"parse.date('Mon Dec 23 00:00:00 2023', 'ansic')",
			0, &simpleDate,
		},
	})
}

func TestParsePlist(t *testing.T) {
	runSimpleTests(t, []simpleTest{
		{
			"parse.plist('/dummy.plist').params['allowdownloadsignedenabled']",
			// validates that the output is not uint64
			0, float64(1),
		},
	})
}
