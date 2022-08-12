//go:build debugtest
// +build debugtest

package ssh_test

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/ssh"
	"go.mondoo.io/mondoo/motor/vault"
)

func TestEc2InstanceConnect(t *testing.T) {
	instanceID := "i-0fed67234fd67e0f2"
	user := "ec2-user"

	endpoint := &providers.TransportConfig{
		Backend: providers.ProviderType_SSH,
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
	endpoint := &providers.TransportConfig{
		Backend: providers.ProviderType_SSH,
		Host:    "192.168.178.26",
		Credentials: []*vault.Credential{{
			Type:   vault.CredentialType_password,
			User:   "chris",
			Secret: []byte("password1!"),
		}},
		Sudo: &providers.Sudo{
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

func TestEc2SSMSession(t *testing.T) {
	instanceID := "i-0335499f012ff1a2b"
	user := "ec2-user"
	profile := "mondoo-dev"
	region := "us-east-1"

	endpoint := &providers.TransportConfig{
		Backend: providers.ProviderType_SSH,
		Host:    instanceID,
		Credentials: []*vault.Credential{{
			Type: vault.CredentialType_aws_ec2_ssm_session,
			User: user,
		}},
		Insecure: true,
		Options: map[string]string{
			"region":  region,
			"profile": profile,
		},
	}

	conn, err := ssh.New(endpoint)
	require.NoError(t, err)

	fi, err := conn.FS().Stat("/etc/os-release")
	require.NoError(t, err)
	assert.NotNil(t, fi)
	f, err := conn.FS().Open("/etc/os-release")
	require.NoError(t, err)
	content, err := ioutil.ReadAll(f)
	require.NoError(t, err)
	assert.NotEqual(t, "", string(content))

	// close ssh connection
	conn.Close()
}
