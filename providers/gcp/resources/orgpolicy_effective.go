// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	orgpolicy "cloud.google.com/go/orgpolicy/apiv2"
	"cloud.google.com/go/orgpolicy/apiv2/orgpolicypb"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/gcp/connection"
	"google.golang.org/api/option"
)

type mqlGcpProjectEffectiveOrgPolicyInternal struct {
	policy    *orgpolicypb.Policy
	policyErr error
	loaded    atomic.Bool
	lock      sync.Mutex
}

// normalizeConstraintName accepts a constraint either as it appears in an org
// policy resource path (compute.requireOsLogin) or with the "constraints/"
// prefix used throughout the org policy documentation and the
// gcp.orgPolicy.constraint resource, and returns the bare form the policy path
// expects.
func normalizeConstraintName(constraint string) string {
	return strings.TrimPrefix(constraint, "constraints/")
}

// initGcpProjectEffectiveOrgPolicy scopes the lookup to the connected project and
// derives the cache key from the constraint under test. Both selectors are
// declared as real fields so the named-argument form compiles.
func initGcpProjectEffectiveOrgPolicy(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if args == nil {
		args = make(map[string]*llx.RawData)
	}

	conn, ok := runtime.Connection.(*connection.GcpConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not a GCP connection")
	}

	if args["projectId"] == nil {
		args["projectId"] = llx.StringData(conn.ResourceID())
	}
	if args["constraint"] == nil {
		args["constraint"] = llx.StringData("")
	}

	// Parameterized resource: the cache key must carry the constraint or every
	// constraint queried in one scan resolves to the first one's answer.
	args["__id"] = llx.StringData(fmt.Sprintf("%s\x00effectiveOrgPolicy\x00%s",
		rawDataString(args["projectId"]), normalizeConstraintName(rawDataString(args["constraint"]))))

	return args, nil, nil
}

// effectivePolicy resolves the constraint's policy with inheritance applied. All
// the derived fields read from this one call.
func (g *mqlGcpProjectEffectiveOrgPolicy) effectivePolicy() (*orgpolicypb.Policy, error) {
	if g.loaded.Load() {
		return g.policy, g.policyErr
	}
	g.lock.Lock()
	defer g.lock.Unlock()
	if g.loaded.Load() {
		return g.policy, g.policyErr
	}
	defer g.loaded.Store(true)

	g.policy, g.policyErr = g.fetchEffectivePolicy()
	return g.policy, g.policyErr
}

func (g *mqlGcpProjectEffectiveOrgPolicy) fetchEffectivePolicy() (*orgpolicypb.Policy, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	if g.Constraint.Error != nil {
		return nil, g.Constraint.Error
	}
	projectId := g.ProjectId.Data
	constraint := normalizeConstraintName(g.Constraint.Data)

	if constraint == "" {
		return nil, errors.New("gcp.project.effectiveOrgPolicy requires a constraint, e.g. " +
			`gcp.project.effectiveOrgPolicy(constraint: "constraints/compute.requireOsLogin")`)
	}

	conn, ok := g.MqlRuntime.Connection.(*connection.GcpConnection)
	if !ok {
		return nil, errors.New("invalid connection provided, it is not a GCP connection")
	}
	creds, err := conn.Credentials(orgpolicy.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := orgpolicy.NewClient(ctx, option.WithCredentials(creds), connection.GRPCClientTraceOption())
	if err != nil {
		return nil, err
	}
	defer client.Close()

	policy, err := client.GetEffectivePolicy(ctx, &orgpolicypb.GetEffectivePolicyRequest{
		Name: fmt.Sprintf("projects/%s/policies/%s", projectId, constraint),
	})
	if err != nil {
		return nil, err
	}
	return policy, nil
}

// interpret decodes the effective spec into the scalar predicates. The effective
// policy carries a single merged spec, so unlike gcp.orgPolicy there is no
// separate dry-run spec to distinguish.
func (g *mqlGcpProjectEffectiveOrgPolicy) interpret() (enforced, allowAll, denyAll bool, allowedValues, deniedValues []any, err error) {
	policy, err := g.effectivePolicy()
	if err != nil {
		return false, false, false, nil, nil, err
	}
	enforced, allowAll, denyAll, _, allowedValues, deniedValues = interpretPolicySpec(policy.GetSpec())
	return enforced, allowAll, denyAll, allowedValues, deniedValues, nil
}

func (g *mqlGcpProjectEffectiveOrgPolicy) enforced() (bool, error) {
	enforced, _, _, _, _, err := g.interpret()
	return enforced, err
}

func (g *mqlGcpProjectEffectiveOrgPolicy) allowAll() (bool, error) {
	_, allowAll, _, _, _, err := g.interpret()
	return allowAll, err
}

func (g *mqlGcpProjectEffectiveOrgPolicy) denyAll() (bool, error) {
	_, _, denyAll, _, _, err := g.interpret()
	return denyAll, err
}

func (g *mqlGcpProjectEffectiveOrgPolicy) allowedValues() ([]any, error) {
	_, _, _, allowedValues, _, err := g.interpret()
	return allowedValues, err
}

func (g *mqlGcpProjectEffectiveOrgPolicy) deniedValues() ([]any, error) {
	_, _, _, _, deniedValues, err := g.interpret()
	return deniedValues, err
}

func (g *mqlGcpProjectEffectiveOrgPolicy) spec() (any, error) {
	policy, err := g.effectivePolicy()
	if err != nil {
		return nil, err
	}
	return protoToDict(policy.GetSpec())
}
