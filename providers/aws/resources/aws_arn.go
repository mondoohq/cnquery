// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"
	"slices"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// arnSpec declares how one MQL resource resolves the ARN that identifies it.
//
// Resource init functions adopt the scanned asset's ARN when invoked with no
// arguments, so that a bare singular query like `aws.msk.cluster` resolves to
// the asset being scanned. Without a service check, an asset from an unrelated
// service (e.g. a Route53 hosted zone being synchronized while a policy
// references aws.msk.cluster) would be adopted as this resource's own ARN,
// fabricating a bogus husk resource keyed on the wrong ARN. Declaring the
// accepted services once, next to the init that uses them, applies the same
// check to the asset ARN and to an ARN the caller supplied explicitly.
//
// Declare one as a package-level var next to the init function it serves.
type arnSpec struct {
	// resource is the MQL resource name, given as the generated constant for
	// it (ResourceAwsMskCluster) so a typo cannot compile. It names the
	// resource in every error message.
	resource string

	// services lists the ARN service tokens this resource accepts ("kafka").
	// Leaving it empty rejects every ARN, making the resource unresolvable.
	services []string

	// prefix optionally requires the ARN's resource segment to start with a
	// given string ("cluster/"), and is trimmed off arnRef.ResourceID.
	//
	// Set it only where a mismatched segment is already an error today. Inits
	// that treat the segment as a hint and fall back to scanning the parent
	// collection (aws.vpc, aws.ec2.snapshot, aws.ecr.repository) must leave it
	// empty and keep testing arnRef.Resource themselves, so a soft fallback
	// does not turn into a hard failure.
	prefix string

	// altKeys lists the other init args that can identify this resource
	// ("name", "id"). When it is empty the ARN is mandatory; otherwise resolve
	// succeeds with a zero arnRef as long as one of these args is present, and
	// the init resolves by that key instead.
	altKeys []string

	// allowRef permits the "arn" arg to carry a non-ARN reference. Only
	// aws.kms.key needs this: it accepts a bare key ID or an "alias/..." name
	// in the same arg, and normalizes them itself.
	allowRef bool
}

// arnRef is an ARN that satisfied its arnSpec. Raw is "" when the spec allowed
// the ARN to be absent because an altKey was supplied instead.
type arnRef struct {
	// ARN is the parsed form, for Region, AccountID and Resource. It is the
	// zero value when Raw is empty or held a non-ARN reference (see allowRef).
	arn.ARN

	// RawArn is the ARN exactly as supplied, never reconstructed from ARN.
	RawArn string

	// ResourceID is ARN.Resource with the spec's prefix trimmed off, so
	// "cluster/my-cluster" reads back as "my-cluster".
	ResourceID string
}

// String returns the ARN as supplied, shadowing arn.ARN.String so a ref never
// stringifies to a reconstruction of itself.
func (r arnRef) String() string { return r.RawArn }

// resolve produces the ARN identifying this resource, adopting the scanned
// asset's ARN when args is empty. On a nil error the returned ref is either
// well-formed and belongs to one of the spec's services, or empty because the
// spec accepts an altKey that args supplied instead.
//
// It writes an adopted ARN back into args["arn"], so callers that pass args on
// to CreateResource see it.
func (s arnSpec) resolve(runtime *plugin.Runtime, args map[string]*llx.RawData) (arnRef, error) {
	if len(args) == 0 {
		if ref := s.assetRef(runtime); ref.RawArn != "" {
			args["arn"] = llx.StringData(ref.RawArn)
		}
	}

	raw := args["arn"]
	if raw == nil {
		if s.hasAltKey(args) {
			return arnRef{}, nil
		}
		return arnRef{}, fmt.Errorf("%s required to fetch %s", s.keyList(), s.resource)
	}

	// Nothing can reach this today: the schema types arn as a string, so mqlc
	// rejects anything else before the provider runs, and every NewResource
	// caller that builds an arn from a *string nil-checks it first (a nil one
	// would arrive as llx.NilData, whose Value is nil). Reading it with
	// comma-ok keeps that true without depending on the convention holding:
	// the alternative, a bare args["arn"].Value.(string), turns a future
	// unguarded caller into an interface-conversion panic that GetData's
	// recover reports as an Internal error naming no resource or field.
	val, ok := raw.Value.(string)
	if !ok {
		return arnRef{}, fmt.Errorf("arn for %s must be a string, got %T", s.resource, raw.Value)
	}
	return s.parse(val)
}

// assetRef returns the scanned asset's ARN when it satisfies the spec, and the
// zero arnRef otherwise. A zero ref leaves args["arn"] unset so resolve falls
// through to its "arn required" path instead of adopting a foreign ARN.
//
// Call it directly instead of resolve in inits that identify the resource by
// args derived from the asset ARN rather than by the ARN itself (region+name
// for aws.codebuild.project, directoryId for aws.directoryservice.directory),
// so that args does not gain an "arn" key those inits never read.
func (s arnSpec) assetRef(runtime *plugin.Runtime) arnRef {
	assetArn := getAssetIdentifier(runtime)
	if assetArn == "" {
		return arnRef{}
	}
	ref, err := s.parse(assetArn)
	if err != nil {
		return arnRef{}
	}
	return ref
}

// parse validates a single ARN string against the spec.
func (s arnSpec) parse(val string) (arnRef, error) {
	if s.allowRef && !strings.HasPrefix(val, "arn:") {
		return arnRef{RawArn: val}, nil
	}

	parsed, err := arn.Parse(val)
	if err != nil {
		return arnRef{}, fmt.Errorf("invalid arn %q for %s: %w", val, s.resource, err)
	}
	if !slices.Contains(s.services, parsed.Service) {
		return arnRef{}, fmt.Errorf("arn %q is not an %s arn: service is %q, expected %s",
			val, s.resource, parsed.Service, strings.Join(s.services, " or "))
	}
	if s.prefix != "" && !strings.HasPrefix(parsed.Resource, s.prefix) {
		return arnRef{}, fmt.Errorf("arn %q is not an %s arn: resource is %q, expected it to start with %q",
			val, s.resource, parsed.Resource, s.prefix)
	}

	return arnRef{
		ARN:        parsed,
		RawArn:     val,
		ResourceID: strings.TrimPrefix(parsed.Resource, s.prefix),
	}, nil
}

// hasAltKey reports whether args carries one of the alternative identifying
// keys this resource accepts in place of an ARN.
func (s arnSpec) hasAltKey(args map[string]*llx.RawData) bool {
	for _, key := range s.altKeys {
		if args[key] != nil {
			return true
		}
	}
	return false
}

// keyList renders the accepted identifying args for an error message, as
// "arn", "arn or name", or "arn, id, or name".
func (s arnSpec) keyList() string {
	keys := append([]string{"arn"}, s.altKeys...)
	switch len(keys) {
	case 1:
		return keys[0]
	case 2:
		return keys[0] + " or " + keys[1]
	default:
		return strings.Join(keys[:len(keys)-1], ", ") + ", or " + keys[len(keys)-1]
	}
}
