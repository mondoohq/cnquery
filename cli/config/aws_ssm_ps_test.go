// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// aws-ssm-ps://region/us-east-2/parameter/MondooAgentConfig?decrypt=true&account=12345678

func TestNewSSMParameter(t *testing.T) {
	param, err := newSsmParameter("us-west-1", "test-name")
	require.NoError(t, err)
	assert.Equal(t, &SsmParameter{Region: "us-west-1", Parameter: "test-name"}, param)
	assert.Equal(t, "region/us-west-1/parameter/test-name", param.String())
}

func TestParseSSMParameterPath(t *testing.T) {
	ssmParam, err := parseSsmParameterPath("region/us-west-2/parameter/test-param-name")
	require.NoError(t, err)
	assert.Equal(t, &SsmParameter{Parameter: "test-param-name", Region: "us-west-2"}, ssmParam)
}

func TestNewSSMParameterPathReturnsErrWhenNoRegion(t *testing.T) {
	_, err := newSsmParameter("", "test-name")
	require.Error(t, err)
	assert.EqualError(t, err, "invalid parameter. region and parameter name required.")
}

func TestParseSSMParameterPathBadPathReturnsError(t *testing.T) {
	_, err := parseSsmParameterPath("region/us-west-2/parameter")
	require.Error(t, err)
	assert.EqualError(t, err, "invalid parameter path. expected region/<region-val>/parameter/<parameter-name>")
	_, err = parseSsmParameterPath("region//parameter/testname")
	require.Error(t, err)
	assert.EqualError(t, err, "invalid parameter path. expected region/<region-val>/parameter/<parameter-name>")
	_, err = parseSsmParameterPath("region/us-west-1/parameter/")
	require.Error(t, err)
	assert.EqualError(t, err, "invalid parameter path. expected region/<region-val>/parameter/<parameter-name>")
}
