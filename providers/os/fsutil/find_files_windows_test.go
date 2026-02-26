// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build windows

package fsutil

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleFsError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantSkip bool
		wantErr  error
	}{
		{
			name:     "nil error",
			err:      nil,
			wantSkip: false,
			wantErr:  nil,
		},
		{
			name:     "permission denied is skipped",
			err:      os.ErrPermission,
			wantSkip: true,
			wantErr:  nil,
		},
		{
			name:     "not exist is skipped",
			err:      os.ErrNotExist,
			wantSkip: true,
			wantErr:  nil,
		},
		{
			name:     "invalid is skipped",
			err:      os.ErrInvalid,
			wantSkip: true,
			wantErr:  nil,
		},
		{
			name:     "other error is propagated",
			err:      errors.New("some other error"),
			wantSkip: true,
			wantErr:  errors.New("some other error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skip, err := handleFsError(tt.err)
			assert.Equal(t, tt.wantSkip, skip)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.Equal(t, tt.wantErr.Error(), err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
