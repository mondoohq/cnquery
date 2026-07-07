// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSSHPublicKey(t *testing.T) {
	tests := []struct {
		name          string
		publicKey     string
		wantAlgorithm string
		wantBits      int64
	}{
		{
			name:          "ed25519",
			publicKey:     "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIATxrDDkLHMi0EMdsCES8icnsrbj+2ra3lsm2cjefUA7 test",
			wantAlgorithm: "ssh-ed25519",
			wantBits:      256,
		},
		{
			name:          "rsa 2048",
			publicKey:     "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCzWUJ5o5nE8AvOzLTV45xJ7b7U3vcH3idgxSQRGumbgXR0JY2O36R6Da6reu+sy1Nio3QjHNX/0prUscJli/N1F8g512wDFwhbFtIUW5J3J4Np2PuGJftxjD119w4uanp47E7kj76IX1TuS86RyZXAAlOJ3BWIrDR/TCoCEdYfCl8yydaIv8Ook5ip1qjZi7JP/RtXagiFOTtvbOmUBeTFz2hDalk6un+l0V4sOE3taULMfaFRwhBi+XeNlnfSmsl1+Bp/T3qJMxLsgRV1AEhwg6hestMGN49TB/YIogiMhAcM8ACF47HIxDMeoNxZJmuRg2tTkDLh4DAG02czba0l test",
			wantAlgorithm: "ssh-rsa",
			wantBits:      2048,
		},
		{
			name:          "ecdsa nistp256",
			publicKey:     "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHoiyNRU8Tk5rBs6ve4wExUzriltXZ/bl8YiuPfd7nF1kC894F306jKgJE0/0JIWSIcMWKeu1o6+e6fa64TwYig= test",
			wantAlgorithm: "ecdsa-sha2-nistp256",
			wantBits:      256,
		},
		{
			name:          "unparseable",
			publicKey:     "not a valid key",
			wantAlgorithm: "",
			wantBits:      0,
		},
		{
			name:          "empty",
			publicKey:     "",
			wantAlgorithm: "",
			wantBits:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			algorithm, bits := parseSSHPublicKey(tt.publicKey)
			assert.Equal(t, tt.wantAlgorithm, algorithm)
			assert.Equal(t, tt.wantBits, bits)
		})
	}
}
