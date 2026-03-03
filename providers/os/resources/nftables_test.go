// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sampleNftJSON = `{
  "nftables": [
    {"metainfo": {"version": "1.0.2", "release_name": "Lester Gooch", "json_schema_version": 1}},
    {"table": {"family": "inet", "name": "filter", "handle": 1}},
    {"chain": {"family": "inet", "table": "filter", "name": "input", "handle": 1, "type": "filter", "hook": "input", "prio": 0, "policy": "accept"}},
    {"chain": {"family": "inet", "table": "filter", "name": "forward", "handle": 2, "type": "filter", "hook": "forward", "prio": 0, "policy": "drop"}},
    {"chain": {"family": "inet", "table": "filter", "name": "my_chain", "handle": 3}},
    {"rule": {"family": "inet", "table": "filter", "chain": "input", "handle": 4, "expr": [{"match": {"left": {"meta": {"key": "iifname"}}, "right": "lo", "op": "=="}}, {"accept": null}]}},
    {"rule": {"family": "inet", "table": "filter", "chain": "input", "handle": 5, "expr": [{"match": {"left": {"ct": {"key": "state"}}, "right": ["established", "related"]}}, {"accept": null}], "comment": "allow established"}},
    {"rule": {"family": "inet", "table": "filter", "chain": "input", "handle": 6, "expr": [{"match": {"left": {"payload": {"protocol": "tcp", "field": "dport"}}, "right": 22, "op": "=="}}, {"accept": null}], "comment": "allow ssh"}},
    {"table": {"family": "ip", "name": "nat", "handle": 2}},
    {"chain": {"family": "ip", "table": "nat", "name": "postrouting", "handle": 1, "type": "nat", "hook": "postrouting", "prio": 100, "policy": "accept"}},
    {"rule": {"family": "ip", "table": "nat", "chain": "postrouting", "handle": 2, "expr": [{"match": {"left": {"payload": {"protocol": "ip", "field": "saddr"}}, "right": {"prefix": {"addr": "192.168.1.0", "len": 24}}, "op": "=="}}, {"masquerade": null}]}}
  ]
}`

func TestParseNftRuleset(t *testing.T) {
	ruleset, err := parseNftRuleset([]byte(sampleNftJSON))
	require.NoError(t, err)
	require.NotNil(t, ruleset)

	var tableCount, chainCount, ruleCount int
	for _, obj := range ruleset.Nftables {
		if obj.Table != nil {
			tableCount++
		}
		if obj.Chain != nil {
			chainCount++
		}
		if obj.Rule != nil {
			ruleCount++
		}
	}
	assert.Equal(t, 2, tableCount)
	assert.Equal(t, 4, chainCount)
	assert.Equal(t, 4, ruleCount)
}

func TestParseNftRuleset_Tables(t *testing.T) {
	ruleset, err := parseNftRuleset([]byte(sampleNftJSON))
	require.NoError(t, err)

	var tables []*nftTable
	for _, obj := range ruleset.Nftables {
		if obj.Table != nil {
			tables = append(tables, obj.Table)
		}
	}

	require.Len(t, tables, 2)

	assert.Equal(t, "inet", tables[0].Family)
	assert.Equal(t, "filter", tables[0].Name)
	assert.Equal(t, int64(1), tables[0].Handle)

	assert.Equal(t, "ip", tables[1].Family)
	assert.Equal(t, "nat", tables[1].Name)
	assert.Equal(t, int64(2), tables[1].Handle)
}

func TestParseNftRuleset_BaseChain(t *testing.T) {
	ruleset, err := parseNftRuleset([]byte(sampleNftJSON))
	require.NoError(t, err)

	var chains []*nftChain
	for _, obj := range ruleset.Nftables {
		if obj.Chain != nil {
			chains = append(chains, obj.Chain)
		}
	}

	require.Len(t, chains, 4)

	// input chain - base chain
	assert.Equal(t, "input", chains[0].Name)
	assert.Equal(t, "filter", chains[0].Type)
	assert.Equal(t, "input", chains[0].Hook)
	assert.Equal(t, int64(0), chains[0].Prio)
	assert.Equal(t, "accept", chains[0].Policy)

	// forward chain - base chain with drop policy
	assert.Equal(t, "forward", chains[1].Name)
	assert.Equal(t, "drop", chains[1].Policy)

	// my_chain - regular chain (no type/hook/prio/policy)
	assert.Equal(t, "my_chain", chains[2].Name)
	assert.Equal(t, "", chains[2].Type)
	assert.Equal(t, "", chains[2].Hook)
	assert.Equal(t, int64(0), chains[2].Prio)
	assert.Equal(t, "", chains[2].Policy)

	// postrouting - nat base chain
	assert.Equal(t, "postrouting", chains[3].Name)
	assert.Equal(t, "nat", chains[3].Type)
	assert.Equal(t, "postrouting", chains[3].Hook)
	assert.Equal(t, int64(100), chains[3].Prio)
}

func TestParseNftRuleset_Rules(t *testing.T) {
	ruleset, err := parseNftRuleset([]byte(sampleNftJSON))
	require.NoError(t, err)

	var rules []*nftRule
	for _, obj := range ruleset.Nftables {
		if obj.Rule != nil {
			rules = append(rules, obj.Rule)
		}
	}

	require.Len(t, rules, 4)

	// First rule: loopback accept
	assert.Equal(t, "inet", rules[0].Family)
	assert.Equal(t, "filter", rules[0].Table)
	assert.Equal(t, "input", rules[0].Chain)
	assert.Equal(t, int64(4), rules[0].Handle)
	assert.Len(t, rules[0].Expr, 2)
	assert.Equal(t, "", rules[0].Comment)

	// Second rule: established/related with comment
	assert.Equal(t, int64(5), rules[1].Handle)
	assert.Equal(t, "allow established", rules[1].Comment)

	// Third rule: allow ssh — verify integer port number is preserved as int64
	assert.Equal(t, int64(6), rules[2].Handle)
	assert.Equal(t, "allow ssh", rules[2].Comment)
	require.Len(t, rules[2].Expr, 2)
	matchExpr, ok := rules[2].Expr[0].(map[string]any)
	require.True(t, ok)
	matchInner, ok := matchExpr["match"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, int64(22), matchInner["right"])

	// NAT rule
	assert.Equal(t, "ip", rules[3].Family)
	assert.Equal(t, "nat", rules[3].Table)
	assert.Equal(t, "postrouting", rules[3].Chain)
	assert.Equal(t, int64(2), rules[3].Handle)
	assert.Len(t, rules[3].Expr, 2)
}

func TestParseNftRuleset_EmptyRuleset(t *testing.T) {
	data := `{"nftables": [{"metainfo": {"version": "1.0.2", "release_name": "Lester Gooch", "json_schema_version": 1}}]}`
	ruleset, err := parseNftRuleset([]byte(data))
	require.NoError(t, err)
	require.NotNil(t, ruleset)

	var tableCount int
	for _, obj := range ruleset.Nftables {
		if obj.Table != nil {
			tableCount++
		}
	}
	assert.Equal(t, 0, tableCount)
}

func TestParseNftRuleset_InvalidJSON(t *testing.T) {
	_, err := parseNftRuleset([]byte("not json"))
	require.Error(t, err)
}

func TestNftTableParseFlags(t *testing.T) {
	t.Run("no flags", func(t *testing.T) {
		tbl := &nftTable{}
		assert.Nil(t, tbl.parseFlags())
	})

	t.Run("null flags", func(t *testing.T) {
		tbl := &nftTable{Flags: json.RawMessage(`null`)}
		assert.Nil(t, tbl.parseFlags())
	})

	t.Run("single string flag", func(t *testing.T) {
		tbl := &nftTable{Flags: json.RawMessage(`"dormant"`)}
		assert.Equal(t, []string{"dormant"}, tbl.parseFlags())
	})

	t.Run("array flags", func(t *testing.T) {
		tbl := &nftTable{Flags: json.RawMessage(`["dormant", "owner"]`)}
		assert.Equal(t, []string{"dormant", "owner"}, tbl.parseFlags())
	})
}
