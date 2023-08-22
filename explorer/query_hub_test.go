// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package explorer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueryPackMrn(t *testing.T) {
	// given
	namespace := "test-namespace"
	uid := "test-uid"

	// when
	mrn := NewQueryPackMrn(namespace, uid)

	// then
	assert.Equal(t, "//registry.mondoo.com/namespace/test-namespace/querypacks/test-uid", mrn)
}
