package inventory

import (
	"context"
	"testing"

	v1 "go.mondoo.com/cnquery/motor/inventory/v1"

	"github.com/stretchr/testify/require"
)

func TestInventoryIdempotent(t *testing.T) {
	v1inventory, err := v1.InventoryFromFile("./v1/testdata/k8s_mount.yaml")
	require.NoError(t, err)

	ctx := context.Background()

	im, err := New(WithInventory(v1inventory))
	require.NoError(t, err)
	// runs resolve step, especially the creds resolution
	im.Resolve(ctx)

	im, err = New(WithInventory(v1inventory))
	require.NoError(t, err)
	// runs resolve step, especially the creds resolution
	im.Resolve(ctx)
}
