// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package inventory

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddMondooLabels(t *testing.T) {
	asset := &Asset{
		Labels: map[string]string{
			"foo": "bar",
		},
	}

	rootAsset := &Asset{
		Labels: map[string]string{
			"k8s.mondoo.com/test": "val",
			"mondoo.com/sample":   "example",
			"random":              "random-val",
		},
	}

	asset.AddMondooLabels(rootAsset)
	assert.Equal(
		t,
		asset.Labels,
		map[string]string{
			"foo":                 "bar",
			"k8s.mondoo.com/test": "val",
			"mondoo.com/sample":   "example",
		})
}
