// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package logger

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func TestUseJSONLogging_IncludesCaller(t *testing.T) {
	saved := log.Logger
	t.Cleanup(func() { log.Logger = saved })

	var buf bytes.Buffer
	UseJSONLogging(&buf)
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Info().Msg("hello")

	if !strings.Contains(buf.String(), `"caller"`) {
		t.Errorf("expected a caller field in the json log, got %q", buf.String())
	}
}

func TestUseGCPJSONLogging_IncludesCaller(t *testing.T) {
	saved := log.Logger
	t.Cleanup(func() { log.Logger = saved })

	var buf bytes.Buffer
	UseGCPJSONLogging(&buf)
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Info().Msg("hello")

	if !strings.Contains(buf.String(), `"caller"`) {
		t.Errorf("expected a caller field in the gcp json log, got %q", buf.String())
	}
}
