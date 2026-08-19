// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/fileshares/armfileshares"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The regression this guards: a nil element in the allowed-subnet list would
// become an empty subnet ID, and a typed reference built from one resolves to
// nothing — so a share would appear to admit a subnet that does not exist.
func TestFileShareAllowedSubnetIds(t *testing.T) {
	str := func(s string) *string { return &s }
	subnetA := "/subscriptions/s/resourceGroups/rg/providers/Microsoft.Network/virtualNetworks/vnet/subnets/a"
	subnetB := "/subscriptions/s/resourceGroups/rg/providers/Microsoft.Network/virtualNetworks/vnet/subnets/b"

	t.Run("no public access properties", func(t *testing.T) {
		assert.Empty(t, fileShareAllowedSubnetIds(nil))
	})

	t.Run("no restriction configured", func(t *testing.T) {
		assert.Empty(t, fileShareAllowedSubnetIds(&armfileshares.PublicAccessProperties{}))
	})

	t.Run("nil and empty entries are dropped", func(t *testing.T) {
		res := fileShareAllowedSubnetIds(&armfileshares.PublicAccessProperties{
			AllowedSubnets: []*string{nil, str(""), str(subnetA)},
		})
		assert.Equal(t, []string{subnetA}, res)
	})

	t.Run("every admitted subnet is reported, in order", func(t *testing.T) {
		res := fileShareAllowedSubnetIds(&armfileshares.PublicAccessProperties{
			AllowedSubnets: []*string{str(subnetA), str(subnetB)},
		})
		assert.Equal(t, []string{subnetA, subnetB}, res)
	})
}

// The regression this guards: a share that states neither NFS protection has to
// read as unset, not as disabled. Collapsing the two would report a share
// nobody configured as one where encryption in transit was deliberately turned
// off, and vice versa on a policy written the other way round.
func TestFileShareNfsProtocol(t *testing.T) {
	enc := func(e armfileshares.EncryptionInTransitRequired) *armfileshares.EncryptionInTransitRequired {
		return &e
	}
	squash := func(s armfileshares.ShareRootSquash) *armfileshares.ShareRootSquash { return &s }

	t.Run("no NFS properties", func(t *testing.T) {
		encryption, rootSquash := fileShareNfsProtocol(nil)
		assert.Equal(t, "", encryption)
		assert.Equal(t, "", rootSquash)
	})

	t.Run("properties present but both unset", func(t *testing.T) {
		encryption, rootSquash := fileShareNfsProtocol(&armfileshares.NfsProtocolProperties{})
		assert.Equal(t, "", encryption)
		assert.Equal(t, "", rootSquash)
	})

	t.Run("the hardened combination", func(t *testing.T) {
		encryption, rootSquash := fileShareNfsProtocol(&armfileshares.NfsProtocolProperties{
			EncryptionInTransitRequired: enc(armfileshares.EncryptionInTransitRequiredEnabled),
			RootSquash:                  squash(armfileshares.ShareRootSquashAllSquash),
		})
		assert.Equal(t, "Enabled", encryption)
		assert.Equal(t, "AllSquash", rootSquash)
	})

	t.Run("the exposed combination", func(t *testing.T) {
		encryption, rootSquash := fileShareNfsProtocol(&armfileshares.NfsProtocolProperties{
			EncryptionInTransitRequired: enc(armfileshares.EncryptionInTransitRequiredDisabled),
			RootSquash:                  squash(armfileshares.ShareRootSquashNoRootSquash),
		})
		assert.Equal(t, "Disabled", encryption)
		assert.Equal(t, "NoRootSquash", rootSquash)
	})

	t.Run("each is read independently", func(t *testing.T) {
		encryption, rootSquash := fileShareNfsProtocol(&armfileshares.NfsProtocolProperties{
			RootSquash: squash(armfileshares.ShareRootSquashRootSquash),
		})
		require.Equal(t, "", encryption)
		assert.Equal(t, "RootSquash", rootSquash)
	})
}
