// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package logger

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/hokaccha/go-prettyjson"
	"github.com/rs/zerolog/log"
	"sigs.k8s.io/yaml"
)

var DumpLocal string

// DebugJSON prints a prettified JSON of the data to CLI on debug mode
func DebugJSON(obj interface{}) {
	if !log.Debug().Enabled() {
		return
	}

	fmt.Fprintln(LogOutputWriter, PrettyJSON(obj))
}

func TraceJSON(obj interface{}) {
	if !log.Trace().Enabled() {
		return
	}

	fmt.Fprintln(LogOutputWriter, PrettyJSON(obj))
}

// PrettyJSON turns any object into its prettified json representation
func PrettyJSON(obj interface{}) string {
	s, _ := prettyjson.Marshal(obj)
	return string(s)
}

// DebugDumpJSON will write a JSON dump if the Debug or Trace mode is active and
// the DumpLocal prefix is defined.
func DebugDumpJSON(name string, obj interface{}) {
	if !log.Debug().Enabled() {
		return
	}

	if DumpLocal == "" {
		if val, ok := os.LookupEnv("DEBUG"); ok && (val == "1" || val == "true") {
			DumpLocal = "./mondoo-debug-"
		} else if val, ok := os.LookupEnv("TRACE"); ok && (val == "1" || val == "true") {
			DumpLocal = "./mondoo-debug-"
		} else {
			return
		}
	}

	raw, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		log.Error().Err(err).Msg("failed to dump JSON")
	}

	err = os.WriteFile(DumpLocal+name+".json", []byte(raw), 0o644)
	if err != nil {
		log.Error().Err(err).Msg("failed to dump JSON")
	}
}

// DebugDumpYAML will write a YAML dump if the Debug or Trace mode is active and
// the DumpLocal prefix is defined.
func DebugDumpYAML(name string, obj interface{}) {
	if !log.Debug().Enabled() {
		return
	}

	if DumpLocal == "" {
		if val, ok := os.LookupEnv("DEBUG"); ok && (val == "1" || val == "true") {
			DumpLocal = "./mondoo-debug-"
		} else if val, ok := os.LookupEnv("TRACE"); ok && (val == "1" || val == "true") {
			DumpLocal = "./mondoo-debug-"
		} else {
			return
		}
	}

	raw, err := yaml.Marshal(obj)
	if err != nil {
		log.Error().Err(err).Msg("failed to dump YAML")
	}

	err = os.WriteFile(DumpLocal+name+".yaml", []byte(raw), 0o644)
	if err != nil {
		log.Error().Err(err).Msg("failed to dump JSON")
	}
}
