// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package runtime

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/stream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers/os/connection/shared"
)

func TestRuntimeImageLazyFlattenBudgetCleansAndReleasesSlot(t *testing.T) {
	tmpRoot := t.TempDir()
	fixtureDir := t.TempDir()
	t.Setenv("MONDOO_TMP_DIR", tmpRoot)

	// The compressed OCI layer is small, but its flattened filesystem tar is
	// deliberately larger than the remaining budget.
	img, err := mutate.AppendLayers(empty.Image, stream.NewLayer(io.NopCloser(strings.NewReader(strings.Repeat("x", 1<<20)))))
	require.NoError(t, err)
	layers, err := img.Layers()
	require.NoError(t, err)
	_, err = layers[0].Digest()
	require.NoError(t, err)
	fixture := filepath.Join(fixtureDir, "image.tar")
	imageDigest := writeTestOCILayoutTarFromImage(t, fixture, img)
	fixtureInfo, err := os.Stat(fixture)
	require.NoError(t, err)
	maxBytes := fixtureInfo.Size() + 4096

	oldExporter := exportRuntimeImage
	exportRuntimeImage = func(_ context.Context, _, _, path, _, _ string) error {
		in, err := os.Open(fixture)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.Create(path)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, in)
		return err
	}
	t.Cleanup(func() { exportRuntimeImage = oldExporter })

	config := func(id uint32) *inventory.Config {
		return &inventory.Config{
			Type: shared.Type_RuntimeImage.String(),
			Host: "registry.example.com/team/app:1.2.3",
			Options: map[string]string{
				OPTION_RUNTIME_IMAGE_KIND:       "containerd",
				OPTION_RUNTIME_IMAGE_DIGEST:     imageDigest,
				OPTION_RUNTIME_IMAGE_ENDPOINT:   "unix:///run/containerd.sock",
				OPTION_RUNTIME_IMAGE_ALLOW_PULL: "false",
				OPTION_RUNTIME_IMAGE_MAX_BYTES:  strconv.FormatInt(maxBytes, 10),
				OPTION_RUNTIME_IMAGE_MAX_IMAGES: "1",
				"disable-cache":                 "true",
			},
		}
	}

	conn, err := NewRuntimeImage(1, config(1), &inventory.Asset{})
	require.NoError(t, err)
	require.ErrorIs(t, conn.Fetch(), errRuntimeImageTooLarge)
	entries, err := os.ReadDir(tmpRoot)
	require.NoError(t, err)
	assert.Empty(t, entries, "lazy extraction failure must remove every runtime-image temporary")

	// The failed connection was never explicitly closed. Reacquisition proves
	// the lazy failure released the max-concurrent-images slot exactly once.
	second, err := NewRuntimeImage(2, config(2), &inventory.Asset{})
	require.NoError(t, err)
	second.Close()
}
