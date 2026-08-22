// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/identity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/utils/syncx"
)

func testRuntime() *plugin.Runtime {
	return &plugin.Runtime{Resources: &syncx.Map[plugin.Resource]{}}
}

func testCompartment(id, name string) identity.Compartment {
	return identity.Compartment{
		Id:             common.String(id),
		Name:           common.String(name),
		Description:    common.String(name + " compartment"),
		LifecycleState: identity.CompartmentLifecycleStateActive,
	}
}

// TestResolveCompartment covers the reason this resolver exists.
//
// Going through NewResource runs initOciCompartment before the runtime cache is
// consulted, so every resource reporting an owner cost a GetCompartment call -
// five hundred instances in five compartments issued five hundred of them. The
// tenancy tree the connection already holds answers nearly all of those, and
// the direct read stays only for the OCIDs it cannot cover.
func TestResolveCompartment(t *testing.T) {
	const ocid = "ocid1.compartment.oc1..aaaaaaaaprod"

	t.Run("a compartment in the tree resolves without a direct read", func(t *testing.T) {
		compartment := testCompartment(ocid, "prod")
		calls := 0
		lookup := func(id string) (*identity.Compartment, error) {
			calls++
			assert.Equal(t, ocid, id)
			return &compartment, nil
		}

		var field plugin.TValue[*mqlOciCompartment]
		got, err := resolveCompartment(testRuntime(), lookup, ocid, &field)

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, ocid, got.Id.Data)
		assert.Equal(t, "prod", got.Name.Data)
		assert.Equal(t, "ACTIVE", got.State.Data)
		assert.Equal(t, 1, calls)
		// The runtime marks the field once the accessor returns; a resolved
		// reference must not have been pre-marked null on the way out.
		assert.Equal(t, plugin.State(0), field.State)
	})

	t.Run("resolving the same compartment twice yields one instance", func(t *testing.T) {
		// The point of building the resource from the same args the lister
		// uses: the __id matches, so CreateResource hands back the instance
		// the runtime already holds instead of a second copy of it.
		compartment := testCompartment(ocid, "prod")
		lookup := func(string) (*identity.Compartment, error) { return &compartment, nil }
		runtime := testRuntime()

		var first, second plugin.TValue[*mqlOciCompartment]
		a, err := resolveCompartment(runtime, lookup, ocid, &first)
		require.NoError(t, err)
		b, err := resolveCompartment(runtime, lookup, ocid, &second)
		require.NoError(t, err)

		assert.Same(t, a, b)
	})

	t.Run("identity matches what oci.compartments produces", func(t *testing.T) {
		// A resolved compartment has to be the same resource the lister
		// created, or the runtime carries two of every compartment and a
		// query that reaches one of them re-fetches its fields.
		compartment := testCompartment(ocid, "prod")
		runtime := testRuntime()

		listed, err := CreateResource(runtime, "oci.compartment", ociCompartmentArgs(compartment))
		require.NoError(t, err)

		var field plugin.TValue[*mqlOciCompartment]
		got, err := resolveCompartment(runtime, func(string) (*identity.Compartment, error) {
			return &compartment, nil
		}, ocid, &field)
		require.NoError(t, err)

		assert.Same(t, listed, got)
	})

	t.Run("an ocid outside the tree falls through to the direct read", func(t *testing.T) {
		// A compartment in another tenancy, or one deleted since the listing,
		// is not in the tree and must still be resolvable. The runtime here
		// carries no OCI connection, so reaching the direct read surfaces as
		// an error rather than a resource - which is the observation.
		lookup := func(string) (*identity.Compartment, error) { return nil, nil }

		var field plugin.TValue[*mqlOciCompartment]
		got, err := resolveCompartment(testRuntime(), lookup, ocid, &field)

		require.Error(t, err)
		assert.Nil(t, got)
	})

	t.Run("an unreadable tree falls through to the direct read", func(t *testing.T) {
		// A throttled or denied ListCompartments says nothing about this
		// compartment, so it must not be reported as absent.
		lookup := func(string) (*identity.Compartment, error) {
			return nil, errors.New("too many requests")
		}

		var field plugin.TValue[*mqlOciCompartment]
		got, err := resolveCompartment(testRuntime(), lookup, ocid, &field)

		require.Error(t, err)
		assert.Nil(t, got)
	})

	t.Run("an empty id is null and never reaches the lookup", func(t *testing.T) {
		lookup := func(string) (*identity.Compartment, error) {
			t.Fatal("an absent reference must not be looked up")
			return nil, nil
		}

		var field plugin.TValue[*mqlOciCompartment]
		got, err := resolveCompartment(nil, lookup, "", &field)

		require.NoError(t, err)
		assert.Nil(t, got)
		assert.Equal(t, plugin.StateIsSet|plugin.StateIsNull, field.State,
			"an absent compartment must be reported as null, not left unset")
	})

	t.Run("a non-ocid sentinel is null and never reaches the lookup", func(t *testing.T) {
		// ocidOrEmpty at the call site turns OCI's placeholders into the empty
		// case; together they must still produce a null rather than a lookup
		// for an id no tree can contain.
		lookup := func(string) (*identity.Compartment, error) {
			t.Fatal("a placeholder must not be looked up")
			return nil, nil
		}

		var field plugin.TValue[*mqlOciCompartment]
		got, err := resolveCompartment(nil, lookup, ocidOrEmpty("ORACLE_MANAGED_KEY"), &field)

		require.NoError(t, err)
		assert.Nil(t, got)
		assert.Equal(t, plugin.StateIsSet|plugin.StateIsNull, field.State)
	})

	t.Run("without a lookup the empty id is still null", func(t *testing.T) {
		var field plugin.TValue[*mqlOciCompartment]
		got, err := resolveCompartment(nil, nil, "", &field)

		require.NoError(t, err)
		assert.Nil(t, got)
		assert.Equal(t, plugin.StateIsSet|plugin.StateIsNull, field.State)
	})
}

// TestCachedCompartment pins the distinction the resolver is built on: a tree
// that cannot answer is not a failure, it is a fallback.
func TestCachedCompartment(t *testing.T) {
	const ocid = "ocid1.compartment.oc1..aaaaaaaaprod"

	t.Run("no lookup means no cached answer", func(t *testing.T) {
		got, err := cachedCompartment(testRuntime(), nil, ocid)
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("an empty id means no cached answer", func(t *testing.T) {
		lookup := func(string) (*identity.Compartment, error) {
			t.Fatal("an empty id must not be looked up")
			return nil, nil
		}
		got, err := cachedCompartment(testRuntime(), lookup, "")
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("a lookup failure is a fallback, not an error", func(t *testing.T) {
		// Reporting the error here would turn a throttled ListCompartments
		// into a failed compartment field on every resource in the scan.
		got, err := cachedCompartment(testRuntime(), func(string) (*identity.Compartment, error) {
			return nil, errors.New("too many requests")
		}, ocid)
		require.NoError(t, err)
		assert.Nil(t, got)
	})
}

// TestOciCompartmentRef covers the lister path, where the compartment is
// resolved while the list is built and so is paid for whether or not the field
// is ever read.
func TestOciCompartmentRef(t *testing.T) {
	const ocid = "ocid1.compartment.oc1..aaaaaaaaprod"

	t.Run("a compartment in the tree is built from the tree", func(t *testing.T) {
		compartment := testCompartment(ocid, "prod")
		runtime := testRuntime()
		listed, err := CreateResource(runtime, "oci.compartment", ociCompartmentArgs(compartment))
		require.NoError(t, err)

		got, err := compartmentRef(runtime, func(string) (*identity.Compartment, error) {
			return &compartment, nil
		}, common.String(ocid))

		require.NoError(t, err)
		assert.Same(t, listed, got)
	})

	t.Run("an ocid outside the tree falls through to the direct read", func(t *testing.T) {
		got, err := compartmentRef(testRuntime(), func(string) (*identity.Compartment, error) {
			return nil, nil
		}, common.String(ocid))

		require.Error(t, err)
		assert.Nil(t, got)
	})

	t.Run("a nil compartment id falls through to the direct read", func(t *testing.T) {
		// A list entry with no compartment id keeps the behavior it had: the
		// direct read decides what an id-less compartment resolves to, which
		// is a resource whose id is null rather than an error.
		got, err := compartmentRef(testRuntime(), func(string) (*identity.Compartment, error) {
			t.Fatal("a nil id must not be looked up")
			return nil, nil
		}, nil)

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Empty(t, got.(*mqlOciCompartment).Id.Data)
	})
}

func TestOciCompartmentLookup(t *testing.T) {
	// A runtime with no OCI connection has no tree to consult. Returning nil
	// rather than a lookup that panics on the assertion is what keeps the
	// resolver falling back instead of ending the scan.
	assert.Nil(t, ociCompartmentLookup(nil))
	assert.Nil(t, ociCompartmentLookup(&plugin.Runtime{}))
}
