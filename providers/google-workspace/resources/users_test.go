// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/api/googleapi"
)

func TestShouldCheckEarlierDateForReport(t *testing.T) {
	err := errors.New("some random error message")
	require.False(t, shouldCheckEarlierDateForReport(err))

	err = errors.New("Error 400: Another err (bad request)")
	require.False(t, shouldCheckEarlierDateForReport(err))

	err = errors.New("Error 400: Start date can not be later than 2024-07-29, invalid")
	require.True(t, shouldCheckEarlierDateForReport(err))

	err = errors.New("Error 400: Data for dates later than 2024-07-26 is not yet available. Please check back later, invalid")
	require.True(t, shouldCheckEarlierDateForReport(err))
}

func TestParseInt64(t *testing.T) {
	require.Equal(t, int64(0), parseInt64(""))
	require.Equal(t, int64(0), parseInt64("not a number"))
	require.Equal(t, int64(42), parseInt64("42"))
	require.Equal(t, int64(-7), parseInt64("-7"))
	require.Equal(t, int64(9223372036854775807), parseInt64("9223372036854775807"))
	// overflow falls back to zero
	require.Equal(t, int64(0), parseInt64("99999999999999999999"))
}

func TestCustomSchemasToDict(t *testing.T) {
	require.Nil(t, customSchemasToDict(nil))
	require.Nil(t, customSchemasToDict(map[string]googleapi.RawMessage{}))

	in := map[string]googleapi.RawMessage{
		"badge":   googleapi.RawMessage(`{"color":"blue","tier":3}`),
		"broken":  googleapi.RawMessage(`not-json`),
		"empty":   googleapi.RawMessage(`{}`),
		"nested":  googleapi.RawMessage(`{"team":{"name":"core"}}`),
		"numeric": googleapi.RawMessage(`{"score":42}`),
	}
	got := customSchemasToDict(in)
	require.Len(t, got, 4, "broken entry should be skipped, all others kept")
	require.Equal(t, map[string]any{"color": "blue", "tier": float64(3)}, got["badge"])
	require.Equal(t, map[string]any{}, got["empty"])
	require.Equal(t, map[string]any{"team": map[string]any{"name": "core"}}, got["nested"])
	require.Equal(t, map[string]any{"score": float64(42)}, got["numeric"])
	require.NotContains(t, got, "broken")
}

func TestUnmarshalUserMultiValue(t *testing.T) {
	type entry struct {
		Address string `json:"address"`
	}

	// nil collapses to empty
	require.Nil(t, unmarshalUserMultiValue[entry](nil))

	// list response shape
	listShape := []any{
		map[string]any{"address": "a@b.com"},
		map[string]any{"address": "c@d.com"},
	}
	got := unmarshalUserMultiValue[entry](listShape)
	require.Equal(t, []entry{{Address: "a@b.com"}, {Address: "c@d.com"}}, got)

	// non-list payload (e.g. single-entry create shape) does not panic; returns nil
	require.Nil(t, unmarshalUserMultiValue[entry](map[string]any{"address": "x@y.com"}))
}
