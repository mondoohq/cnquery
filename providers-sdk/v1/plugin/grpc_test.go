// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRecoverPanic(t *testing.T) {
	t.Run("recovers string panic", func(t *testing.T) {
		var err error
		func() {
			defer recoverPanic("TestMethod", &err)
			panic("something went wrong")
		}()

		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
		assert.Equal(t, "panic in provider TestMethod: something went wrong", st.Message())
		assert.NotContains(t, st.Message(), "goroutine") // stack trace stays in log, not on the wire
	})

	t.Run("recovers nil pointer panic", func(t *testing.T) {
		var err error
		func() {
			defer recoverPanic("GetData", &err)
			var s *string
			_ = *s // nil pointer dereference
		}()

		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
		assert.Contains(t, st.Message(), "panic in provider GetData")
	})

	t.Run("no panic leaves error nil", func(t *testing.T) {
		var err error
		func() {
			defer recoverPanic("TestMethod", &err)
			// no panic
		}()

		assert.NoError(t, err)
	})
}
