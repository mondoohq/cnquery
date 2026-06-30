// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package logger

import (
	"bytes"
	"encoding/json"
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

func TestUseGCPJSONLogging_WithLabels(t *testing.T) {
	saved := log.Logger
	t.Cleanup(func() { log.Logger = saved })

	var buf bytes.Buffer
	UseGCPJSONLogging(&buf, WithGCPLabels(map[string]string{"project_id": "scanned-proj", "scan_type": "gcp"}))
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Info().Msg("hello")

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("log line is not valid json: %v (%q)", err, buf.String())
	}
	// Cloud Logging lifts this special field into LogEntry.labels.
	labels, ok := entry["logging.googleapis.com/labels"].(map[string]any)
	if !ok {
		t.Fatalf("expected a logging.googleapis.com/labels object, got %v", entry["logging.googleapis.com/labels"])
	}
	if labels["project_id"] != "scanned-proj" || labels["scan_type"] != "gcp" {
		t.Errorf("unexpected labels: %v", labels)
	}
}

func TestUseGCPJSONLogging_NoLabelsByDefault(t *testing.T) {
	saved := log.Logger
	t.Cleanup(func() { log.Logger = saved })

	var buf bytes.Buffer
	UseGCPJSONLogging(&buf)
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Info().Msg("hello")

	if strings.Contains(buf.String(), "logging.googleapis.com/labels") {
		t.Errorf("did not expect a labels field without WithGCPLabels, got %q", buf.String())
	}
}
