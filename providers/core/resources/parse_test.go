package resources_test

import (
	"testing"
	"time"

	"go.mondoo.com/cnquery/providers-sdk/v1/testutils"
)

func TestParse_Date(t *testing.T) {
	simpleDate, err := time.Parse("2006-01-02", "2023-12-23")
	if err != nil {
		panic("cannot parse time needed for testing")
	}

	x.TestSimple(t, []testutils.SimpleTest{
		{
			Code:        "parse.date('2023-12-23T00:00:00Z')",
			ResultIndex: 0,
			Expectation: &simpleDate,
		},
		{
			Code:        "parse.date('2023/12/23', '2006/01/02')",
			ResultIndex: 0,
			Expectation: &simpleDate,
		},
		{
			Code:        "parse.date('Mon Dec 23 00:00:00 2023', 'ansic')",
			ResultIndex: 0,
			Expectation: &simpleDate,
		},
	})
}
