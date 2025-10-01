// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateMacieSessionArn(t *testing.T) {
	arn := generateMacieSessionArn("123456789012", "us-east-1")
	expected := "arn:aws:macie2:us-east-1:123456789012:session"
	require.Equal(t, expected, arn)
}

func TestGenerateClassificationJobArn(t *testing.T) {
	arn := generateClassificationJobArn("123456789012", "us-west-2", "job-123456")
	expected := "arn:aws:macie2:us-west-2:123456789012:classification-job/job-123456"
	require.Equal(t, expected, arn)
}

func TestGenerateFindingArn(t *testing.T) {
	arn := generateFindingArn("123456789012", "eu-west-1", "finding-123456")
	expected := "arn:aws:macie2:eu-west-1:123456789012:finding/finding-123456"
	require.Equal(t, expected, arn)
}

func TestMacieMqlResourceIds(t *testing.T) {
	// Test that ID methods return expected values
	// This tests the basic structure without requiring AWS API calls

	t.Run("test macie session id", func(t *testing.T) {
		session := &mqlAwsMacieSession{}
		session.Arn.Data = "arn:aws:macie2:us-east-1:123456789012:session"
		session.Arn.State = 1

		id, err := session.id()
		require.NoError(t, err)
		require.Equal(t, "arn:aws:macie2:us-east-1:123456789012:session", id)
	})

	t.Run("test classification job id", func(t *testing.T) {
		job := &mqlAwsMacieClassificationJob{}
		job.Arn.Data = "arn:aws:macie2:us-east-1:123456789012:classification-job/job-123"
		job.Arn.State = 1

		id, err := job.id()
		require.NoError(t, err)
		require.Equal(t, "arn:aws:macie2:us-east-1:123456789012:classification-job/job-123", id)
	})

	t.Run("test finding id", func(t *testing.T) {
		finding := &mqlAwsMacieFinding{}
		finding.Id.Data = "finding-12345"
		finding.Id.State = 1

		id, err := finding.id()
		require.NoError(t, err)
		require.Equal(t, "finding-12345", id)
	})

	t.Run("test custom data identifier id", func(t *testing.T) {
		identifier := &mqlAwsMacieCustomDataIdentifier{}
		identifier.Id.Data = "identifier-12345"
		identifier.Id.State = 1

		id, err := identifier.id()
		require.NoError(t, err)
		require.Equal(t, "identifier-12345", id)
	})
}
