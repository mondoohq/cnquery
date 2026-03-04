// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"go.mondoo.com/mql/v13/types"
)

// nftRuleset is the top-level JSON envelope from `nft -j list ruleset`.
type nftRuleset struct {
	Nftables []nftObject `json:"nftables"`
}

// nftObject represents one element in the nftables array.
// Exactly one field will be non-nil per object.
type nftObject struct {
	Metainfo *nftMetainfo `json:"metainfo,omitempty"`
	Table    *nftTable    `json:"table,omitempty"`
	Chain    *nftChain    `json:"chain,omitempty"`
	Rule     *nftRule     `json:"rule,omitempty"`
}

type nftMetainfo struct {
	Version           string `json:"version"`
	ReleaseName       string `json:"release_name"`
	JSONSchemaVersion int    `json:"json_schema_version"`
}

type nftTable struct {
	Family string          `json:"family"`
	Name   string          `json:"name"`
	Handle int64           `json:"handle"`
	Flags  json.RawMessage `json:"flags,omitempty"`
}

// parseFlags normalizes nftables table flags from JSON.
// Flags can be a single string or an array of strings depending on the nft version.
func (t *nftTable) parseFlags() []string {
	if t.Flags == nil || string(t.Flags) == "null" {
		return nil
	}
	var arr []string
	if err := json.Unmarshal(t.Flags, &arr); err == nil {
		return arr
	}
	var s string
	if err := json.Unmarshal(t.Flags, &s); err == nil {
		return []string{s}
	}
	return nil
}

type nftChain struct {
	Family string `json:"family"`
	Table  string `json:"table"`
	Name   string `json:"name"`
	Handle int64  `json:"handle"`
	Type   string `json:"type,omitempty"`
	Hook   string `json:"hook,omitempty"`
	Prio   int64  `json:"prio,omitempty"`
	Policy string `json:"policy,omitempty"`
}

type nftRule struct {
	Family  string `json:"family"`
	Table   string `json:"table"`
	Chain   string `json:"chain"`
	Handle  int64  `json:"handle"`
	Expr    []any  `json:"expr,omitempty"`
	Comment string `json:"comment,omitempty"`
}

func parseNftRuleset(data []byte) (*nftRuleset, error) {
	var ruleset nftRuleset
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err := dec.Decode(&ruleset); err != nil {
		return nil, fmt.Errorf("failed to parse nftables JSON: %w", err)
	}
	// Convert json.Number values in rule expressions to native Go types
	// so they are compatible with llx dict handling (which expects int64/float64).
	for i := range ruleset.Nftables {
		if r := ruleset.Nftables[i].Rule; r != nil {
			for j := range r.Expr {
				r.Expr[j] = convertJSONNumbers(r.Expr[j])
			}
		}
	}
	return &ruleset, nil
}

// convertJSONNumbers recursively walks a value decoded with UseNumber()
// and replaces json.Number with int64 (preferred) or float64.
func convertJSONNumbers(v any) any {
	switch x := v.(type) {
	case json.Number:
		if n, err := x.Int64(); err == nil {
			return n
		}
		if f, err := x.Float64(); err == nil {
			return f
		}
		return x.String()
	case map[string]any:
		for k, val := range x {
			x[k] = convertJSONNumbers(val)
		}
		return x
	case []any:
		for i, val := range x {
			x[i] = convertJSONNumbers(val)
		}
		return x
	default:
		return v
	}
}

func (n *mqlNftables) id() (string, error) {
	return "nftables", nil
}

func (t *mqlNftablesTable) id() (string, error) {
	return t.Family.Data + "/" + t.Name.Data, nil
}

func (c *mqlNftablesChain) id() (string, error) {
	return c.Family.Data + "/" + c.Table.Data + "/" + c.Name.Data, nil
}

func (r *mqlNftablesRule) id() (string, error) {
	return r.Family.Data + "/" + r.Table.Data + "/" + r.Chain.Data + "/" + strconv.FormatInt(r.Handle.Data, 10), nil
}

func (n *mqlNftables) tables() ([]any, error) {
	conn, ok := n.MqlRuntime.Connection.(shared.Connection)
	if !ok || !conn.Capabilities().Has(shared.Capability_RunCommand) {
		return nil, nil
	}

	o, err := CreateResource(n.MqlRuntime, "command", map[string]*llx.RawData{
		"command": llx.StringData("nft -j list ruleset"),
	})
	if err != nil {
		return nil, err
	}
	cmd := o.(*mqlCommand)
	if exit := cmd.GetExitcode(); exit.Data != 0 {
		return nil, fmt.Errorf("nft command failed (exit %d): %s", exit.Data, cmd.Stderr.Data)
	}

	ruleset, err := parseNftRuleset([]byte(cmd.Stdout.Data))
	if err != nil {
		return nil, err
	}

	tables := []any{}
	for _, obj := range ruleset.Nftables {
		if obj.Table == nil {
			continue
		}
		t := obj.Table

		parsedFlags := t.parseFlags()
		flags := make([]any, len(parsedFlags))
		for i, f := range parsedFlags {
			flags[i] = f
		}

		// Collect chains for this table
		chains := []any{}
		for _, o := range ruleset.Nftables {
			if o.Chain == nil || o.Chain.Family != t.Family || o.Chain.Table != t.Name {
				continue
			}
			ch := o.Chain

			chainRules, err := nftCollectRules(n.MqlRuntime, ruleset, t.Family, t.Name, ch.Name)
			if err != nil {
				return nil, err
			}

			isBase := ch.Type != ""
			chainRes, err := CreateResource(n.MqlRuntime, "nftables.chain", map[string]*llx.RawData{
				"family":      llx.StringData(ch.Family),
				"table":       llx.StringData(ch.Table),
				"name":        llx.StringData(ch.Name),
				"handle":      llx.IntData(ch.Handle),
				"type":        llx.StringData(ch.Type),
				"hook":        llx.StringData(ch.Hook),
				"prio":        llx.IntData(ch.Prio),
				"policy":      llx.StringData(ch.Policy),
				"isBaseChain": llx.BoolData(isBase),
				"rules":       llx.ArrayData(chainRules, types.Resource("nftables.rule")),
			})
			if err != nil {
				return nil, err
			}
			chains = append(chains, chainRes)
		}

		// Collect all rules for this table across all chains
		tableRules, err := nftCollectRules(n.MqlRuntime, ruleset, t.Family, t.Name, "")
		if err != nil {
			return nil, err
		}

		tableRes, err := CreateResource(n.MqlRuntime, "nftables.table", map[string]*llx.RawData{
			"family": llx.StringData(t.Family),
			"name":   llx.StringData(t.Name),
			"handle": llx.IntData(t.Handle),
			"flags":  llx.ArrayData(flags, types.String),
			"chains": llx.ArrayData(chains, types.Resource("nftables.chain")),
			"rules":  llx.ArrayData(tableRules, types.Resource("nftables.rule")),
		})
		if err != nil {
			return nil, err
		}
		tables = append(tables, tableRes)
	}

	return tables, nil
}

// nftCollectRules creates rule resources filtered by family/table and optionally chain.
// If chain is empty, all rules for the table are returned.
func nftCollectRules(runtime *plugin.Runtime, ruleset *nftRuleset, family, table, chain string) ([]any, error) {
	var rules []any
	for _, obj := range ruleset.Nftables {
		if obj.Rule == nil {
			continue
		}
		r := obj.Rule
		if r.Family != family || r.Table != table {
			continue
		}
		if chain != "" && r.Chain != chain {
			continue
		}

		exprDicts := make([]any, len(r.Expr))
		copy(exprDicts, r.Expr)

		ruleRes, err := CreateResource(runtime, "nftables.rule", map[string]*llx.RawData{
			"family":  llx.StringData(r.Family),
			"table":   llx.StringData(r.Table),
			"chain":   llx.StringData(r.Chain),
			"handle":  llx.IntData(r.Handle),
			"expr":    llx.ArrayData(exprDicts, types.Dict),
			"comment": llx.StringData(r.Comment),
		})
		if err != nil {
			return nil, err
		}
		rules = append(rules, ruleRes)
	}
	return rules, nil
}
