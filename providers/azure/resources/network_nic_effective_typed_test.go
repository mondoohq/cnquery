// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"

	network "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v10"
)

func TestRuleString(t *testing.T) {
	rule := map[string]any{"access": "Allow", "priority": float64(100)}

	assert.Equal(t, "Allow", ruleString(rule, "access"))
	// An absent key and a key holding a non-string both yield "" rather than
	// panicking or rendering the wrong type into the schema.
	assert.Equal(t, "", ruleString(rule, "missing"))
	assert.Equal(t, "", ruleString(rule, "priority"))
}

func TestRuleStringSlice(t *testing.T) {
	rule := map[string]any{
		"sourceAddressPrefixes": []any{"10.0.0.0/8", "192.168.0.0/16"},
		"mixed":                 []any{"10.0.0.0/8", float64(42), nil},
		"notASlice":             "10.0.0.0/8",
	}

	assert.Equal(t, []any{"10.0.0.0/8", "192.168.0.0/16"}, ruleStringSlice(rule, "sourceAddressPrefixes"))

	// Non-string entries are dropped rather than coerced: an empty string in a
	// prefix list would read as a rule covering an unnamed range.
	assert.Equal(t, []any{"10.0.0.0/8"}, ruleStringSlice(rule, "mixed"))

	// An absent key or a wrongly-typed value yields an empty list, never nil,
	// so the schema reports "no prefixes" rather than an unset field.
	assert.Equal(t, []any{}, ruleStringSlice(rule, "missing"))
	assert.Equal(t, []any{}, ruleStringSlice(rule, "notASlice"))
}

func TestEnumString(t *testing.T) {
	active := network.EffectiveRouteStateActive
	assert.Equal(t, "Active", enumString(&active))

	// A nil enum means Azure did not state the value; it must stay empty rather
	// than defaulting to a value the API never returned.
	var nilState *network.EffectiveRouteState
	assert.Equal(t, "", enumString(nilState))

	hop := network.RouteNextHopTypeInternet
	assert.Equal(t, "Internet", enumString(&hop))
}

func TestStrPtrSliceToAny(t *testing.T) {
	a, b := "10.0.0.0/8", "0.0.0.0/0"

	assert.Equal(t, []any{a, b}, strPtrSliceToAny([]*string{&a, &b}))

	// A nil element is skipped rather than dereferenced. convert.SliceStrPtrToStr
	// panics on this input, which is why the conversion is written out here.
	assert.Equal(t, []any{a}, strPtrSliceToAny([]*string{&a, nil}))

	assert.Equal(t, []any{}, strPtrSliceToAny(nil))
}
