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

func TestAddAnnotations(t *testing.T) {
	t.Run("AddAnnotations", func(t *testing.T) {
		asset := &Asset{
			Labels: map[string]string{
				"foo": "bar",
			},
		}
		asset.AddAnnotations(map[string]string{})
		assert.Equal(t, nil, asset.Annotations)
	})

	t.Run("test nil", func(t *testing.T) {
		asset := &Asset{
			Labels: map[string]string{
				"foo": "bar",
			},
		}
		asset.AddAnnotations(nil)
		assert.Equal(t, nil, asset.Annotations)
	})

	t.Run("test merge", func(t *testing.T) {
		asset := &Asset{
			Annotations: map[string]string{
				"foo": "bar",
			},
		}
		asset.AddAnnotations(map[string]string{
			"fruit": "banana",
		})
		assert.Equal(t, map[string]string{
			"foo":   "bar",
			"fruit": "banana",
		}, asset.Annotations)
	})

	t.Run("test overwrite", func(t *testing.T) {
		asset := &Asset{
			Annotations: map[string]string{
				"foo": "bar",
			},
		}
		asset.AddAnnotations(map[string]string{
			"foo": "not-bar",
		})
		assert.Equal(t, map[string]string{
			"foo": "not-bar",
		}, asset.Annotations)
	})
}
