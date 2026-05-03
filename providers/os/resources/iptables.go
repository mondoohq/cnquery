// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"fmt"
	"io"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"go.mondoo.com/mql/v13/types"
)

// Stat represents a structured statistic entry.
type Stat struct {
	LineNumber  int64
	Packets     int64
	Bytes       int64
	Target      string
	Protocol    string
	Opt         string
	Input       string
	Output      string
	Source      string
	Destination string
	Options     string
}

// ChainResult holds the parsed entries and default policy for a single chain.
type ChainResult struct {
	Policy  string // e.g., "ACCEPT", "DROP", "REJECT"
	Entries []Stat
}

func (ie *mqlIptablesEntry) id() (string, error) {
	return strconv.FormatInt(ie.LineNumber.Data, 10) + ie.Chain.Data, nil
}

type mqlIptablesTableInternal struct {
	ipVersion string
}

type mqlIptablesChainInternal struct {
	ipVersion string
}

func (t *mqlIptablesTable) id() (string, error) {
	return t.ipVersion + "/" + t.Name.Data, nil
}

func (c *mqlIptablesChain) id() (string, error) {
	return c.ipVersion + "/" + c.Table.Data + "/" + c.Name.Data, nil
}

// statToRawData converts a Stat into the llx data map for creating an iptables.entry resource.
func statToRawData(stat Stat, chain string) map[string]*llx.RawData {
	return map[string]*llx.RawData{
		"lineNumber":  llx.IntData(stat.LineNumber),
		"packets":     llx.IntData(stat.Packets),
		"bytes":       llx.IntData(stat.Bytes),
		"target":      llx.StringData(stat.Target),
		"protocol":    llx.StringData(stat.Protocol),
		"opt":         llx.StringData(stat.Opt),
		"in":          llx.StringData(stat.Input),
		"out":         llx.StringData(stat.Output),
		"source":      llx.StringData(stat.Source),
		"destination": llx.StringData(stat.Destination),
		"options":     llx.StringData(stat.Options),
		"chain":       llx.StringData(chain),
	}
}

// chainCache stores the result of a single chain query so that both the
// entries and policy fields can be served from one shell command.
// Fields are written inside the sync.Once callback and read after Do returns;
// sync.Once guarantees happens-before visibility of all writes to subsequent callers.
type chainCache struct {
	once    sync.Once
	entries []any
	policy  string
	err     error
}

// mqlIptablesInternal caches chain results to avoid running the same
// iptables command twice when both entries and policy are queried.
type mqlIptablesInternal struct {
	inputCache   chainCache
	outputCache  chainCache
	forwardCache chainCache
	tablesOnce   sync.Once
	tablesCache  []any
	tablesErr    error
}

// mqlIp6tablesInternal caches chain results to avoid running the same
// ip6tables command twice when both entries and policy are queried.
type mqlIp6tablesInternal struct {
	inputCache   chainCache
	outputCache  chainCache
	forwardCache chainCache
	tablesOnce   sync.Once
	tablesCache  []any
	tablesErr    error
}

// fetchChain runs an iptables/ip6tables command for a chain, parses the output,
// and creates MQL entry resources. Returns entries, policy, and any error.
func fetchChain(runtime *plugin.Runtime, conn shared.Connection, binary, chainName, mqlChainID string, ipv6 bool) ([]any, string, error) {
	cmd, err := conn.RunCommand(fmt.Sprintf("%s -L %s -v -n -x --line-numbers", binary, chainName))
	if err != nil {
		return nil, "", err
	}
	data, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		return nil, "", err
	}
	if cmd.ExitStatus != 0 {
		outErr, _ := io.ReadAll(cmd.Stderr)
		return nil, "", errors.New(string(outErr))
	}

	lines := getLines(string(data))
	result, err := ParseChain(lines, ipv6)
	if err != nil {
		return nil, "", err
	}

	entries := make([]any, 0, len(result.Entries))
	for _, stat := range result.Entries {
		entry, err := CreateResource(runtime, "iptables.entry", statToRawData(stat, mqlChainID))
		if err != nil {
			return nil, "", err
		}
		entries = append(entries, entry.(*mqlIptablesEntry))
	}
	return entries, result.Policy, nil
}

// iptablesTableNames lists the standard tables to query.
var iptablesTableNames = []string{"filter", "nat", "mangle", "raw"}

// fetchAllTables runs `iptables -t <table> -L -v -n -x --line-numbers` for each
// table and returns MQL table resources containing chains and entries.
func fetchAllTables(runtime *plugin.Runtime, conn shared.Connection, binary string, ipv6 bool) ([]any, error) {
	ver := "ipv4"
	if ipv6 {
		ver = "ipv6"
	}

	var tables []any
	for _, tableName := range iptablesTableNames {
		// tableName comes from the compile-time iptablesTableNames constant.
		cmd, err := conn.RunCommand(fmt.Sprintf("%s -t %s -L -v -n -x --line-numbers", binary, tableName))
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(cmd.Stdout)
		if err != nil {
			return nil, err
		}
		if cmd.ExitStatus != 0 {
			stderr, _ := io.ReadAll(cmd.Stderr)
			errMsg := strings.TrimSpace(string(stderr))
			// Table may not exist on this kernel (e.g., raw or mangle not loaded)
			if strings.Contains(errMsg, "does not exist") ||
				strings.Contains(errMsg, "No such file or directory") ||
				strings.Contains(errMsg, "can't initialize") {
				continue
			}
			return nil, fmt.Errorf("%s -t %s failed: %s", binary, tableName, errMsg)
		}

		chains, err := parseAllChains(runtime, string(data), tableName, ver, ipv6)
		if err != nil {
			return nil, err
		}
		if len(chains) == 0 {
			continue
		}

		tableRes, err := CreateResource(runtime, "iptables.table", map[string]*llx.RawData{
			"name":   llx.StringData(tableName),
			"chains": llx.ArrayData(chains, types.Resource("iptables.chain")),
		})
		if err != nil {
			return nil, err
		}
		tableRes.(*mqlIptablesTable).ipVersion = ver
		tables = append(tables, tableRes)
	}
	return tables, nil
}

// parseAllChains parses the full output of `iptables -t <table> -L` which
// contains multiple chain blocks separated by blank lines.
func parseAllChains(runtime *plugin.Runtime, output, tableName, ipVersion string, ipv6 bool) ([]any, error) {
	blocks := splitChainBlocks(output)
	var chains []any

	for _, block := range blocks {
		lines := getLines(block)
		if len(lines) < 2 {
			continue
		}

		chainName := parseChainName(lines[0])
		if chainName == "" {
			continue
		}

		result, err := ParseChain(lines, ipv6)
		if err != nil {
			return nil, err
		}

		chainID := ipVersion + "/" + tableName + "/" + chainName
		entries := make([]any, 0, len(result.Entries))
		for _, stat := range result.Entries {
			entry, err := CreateResource(runtime, "iptables.entry", statToRawData(stat, chainID))
			if err != nil {
				return nil, err
			}
			entries = append(entries, entry)
		}

		chainRes, err := CreateResource(runtime, "iptables.chain", map[string]*llx.RawData{
			"table":  llx.StringData(tableName),
			"name":   llx.StringData(chainName),
			"policy": llx.StringData(result.Policy),
			"rules":  llx.ArrayData(entries, types.Resource("iptables.entry")),
		})
		if err != nil {
			return nil, err
		}
		chainRes.(*mqlIptablesChain).ipVersion = ipVersion
		chains = append(chains, chainRes)
	}
	return chains, nil
}

// splitChainBlocks splits the output of `iptables -L` (all chains) into
// individual chain blocks. Each block starts with "Chain ..." and is
// separated by empty lines.
func splitChainBlocks(output string) []string {
	var blocks []string
	var current []string

	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "Chain ") && len(current) > 0 {
			blocks = append(blocks, strings.Join(current, "\n"))
			current = nil
		}
		if line == "" && len(current) > 0 {
			continue
		}
		if line != "" || len(current) > 0 {
			current = append(current, line)
		}
	}
	if len(current) > 0 {
		blocks = append(blocks, strings.Join(current, "\n"))
	}
	return blocks
}

// reChainName matches "Chain CHAINNAME (...)" and extracts the chain name.
var reChainName = regexp.MustCompile(`^Chain\s+(\S+)`)

// parseChainName extracts the chain name from a chain header line.
func parseChainName(line string) string {
	m := reChainName.FindStringSubmatch(line)
	if m == nil {
		return ""
	}
	return m[1]
}

// --- iptables (IPv4) ---

func (i *mqlIptables) fetchInput() {
	conn := i.MqlRuntime.Connection.(shared.Connection)
	i.inputCache.entries, i.inputCache.policy, i.inputCache.err =
		fetchChain(i.MqlRuntime, conn, "iptables", "INPUT", "input", false)
}

func (i *mqlIptables) fetchOutput() {
	conn := i.MqlRuntime.Connection.(shared.Connection)
	i.outputCache.entries, i.outputCache.policy, i.outputCache.err =
		fetchChain(i.MqlRuntime, conn, "iptables", "OUTPUT", "output", false)
}

func (i *mqlIptables) fetchForward() {
	conn := i.MqlRuntime.Connection.(shared.Connection)
	i.forwardCache.entries, i.forwardCache.policy, i.forwardCache.err =
		fetchChain(i.MqlRuntime, conn, "iptables", "FORWARD", "forward", false)
}

func (i *mqlIptables) input() ([]any, error) {
	i.inputCache.once.Do(i.fetchInput)
	return i.inputCache.entries, i.inputCache.err
}

func (i *mqlIptables) output() ([]any, error) {
	i.outputCache.once.Do(i.fetchOutput)
	return i.outputCache.entries, i.outputCache.err
}

func (i *mqlIptables) forward() ([]any, error) {
	i.forwardCache.once.Do(i.fetchForward)
	return i.forwardCache.entries, i.forwardCache.err
}

func (i *mqlIptables) inputPolicy() (string, error) {
	i.inputCache.once.Do(i.fetchInput)
	return i.inputCache.policy, i.inputCache.err
}

func (i *mqlIptables) outputPolicy() (string, error) {
	i.outputCache.once.Do(i.fetchOutput)
	return i.outputCache.policy, i.outputCache.err
}

func (i *mqlIptables) forwardPolicy() (string, error) {
	i.forwardCache.once.Do(i.fetchForward)
	return i.forwardCache.policy, i.forwardCache.err
}

func (i *mqlIptables) tables() ([]any, error) {
	i.tablesOnce.Do(func() {
		conn := i.MqlRuntime.Connection.(shared.Connection)
		i.tablesCache, i.tablesErr = fetchAllTables(i.MqlRuntime, conn, "iptables", false)
	})
	return i.tablesCache, i.tablesErr
}

// --- ip6tables (IPv6) ---

func (i *mqlIp6tables) fetchInput() {
	conn := i.MqlRuntime.Connection.(shared.Connection)
	i.inputCache.entries, i.inputCache.policy, i.inputCache.err =
		fetchChain(i.MqlRuntime, conn, "ip6tables", "INPUT", "input6", true)
}

func (i *mqlIp6tables) fetchOutput() {
	conn := i.MqlRuntime.Connection.(shared.Connection)
	i.outputCache.entries, i.outputCache.policy, i.outputCache.err =
		fetchChain(i.MqlRuntime, conn, "ip6tables", "OUTPUT", "output6", true)
}

func (i *mqlIp6tables) fetchForward() {
	conn := i.MqlRuntime.Connection.(shared.Connection)
	i.forwardCache.entries, i.forwardCache.policy, i.forwardCache.err =
		fetchChain(i.MqlRuntime, conn, "ip6tables", "FORWARD", "forward6", true)
}

func (i *mqlIp6tables) input() ([]any, error) {
	i.inputCache.once.Do(i.fetchInput)
	return i.inputCache.entries, i.inputCache.err
}

func (i *mqlIp6tables) output() ([]any, error) {
	i.outputCache.once.Do(i.fetchOutput)
	return i.outputCache.entries, i.outputCache.err
}

func (i *mqlIp6tables) forward() ([]any, error) {
	i.forwardCache.once.Do(i.fetchForward)
	return i.forwardCache.entries, i.forwardCache.err
}

func (i *mqlIp6tables) inputPolicy() (string, error) {
	i.inputCache.once.Do(i.fetchInput)
	return i.inputCache.policy, i.inputCache.err
}

func (i *mqlIp6tables) outputPolicy() (string, error) {
	i.outputCache.once.Do(i.fetchOutput)
	return i.outputCache.policy, i.outputCache.err
}

func (i *mqlIp6tables) forwardPolicy() (string, error) {
	i.forwardCache.once.Do(i.fetchForward)
	return i.forwardCache.policy, i.forwardCache.err
}

func (i *mqlIp6tables) tables() ([]any, error) {
	i.tablesOnce.Do(func() {
		conn := i.MqlRuntime.Connection.(shared.Connection)
		i.tablesCache, i.tablesErr = fetchAllTables(i.MqlRuntime, conn, "ip6tables", true)
	})
	return i.tablesCache, i.tablesErr
}

// Credit to github.com/coreos/go-iptables for some of the parsing logic
func getLines(data string) []string {
	rules := strings.Split(data, "\n")

	// strip trailing newline
	if len(rules) > 0 && rules[len(rules)-1] == "" {
		rules = rules[:len(rules)-1]
	}

	return rules
}

// reChainPolicy matches the chain header line, e.g.:
//
//	"Chain INPUT (policy DROP 227 packets, 12904 bytes)"
var reChainPolicy = regexp.MustCompile(`^Chain\s+\S+\s+\(policy\s+([A-Z]+)`)

// ParseChainPolicy extracts the default policy from the first line of
// iptables/ip6tables -L output. Returns "" for user-defined chains or
// missing headers.
func ParseChainPolicy(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	if m := reChainPolicy.FindStringSubmatch(lines[0]); m != nil {
		return m[1]
	}
	return ""
}

// ParseChain parses the full output of an iptables/ip6tables -L command,
// returning both the chain's default policy and its rule entries.
func ParseChain(lines []string, ipv6 bool) (ChainResult, error) {
	result := ChainResult{
		Policy: ParseChainPolicy(lines),
	}
	var err error
	result.Entries, err = ParseStat(lines, ipv6)
	return result, err
}

// ParseStat parses the tabular rule entries from iptables/ip6tables -L output.
// Lines 0-1 (chain header + column names) are skipped.
func ParseStat(lines []string, ipv6 bool) ([]Stat, error) {
	entries := []Stat{}
	for i, line := range lines {
		// Skip over chain name and field header
		if i < 2 {
			continue
		}

		// Fields:
		// 0=linenumber 1=pkts 2=bytes 3=target 4=prot 5=opt 6=in 7=out 8=source 9=destination 10=options
		line = strings.TrimSpace(line)
		fields := strings.Fields(line)

		// The ip6tables verbose output cannot be naively split due to the default "opt"
		// field containing 2 single spaces.
		if ipv6 {
			// Check if field 7 is "out" or "source" address
			dest := fields[7]
			ip, _, _ := net.ParseCIDR(dest)
			if ip == nil {
				ip = net.ParseIP(dest)
			}

			// If we detected a CIDR or IP, the "opt" field is empty.. insert it.
			if ip != nil {
				f := []string{}
				f = append(f, fields[:5]...)
				f = append(f, "  ") // Empty "opt" field for ip6tables
				f = append(f, fields[5:]...)
				fields = f
			}
		}
		ln, err := strconv.ParseInt(fields[0], 0, 64)
		if err != nil {
			return entries, fmt.Errorf(err.Error(), "could not parse line number")
		}
		pkts, err := strconv.ParseInt(fields[1], 0, 64)
		if err != nil {
			return entries, fmt.Errorf(err.Error(), "could not parse packets")
		}
		bts, err := strconv.ParseInt(fields[2], 0, 64)
		if err != nil {
			return entries, fmt.Errorf(err.Error(), "could not parse bytes")
		}
		var opts string
		// combine options if they exist
		if len(fields) > 10 {
			o := fields[10:]
			opts = strings.Join(o, " ")
		}
		entry := Stat{
			LineNumber: ln,
			Packets:    pkts,
			Bytes:      bts,
			Target:     fields[3],
			Protocol:   fields[4],
			Opt:        fields[5],
			Input:      fields[6],
			Output:     fields[7],
			Source:     fields[8],
			Options:    opts,
		}

		if len(fields) > 9 {
			entry.Destination = fields[9]
		}
		entries = append(entries, entry)
	}
	return entries, nil
}
