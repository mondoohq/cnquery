// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package stackdriver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithLabels(t *testing.T) {
	t.Run("sets labels on options", func(t *testing.T) {
		labels := map[string]string{"env": "prod", "team": "security"}
		o := &options{}
		WithLabels(labels)(o)
		assert.Equal(t, labels, o.labels)
	})

	t.Run("no options leaves labels empty", func(t *testing.T) {
		o := &options{}
		assert.Empty(t, o.labels)
	})
}

//import (
//	"errors"
//	"github.com/rs/zerolog/log"
//	"github.com/stretchr/testify/require"
//	"testing"
//)
//
//func TestStackdriverLogging(t *testing.T) {
//	projectId := "mondoo-dev-12345"
//	gcpWriter, err := NewStackdriverWriter(projectId, "testing")
//	require.NoError(t, err)
//	log.Logger = log.Output(gcpWriter)
//
//	log.Info().Msg("info")
//	log.Warn().Msg("warn")
//	log.Error().Err(errors.New("something went wrong")).Msg("err")
//	log.Debug().Msg("debug")
//	log.Fatal().Msg("fatal")
//}
