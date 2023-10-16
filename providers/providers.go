// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"archive/tar"
	"encoding/json"
	"go.mondoo.com/ranger-rpc"
	"io"
	"net/http"
	"os"
	osfs "os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"github.com/ulikunitz/xz"
	"go.mondoo.com/cnquery/v9/cli/config"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/resources"
	"golang.org/x/exp/slices"
)

var (
	SystemPath string
	HomePath   string
	// this is the default path for providers, it's either system or home path, if the user is root the system path is used
	DefaultPath string
	// CachedProviders contains all providers that have been loaded the last time
	// ListActive or ListAll have been called
	CachedProviders []*Provider
)

func init() {
	SystemPath = config.SystemDataPath("providers")
	DefaultPath = SystemPath
	if os.Geteuid() != 0 {
		HomePath, _ = config.HomePath("providers")
		DefaultPath = HomePath
	}
}

type Providers map[string]*Provider

type Provider struct {
	*plugin.Provider
	Schema *resources.Schema
	Path   string
}

func httpClient() (*http.Client, error) {
	proxy, err := config.GetAPIProxy()
	if err != nil {
		log.Fatal().Err(err).Msg("could not parse proxy URL")
	}
	return ranger.NewHttpClient(ranger.WithProxy(proxy)), nil
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

	// useful for caching; even if the structure gets updated with new providers
	Coordinator.Providers = res
	return res, nil
}

// ListAll available providers, including duplicates between builtin, user,
// and system providers. We only return errors when the things we are trying
// to load don't work.
// Note: We load providers from cache so these expensive calls don't have
// to be repeated. If you want to force a refresh, you can nil out the cache.
func ListAll() ([]*Provider, error) {
	if CachedProviders != nil {
		return CachedProviders, nil
	}

	all := []*Provider{}
	CachedProviders = all

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
		msg.Msg("can't find any paths for providers, none are configured")
	}

	if sysOk {
		cur, err := findProviders(SystemPath)
		if err != nil {
			log.Warn().Str("path", SystemPath).Err(err).Msg("failed to get providers from system path")
		}
		all = append(all, cur...)
	}

	if homeOk {
		cur, err := findProviders(HomePath)
		if err != nil {
			log.Warn().Str("path", HomePath).Err(err).Msg("failed to get providers from home path")
		}
		all = append(all, cur...)
	}

	for _, x := range builtinProviders {
		all = append(all, &Provider{
			Provider: x.Config,
		})
	}

	var res []*Provider
	for i := range all {
		provider := all[i]

		// builtin providers don't need to be loaded, so they are ok to be returned
		if provider.Path == "" {
			res = append(res, provider)
			continue
		}

		// we only add a provider if we can load it, otherwise it has bad
		// consequences for other mechanisms (like attaching shell, listing etc)
		if err := provider.LoadJSON(); err != nil {
			log.Error().Err(err).
				Str("provider", provider.Name).
				Str("path", provider.Path).
				Msg("failed to load provider")
		} else {
			res = append(res, provider)
		}
	}

	CachedProviders = res
	return res, nil
}

// EnsureProvider makes sure that a given provider exists and returns it.
// You can supply providers either via:
//  1. connectorName, which is what you see in the CLI e.g. "local", "ssh", ...
//  2. connectorType, which is how assets define the connector type when
//     they are moved between discovery and execution, e.g. "registry-image".
//
// If you disable autoUpdate, it will neither update NOR install missing providers.
//
// If you don't supply existing providers, it will look for alist of all
// active providers first.
func EnsureProvider(connectorName string, connectorType string, autoUpdate bool, existing Providers) (*Provider, error) {
	if existing == nil {
		var err error
		existing, err = ListActive()
		if err != nil {
			return nil, err
		}
	}

	provider := existing.ForConnection(connectorName, connectorType)
	if provider != nil {
		return provider, nil
	}

	if connectorName == "mock" || connectorType == "mock" {
		existing.Add(&mockProvider)
		return &mockProvider, nil
	}

	upstream := DefaultProviders.ForConnection(connectorName, connectorType)
	if upstream == nil {
		// we can't find any provider for this connector in our default set
		// FIXME: This causes a panic in the CLI, we should handle this better
		return nil, nil
	}

	if !autoUpdate {
		return nil, errors.New("cannot find installed provider for connection " + connectorName)
	}

	nu, err := Install(upstream.Name, "")
	if err != nil {
		return nil, err
	}
	existing.Add(nu)
	PrintInstallResults([]*Provider{nu})
	return nu, nil
}

func Install(name string, version string) (*Provider, error) {
	if version == "" {
		// if no version is specified, we default to installing the latest one
		latestVersion, err := LatestVersion(name)
		if err != nil {
			return nil, err
		}
		version = latestVersion
	}

	log.Info().
		Str("version", version).
		Msg("installing provider '" + name + "'")
	return installVersion(name, version)
}

// This is the default installation source for core providers.
const upstreamURL = "https://releases.mondoo.com/providers/{NAME}/{VERSION}/{NAME}_{VERSION}_{OS}_{ARCH}.tar.xz"

func installVersion(name string, version string) (*Provider, error) {
	url := upstreamURL
	url = strings.ReplaceAll(url, "{NAME}", name)
	url = strings.ReplaceAll(url, "{VERSION}", version)
	url = strings.ReplaceAll(url, "{OS}", runtime.GOOS)
	url = strings.ReplaceAll(url, "{ARCH}", runtime.GOARCH)

	log.Debug().Str("url", url).Msg("installing provider from URL")
	client, err := httpClient()
	if err != nil {
		return nil, err
	}

	res, err := client.Get(url)
	if err != nil {
		log.Debug().Str("url", url).Msg("failed to install from URL (get request)")
		return nil, errors.Wrap(err, "failed to install "+name+"-"+version)
	}

	if res.StatusCode == http.StatusNotFound {
		return nil, errors.New("cannot find provider " + name + "-" + version + " under url " + url)
	} else if res.StatusCode != http.StatusOK {
		log.Debug().Str("url", url).Int("status", res.StatusCode).Msg("failed to install from URL (status code)")
		return nil, errors.New("failed to install " + name + "-" + version + ", received status code: " + res.Status)
	}

	// else we know we got a 200 response, we can safely install
	installed, err := InstallIO(res.Body, InstallConf{
		Dst: DefaultPath,
	})
	if err != nil {
		log.Debug().Str("url", url).Msg("failed to install form URL (download)")
		return nil, errors.Wrap(err, "failed to install "+name+"-"+version)
	}

	if len(installed) == 0 {
		return nil, errors.New("couldn't find installed provider")
	}
	if len(installed) > 1 {
		log.Warn().Msg("too many providers were installed")
	}
	if installed[0].Version != version {
		return nil, errors.New("version for provider didn't match expected install version: expected " + version + ", installed: " + installed[0].Version)
	}

	// we need to clear out the cache now, because we installed something new,
	// otherwise it will load old data
	CachedProviders = nil

	return installed[0], nil
}

func LatestVersion(name string) (string, error) {
	client, err := httpClient()
	if err != nil {
		return "", err
	}
	client.Timeout = time.Duration(5 * time.Second)

	res, err := client.Get("https://releases.mondoo.com/providers/latest.json")
	if err != nil {
		return "", err
	}
	data, err := io.ReadAll(res.Body)
	if err != nil {
		log.Debug().Err(err).Msg("reading latest.json failed")
		return "", errors.New("failed to read response from upstream provider versions")
	}

	var upstreamVersions ProviderVersions
	err = json.Unmarshal(data, &upstreamVersions)
	if err != nil {
		log.Debug().Err(err).Msg("parsing latest.json failed")
		return "", errors.New("failed to parse response from upstream provider versions")
	}

	var latestVersion string
	for i := range upstreamVersions.Providers {
		if upstreamVersions.Providers[i].Name == name {
			latestVersion = upstreamVersions.Providers[i].Version
			break
		}
	}

	if latestVersion == "" {
		return "", errors.New("cannot determine latest version of provider '" + name + "'")
	}
	return latestVersion, nil
}

func PrintInstallResults(providers []*Provider) {
	for i := range providers {
		provider := providers[i]
		log.Info().
			Str("version", provider.Version).
			Str("path", provider.Path).
			Msg("successfully installed " + provider.Name + " provider")
	}
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
		conf.Dst = DefaultPath
	}

	if !config.ProbeDir(conf.Dst) {
		log.Debug().Str("path", conf.Dst).Msg("creating providers directory")
		if err := os.MkdirAll(conf.Dst, 0o755); err != nil {
			return nil, errors.New("failed to create " + conf.Dst)
		}
		if !config.ProbeDir(conf.Dst) {
			return nil, errors.New("cannot write to " + conf.Dst)
		}
	}

	log.Debug().Msg("create temp directory to unpack providers")
	tmpdir, err := os.MkdirTemp(conf.Dst, ".providers-unpack")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create temporary directory to unpack files")
	}

	log.Debug().Str("path", tmpdir).Msg("unpacking providers")
	files := map[string]struct{}{}
	err = walkTarXz(reader, func(reader *tar.Reader, header *tar.Header) error {
		files[header.Name] = struct{}{}
		dst := filepath.Join(tmpdir, header.Name)
		log.Debug().Str("name", header.Name).Str("dest", dst).Msg("unpacking file")
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

	log.Debug().Msg("move provider to destination")
	providerDirs := []string{}
	for name := range files {
		// we only want to identify the binary and then all associated files from it
		// NOTE: we need special handling for windows since binaries have the .exe extension
		if !strings.HasSuffix(name, ".exe") && strings.Contains(name, ".") {
			continue
		}

		providerName := name
		if strings.HasSuffix(name, ".exe") {
			providerName = strings.TrimSuffix(name, ".exe")
		}

		if _, ok := files[providerName+".json"]; !ok {
			return nil, errors.New("cannot find " + providerName + ".json in the archive")
		}
		if _, ok := files[providerName+".resources.json"]; !ok {
			return nil, errors.New("cannot find " + providerName + ".resources.json in the archive")
		}

		dstPath := filepath.Join(conf.Dst, providerName)
		if err = os.MkdirAll(dstPath, 0o755); err != nil {
			return nil, err
		}

		// move the binary and the associated files
		srcBin := filepath.Join(tmpdir, name)
		dstBin := filepath.Join(dstPath, name)
		log.Debug().Str("src", srcBin).Str("dst", dstBin).Msg("move provider binary")
		if err = os.Rename(srcBin, dstBin); err != nil {
			return nil, err
		}
		if err = os.Chmod(dstBin, 0o755); err != nil {
			return nil, err
		}

		srcMeta := filepath.Join(tmpdir, providerName)
		dstMeta := filepath.Join(dstPath, providerName)
		if err = os.Rename(srcMeta+".json", dstMeta+".json"); err != nil {
			return nil, err
		}
		if err = os.Rename(srcMeta+".resources.json", dstMeta+".resources.json"); err != nil {
			return nil, err
		}

		providerDirs = append(providerDirs, dstPath)
	}

	log.Debug().Msg("loading providers")
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
	// We don't check the permissions for windows
	if runtime.GOOS != "windows" && mode&0o022 != 0 {
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
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	conf := filepath.Join(pdir, name+".json")
	resources := filepath.Join(pdir, name+".resources.json")

	if !config.ProbeFile(bin) {
		log.Debug().Str("path", bin).Msg("ignoring provider, can't access the plugin")
		return nil, nil
	}
	if !config.ProbeFile(conf) {
		log.Debug().Str("path", conf).Msg("ignoring provider, can't access the plugin config")
		return nil, nil
	}
	if !config.ProbeFile(resources) {
		log.Debug().Str("path", resources).Msg("ignoring provider, can't access the plugin schema")
		return nil, nil
	}

	return &Provider{
		Provider: &plugin.Provider{
			Name: name,
		},
		Path: pdir,
	}, nil
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
	name := p.Name
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(p.Path, name)
}

func (p Providers) ForConnection(name string, typ string) *Provider {
	if name != "" {
		for _, provider := range p {
			for i := range provider.Connectors {
				if provider.Connectors[i].Name == name {
					return provider
				}
				if slices.Contains(provider.Connectors[i].Aliases, name) {
					return provider
				}
			}
		}
	}

	if typ != "" {
		for _, provider := range p {
			if slices.Contains(provider.ConnectionTypes, typ) {
				return provider
			}
			for i := range provider.Connectors {
				if slices.Contains(provider.Connectors[i].Aliases, typ) {
					return provider
				}
			}
		}
	}

	return nil
}

func (p Providers) Add(nu *Provider) {
	if nu != nil {
		p[nu.ID] = nu
	}
}

func MustLoadSchema(name string, data []byte) *resources.Schema {
	var res resources.Schema
	if err := json.Unmarshal(data, &res); err != nil {
		panic("failed to embed schema for " + name)
	}
	return &res
}

func MustLoadSchemaFromFile(name string, path string) *resources.Schema {
	raw, err := osfs.ReadFile(path)
	if err != nil {
		panic("cannot read schema file: " + path)
	}
	return MustLoadSchema(name, raw)
}
