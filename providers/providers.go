package providers

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/cli/config"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers-sdk/v1/resources"
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

type Providers map[string]*Provider

type Provider struct {
	*plugin.Provider
	Schema *resources.Schema
	Path   string
}

// List providers that are going to be used in their default order:
// builtin > user > system. The providers are also loaded and provider their
// metadata/configuration.
func ListActive() (Providers, error) {
	all, err := ListAll()
	if err != nil {
		return nil, err
	}

	var res Providers = make(map[string]*Provider, len(all))
	for _, v := range all {
		res[v.ID] = v
	}

	for _, v := range res {
		// only happens for builtin providers
		if v.Path == "" {
			continue
		}

		if err := v.LoadJSON(); err != nil {
			return nil, err
		}
	}

	// useful for caching; even if the structure gets updated with new providers
	Coordinator.Providers = res
	return res, nil
}

// ListAll available providers, including duplicates between builtin, user,
// and system providers. We only return errors when the things we are trying
// to load don't work. Note that the providers are not loaded yet.
func ListAll() ([]*Provider, error) {
	// This really shouldn't happen, but just in case it does...
	if SystemPath == "" && HomePath == "" {
		log.Warn().Msg("can't find any paths for providers, none are configured")
		return nil, nil
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
		return nil, nil
	}

	var res []*Provider
	if sysOk {
		cur, err := findProviders(SystemPath)
		if err != nil {
			log.Warn().Str("path", SystemPath).Msg("failed to get providers from system path")
		}
		res = append(res, cur...)
	}

	if homeOk {
		cur, err := findProviders(HomePath)
		if err != nil {
			log.Warn().Str("path", HomePath).Msg("failed to get providers from home path")
		}
		res = append(res, cur...)
	}

	for _, x := range builtinProviders {
		res = append(res, &Provider{
			Provider: x.Config,
		})
	}

	return res, nil
}

func isOverlyPermissive(path string) (bool, error) {
	stat, err := config.AppFs.Stat(path)
	if err != nil {
		return true, errors.New("failed to analyze " + path)
	}

	mode := stat.Mode()
	if mode&0o022 != 0 {
		return true, nil
	}

	return false, nil
}

func findProviders(path string) ([]*Provider, error) {
	overlyPermissive, err := isOverlyPermissive(path)
	if err != nil {
		return nil, err
	}
	if overlyPermissive {
		return nil, errors.New("path is overly permissive, make sure it is not writable to others or the group: " + path)
	}

	log.Debug().Str("path", path).Msg("searching providers in path")
	files, err := afero.ReadDir(config.AppFs, path)
	if err != nil {
		return nil, err
	}

	candidates := map[string]struct{}{}
	for i := range files {
		file := files[i]
		if file.Mode().IsDir() {
			candidates[file.Name()] = struct{}{}
		}
	}

	var res []*Provider
	for name := range candidates {
		pdir := filepath.Join(path, name)

		bin := filepath.Join(pdir, name)
		conf := filepath.Join(pdir, name+".json")
		resources := filepath.Join(pdir, name+".resources.json")

		if !config.ProbeFile(bin) {
			log.Debug().Str("path", bin).Msg("ignoring provider because can't access the plugin")
			continue
		}
		if !config.ProbeFile(conf) {
			log.Debug().Str("path", bin).Msg("ignoring provider because can't access the plugin config")
			continue
		}
		if !config.ProbeFile(resources) {
			log.Debug().Str("path", bin).Msg("ignoring provider because can't access the plugin schema")
			continue
		}

		res = append(res, &Provider{
			Provider: &plugin.Provider{
				Name: name,
			},
			Path: filepath.Join(path, name),
		})
	}

	return res, nil
}

// This is the default installation source for core providers.
const upstreamURL = "https://releases.mondoo.com/providers/{NAME}/{VERSION}/{NAME}_{VERSION}_{BUILD}.tar.xz"

func Install(name string) (*Provider, error) {
	panic("INSTALL")
}

func (p *Provider) LoadJSON() error {
	path := filepath.Join(p.Path, p.Name+".json")
	res, err := afero.ReadFile(config.AppFs, path)
	if err != nil {
		return errors.New("failed to read provider json from " + path + ": " + err.Error())
	}

	if err := json.Unmarshal(res, &p.Provider); err != nil {
		return errors.New("failed to parse provider json from " + path + ": " + err.Error())
	}
	return nil
}

func (p *Provider) LoadResources() error {
	path := filepath.Join(p.Path, p.Name+".resources.json")
	res, err := afero.ReadFile(config.AppFs, path)
	if err != nil {
		return errors.New("failed to read provider resources json from " + path + ": " + err.Error())
	}

	if err := json.Unmarshal(res, &p.Schema); err != nil {
		return errors.New("failed to parse provider resources json from " + path + ": " + err.Error())
	}
	return nil
}

func (p *Provider) binPath() string {
	return filepath.Join(p.Path, p.Name)
}

func (p Providers) ForConnection(name string) *Provider {
	for _, provider := range p {
		for i := range provider.Connectors {
			connector := provider.Connectors[i]
			if connector.Name == name {
				return provider
			}
		}
	}

	return nil
}

func (p Providers) Add(nu *Provider) {
	if nu != nil {
		p[nu.Name] = nu
	}
}
