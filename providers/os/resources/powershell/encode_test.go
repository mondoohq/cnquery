// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package powershell_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/v10/providers/os/resources/powershell"
)

func TestPowershellEncoding(t *testing.T) {
	expected := "powershell.exe -NoProfile -EncodedCommand JABQAHIAbwBnAHIAZQBzAHMAUAByAGUAZgBlAHIAZQBuAGMAZQA9ACcAUwBpAGwAZQBuAHQAbAB5AEMAbwBuAHQAaQBuAHUAZQAnADsAZABpAHIAIAAiAGMAOgBcAHAAcgBvAGcAcgBhAG0AIABmAGkAbABlAHMAIgAgAA=="
	cmd := string("dir \"c:\\program files\" ")
	assert.Equal(t, expected, powershell.Encode(cmd))
}
