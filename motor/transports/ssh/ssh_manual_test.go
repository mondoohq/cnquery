// +build debugtest

package ssh_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/ssh"
	"go.mondoo.io/mondoo/motor/vault"
)

func TestEc2InstanceConnect(t *testing.T) {
	instanceID := "i-0fed67234fd67e0f2"
	user := "ec2-user"

	endpoint := &transports.TransportConfig{
		Backend: transports.TransportBackend_CONNECTION_SSH,
		Host:    instanceID,
		Credentials: []*vault.Credential{{
			Type: vault.CredentialType_aws_ec2_instance_connect,
			User: user,
		}},
		Insecure: true,
	}

	err := ssh.VerifyConfig(endpoint)
	assert.Nil(t, err)

	_, err = ssh.New(endpoint)
	require.NoError(t, err)
}

func TestSudoConnect(t *testing.T) {
	endpoint := &transports.TransportConfig{
		Backend: transports.TransportBackend_CONNECTION_SSH,
		Host:    "192.168.178.26",
		Credentials: []*vault.Credential{{
			Type:   vault.CredentialType_password,
			User:   "chris",
			Secret: []byte("password1!"),
		}},
		Sudo: &transports.Sudo{
			Active: true,
		},
		Insecure: true,
	}

	conn, err := ssh.New(endpoint)
	require.NoError(t, err)
	defer conn.Close()

	err = conn.VerifyConnection()
	require.NoError(t, err)

	fi, err := conn.FS().Stat("/etc/os-release")
	require.NoError(t, err)
	assert.NotNil(t, fi)
}
