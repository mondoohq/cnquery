// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/mqlr/lrcore"
)

var defaultLrDocsEntry = &lrcore.LrDocsEntry{
	Fields:           map[string]*lrcore.LrDocsField{},
	MinMondooVersion: "9.1.0",
}

func TestPlatformMapping(t *testing.T) {
	res := ensureDefaults("terraform.plan.configuration", defaultLrDocsEntry, "9.1.0")
	assert.Equal(t, "terraform-plan", res.Platform.Name[0])

	res = ensureDefaults("terraform.plan.proposedChange", defaultLrDocsEntry, "9.1.0")
	assert.Equal(t, "terraform-plan", res.Platform.Name[0])

	res = ensureDefaults("terraform.state.module", defaultLrDocsEntry, "9.1.0")
	assert.Equal(t, "terraform-state", res.Platform.Name[0])

	res = ensureDefaults("terraform.block", defaultLrDocsEntry, "9.1.0")
	assert.Equal(t, "terraform-hcl", res.Platform.Name[0])
}
