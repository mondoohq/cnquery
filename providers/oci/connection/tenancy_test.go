// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"sync"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/identity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testCompartment(id, name string) identity.Compartment {
	return identity.Compartment{Id: common.String(id), Name: common.String(name)}
}

// seededConnection is a connection whose compartment tree has already been
// fetched, so the lookups below never reach the Identity API.
func seededConnection(compartments ...identity.Compartment) *OciConnection {
	return &OciConnection{
		compartmentList:  compartments,
		compartmentsDone: true,
	}
}

func TestCompartmentIndexByID(t *testing.T) {
	root := testCompartment("ocid1.tenancy.oc1..root", "tenancy")
	prod := testCompartment("ocid1.compartment.oc1..prod", "prod")

	t.Run("keys every compartment by its ocid", func(t *testing.T) {
		index := compartmentIndexByID([]identity.Compartment{root, prod})

		require.Len(t, index, 2)
		assert.Equal(t, "tenancy", *index["ocid1.tenancy.oc1..root"].Name)
		assert.Equal(t, "prod", *index["ocid1.compartment.oc1..prod"].Name)
	})

	t.Run("no compartments", func(t *testing.T) {
		assert.Empty(t, compartmentIndexByID(nil))
	})

	t.Run("an entry without an ocid is skipped, not indexed under the empty key", func(t *testing.T) {
		// An unkeyed entry stored under "" would be handed back for any id the
		// caller passes that also happens to be empty, reporting the wrong
		// compartment rather than a miss.
		index := compartmentIndexByID([]identity.Compartment{
			{Name: common.String("no id")},
			{Id: common.String(""), Name: common.String("empty id")},
			prod,
		})

		require.Len(t, index, 1)
		assert.Contains(t, index, "ocid1.compartment.oc1..prod")
	})

	t.Run("a repeated ocid keeps the first entry", func(t *testing.T) {
		index := compartmentIndexByID([]identity.Compartment{
			prod,
			testCompartment("ocid1.compartment.oc1..prod", "prod-again"),
		})

		require.Len(t, index, 1)
		assert.Equal(t, "prod", *index["ocid1.compartment.oc1..prod"].Name)
	})
}

func TestCompartmentByID(t *testing.T) {
	prod := testCompartment("ocid1.compartment.oc1..prod", "prod")
	ctx := context.Background()

	t.Run("a compartment in the tree is found", func(t *testing.T) {
		conn := seededConnection(prod)

		got, err := conn.CompartmentByID(ctx, "ocid1.compartment.oc1..prod")

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, "prod", *got.Name)
	})

	t.Run("an ocid outside the tree misses without an error", func(t *testing.T) {
		// Cross-tenancy, or deleted since the listing. The caller resolves
		// those directly; a miss is not a failure.
		conn := seededConnection(prod)

		got, err := conn.CompartmentByID(ctx, "ocid1.compartment.oc1..elsewhere")

		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("an empty id misses before the tree is touched", func(t *testing.T) {
		// Not seeded, so a connection that reached GetCompartments here would
		// try to build an Identity client instead of returning.
		conn := &OciConnection{}

		got, err := conn.CompartmentByID(ctx, "")

		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("the index is built once and reused", func(t *testing.T) {
		conn := seededConnection(prod)

		_, err := conn.CompartmentByID(ctx, "ocid1.compartment.oc1..prod")
		require.NoError(t, err)
		first := conn.compartmentIndex
		require.NotNil(t, first)

		_, err = conn.CompartmentByID(ctx, "ocid1.compartment.oc1..prod")
		require.NoError(t, err)

		assert.True(t, sameMap(first, conn.compartmentIndex),
			"the index must be memoized, not rebuilt per lookup")
	})

	t.Run("the returned record does not alias the memoized tree", func(t *testing.T) {
		conn := seededConnection(prod)

		got, err := conn.CompartmentByID(ctx, "ocid1.compartment.oc1..prod")
		require.NoError(t, err)
		got.Name = common.String("mutated")

		again, err := conn.CompartmentByID(ctx, "ocid1.compartment.oc1..prod")
		require.NoError(t, err)
		assert.Equal(t, "prod", *again.Name)
	})

	t.Run("concurrent lookups share one index", func(t *testing.T) {
		// Every resource in the provider resolves its compartment, and the
		// executor resolves fields in goroutines, so this runs concurrently by
		// construction. Run under -race.
		conn := seededConnection(prod, testCompartment("ocid1.compartment.oc1..dev", "dev"))

		var wg sync.WaitGroup
		for i := 0; i < 32; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				id := "ocid1.compartment.oc1..prod"
				if i%2 == 0 {
					id = "ocid1.compartment.oc1..dev"
				}
				got, err := conn.CompartmentByID(ctx, id)
				assert.NoError(t, err)
				assert.NotNil(t, got)
			}(i)
		}
		wg.Wait()

		assert.Len(t, conn.compartmentIndex, 2)
	})
}

// sameMap reports whether two maps are the same underlying map, which is what
// distinguishes a memoized index from one rebuilt on every lookup.
func sameMap(a, b map[string]identity.Compartment) bool {
	if len(a) != len(b) {
		return false
	}
	const probe = "\x00sameMap-probe"
	a[probe] = identity.Compartment{}
	_, ok := b[probe]
	delete(a, probe)
	return ok
}
