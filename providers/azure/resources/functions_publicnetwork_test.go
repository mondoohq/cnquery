// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	web "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appservice/armappservice/v6"
	"github.com/stretchr/testify/assert"
)

func TestFunctionAppPublicNetworkAccess(t *testing.T) {
	t.Run("nil properties returns empty", func(t *testing.T) {
		assert.Equal(t, "", functionAppPublicNetworkAccess(nil))
	})
	t.Run("nil field returns empty", func(t *testing.T) {
		assert.Equal(t, "", functionAppPublicNetworkAccess(&web.SiteProperties{}))
	})
	t.Run("disabled value is returned", func(t *testing.T) {
		v := "Disabled"
		assert.Equal(t, "Disabled", functionAppPublicNetworkAccess(&web.SiteProperties{PublicNetworkAccess: &v}))
	})
}
