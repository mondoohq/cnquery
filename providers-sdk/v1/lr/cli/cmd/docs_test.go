// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/lr/docs"
	"testing"
)

var defaultLrDocsEntry = &docs.LrDocsEntry{
	Fields:           map[string]*docs.LrDocsField{},
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
