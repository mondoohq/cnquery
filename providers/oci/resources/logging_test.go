// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertLogConfiguration(t *testing.T) {
	t.Run("nil config returns nil", func(t *testing.T) {
		result, err := convertLogConfiguration(nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("empty config returns empty map", func(t *testing.T) {
		cfg := &logging.Configuration{}
		result, err := convertLogConfiguration(cfg)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result)
	})

	t.Run("config with compartment ID", func(t *testing.T) {
		cfg := &logging.Configuration{
			CompartmentId: common.String("ocid1.compartment.oc1..example"),
		}
		result, err := convertLogConfiguration(cfg)
		require.NoError(t, err)
		assert.Equal(t, "ocid1.compartment.oc1..example", result["compartmentId"])
	})

	t.Run("config with source", func(t *testing.T) {
		cfg := &logging.Configuration{
			Source: &logging.OciService{
				Service:  common.String("flowlogs"),
				Resource: common.String("ocid1.subnet.oc1..example"),
				Category: common.String("all"),
			},
		}
		result, err := convertLogConfiguration(cfg)
		require.NoError(t, err)
		source, ok := result["source"].(map[string]interface{})
		require.True(t, ok, "source should be a map")
		assert.Equal(t, "flowlogs", source["service"])
		assert.Equal(t, "ocid1.subnet.oc1..example", source["resource"])
		assert.Equal(t, "all", source["category"])
	})

	t.Run("config with all fields", func(t *testing.T) {
		cfg := &logging.Configuration{
			CompartmentId: common.String("ocid1.compartment.oc1..example"),
			Source: &logging.OciService{
				Service:  common.String("objectstorage"),
				Resource: common.String("ocid1.bucket.oc1..example"),
				Category: common.String("write"),
			},
		}
		result, err := convertLogConfiguration(cfg)
		require.NoError(t, err)
		assert.Equal(t, "ocid1.compartment.oc1..example", result["compartmentId"])
		assert.NotNil(t, result["source"])
	})
}
