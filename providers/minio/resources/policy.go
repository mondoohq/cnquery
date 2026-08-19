// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"sort"
	"strings"
)

// wildcardPrincipal is the principal MinIO treats as "anyone, including an
// unauthenticated caller".
const wildcardPrincipal = "*"

// adminActionPrefix marks an action in the deployment's admin API namespace,
// which covers reconfiguring the deployment and creating further identities.
const adminActionPrefix = "admin:"

// iamPolicy is a MinIO access policy document. The same shape is used for a
// bucket's access policy and for a policy defined by name in the identity
// store; the difference is that a bucket policy names its principals while a
// named policy leaves them to the attachment.
type iamPolicy struct {
	Version    string            `json:"Version"`
	ID         string            `json:"ID,omitempty"`
	Statements []policyStatement `json:"Statement"`
}

// policyStatement is one statement of an access policy.
type policyStatement struct {
	SID          string                          `json:"Sid,omitempty"`
	Effect       string                          `json:"Effect"`
	Principal    policyPrincipal                 `json:"Principal,omitempty"`
	NotPrincipal policyPrincipal                 `json:"NotPrincipal,omitempty"`
	Action       stringSet                       `json:"Action,omitempty"`
	NotAction    stringSet                       `json:"NotAction,omitempty"`
	Resource     stringSet                       `json:"Resource,omitempty"`
	NotResource  stringSet                       `json:"NotResource,omitempty"`
	Condition    map[string]map[string]stringSet `json:"Condition,omitempty"`
}

// stringSet decodes a policy field that is written either as one string or as a
// list of strings. Policy documents use both spellings interchangeably, and a
// plain []string tag silently fails on the single-string form, which would
// report "no actions" on a statement that has one.
type stringSet []string

func (s *stringSet) UnmarshalJSON(data []byte) error {
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*s = stringSet{single}
		return nil
	}
	var many []string
	if err := json.Unmarshal(data, &many); err != nil {
		return err
	}
	*s = stringSet(many)
	return nil
}

// policyPrincipal decodes the Principal field, which is written either as the
// bare string "*" or as an object keyed by principal type, for example
// {"AWS": ["*"]}. Values inside the object are themselves string-or-list.
type policyPrincipal struct {
	// Values holds every principal named by the statement, flattened across
	// principal types. The type is not retained because MinIO evaluates the
	// wildcard the same way regardless of which key carries it.
	Values []string
}

func (p *policyPrincipal) UnmarshalJSON(data []byte) error {
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		if single != "" {
			p.Values = []string{single}
		}
		return nil
	}

	var byType map[string]stringSet
	if err := json.Unmarshal(data, &byType); err != nil {
		// A principal shape we do not recognize leaves the statement with no
		// principals rather than failing the whole document, so one odd
		// statement cannot blind every other statement in the policy.
		return nil
	}

	keys := make([]string, 0, len(byType))
	for k := range byType {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		p.Values = append(p.Values, byType[k]...)
	}
	return nil
}

// UnmarshalJSON on the policy handles the Statement field being written as a
// single object rather than a list, which hand-written policies do.
func (p *iamPolicy) UnmarshalJSON(data []byte) error {
	type rawPolicy struct {
		Version   string          `json:"Version"`
		ID        string          `json:"ID"`
		Statement json.RawMessage `json:"Statement"`
	}
	var raw rawPolicy
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	p.Version = raw.Version
	p.ID = raw.ID
	if len(raw.Statement) == 0 {
		return nil
	}

	var list []policyStatement
	if err := json.Unmarshal(raw.Statement, &list); err == nil {
		p.Statements = list
		return nil
	}
	var one policyStatement
	if err := json.Unmarshal(raw.Statement, &one); err != nil {
		return err
	}
	p.Statements = []policyStatement{one}
	return nil
}

// parsePolicyDocument decodes a policy document. An empty document is not an
// error: MinIO answers a bucket with no access policy with an empty body, and
// that is a real answer meaning "no policy" rather than a failure to read one.
func parsePolicyDocument(document string) (*iamPolicy, error) {
	if strings.TrimSpace(document) == "" {
		return nil, nil
	}
	var policy iamPolicy
	if err := json.Unmarshal([]byte(document), &policy); err != nil {
		return nil, err
	}
	return &policy, nil
}

func (s policyStatement) isAllow() bool { return strings.EqualFold(s.Effect, "Allow") }

func (s policyStatement) isDeny() bool { return strings.EqualFold(s.Effect, "Deny") }

// policyHasWildcardPrincipal reports whether any statement names the wildcard
// principal, whatever its effect. A Deny statement naming "*" is how a policy
// forbids something for everyone, so this is not on its own an exposure.
func policyHasWildcardPrincipal(policy *iamPolicy) bool {
	if policy == nil {
		return false
	}
	for _, statement := range policy.Statements {
		if containsValue(statement.Principal.Values, wildcardPrincipal) {
			return true
		}
	}
	return false
}

// policyGrantsAnonymousAccess reports whether an Allow statement names the
// wildcard principal, which is what makes a bucket reachable without
// credentials.
func policyGrantsAnonymousAccess(policy *iamPolicy) bool {
	if policy == nil {
		return false
	}
	for _, statement := range policy.Statements {
		if !statement.isAllow() {
			continue
		}
		if containsValue(statement.Principal.Values, wildcardPrincipal) {
			return true
		}
	}
	return false
}

// policyHasWildcardAction reports whether an Allow statement grants an action
// carrying a wildcard, for example s3:* or s3:Get*.
func policyHasWildcardAction(policy *iamPolicy) bool {
	if policy == nil {
		return false
	}
	for _, statement := range policy.Statements {
		if !statement.isAllow() {
			continue
		}
		for _, action := range statement.Action {
			if strings.Contains(action, "*") {
				return true
			}
		}
	}
	return false
}

// policyHasWildcardResource reports whether an Allow statement applies to a
// resource carrying a wildcard, for example arn:aws:s3:::*.
func policyHasWildcardResource(policy *iamPolicy) bool {
	if policy == nil {
		return false
	}
	for _, statement := range policy.Statements {
		if !statement.isAllow() {
			continue
		}
		for _, resource := range statement.Resource {
			if strings.Contains(resource, "*") {
				return true
			}
		}
	}
	return false
}

// policyGrantsAdminAccess reports whether an Allow statement grants any action
// in the admin API namespace.
func policyGrantsAdminAccess(policy *iamPolicy) bool {
	if policy == nil {
		return false
	}
	for _, statement := range policy.Statements {
		if !statement.isAllow() {
			continue
		}
		for _, action := range statement.Action {
			if strings.HasPrefix(strings.ToLower(action), adminActionPrefix) {
				return true
			}
			// A bare "*" grants every namespace, the admin one included.
			if action == "*" {
				return true
			}
		}
	}
	return false
}

// policyEnforcesSslOnly reports whether the policy contains a Deny statement
// conditioned on aws:SecureTransport being false, which rejects any request
// that does not use HTTPS.
func policyEnforcesSslOnly(policy *iamPolicy) bool {
	if policy == nil {
		return false
	}
	for _, statement := range policy.Statements {
		if !statement.isDeny() {
			continue
		}
		for operator, keys := range statement.Condition {
			if !strings.EqualFold(operator, "Bool") {
				continue
			}
			for key, values := range keys {
				if !strings.EqualFold(key, "aws:SecureTransport") {
					continue
				}
				for _, v := range values {
					if strings.EqualFold(v, "false") {
						return true
					}
				}
			}
		}
	}
	return false
}

func containsValue(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}

// conditionsToDict renders a statement's conditions for the schema. Values are
// always emitted as lists, matching how MinIO returns a policy it stored: a
// condition written with a single value comes back wrapped in a list.
func conditionsToDict(conditions map[string]map[string]stringSet) map[string]any {
	if len(conditions) == 0 {
		return nil
	}
	out := make(map[string]any, len(conditions))
	for operator, keys := range conditions {
		inner := make(map[string]any, len(keys))
		for key, values := range keys {
			list := make([]any, 0, len(values))
			for _, v := range values {
				list = append(list, v)
			}
			inner[key] = list
		}
		out[operator] = inner
	}
	return out
}
