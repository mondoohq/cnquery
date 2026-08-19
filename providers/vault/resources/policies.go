// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"sort"
	"strings"
	"sync"

	"github.com/hashicorp/hcl"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
)

// BuiltinRootPolicy is the name Vault reserves for its root policy. The policy
// cannot be modified or deleted, and its document is empty because the server
// grants it every capability on every path implicitly.
const BuiltinRootPolicy = "root"

// mqlVaultPolicyInternal caches the policy document. Vault has no endpoint that
// returns every policy body at once, so the list costs one call per policy if
// the documents are read eagerly. Listing policies is the common query, so the
// document is fetched only when a field that needs it is touched, and shared by
// all five of them once it is.
type mqlVaultPolicyInternal struct {
	lock         sync.Mutex
	fetched      bool
	cachedRules  string
	cachedParsed []policyRule
}

func (r *mqlVault) policies() ([]any, error) {
	client, err := vaultClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	names, err := client.Sys().ListPolicies()
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(names))
	for _, name := range names {
		mqlPolicy, err := CreateResource(r.MqlRuntime, "vault.policy", map[string]*llx.RawData{
			"__id": llx.StringData(name),
			"name": llx.StringData(name),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlPolicy)
	}
	return res, nil
}

// fetchPolicy reads the policy document once, under a lock, and parses it. The
// double check keeps a concurrent second reader from issuing the same call.
func (r *mqlVaultPolicy) fetchPolicy() (string, []policyRule, error) {
	if r.fetched {
		return r.cachedRules, r.cachedParsed, nil
	}

	r.lock.Lock()
	defer r.lock.Unlock()
	if r.fetched {
		return r.cachedRules, r.cachedParsed, nil
	}

	client, err := vaultClient(r.MqlRuntime)
	if err != nil {
		return "", nil, err
	}

	name := r.Name.Data
	rules, err := client.Sys().GetPolicy(name)
	if err != nil {
		return "", nil, err
	}

	r.cachedRules = rules
	r.cachedParsed = parsePolicyPaths(rules)
	r.fetched = true
	return r.cachedRules, r.cachedParsed, nil
}

func (r *mqlVaultPolicy) rules() (string, error) {
	rules, _, err := r.fetchPolicy()
	return rules, err
}

func (r *mqlVaultPolicy) paths() ([]any, error) {
	_, parsed, err := r.fetchPolicy()
	if err != nil {
		return nil, err
	}
	return convert.JsonToDictSlice(parsed)
}

func (r *mqlVaultPolicy) grantsSudo() (bool, error) {
	if isBuiltinRootPolicy(r.Name.Data) {
		return true, nil
	}
	_, parsed, err := r.fetchPolicy()
	if err != nil {
		return false, err
	}
	return grantsCapability(parsed, "sudo"), nil
}

func (r *mqlVaultPolicy) grantsRootPath() (bool, error) {
	if isBuiltinRootPolicy(r.Name.Data) {
		return true, nil
	}
	_, parsed, err := r.fetchPolicy()
	if err != nil {
		return false, err
	}
	return grantsRootPath(parsed), nil
}

func (r *mqlVaultPolicy) grantsWildcardPath() (bool, error) {
	if isBuiltinRootPolicy(r.Name.Data) {
		return true, nil
	}
	_, parsed, err := r.fetchPolicy()
	if err != nil {
		return false, err
	}
	return grantsWildcardPath(parsed), nil
}

// isBuiltinRootPolicy reports whether the policy is Vault's reserved root
// policy, whose grants do not appear in its document. Reading them off the text
// would report the most privileged policy on the server as granting nothing.
func isBuiltinRootPolicy(name string) bool {
	return name == BuiltinRootPolicy
}

// policyRule is one path block of a Vault ACL policy.
type policyRule struct {
	Path         string   `json:"path"`
	Capabilities []string `json:"capabilities"`
}

// parsePolicyPaths turns a policy document into its path rules. A policy that
// does not parse yields no rules rather than an error: Vault itself accepted
// the document, so a parse failure here is our limitation, and failing the
// whole policy list over one unreadable policy would hide every other one.
func parsePolicyPaths(rules string) []policyRule {
	if strings.TrimSpace(rules) == "" {
		return nil
	}

	var doc struct {
		Path []map[string]struct {
			Capabilities []string `hcl:"capabilities"`
		} `hcl:"path"`
	}
	if err := hcl.Decode(&doc, rules); err != nil {
		return nil
	}

	out := []policyRule{}
	for _, block := range doc.Path {
		for path, body := range block {
			capabilities := make([]string, 0, len(body.Capabilities))
			for _, capability := range body.Capabilities {
				capabilities = append(capabilities, strings.ToLower(strings.TrimSpace(capability)))
			}
			sort.Strings(capabilities)
			out = append(out, policyRule{Path: path, Capabilities: capabilities})
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// grantsCapability reports whether any rule carries the named capability.
func grantsCapability(rules []policyRule, capability string) bool {
	for _, rule := range rules {
		for _, granted := range rule.Capabilities {
			if granted == capability {
				return true
			}
		}
	}
	return false
}

// grantsRootPath reports whether a rule spans the whole secret namespace. A
// "deny" grant on such a path restricts rather than widens access, so it does
// not count.
func grantsRootPath(rules []policyRule) bool {
	for _, rule := range rules {
		path := strings.TrimPrefix(rule.Path, "/")
		if path == "*" && !isDenyOnly(rule) {
			return true
		}
	}
	return false
}

// grantsWildcardPath reports whether a rule matches more than one literal path,
// through either a trailing "*" or a "+" segment matcher. Deny-only rules do
// not count, for the same reason as grantsRootPath.
func grantsWildcardPath(rules []policyRule) bool {
	for _, rule := range rules {
		if isDenyOnly(rule) {
			continue
		}
		if strings.HasSuffix(rule.Path, "*") || hasSegmentMatcher(rule.Path) {
			return true
		}
	}
	return false
}

// hasSegmentMatcher reports whether the path uses "+" as a whole segment, which
// matches any single segment. A "+" inside a segment is a literal character.
func hasSegmentMatcher(path string) bool {
	for _, segment := range strings.Split(path, "/") {
		if segment == "+" {
			return true
		}
	}
	return false
}

// isDenyOnly reports whether the rule's only capability is deny. Vault treats
// deny as absolute, so such a rule can never widen access.
func isDenyOnly(rule policyRule) bool {
	if len(rule.Capabilities) == 0 {
		return false
	}
	for _, capability := range rule.Capabilities {
		if capability != "deny" {
			return false
		}
	}
	return true
}
