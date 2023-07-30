package providers

import (
	"archive/tar"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"github.com/ulikunitz/xz"
	"go.mondoo.com/cnquery/cli/config"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers-sdk/v1/resources"
)

var (
	SystemPath string
	HomePath   string
	// CachedProviders contains all providers that have been loaded the last time
	// ListActive or ListAll have been called
	CachedProviders []*Provider
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
// to load don't work.
// Note: That the providers are not loaded yet.
// Note: We load providers from cache so these expensive calls don't have
// to be repeated. If you want to force a refresh, you can nil out the cache.
func ListAll() ([]*Provider, error) {
	if CachedProviders != nil {
		return CachedProviders, nil
	}

	res := []*Provider{}
	CachedProviders = res

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
	}

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

	CachedProviders = res
	return res, nil
}

func Install(name string) (*Provider, error) {
	panic("INSTALL")
}

type InstallConf struct {
	// Dst specify which path to install into.
	Dst string
}

func InstallFile(path string, conf InstallConf) ([]*Provider, error) {
	if !config.ProbeFile(path) {
		return nil, errors.New("please provide a regular file when installing providers")
	}

	reader, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return InstallIO(reader, conf)
}

func InstallIO(reader io.ReadCloser, conf InstallConf) ([]*Provider, error) {
	if conf.Dst == "" {
		conf.Dst = HomePath
	}
	if !config.ProbeDir(conf.Dst) {
		if err := os.MkdirAll(conf.Dst, 0o755); err != nil {
			return nil, errors.New("failed to create " + conf.Dst)
		}
		if !config.ProbeDir(conf.Dst) {
			return nil, errors.New("cannot write to " + conf.Dst)
		}
	}

	tmpdir, err := os.MkdirTemp(conf.Dst, ".providers-unpack")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create temporary directory to unpack files")
	}

	files := map[string]struct{}{}
	err = walkTarXz(reader, func(reader *tar.Reader, header *tar.Header) error {
		files[header.Name] = struct{}{}
		dst := filepath.Join(tmpdir, header.Name)
		writer, err := os.Create(dst)
		if err != nil {
			return err
		}
		defer writer.Close()

		_, err = io.Copy(writer, reader)
		return err
	})
	if err != nil {
		return nil, err
	}

	// If for any reason we drop here, it's best to clean up all temporary files
	// so we don't spam the system with unnecessary data. Optionally we could
	// keep them and re-use them, so they don't have to download again.
	defer func() {
		if err = os.RemoveAll(tmpdir); err != nil {
			log.Error().Err(err).Msg("failed to remove temporary folder for unpacked provider")
		}
	}()

	providerDirs := []string{}
	for name := range files {
		// we only want to identify the binary and then all associated files from it
		if strings.Contains(name, ".") {
			continue
		}

		if _, ok := files[name+".json"]; !ok {
			return nil, errors.New("cannot find " + name + ".json in the archive")
		}
		if _, ok := files[name+".resources.json"]; !ok {
			return nil, errors.New("cannot find " + name + ".resources.json in the archive")
		}

		dstPath := filepath.Join(conf.Dst, name)
		if err = os.Mkdir(dstPath, 0o755); err != nil {
			return nil, err
		}

		srcBin := filepath.Join(tmpdir, name)
		dstBin := filepath.Join(dstPath, name)
		if err = os.Rename(srcBin, dstBin); err != nil {
			return nil, err
		}
		if err = os.Rename(srcBin+".json", dstBin+".json"); err != nil {
			return nil, err
		}
		if err = os.Rename(srcBin+".resources.json", dstBin+".resources.json"); err != nil {
			return nil, err
		}

		providerDirs = append(providerDirs, dstPath)
	}

	res := []*Provider{}
	for i := range providerDirs {
		pdir := providerDirs[i]
		provider, err := readProviderDir(pdir)
		if err != nil {
			return nil, err
		}

		if provider == nil {
			log.Error().Err(err).Str("path", pdir).Msg("failed to read provider, please remove or fix it")
			continue
		}

		if err := provider.LoadJSON(); err != nil {
			log.Error().Err(err).Str("path", pdir).Msg("failed to read provider metadata, please remove or fix it")
			continue
		}

		res = append(res, provider)
	}

	return res, nil
}

func walkTarXz(reader io.Reader, callback func(reader *tar.Reader, header *tar.Header) error) error {
	r, err := xz.NewReader(reader)
	if err != nil {
		return errors.Wrap(err, "failed to read xz")
	}

	tarReader := tar.NewReader(r)
	for {
		header, err := tarReader.Next()
		// end of archive
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, "failed to read tar")
		}

		switch header.Typeflag {
		case tar.TypeReg:
			if err = callback(tarReader, header); err != nil {
				return err
			}

		default:
			log.Warn().Str("name", header.Name).Msg("encounter a file in archive that is not supported, skipping it")
		}
	}
	return nil
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
		provider, err := readProviderDir(pdir)
		if err != nil {
			return nil, err
		}
		if provider != nil {
			res = append(res, provider)
		}
	}

	return res, nil
}

func readProviderDir(pdir string) (*Provider, error) {
	name := filepath.Base(pdir)
	bin := filepath.Join(pdir, name)
	conf := filepath.Join(pdir, name+".json")
	resources := filepath.Join(pdir, name+".resources.json")

	if !config.ProbeFile(bin) {
		log.Debug().Str("path", bin).Msg("ignoring provider, can't access the plugin")
		return nil, nil
	}
	if !config.ProbeFile(conf) {
		log.Debug().Str("path", bin).Msg("ignoring provider, can't access the plugin config")
		return nil, nil
	}
	if !config.ProbeFile(resources) {
		log.Debug().Str("path", bin).Msg("ignoring provider, can't access the plugin schema")
		return nil, nil
	}

	return &Provider{
		Provider: &plugin.Provider{
			Name: name,
		},
		Path: pdir,
	}, nil
}

// This is the default installation source for core providers.
const upstreamURL = "https://releases.mondoo.com/providers/{NAME}/{VERSION}/{NAME}_{VERSION}_{BUILD}.tar.xz"

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
