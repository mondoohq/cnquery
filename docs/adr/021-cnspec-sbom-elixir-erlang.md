# ADR: mql SBOM Cataloger for Elixir and Erlang (Hex)

**Status:** Proposed
**Date:** 2026-04-19

---

## Context

cnspec needs native SBOM cataloging for Elixir and Erlang packages. Both languages use the Hex package ecosystem (hex.pm), share the same PURL scheme (`pkg:hex`), and have similar lock file formats based on their respective term syntaxes.

## File Formats

### Elixir mix.lock

Elixir term format — a map of package names to tuples.

```elixir
%{
  "jason": {:hex, :jason, "1.4.1", "af1chabc...", [:mix], [], "hexpm", "fdfhash..."},
  "plug": {:hex, :plug, "1.15.3", "abc123...", [:mix], [{:mime, "~> 2.0", [hex: :mime, repo: "hexpm"]}], "hexpm", "def456..."},
  "mime": {:hex, :mime, "2.0.5", "ghi789...", [:mix], [], "hexpm", "jkl012..."},
}
```

Each entry: `"name": {:hex, :name, "version", "hash", build_tools, deps, "repo", "outer_hash"}`

Key fields: package name (string key), version (3rd tuple element).

### Erlang rebar.lock

Erlang term format — a list of package tuples.

```erlang
[{<<"cowboy">>,{pkg,<<"cowboy">>,<<"2.10.0">>,<<"hash...">>},0},
 {<<"cowlib">>,{pkg,<<"cowlib">>,<<"2.12.1">>,<<"hash...">>},0},
 {<<"ranch">>,{pkg,<<"ranch">>,<<"2.1.0">>,<<"hash...">>},0}].
```

Each entry: `{<<"name">>, {pkg, <<"name">>, <<"version">>, <<"hash">>}, level}`

## Parsing Strategy

Both formats require custom parsers since they use language-specific term syntax (not JSON/YAML/TOML).

### Elixir mix.lock
1. Read line by line
2. Match `"name": {:hex, :name, "version", ...}` pattern via regex
3. Extract name and version from each matched line

### Erlang rebar.lock  
1. Read line by line
2. Match `{<<"name">>,{pkg,<<"name">>,<<"version">>` pattern via regex
3. Extract name and version from each matched line

Both parsers use regex-based extraction rather than full term parsers — this is simpler, more robust, and sufficient for SBOM purposes.

## Package Identification

Both use the Hex PURL scheme:
- **PURL:** `pkg:hex/<name>@<version>`
- Shared helpers in `providers/os/resources/languages/hex/`

## Verification

- Fixture-based tests for both formats
- Interactive: `mql run os -c "elixir.packages(path: '.') { list { name version purl } }"`
- Interactive: `mql run os -c "erlang.packages(path: '.') { list { name version purl } }"`
