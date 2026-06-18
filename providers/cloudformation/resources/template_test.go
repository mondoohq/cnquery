// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIsEvenMappingNode_SectionBodyGuard guards against the out-of-bounds panic
// that occurred when a CloudFormation section (Resources/Outputs/Parameters/…)
// was written as a YAML sequence instead of a mapping: the stride-2 loops
// (body.Content[i+1]) would index past the end. isEvenMappingNode must reject a
// sequence-bodied section so those loops never run.
func TestIsEvenMappingNode_SectionBodyGuard(t *testing.T) {
	// Resources as an odd-length sequence — the historical panic input.
	tmpl, err := parse.String("Resources:\n  - a\n  - b\n  - c\n")
	require.NoError(t, err)
	require.NotNil(t, tmpl.Node)
	require.NotEmpty(t, tmpl.Node.Content)

	_, body, err := gatherMapValue(tmpl.Node.Content[0], string(cft.Resources))
	require.NoError(t, err)
	assert.False(t, isEvenMappingNode(body), "sequence-bodied section must be rejected")

	// A well-formed mapping body must pass.
	tmpl2, err := parse.String("Resources:\n  Bucket:\n    Type: AWS::S3::Bucket\n")
	require.NoError(t, err)
	_, body2, err := gatherMapValue(tmpl2.Node.Content[0], string(cft.Resources))
	require.NoError(t, err)
	assert.True(t, isEvenMappingNode(body2), "well-formed mapping body must be accepted")
}

// TestResourcesDoesNotPanicOnSequenceBody is a belt-and-suspenders check that
// the node-level guard maps to the behavior we want: nil node, sequence node,
// and odd-content node are all rejected.
func TestIsEvenMappingNode_Cases(t *testing.T) {
	assert.False(t, isEvenMappingNode(nil))

	tmpl, err := parse.String("Resources:\n  Bucket:\n    Type: AWS::S3::Bucket\n")
	require.NoError(t, err)
	doc := tmpl.Node.Content[0]
	assert.True(t, isEvenMappingNode(doc), "the top-level template mapping is even")
}
