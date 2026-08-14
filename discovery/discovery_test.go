// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package discovery

import (
	"testing"

	"github.com/stretchr/testify/assert"
	inventory "go.mondoo.com/mql/providers-sdk/v1/inventory"
)

// TestRestoreRequestedName covers the name precedence between the caller and the
// provider. Connecting replaces the root asset with the provider's connection asset,
// and providers name that asset in their detect step — several of them unconditionally.
// A name the caller asked for (cnspec's --asset-name, or a `name:` in an inventory
// file) has to survive that.
func TestRestoreRequestedName(t *testing.T) {
	t.Run("restores a name the provider overwrote", func(t *testing.T) {
		// what a `cnspec scan terraform plan tf.json --asset-name ...` looks like
		// after the terraform provider's detect step has run
		asset := &inventory.Asset{Name: "Terraform Plan tf"}

		restoreRequestedName(asset, "tf-plan-team-sandbox")

		assert.Equal(t, "tf-plan-team-sandbox", asset.Name)
	})

	t.Run("keeps the detected name when none was requested", func(t *testing.T) {
		asset := &inventory.Asset{Name: "Terraform Plan tf"}

		restoreRequestedName(asset, "")

		assert.Equal(t, "Terraform Plan tf", asset.Name)
	})

	t.Run("names a provider left empty", func(t *testing.T) {
		asset := &inventory.Asset{}

		restoreRequestedName(asset, "tf-plan-team-sandbox")

		assert.Equal(t, "tf-plan-team-sandbox", asset.Name)
	})

	t.Run("is a no-op when the provider agrees with the request", func(t *testing.T) {
		asset := &inventory.Asset{Name: "tf-plan-team-sandbox"}

		restoreRequestedName(asset, "tf-plan-team-sandbox")

		assert.Equal(t, "tf-plan-team-sandbox", asset.Name)
	})

	t.Run("tolerates a nil asset", func(t *testing.T) {
		assert.NotPanics(t, func() {
			restoreRequestedName(nil, "tf-plan-team-sandbox")
		})
	})
}
