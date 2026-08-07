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
