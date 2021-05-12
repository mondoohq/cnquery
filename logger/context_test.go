package logger

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRequestIDLoggingContext(t *testing.T) {
	type testLogMsg struct {
		RequestID string `json:"req-id"`
	}
	t.Run("outputs the provided request id with the log message", func(t *testing.T) {
		testRequestID := "test-req-id"
		twriter := &strings.Builder{}
		ctx := RequestScopedContext(context.Background(), testRequestID)
		log := FromContext(ctx).Output(twriter)

		log.Debug().Msg("hello")
		msg := testLogMsg{}
		err := json.Unmarshal([]byte(twriter.String()), &msg)
		require.NoError(t, err)
		require.Equal(t, testRequestID, msg.RequestID)
	})

	t.Run("generates a request id if one is not provided", func(t *testing.T) {
		twriter := &strings.Builder{}
		ctx := RequestScopedContext(context.Background(), "")
		log := FromContext(ctx).Output(twriter)

		log.Debug().Msg("hello")
		msg := testLogMsg{}
		err := json.Unmarshal([]byte(twriter.String()), &msg)
		require.NoError(t, err)
		require.True(t, len(msg.RequestID) > 1)
		require.True(t, strings.HasPrefix(msg.RequestID, "_"))
	})
}
