package domainlist

import (
	"os"
	"testing"

	"go.mondoo.com/cnquery/motor/providers"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestParseInventory(t *testing.T) {
	f, err := os.Open("./testdata/input.txt")
	assert.Nil(t, err)
	defer f.Close()

	inventory, err := Parse(f)
	require.NoError(t, err)
	assert.Equal(t, inventory.Hosts, []string{"example.com:443", "my-example.com:4443", "sub.example.com:8443", "my-example.com:8443", "anotherdomain.com"})

	out := inventory.ToV1Inventory()
	assert.Equal(t, 5, len(out.Spec.Assets))
	assert.Equal(t, "example.com:443", out.Spec.Assets[0].Name)
	assert.Equal(t, "example.com", out.Spec.Assets[0].Connections[0].Host)
	assert.Equal(t, int32(443), out.Spec.Assets[0].Connections[0].Port)
	assert.Equal(t, providers.ProviderType_HOST, out.Spec.Assets[0].Connections[0].Backend)
}
