// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Guards against the inverted nil-check the function used to carry —
// it now returns nil on a nil input instead of panicking on the
// dereference inside the constructor.
func TestNewAdminConsentRequestPolicy_NilInput(t *testing.T) {
	assert.Nil(t, newAdminConsentRequestPolicy(nil))
}
