// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package detector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
)

func TestFamilyParents(t *testing.T) {
	test := []struct {
		Platform string
		Expected []string
	}{
		{
			Platform: "redhat",
			Expected: []string{"os", "unix", "linux", "redhat"},
		},
		{
			Platform: "centos",
			Expected: []string{"os", "unix", "linux", "redhat"},
		},
		{
			Platform: "debian",
			Expected: []string{"os", "unix", "linux", "debian"},
		},
		{
			Platform: "ubuntu",
			Expected: []string{"os", "unix", "linux", "debian"},
		},
	}

	for i := range test {
		assert.Equal(t, test[i].Expected, Family(test[i].Platform), test[i].Platform)
	}
}

func TestIsFamily(t *testing.T) {
	test := []struct {
		Val      bool
		Expected bool
	}{
		{
			Val:      IsFamily("redhat", inventory.FAMILY_LINUX),
			Expected: true,
		},
		{
			Val:      IsFamily("redhat", inventory.FAMILY_UNIX),
			Expected: true,
		},
		{
			Val:      IsFamily("redhat", "redhat"),
			Expected: true,
		},
		{
			Val:      IsFamily("centos", inventory.FAMILY_LINUX),
			Expected: true,
		},
		{
			Val:      IsFamily("centos", "redhat"),
			Expected: true,
		},
	}

	for i := range test {
		assert.Equal(t, test[i].Expected, test[i].Val, i)
	}
}

func TestFamilies(t *testing.T) {
	di := &inventory.Platform{}
	di.Family = []string{"unix", "bsd", "darwin"}

	assert.Equal(t, true, di.IsFamily("unix"), "unix should be a family")
	assert.Equal(t, true, di.IsFamily("bsd"), "bsd should be a family")
	assert.Equal(t, true, di.IsFamily("darwin"), "darwin should be a family")
}
