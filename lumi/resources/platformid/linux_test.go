package platformid

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	motor "go.mondoo.io/mondoo/motor/motoros"
	"go.mondoo.io/mondoo/motor/motoros/mock/toml"
	"go.mondoo.io/mondoo/motor/motoros/types"
)

func TestLinuxMachineId(t *testing.T) {
	filepath, _ := filepath.Abs("./linux_test.toml")
	trans, err := toml.New(&types.Endpoint{Backend: "mock", Path: filepath})
	if err != nil {
		t.Fatal(err)
	}

	m, err := motor.New(trans)
	if err != nil {
		t.Fatal(err)
	}

	lid := LinuxIdProvider{Motor: m}
	id, err := lid.ID()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "39827700b8d246eb9446947c573ecff2", id, "machine id is properly detected")
}
