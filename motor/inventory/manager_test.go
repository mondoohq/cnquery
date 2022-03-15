package inventory

import (
	"testing"

	v1 "go.mondoo.io/mondoo/motor/inventory/v1"

	"github.com/stretchr/testify/require"
)

func TestInventoryIdempotent(t *testing.T) {
	v1inventory, err := v1.InventoryFromFile("./v1/testdata/k8s_mount.yaml")
	require.NoError(t, err)

	im, err := New(WithInventory(v1inventory))
	require.NoError(t, err)
	// runs resolve step, especially the creds resolution
	im.Resolve()

	im, err = New(WithInventory(v1inventory))
	require.NoError(t, err)
	// runs resolve step, especially the creds resolution
	im.Resolve()
}
