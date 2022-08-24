package core_test

import (
	"testing"
	"time"

	"go.mondoo.com/cnquery/resources/packs/testutils"
)

func TestParse_Date(t *testing.T) {
	simpleDate, err := time.Parse("2006-01-02", "2023-12-23")
	if err != nil {
		panic("cannot parse time needed for testing")
	}

	x.TestSimple(t, []testutils.SimpleTest{
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
	x.TestSimple(t, []testutils.SimpleTest{
		{
			"parse.plist('/dummy.plist').params['allowdownloadsignedenabled']",
			// validates that the output is not uint64
			0, float64(1),
		},
	})
}

func TestParseJson(t *testing.T) {
	x.TestSimple(t, []testutils.SimpleTest{
		{
			"parse.json(content: '{\"a\": 1}').params",
			0,
			map[string]interface{}{"a": float64(1)},
		},
		{
			"parse.json(content: '[{\"a\": 1}]').params[0]",
			0,
			map[string]interface{}{"a": float64(1)},
		},
	})
}
