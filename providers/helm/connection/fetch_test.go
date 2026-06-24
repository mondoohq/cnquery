// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// httpGetBytes must reject a response that exceeds maxChartDownloadSize rather
// than silently truncating it (io.LimitReader returns EOF at the cap with no
// error). A body at or under the cap is returned intact.
func TestHttpGetBytesSizeLimit(t *testing.T) {
	orig := maxChartDownloadSize
	maxChartDownloadSize = 64
	defer func() { maxChartDownloadSize = orig }()
	capN := int(maxChartDownloadSize)

	t.Run("oversized response errors", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Write([]byte(strings.Repeat("A", capN+10)))
		}))
		defer srv.Close()

		_, err := httpGetBytes(srv.URL, "", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "maximum allowed size")
	})

	t.Run("body within the cap is returned intact", func(t *testing.T) {
		body := strings.Repeat("B", capN)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Write([]byte(body))
		}))
		defer srv.Close()

		data, err := httpGetBytes(srv.URL, "", "")
		require.NoError(t, err)
		assert.Equal(t, body, string(data))
	})
}
