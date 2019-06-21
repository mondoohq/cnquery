package awsec2_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.mondoo.io/mondoo/motor/motorid/awsec2"
	motor "go.mondoo.io/mondoo/motor/motoros"
	"go.mondoo.io/mondoo/motor/motoros/mock/toml"
	"go.mondoo.io/mondoo/motor/motoros/types"
)

func TestEC2RoleProviderInstanceIdentityUnix(t *testing.T) {
	trans, err := toml.New(&types.Endpoint{Backend: "mock", Path: "./testdata/instance-identity_document_linux.toml"})
	if err != nil {
		t.Fatal(err)
	}

	m, err := motor.New(trans)
	if err != nil {
		t.Fatal(err)
	}

	metadata := awsec2.NewUnix(m)
	mrn, err := metadata.InstanceID()
	assert.Nil(t, err)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/aws/ec2/v1/accounts/123456789012/regions/us-west-2/instances/i-1234567890abcdef0", mrn)

}
