// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	tea "github.com/alibabacloud-go/tea/tea"
	"github.com/stretchr/testify/assert"
)

func TestCloudssoStatusEnabled(t *testing.T) {
	assert.True(t, cloudssoStatusEnabled(tea.String("Enabled")))
	assert.True(t, cloudssoStatusEnabled(tea.String("enabled")))
	assert.True(t, cloudssoStatusEnabled(tea.String(" Enabled ")))
	assert.False(t, cloudssoStatusEnabled(tea.String("Disabled")))
	// a status nobody could read must not report a control that may not be in
	// place
	assert.False(t, cloudssoStatusEnabled(nil))
	assert.False(t, cloudssoStatusEnabled(tea.String("")))
}

func TestCloudssoTaskSucceeded(t *testing.T) {
	assert.True(t, cloudssoTaskSucceeded(tea.String("Success")))
	assert.True(t, cloudssoTaskSucceeded(tea.String("success")))
	// a failed revocation leaves the access it was withdrawing still granted
	assert.False(t, cloudssoTaskSucceeded(tea.String("Failed")))
	// a task still running has not withdrawn anything yet either
	assert.False(t, cloudssoTaskSucceeded(tea.String("InProgress")))
	assert.False(t, cloudssoTaskSucceeded(nil))
}
