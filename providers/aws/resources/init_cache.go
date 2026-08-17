// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// cachedByArn returns a resource already materialized under this ARN, or nil.
//
// NewResource runs a resource's init *before* it consults the runtime cache --
// it has to, because the cache key is the resolved resource's MqlID, which for
// a resource whose identity is computed during init is not knowable in advance.
// An init that fetches unconditionally therefore pays once per referring
// resource and then has its result discarded in favour of the cached instance:
// four EC2 instances in one VPC cost four DescribeVpcs calls for the one VPC,
// and the fan-in on aws.iam.role reaches 94 referring call sites.
//
// The opening exists for callers that already supply the id. These resources
// key their __id on the ARN, and typed references pass the ARN, so the cache
// key is known up front and can be checked before spending a call.
//
// Same shape as the checks already in aws_kms.go, aws_vpc.go and aws_ec2.go;
// this only gives them one name.
func cachedByArn(runtime *plugin.Runtime, resourceName, arn string) plugin.Resource {
	if arn == "" || runtime == nil || runtime.Resources == nil {
		return nil
	}
	if res, ok := runtime.Resources.Get(resourceName + "\x00" + arn); ok {
		return res
	}
	return nil
}

// cachedArgByArn is cachedByArn for the common init shape, reading the ARN out
// of args itself. Returns nil when there is no usable "arn" argument, which
// leaves the caller on its existing path.
func cachedArgByArn(runtime *plugin.Runtime, resourceName string, args map[string]*llx.RawData) plugin.Resource {
	raw, ok := args["arn"]
	if !ok || raw == nil {
		return nil
	}
	arn, ok := raw.Value.(string)
	if !ok {
		return nil
	}
	return cachedByArn(runtime, resourceName, arn)
}
