// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
	"github.com/stretchr/testify/assert"
)

func TestCustomJobSpecFields(t *testing.T) {
	t.Run("nil spec yields zero values", func(t *testing.T) {
		sa, net, web := customJobSpecFields(nil)
		assert.Equal(t, "", sa)
		assert.Equal(t, "", net)
		assert.False(t, web)
	})
	t.Run("fields are surfaced", func(t *testing.T) {
		spec := &aiplatformpb.CustomJobSpec{
			ServiceAccount:  "sa@project.iam.gserviceaccount.com",
			Network:         "projects/123/global/networks/default",
			EnableWebAccess: true,
		}
		sa, net, web := customJobSpecFields(spec)
		assert.Equal(t, "sa@project.iam.gserviceaccount.com", sa)
		assert.Equal(t, "projects/123/global/networks/default", net)
		assert.True(t, web)
	})
}
