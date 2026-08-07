// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackupVaultNameFromArn(t *testing.T) {
	tests := []struct {
		name       string
		arn        string
		wantName   string
		wantRegion string
		wantErr    bool
	}{
		{
			name:       "standard vault arn",
			arn:        "arn:aws:backup:us-east-1:123456789012:backup-vault:Default",
			wantName:   "Default",
			wantRegion: "us-east-1",
		},
		{
			name:       "vault name containing dashes",
			arn:        "arn:aws:backup:eu-central-1:123456789012:backup-vault:prod-daily-vault",
			wantName:   "prod-daily-vault",
			wantRegion: "eu-central-1",
		},
		{
			name:    "not an arn",
			arn:     "Default",
			wantErr: true,
		},
		{
			name:    "empty",
			arn:     "",
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			name, region, err := backupVaultNameFromArn(test.arn)
			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.wantName, name)
			assert.Equal(t, test.wantRegion, region)
		})
	}
}
