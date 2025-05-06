// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package benchmark

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11"
	"go.mondoo.com/cnquery/v11/explorer"
	"go.mondoo.com/cnquery/v11/explorer/scan"
	"go.mondoo.com/cnquery/v11/mqlc"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/testutils"
)

func init() {
	log.Logger = log.Logger.Level(zerolog.Disabled)
	zerolog.SetGlobalLevel(zerolog.Disabled)
}

func BenchmarkScan_SingleAsset(b *testing.B) {
	ctx := context.Background()
	runtime := testutils.Local()
	conf := mqlc.NewConfig(runtime.Schema(), cnquery.DefaultFeatures)
	job := &scan.Job{
		Inventory: &inventory.Inventory{
			Spec: &inventory.InventorySpec{
				Assets: []*inventory.Asset{
					{
						Connections: []*inventory.Config{
							{
								Type: "k8s",
								Options: map[string]string{
									"path": "./testdata/1pod.yaml",
								},
								Discover: &inventory.Discovery{
									Targets: []string{"pods"},
								},
							},
						},
					},
				},
			},
		},
	}

	bundle, err := explorer.BundleFromPaths("./testdata/mondoo-kubernetes-inventory.mql.yaml")
	require.NoError(b, err)

	_, err = bundle.CompileExt(context.Background(), explorer.BundleCompileConf{
		CompilerConfig: conf,
		RemoveFailing:  true,
	})
	require.NoError(b, err)

	job.Bundle = bundle

	scanner := scan.NewLocalScanner(scan.DisableProgressBar())
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		res, err := scanner.RunIncognito(ctx, job)
		require.NoError(b, err)
		require.NotNil(b, res)
	}
}

func BenchmarkScan_MultipleAssets(b *testing.B) {
	ctx := context.Background()
	runtime := testutils.Local()
	conf := mqlc.NewConfig(runtime.Schema(), cnquery.DefaultFeatures)
	job := &scan.Job{
		Inventory: &inventory.Inventory{
			Spec: &inventory.InventorySpec{
				Assets: []*inventory.Asset{
					{
						Connections: []*inventory.Config{
							{
								Type: "k8s",
								Options: map[string]string{
									"path": "./testdata/2pods.yaml",
								},
								Discover: &inventory.Discovery{
									Targets: []string{"pods"},
								},
							},
						},
					},
				},
			},
		},
	}

	bundle, err := explorer.BundleFromPaths("./testdata/mondoo-kubernetes-inventory.mql.yaml")
	require.NoError(b, err)

	_, err = bundle.CompileExt(context.Background(), explorer.BundleCompileConf{
		CompilerConfig: conf,
		RemoveFailing:  true,
	})
	require.NoError(b, err)

	job.Bundle = bundle

	scanner := scan.NewLocalScanner(scan.DisableProgressBar())
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		res, err := scanner.RunIncognito(ctx, job)
		require.NoError(b, err)
		require.NotNil(b, res)
	}
}
