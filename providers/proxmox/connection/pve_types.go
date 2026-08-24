// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"fmt"
	"strconv"
	"strings"
)

// PveBool decodes a Proxmox boolean flag.
//
// Proxmox is written in Perl and has no native JSON boolean, so most flags
// come back as the integers 1 and 0. Its published API schema nonetheless
// declares them as `boolean`, and some endpoints and releases do emit real
// JSON `true`/`false`; a handful of config-derived values arrive quoted.
//
// Neither plain Go type survives that spread. A `bool` field silently stays
// false against the integer form, because encoding/json will not coerce a
// number into a bool, and the zero value is indistinguishable from a real
// false. An `int` field fails outright on the boolean form, and because the
// error propagates out of apiGet it takes the whole list with it.
//
// This accepts every form Proxmox is known to produce, so a flag reports what
// the cluster actually set rather than the Go zero value.
type PveBool bool

// Bool returns the decoded value.
func (b PveBool) Bool() bool { return bool(b) }

func (b *PveBool) UnmarshalJSON(data []byte) error {
	s := strings.Trim(strings.TrimSpace(string(data)), `"`)
	switch s {
	case "true":
		*b = true
		return nil
	case "false", "null", "":
		*b = false
		return nil
	}
	// Numeric form: Proxmox uses 1 and 0, but treat any non-zero as set
	// rather than rejecting a value the cluster clearly considers true.
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		*b = f != 0
		return nil
	}
	return fmt.Errorf("cannot decode %q as a proxmox boolean", s)
}

// ParsePropertyString splits a Proxmox property string into its key/value
// pairs.
//
// Many of the most security-relevant guest settings are not discrete JSON
// fields. Whether a VM carries pre-enrolled Secure Boot keys, which CPU
// mitigation flags reach the guest, and which SEV features are switched on all
// live inside a single comma-delimited config line such as
// `local-lvm:vm-100-disk-1,efitype=4m,pre-enrolled-keys=1`.
//
// A leading element with no `=` is the format's positional value and is stored
// under defaultKey, which is how PVE serializes the volume of a disk, the CPU
// model of `cpu`, the SEV type of `amd-sev`, and the device of `rng0`. An
// explicit `defaultKey=` later in the string wins over the positional form,
// matching how PVE itself parses the line. Elements that are neither are
// skipped rather than stored under an empty key.
//
// Values may themselves contain semicolons (`flags=+spec-ctrl;-ssbd`), which
// survive because only commas separate elements.
func ParsePropertyString(s, defaultKey string) map[string]string {
	out := map[string]string{}
	s = strings.TrimSpace(s)
	if s == "" {
		return out
	}
	for i, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, value, found := strings.Cut(part, "=")
		if !found {
			// Only the first element may be positional; a later bare token is
			// not something PVE emits, so ignore it instead of guessing.
			if i == 0 && defaultKey != "" {
				out[defaultKey] = part
			}
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		out[key] = strings.TrimSpace(value)
	}
	return out
}

// PropBool reads a boolean out of a parsed property string, reporting absence
// as nil.
//
// The distinction matters: a setting the config never mentions and a setting
// explicitly turned off are different facts, and collapsing both into false
// would report "SEV debugging is permitted" about a VM whose config was never
// read. Callers that know PVE publishes a default for the key apply it
// themselves rather than having one invented here.
func PropBool(props map[string]string, key string) *bool {
	raw, ok := props[key]
	if !ok {
		return nil
	}
	return decodePveBoolPtr(raw)
}

// ConfigBool reads a boolean out of a raw guest config map, reporting absence
// as nil.
//
// The map comes straight from encoding/json, so the same flag arrives as a
// float64 1, a string "1", or a real JSON true depending on the endpoint and
// the Proxmox release.
func ConfigBool(cfg map[string]any, key string) *bool {
	if cfg == nil {
		return nil
	}
	v, ok := cfg[key]
	if !ok || v == nil {
		return nil
	}
	switch val := v.(type) {
	case bool:
		return &val
	case float64:
		b := val != 0
		return &b
	case string:
		return decodePveBoolPtr(val)
	}
	return nil
}

func decodePveBoolPtr(raw string) *bool {
	var b PveBool
	if err := b.UnmarshalJSON([]byte(raw)); err != nil {
		return nil
	}
	out := bool(b)
	return &out
}
