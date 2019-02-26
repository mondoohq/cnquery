package sshd_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/lumi/resources/sshd"
	"go.mondoo.io/mondoo/motor/mock/toml"
	"go.mondoo.io/mondoo/motor/types"
)

func TestSSHParser(t *testing.T) {
	path := "./sshd_config.toml"
	trans, err := toml.New(&types.Endpoint{Backend: "mock", Path: path})

	f, err := trans.File("/etc/ssh/sshd_config")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	sshParams, err := sshd.Params(f)
	if err != nil {
		t.Fatalf("cannot request file %v", err)
	}

	assert.NotNil(t, sshParams, "params are not nil")

	// assert.Equal(t, "/etc/ssh/ssh_host_rsa_key", sshParams["HostKey"])
	assert.Equal(t, "/etc/ssh/ssh_host_ed25519_key", sshParams["HostKey"])

}
