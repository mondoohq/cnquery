//go:build debugtest
// +build debugtest

package googleworkspace

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/motor"
	"go.mondoo.com/cnquery/motor/providers"
	google_provider "go.mondoo.com/cnquery/motor/providers/google"
	"go.mondoo.com/cnquery/resources/packs/testutils"
)

var x = testutils.InitTester(googleWorkspaceProvider(), Registry)

func googleWorkspaceProvider() *motor.Motor {
	provider, err := google_provider.New(&providers.Config{
		Backend: providers.ProviderType_GOOGLE_WORKSPACE,
		Options: map[string]string{
			"customer-id": "<add-here>",
		},
	})
	if err != nil {
		panic(err.Error())
	}

	m, err := motor.New(provider)
	if err != nil {
		panic(err.Error())
	}

	return m
}

func TestResource_Domain(t *testing.T) {
	res := x.TestQuery(t, "googleworkspace.users")
	assert.NotEmpty(t, res)
}
