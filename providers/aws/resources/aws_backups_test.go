// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/backup"
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

// TestBackupVaultArgsAreSettableFields pins every argument the vault init
// passes to a field the generated schema can actually set. SetAllData rejects
// an unknown key outright, so a stale argument left behind by a schema change
// does not degrade a field - it fails the whole resource, on every lookup.
func TestBackupVaultArgsAreSettableFields(t *testing.T) {
	args := backupVaultArgs(&backup.DescribeBackupVaultOutput{
		BackupVaultArn:   aws.String("arn:aws:backup:us-east-1:123456789012:backup-vault:Default"),
		BackupVaultName:  aws.String("Default"),
		EncryptionKeyArn: aws.String("arn:aws:kms:us-east-1:123456789012:key/abc"),
		Locked:           aws.Bool(true),
	}, "us-east-1")

	require.NotEmpty(t, args)
	for key := range args {
		_, ok := setDataFields["aws.backup.vault."+key]
		assert.True(t, ok, "aws.backup.vault has no settable field %q", key)
	}

	// The encryption key reaches the resource through the typed encryptionKey
	// accessor, never as a raw ARN argument.
	assert.NotContains(t, args, "encryptionKeyArn")
}
