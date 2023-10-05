// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources_test

import (
	"testing"

	"go.mondoo.com/cnquery/v9/providers-sdk/v1/testutils"
)

func TestUUID(t *testing.T) {
	x.TestSimple(t, []testutils.SimpleTest{
		{
			Code:        "uuid('6ba7b810-9dad-11d1-80b4-00c04fd430c8').value",
			ResultIndex: 0,
			Expectation: "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
		},
		{
			Code:        "uuid('6ba7b810-9dad-11d1-80b4-00c04fd430c8').variant",
			ResultIndex: 0,
			Expectation: "RFC4122",
		},
		{
			Code:        "uuid('6ba7b810-9dad-11d1-80b4-00c04fd430c8').version",
			ResultIndex: 0,
			Expectation: int64(1),
		},
	})
}
