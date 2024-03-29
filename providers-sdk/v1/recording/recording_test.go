// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package recording

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestLoadRecording(t *testing.T) {
	record, err := LoadRecordingFile("testdata/recording.json")
	require.NoError(t, err)
	assert.NotNil(t, record)
}
