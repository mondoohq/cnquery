// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/v10/providers/os/connection/mock"
)

func TestRpmUpdateParser(t *testing.T) {
	mock, err := mock.New("./testdata/updates_rpm.toml", nil)
	if err != nil {
		t.Fatal(err)
	}
	c, err := mock.RunCommand("python")
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, err)

	m, err := ParseRpmUpdates(c.Stdout)
	assert.Nil(t, err)
	assert.Equal(t, 8, len(m), "detected the right amount of package updates")

	update := m["python-libs"]
	assert.Equal(t, "python-libs", update.Name, "pkg name detected")
	assert.Equal(t, "", update.Version, "pkg version detected")
	assert.Equal(t, "0:2.7.5-69.el7_5", update.Available, "pkg available version detected")

	update = m["binutils"]
	assert.Equal(t, "binutils", update.Name, "pkg name detected")
	assert.Equal(t, "", update.Version, "pkg version detected")
	assert.Equal(t, "0:2.27-28.base.el7_5.1", update.Available, "pkg available version detected")
}

func TestZypperUpdateParser(t *testing.T) {
	mock, err := mock.New("./testdata/updates_zypper.toml", nil)
	if err != nil {
		t.Fatal(err)
	}
	c, err := mock.RunCommand("zypper -n --xmlout list-updates")
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, err)

	m, err := ParseZypperUpdates(c.Stdout)
	assert.Nil(t, err)
	assert.Equal(t, 22, len(m), "detected the right amount of package updates")

	update := m["aaa_base"]
	assert.Equal(t, "aaa_base", update.Name, "pkg name detected")
	assert.Equal(t, "13.2+git20140911.61c1681-28.3.1", update.Version, "pkg version detected")

	update = m["bash"]
	assert.Equal(t, "bash", update.Name, "pkg name detected")
	assert.Equal(t, "4.3-83.3.1", update.Version, "pkg version detected")
}
