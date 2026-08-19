// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// epochMillisCutoff separates a timestamp counted in seconds from one counted
// in milliseconds. Anything at or above it cannot be a plausible second count
// (it lands beyond the year 5138), so it is read as milliseconds instead.
const epochMillisCutoff int64 = 1e11

// timeLayouts are the string shapes New Relic serializes a timestamp in. The
// GraphQL DateTime scalar is documented as ISO 8601, which several concrete
// layouts satisfy, and different NerdGraph subgraphs pick different ones.
var timeLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02T15:04:05.000Z0700",
	"2006-01-02T15:04:05Z0700",
	"2006-01-02T15:04:05",
	"2006-01-02",
}

// nrTime decodes a New Relic timestamp, which arrives either as an ISO 8601
// string (the DateTime scalar) or as a number of seconds or milliseconds since
// the epoch (the EpochSeconds and EpochMilliseconds scalars).
//
// An absent, null or empty value decodes to no time at all rather than to the
// zero time. That distinction is the whole point of the type: a zero time.Time
// would surface as 1 January year 1 and a zero epoch as 1 January 1970, and
// both read as a real date that an age or rotation check would then compare
// against.
//
// A value that is present but in none of the recognized shapes is an error, not
// a silent null. A serialization change on New Relic's side has to surface as a
// failure, because reporting "no timestamp" for every record would make every
// key look newly created.
type nrTime struct {
	value *time.Time
}

// Time returns the decoded timestamp, or nil when the field carried no value.
func (n nrTime) Time() *time.Time { return n.value }

// IsZero reports whether the field carried no value.
func (n nrTime) IsZero() bool { return n.value == nil }

func (n *nrTime) UnmarshalJSON(b []byte) error {
	raw := strings.TrimSpace(string(b))
	if raw == "" || raw == "null" {
		n.value = nil
		return nil
	}

	if raw[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		return n.parseString(s)
	}

	var num json.Number
	if err := json.Unmarshal(b, &num); err != nil {
		return fmt.Errorf("could not decode the New Relic timestamp %s: %w", raw, err)
	}
	return n.parseEpoch(num.String())
}

func (n *nrTime) parseString(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		n.value = nil
		return nil
	}

	for _, layout := range timeLayouts {
		if parsed, err := time.Parse(layout, s); err == nil {
			n.value = &parsed
			return nil
		}
	}

	// Some subgraphs return an epoch count inside a string. Try that before
	// giving up, but only for something that is entirely digits, so a malformed
	// date is not coerced into a plausible-looking timestamp.
	if isAllDigits(s) {
		return n.parseEpoch(s)
	}

	return fmt.Errorf("could not decode the New Relic timestamp %q", s)
}

func (n *nrTime) parseEpoch(s string) error {
	epoch, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fmt.Errorf("could not decode the New Relic timestamp %q: %w", s, err)
	}
	if epoch == 0 {
		// A zero epoch is how the API reports "no timestamp" for a numeric
		// field. Reporting it as 1970 would make an original account key look
		// like the oldest credential in the estate.
		n.value = nil
		return nil
	}

	var parsed time.Time
	if epoch >= epochMillisCutoff || epoch <= -epochMillisCutoff {
		parsed = time.UnixMilli(epoch).UTC()
	} else {
		parsed = time.Unix(epoch, 0).UTC()
	}
	n.value = &parsed
	return nil
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 && (r == '-' || r == '+') {
			continue
		}
		if r < '0' || r > '9' {
			return false
		}
	}
	// A bare sign is not a number.
	return strings.IndexFunc(s, func(r rune) bool { return r >= '0' && r <= '9' }) >= 0
}
