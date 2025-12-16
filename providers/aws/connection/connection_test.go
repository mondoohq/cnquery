// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/inventory"
)

func TestNewAwsConnection(t *testing.T) {
	conn, err := NewAwsConnection(123, &inventory.Asset{}, &inventory.Config{})
	require.Nil(t, err)
	require.NotNil(t, conn)
}

func TestGetRegionsFromRegionalTable(t *testing.T) {
	t.Run("Successful region extraction and deduplication", func(t *testing.T) {
		regions, err := getRegionsFromRegionalTable()
		require.NoError(t, err)
		fewExpectedRegions := []string{
			"ap-east-1",
			"ap-northeast-1",
			"ap-south-1",
			"ap-southeast-1",
			"ca-central-1",
			"ca-west-1",
			"eu-central-1",
			"eu-central-2",
			"eu-north-1",
			"eu-south-1",
			"eu-south-2",
			"eu-west-1",
			"eu-west-2",
			"eu-west-3",
			"il-central-1",
			"me-central-1",
			"me-south-1",
			"mx-central-1",
			"sa-east-1",
			"us-east-1",
			"us-east-2",
			"us-gov-east-1",
			"us-gov-west-1",
			"us-west-1",
			"us-west-2",
		}
		for _, expectedRegion := range fewExpectedRegions {
			require.Contains(t, regions, expectedRegion)
		}
	})
}
