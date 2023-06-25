package providers

import (
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/cli/config"
	"go.mondoo.com/cnquery/providers/plugin"
)

var (
	SystemPath string
	HomePath   string
)

func init() {
	SystemPath = config.SystemDataPath("providers")
	if os.Geteuid() != 0 {
		HomePath, _ = config.HomePath("providers")
	}
}

type Provider struct {
	*plugin.Provider
	Path string
}

func List() {
	var providers []*Provider

	// This really shouldn't happen, but just in case it does...
	if SystemPath == "" && HomePath == "" {
		log.Error().Msg("can't find any paths for providers, none are configured")
		return
	}

	sysOk := config.ProbeDir(SystemPath)
	homeOk := config.ProbeDir(HomePath)
	if !sysOk && !homeOk {
		msg := log.Warn()
		if SystemPath != "" {
			msg = msg.Str("system-path", SystemPath)
		}
		if HomePath != "" {
			msg = msg.Str("home-path", HomePath)
		}
		msg.Msg("no provider paths exist")
		return
	}

	if sysOk {
		x, err := listProviders(SystemPath)
		if err != nil {
			log.Warn().Str("path", SystemPath).Msg("failed to get providers from system path")
		} else {
			providers = append(providers, x...)
		}
	}

	if homeOk {
		x, err := listProviders(HomePath)
		if err != nil {
			log.Warn().Str("path", HomePath).Msg("failed to get providers from home path")
		} else {
			providers = append(providers, x...)
		}
	}

	if len(providers) == 0 {
		return
	}
}

func listProviders(path string) ([]*Provider, error) {
	log.Debug().Str("path", path).Msg("searching providers in path")
	files, err := afero.ReadDir(config.AppFs, path)
	if err != nil {
		return nil, err
	}

	panic("STH")
	_ = files
	return nil, nil
}
