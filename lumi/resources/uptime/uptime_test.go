package uptime_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/lumi/resources/uptime"
	motor "go.mondoo.io/mondoo/motor/motoros"
	"go.mondoo.io/mondoo/motor/motoros/mock/toml"
	"go.mondoo.io/mondoo/motor/motoros/types"
)

func TestUptimeOnLinux(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/linux_uptime.toml")
	trans, err := toml.New(&types.Endpoint{Backend: "mock", Path: filepath})
	if err != nil {
		t.Fatal(err)
	}

	m, err := motor.New(trans)
	if err != nil {
		t.Fatal(err)
	}

	ut, err := uptime.New(m)
	if err != nil {
		t.Fatal(err)
	}

	required, err := ut.Duration()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "19m0s", required.String())
}

func TestUptimeOnWindows(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/win_uptime.toml")
	trans, err := toml.New(&types.Endpoint{Backend: "mock", Path: filepath})
	if err != nil {
		t.Fatal(err)
	}

	m, err := motor.New(trans)
	if err != nil {
		t.Fatal(err)
	}

	ut, err := uptime.New(m)
	if err != nil {
		t.Fatal(err)
	}

	required, err := ut.Duration()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "3m45.8270365s", required.String())
}
