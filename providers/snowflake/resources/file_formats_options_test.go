// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// nullString stands in for the single-field wrapper the SDK uses for the
// entries of NULL_IF.
type nullString struct {
	S string
}

// jsonNativeValue feeds a dict field, and llx requires dict values to be
// JSON-native. A value that slips through as some other Go type is not rejected
// at compile time, so each accepted kind is pinned here.
func TestJSONNativeValue(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want any
		ok   bool
	}{
		{"string", "|", "|", true},
		{"empty string", "", "", true},
		{"bool true", true, true, true},
		{"bool false", false, false, true},
		{"int", 3, int64(3), true},
		{"int64", int64(97), int64(97), true},
		{"int32", int32(-1), int64(-1), true},
		{"zero int", 0, int64(0), true},

		// Floats are not among the accepted kinds; the caller must see that
		// rather than receive a silently dropped option.
		{"float is rejected", 1.5, nil, false},
		{"struct is rejected", struct{ A int }{1}, nil, false},
		{"map is rejected", map[string]string{"a": "b"}, nil, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := jsonNativeValue(reflect.ValueOf(tc.in))
			assert.Equal(t, tc.ok, ok)
			if tc.ok {
				assert.Equal(t, tc.want, got)
			}
		})
	}
}

func TestJSONNativeValuePointer(t *testing.T) {
	s := "HEX"
	n := 3
	b := true

	t.Run("a pointer is dereferenced", func(t *testing.T) {
		got, ok := jsonNativeValue(reflect.ValueOf(&s))
		require.True(t, ok)
		assert.Equal(t, "HEX", got)

		got, ok = jsonNativeValue(reflect.ValueOf(&n))
		require.True(t, ok)
		assert.Equal(t, int64(3), got)

		got, ok = jsonNativeValue(reflect.ValueOf(&b))
		require.True(t, ok)
		assert.Equal(t, true, got)
	})

	// An unset option is absent, not an option whose value is the zero value.
	t.Run("a nil pointer is not a value", func(t *testing.T) {
		var p *string
		_, ok := jsonNativeValue(reflect.ValueOf(p))
		assert.False(t, ok)
	})
}

func TestJSONNativeValueSlice(t *testing.T) {
	t.Run("single-field string structs are flattened", func(t *testing.T) {
		// NULL_IF arrives as a list of wrappers; the dict should carry the
		// strings they wrap, not the wrappers.
		in := []nullString{{S: `\N`}, {S: "NULL"}}
		got, ok := jsonNativeValue(reflect.ValueOf(in))
		require.True(t, ok)
		assert.Equal(t, []any{`\N`, "NULL"}, got)
	})

	t.Run("a plain string slice passes through", func(t *testing.T) {
		got, ok := jsonNativeValue(reflect.ValueOf([]string{"a", "b"}))
		require.True(t, ok)
		assert.Equal(t, []any{"a", "b"}, got)
	})

	t.Run("an empty slice is an empty list", func(t *testing.T) {
		got, ok := jsonNativeValue(reflect.ValueOf([]string{}))
		require.True(t, ok)
		assert.Equal(t, []any{}, got)
	})

	// One unrepresentable element fails the whole slice rather than yielding a
	// list that silently omits it.
	t.Run("an unrepresentable element rejects the slice", func(t *testing.T) {
		_, ok := jsonNativeValue(reflect.ValueOf([]float64{1.5}))
		assert.False(t, ok)
	})
}
