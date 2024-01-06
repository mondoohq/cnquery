// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources_test

import (
	"testing"
	"time"

	"go.mondoo.com/cnquery/v9/llx"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/testutils"
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

func TestParse_Duration(t *testing.T) {
	twoSecs := llx.DurationToTime(2)
	tenMin := llx.DurationToTime(10 * 60)
	threeHours := llx.DurationToTime(3 * 60 * 60)
	thirtyDays := llx.DurationToTime(30 * 60 * 60 * 24)
	sevenYears := llx.DurationToTime(7 * 60 * 60 * 24 * 365)
	x.TestSimple(t, []testutils.SimpleTest{
		{
			Code:        "parse.duration('2')",
			ResultIndex: 0,
			Expectation: &twoSecs,
		},
		{
			Code:        "parse.duration('2seconds')",
			ResultIndex: 0,
			Expectation: &twoSecs,
		},
		{
			Code:        "parse.duration('10min')",
			ResultIndex: 0,
			Expectation: &tenMin,
		},
		{
			Code:        "parse.duration('3h')",
			ResultIndex: 0,
			Expectation: &threeHours,
		},
		{
			Code:        "parse.duration('30day')",
			ResultIndex: 0,
			Expectation: &thirtyDays,
		},
		{
			Code:        "parse.duration('7y')",
			ResultIndex: 0,
			Expectation: &sevenYears,
		},
	})
}
