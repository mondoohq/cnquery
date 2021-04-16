package stackdriver

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
