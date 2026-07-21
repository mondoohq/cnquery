// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package powershell_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/mql/v13/providers/os/resources/powershell"
)

func TestPowershellEncoding(t *testing.T) {
	expected := "powershell.exe -NoProfile -EncodedCommand JABQAHIAbwBnAHIAZQBzAHMAUAByAGUAZgBlAHIAZQBuAGMAZQA9ACcAUwBpAGwAZQBuAHQAbAB5AEMAbwBuAHQAaQBuAHUAZQAnADsAZABpAHIAIAAiAGMAOgBcAHAAcgBvAGcAcgBhAG0AIABmAGkAbABlAHMAIgAgAA=="
	cmd := string("dir \"c:\\program files\" ")
	assert.Equal(t, expected, powershell.Encode(cmd))
}

func TestSplitInvocation(t *testing.T) {
	tests := []struct {
		name     string
		cmd      string
		wantArgv []string
		wantOK   bool
	}{
		{
			name:     "Encode output round-trips to direct argv",
			cmd:      powershell.Encode("Get-CimInstance -ClassName Win32_Bios"),
			wantArgv: []string{"powershell.exe", "-NoProfile", "-EncodedCommand", powershell.Encode("Get-CimInstance -ClassName Win32_Bios")[len("powershell.exe -NoProfile -EncodedCommand "):]},
			wantOK:   true,
		},
		{
			name:     "EncodeUnix output round-trips to direct argv",
			cmd:      powershell.EncodeUnix("hostname"),
			wantArgv: []string{"pwsh", "-NoProfile", "-EncodedCommand", powershell.EncodeUnix("hostname")[len("pwsh -NoProfile -EncodedCommand "):]},
			wantOK:   true,
		},
		{
			name:     "Wrap output unwraps to a single -c script",
			cmd:      powershell.Wrap("Get-NetAdapter | ConvertTo-Json"),
			wantArgv: []string{"powershell", "-c", "Get-NetAdapter | ConvertTo-Json"},
			wantOK:   true,
		},
		{
			name:   "plain command is not a powershell invocation",
			cmd:    "hostname",
			wantOK: false,
		},
		{
			name:   "non-powershell binary is left alone",
			cmd:    "cmd /c echo hi",
			wantOK: false,
		},
		{
			name:   "empty encoded payload is rejected",
			cmd:    "powershell.exe -NoProfile -EncodedCommand ",
			wantOK: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			argv, ok := powershell.SplitInvocation(tc.cmd)
			assert.Equal(t, tc.wantOK, ok)
			if tc.wantOK {
				assert.Equal(t, tc.wantArgv, argv)
			}
		})
	}
}
