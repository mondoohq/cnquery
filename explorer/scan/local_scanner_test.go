// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilterPreprocess(t *testing.T) {
	// given
	filters := []string{
		"namespace1/pack1",
		"namespace2/pack2",
		"//registry.mondoo.com/namespace/namespace3/querypacks/pack3",
	}

	// when
	preprocessed := preprocessQueryPackFilters(filters)

	// then
	assert.Equal(t, []string{
		"//registry.mondoo.com/namespace/namespace1/querypacks/pack1",
		"//registry.mondoo.com/namespace/namespace2/querypacks/pack2",
		"//registry.mondoo.com/namespace/namespace3/querypacks/pack3",
	}, preprocessed)
}
