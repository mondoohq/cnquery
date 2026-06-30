// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package loggerconf

import (
	"os"
	"path/filepath"
	"testing"

	"go.mondoo.com/mql/v13/logger"
)

func clearEnvLevel(t *testing.T) {
	t.Helper()
	// ensure DEBUG/TRACE don't override the configured level during the test
	t.Setenv("DEBUG", "")
	t.Setenv("TRACE", "")
}

func TestConfigure_Level(t *testing.T) {
	clearEnvLevel(t)
	if err := Configure(&Options{Writer: "cli", Level: "warn"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := logger.GetLevel(); got != "warn" {
		t.Errorf("expected level warn, got %q", got)
	}
}

func TestConfigure_EmptyWriterDefaultsToCli(t *testing.T) {
	clearEnvLevel(t)
	if err := Configure(&Options{Level: "error"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := logger.GetLevel(); got != "error" {
		t.Errorf("expected level error, got %q", got)
	}
}

func TestConfigure_Nil(t *testing.T) {
	clearEnvLevel(t)
	if err := Configure(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := logger.GetLevel(); got != "info" {
		t.Errorf("expected level info, got %q", got)
	}
}

func TestConfigure_EnvOverridesLevel(t *testing.T) {
	t.Setenv("DEBUG", "true")
	t.Setenv("TRACE", "")
	if err := Configure(&Options{Writer: "cli", Level: "error"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := logger.GetLevel(); got != "debug" {
		t.Errorf("expected env to force debug, got %q", got)
	}
}

func TestConfigure_UnknownWriter(t *testing.T) {
	clearEnvLevel(t)
	if err := Configure(&Options{Writer: "bogus"}); err == nil {
		t.Fatal("expected an error for an unknown writer")
	}
}

func TestConfigure_StackdriverRequiresOptions(t *testing.T) {
	clearEnvLevel(t)
	if err := Configure(&Options{Writer: "stackdriver"}); err == nil {
		t.Fatal("expected an error when project-id is missing")
	}
	if err := Configure(&Options{Writer: "stackdriver", Options: map[string]string{"project-id": "p"}}); err == nil {
		t.Fatal("expected an error when log-id is missing")
	}
}

func TestLoad(t *testing.T) {
	dir := t.TempDir()

	yamlPath := filepath.Join(dir, "logging.yaml")
	if err := os.WriteFile(yamlPath, []byte("writer: cli\nlevel: debug\noptions:\n  format: json\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	opts, err := Load(yamlPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Writer != "cli" || opts.Level != "debug" || opts.Options["format"] != "json" {
		t.Errorf("unexpected options parsed from yaml: %+v", opts)
	}

	jsonPath := filepath.Join(dir, "logging.json")
	if err := os.WriteFile(jsonPath, []byte(`{"writer":"cli","level":"trace"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	opts, err = Load(jsonPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Writer != "cli" || opts.Level != "trace" {
		t.Errorf("unexpected options parsed from json: %+v", opts)
	}

	if _, err := Load(filepath.Join(dir, "does-not-exist.yaml")); err == nil {
		t.Fatal("expected an error for a missing file")
	}
}
