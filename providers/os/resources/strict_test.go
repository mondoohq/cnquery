// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/mqlc"
	"go.mondoo.com/mql/providers-sdk/v1/testutils"
)

// These execute against the arch mock, whose /dummy.json contains a `_` key
// holding null and no key named `NOPE` - the two cases ADR 043 has to tell
// apart: a value that is null because that is what was there, versus a key the
// query claimed exists and does not.
const strictDoc = "parse.json('/dummy.json').params"

// runStrict compiles in the given mode and returns the expression's own value.
// Results are ordered value-first, with the query's truthiness appended, so
// index 0 is the chain's result and the tail is the assertion outcome.
func runStrict(t *testing.T, code string, strict bool) *llx.RawResult {
	t.Helper()
	return runStrictAll(t, code, strict)[0]
}

// runStrictOutcome returns the query's final boolean, which is what a comparison
// resolves to.
func runStrictOutcome(t *testing.T, code string, strict bool) *llx.RawResult {
	t.Helper()
	all := runStrictAll(t, code, strict)
	return all[len(all)-1]
}

func runStrictAll(t *testing.T, code string, strict bool) []*llx.RawResult {
	t.Helper()

	x := testutils.InitTester(testutils.LinuxMock())
	conf := mqlc.NewConfig(x.Runtime.Schema(), testutils.Features)
	conf.Strict = strict

	bundle, err := mqlc.Compile(code, nil, conf)
	require.NoError(t, err, "compiling %q", code)

	results := x.TestMqlc(t, bundle, nil)
	require.NotEmpty(t, results, "no results for %q", code)
	return results
}

func assertValue(t *testing.T, res *llx.RawResult, expected any, code string) {
	t.Helper()
	require.NoError(t, res.Data.Error, "%q should not error", code)
	assert.Equal(t, expected, res.Data.Value, "%q", code)
}

func assertNull(t *testing.T, res *llx.RawResult, code string) {
	t.Helper()
	require.NoError(t, res.Data.Error, "%q should not error", code)
	assert.Nil(t, res.Data.Value, "%q should be null", code)
}

func assertErrors(t *testing.T, res *llx.RawResult, contains string, code string) {
	t.Helper()
	require.Error(t, res.Data.Error, "%q should error", code)
	assert.Contains(t, res.Data.Error.Error(), contains, "%q", code)
}

// TestStrict_nonStrictUnchanged is the compatibility guarantee. Every one of
// these silently yields null today, and must keep doing so.
func TestStrict_nonStrictUnchanged(t *testing.T) {
	for _, code := range []string{
		strictDoc + "['NOPE']",
		strictDoc + "['NOPE']['b']",
		strictDoc + ".NOPE",
		strictDoc + "['_']['b']",
	} {
		t.Run(code, func(t *testing.T) {
			assertNull(t, runStrict(t, code, false), code)
		})
	}

	// the silent false that motivates the whole ADR
	code := strictDoc + "['NOPE'] == 'no'"
	assertValue(t, runStrictOutcome(t, code, false), false, code)
}

// TestStrict_missingKeyErrors covers the case strict mode exists for: a key the
// query names that is not there. Notably it errors even as a terminal lookup,
// which is what catches a typo that only ever feeds a comparison.
func TestStrict_missingKeyErrors(t *testing.T) {
	for _, code := range []string{
		strictDoc + "['NOPE']",
		strictDoc + ".NOPE",
		strictDoc + "['NOPE']['b']",
	} {
		t.Run(code, func(t *testing.T) {
			assertErrors(t, runStrict(t, code, true), "cannot find key \"NOPE\"", code)
		})
	}

	// The one that matters most: a mistyped key compared against a value used to
	// come back as a clean `false`, which reads as a passing check.
	code := strictDoc + "['NOPE'] == 'no'"
	assertErrors(t, runStrict(t, code, true), "cannot find key \"NOPE\"", code)
}

// TestStrict_nullValueIsNotAMissingKey pins the distinction the rule rests on.
// `_` is present and holds null, so reading it resolves; only reading *through*
// it fails.
func TestStrict_nullValueIsNotAMissingKey(t *testing.T) {
	code := strictDoc + "['_']"
	assertNull(t, runStrict(t, code, true), code)

	code = strictDoc + "['_']['b']"
	assertErrors(t, runStrict(t, code, true), "the value it reads from is null", code)
}

// TestStrict_optionalWaivesBothCauses is the double duty: one mark, in one
// position, covers both an absent key and a present-but-null value.
func TestStrict_optionalWaivesBothCauses(t *testing.T) {
	for _, code := range []string{
		strictDoc + "['NOPE']?",       // absent key, terminal
		strictDoc + "['NOPE']?['b']",  // absent key, dereferenced
		strictDoc + "['_']?['b']",     // present but null, dereferenced
		strictDoc + ".NOPE?",          // bare-word sugar
		strictDoc + "?['NOPE']?['b']", // guards stacked
	} {
		t.Run(code, func(t *testing.T) {
			assertNull(t, runStrict(t, code, true), code)
		})
	}
}

// TestStrict_shortCircuitIsNotSticky is the JavaScript property: a guard covers
// the link it is attached to and hands a null down the chain, but it does not
// silence a *different* link further along.
func TestStrict_shortCircuitIsNotSticky(t *testing.T) {
	// `e` exists and is a dict, so the guard on it never fires; `NOPE` inside it
	// is still missing and must still error.
	code := strictDoc + "['e']?['NOPE']"
	assertErrors(t, runStrict(t, code, true), "cannot find key \"NOPE\"", code)

	// with the guard moved to the link that actually misses, it resolves
	code = strictDoc + "['e']?['NOPE']?"
	assertNull(t, runStrict(t, code, true), code)
}

// TestStrict_operatorsStillTakeNull is the exemption. Comparisons receive a null
// as an operand rather than as a failed dereference, so `== null` stays the way
// to test for null and does not become an error.
func TestStrict_operatorsStillTakeNull(t *testing.T) {
	code := strictDoc + "['_'] == null"
	assertValue(t, runStrictOutcome(t, code, true), true, code)

	code = strictDoc + "['_'] != null"
	assertValue(t, runStrictOutcome(t, code, true), false, code)
}

// TestStrict_resolvedChainsAreUnaffected: strict mode must be invisible to a
// query that was already correct.
func TestStrict_resolvedChainsAreUnaffected(t *testing.T) {
	for _, tc := range []struct {
		code     string
		expected any
	}{
		{strictDoc + "['hello']", "hello"},
		{strictDoc + ".hello", "hello"},
		{strictDoc + "['e']['hi']", "hello"},
		{strictDoc + ".e.hi", "hello"},
	} {
		t.Run(tc.code, func(t *testing.T) {
			assertValue(t, runStrict(t, tc.code, true), tc.expected, tc.code)
		})
	}

	code := strictDoc + "['hello'] == 'hello'"
	assertValue(t, runStrictOutcome(t, code, true), true, code)
}
