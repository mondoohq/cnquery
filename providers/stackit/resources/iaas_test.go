// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"testing"

	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
)

// boolPtr is the absent-vs-set distinction these tests are about: a *bool that
// is nil means the API reported nothing, which is not the same as false.
func boolPtr(b bool) *bool { return &b }

func assertBoolPtr(t *testing.T, name string, got, want *bool) {
	t.Helper()
	switch {
	case got == nil && want == nil:
	case got == nil:
		t.Fatalf("%s = null, want %v", name, *want)
	case want == nil:
		t.Fatalf("%s = %v, want null", name, *got)
	case *got != *want:
		t.Fatalf("%s = %v, want %v", name, *got, *want)
	}
}

func TestProtocolLabel(t *testing.T) {
	cases := []struct {
		name      string
		protoName string
		number    int64
		hasNumber bool
		want      string
	}{
		{"name only", "tcp", 0, false, "tcp"},
		{"name wins over number", "udp", 17, true, "udp"},
		{"number fallback (GRE)", "", 47, true, "47"},
		{"number zero is still a protocol", "", 0, true, "0"},
		{"neither set", "", 0, false, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := protocolLabel(tc.protoName, tc.number, tc.hasNumber); got != tc.want {
				t.Fatalf("protocolLabel(%q, %d, %v) = %q, want %q",
					tc.protoName, tc.number, tc.hasNumber, got, tc.want)
			}
		})
	}
}

func TestClassifyVolumeSource(t *testing.T) {
	const id = "11111111-2222-3333-4444-555555555555"
	cases := []struct {
		name                              string
		sourceType                        string
		wantImage, wantSnapshot, wantBkup string
	}{
		{"image source", "image", id, "", ""},
		{"snapshot source", "snapshot", "", id, ""},
		{"backup source stays a backup, not a snapshot", "backup", "", "", id},
		{"volume clone maps to nothing", "volume", "", "", ""},
		{"unknown type maps to nothing", "something-new", "", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotImage, gotSnapshot, gotBackup := classifyVolumeSource(tc.sourceType, id)
			if gotImage != tc.wantImage || gotSnapshot != tc.wantSnapshot || gotBackup != tc.wantBkup {
				t.Fatalf("classifyVolumeSource(%q, id) = (%q, %q, %q), want (%q, %q, %q)",
					tc.sourceType, gotImage, gotSnapshot, gotBackup,
					tc.wantImage, tc.wantSnapshot, tc.wantBkup)
			}
		})
	}
}

// --- server posture fields that must stay tri-state -------------------------

// decodeServer builds an iaas.Server from a payload shaped like the one the
// GetServer endpoint returns, so these tests exercise the same JSON path the
// provider takes rather than a hand-built struct that could not catch a field
// read from the wrong place.
func decodeServer(t *testing.T, payload string) *iaas.Server {
	t.Helper()
	var s iaas.Server
	if err := json.Unmarshal([]byte(payload), &s); err != nil {
		t.Fatalf("decoding server payload: %v", err)
	}
	return &s
}

func TestServerAgentProvisioned(t *testing.T) {
	cases := []struct {
		name    string
		payload string
		want    *bool
	}{
		{
			name:    "agent absent means the image default decides, which we cannot read",
			payload: `{"id":"s1","name":"web","machineType":"g1.1"}`,
			want:    nil,
		},
		{
			name:    "agent object present but provisioning unset is still unreadable",
			payload: `{"id":"s1","name":"web","machineType":"g1.1","agent":{}}`,
			want:    nil,
		},
		{
			name:    "agent provisioned",
			payload: `{"id":"s1","name":"web","machineType":"g1.1","agent":{"provisioned":true}}`,
			want:    boolPtr(true),
		},
		{
			name:    "agent explicitly not provisioned",
			payload: `{"id":"s1","name":"web","machineType":"g1.1","agent":{"provisioned":false}}`,
			want:    boolPtr(false),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := serverAgentProvisioned(decodeServer(t, tc.payload))
			assertBoolPtr(t, "serverAgentProvisioned", got, tc.want)
		})
	}
}

func TestServerAgentProvisionedNilServer(t *testing.T) {
	if got := serverAgentProvisioned(nil); got != nil {
		t.Fatalf("serverAgentProvisioned(nil) = %v, want nil", *got)
	}
}

func TestServerBootVolumeDeleteOnTermination(t *testing.T) {
	cases := []struct {
		name    string
		payload string
		want    *bool
	}{
		{
			name:    "server booted from an image has no boot volume, so the question does not apply",
			payload: `{"id":"s1","name":"web","machineType":"g1.1","imageId":"img-1"}`,
			want:    nil,
		},
		{
			name:    "boot volume present but retention unreported",
			payload: `{"id":"s1","name":"web","machineType":"g1.1","bootVolume":{"id":"v1","size":32}}`,
			want:    nil,
		},
		{
			name:    "disk is destroyed with the server",
			payload: `{"id":"s1","name":"web","machineType":"g1.1","bootVolume":{"id":"v1","deleteOnTermination":true}}`,
			want:    boolPtr(true),
		},
		{
			name:    "disk survives the server",
			payload: `{"id":"s1","name":"web","machineType":"g1.1","bootVolume":{"id":"v1","deleteOnTermination":false}}`,
			want:    boolPtr(false),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := serverBootVolumeDeleteOnTermination(decodeServer(t, tc.payload))
			assertBoolPtr(t, "serverBootVolumeDeleteOnTermination", got, tc.want)
		})
	}
}

func TestServerBootVolumeDeleteOnTerminationNilServer(t *testing.T) {
	if got := serverBootVolumeDeleteOnTermination(nil); got != nil {
		t.Fatalf("serverBootVolumeDeleteOnTermination(nil) = %v, want nil", *got)
	}
}
