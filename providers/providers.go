// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"archive/tar"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"github.com/ulikunitz/xz"
	"go.mondoo.com/cnquery/v10/cli/config"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/resources"
	"golang.org/x/exp/slices"
)

var (
	SystemPath         string
	HomePath           string
	CustomProviderPath string
	// this is the default path for providers, it's either system or home path, if the user is root the system path is used
	DefaultPath string
	// CachedProviders contains all providers that have been loaded the last time
	// ListActive or ListAll have been called
	CachedProviders []*Provider
	// LastProviderInstall keeps track of when the last provider installation
	// took place relative to this runtime. It is initialized to a non-zero
	// timestamp during this file's init() method. Timestamps are unix seconds.
	LastProviderInstall int64
)

func init() {
	SystemPath = config.SystemDataPath("providers")
	DefaultPath = SystemPath
	if os.Geteuid() != 0 {
		HomePath, _ = config.HomePath("providers")
		DefaultPath = HomePath
	}
	CustomProviderPath = os.Getenv("PROVIDERS_PATH")
	if CustomProviderPath != "" {
		DefaultPath = CustomProviderPath
	}

	LastProviderInstall = time.Now().Unix()

	// Initialize the global coordinator instance
	coordinator := newCoordinator()
	Coordinator = coordinator
}

type ProviderLookup struct {
	ID       string
	ConnName string
	ConnType string
}

func (s ProviderLookup) String() string {
	res := []string{}
	if s.ID != "" {
		res = append(res, "id="+s.ID)
	}
	if s.ConnName != "" {
		res = append(res, "name="+s.ConnName)
	}
	if s.ConnType != "" {
		res = append(res, "name="+s.ConnType)
	}
	return strings.Join(res, " ")
}

type Providers map[string]*Provider

// FIXME: DEPRECATED, remove in v12.0 vv
// Unlike lookup, which searches through providers by ID, connection type and
// connection names, this function only cycles through the index of providers
// (which is based on IDs) in order and returns the first found provider.
// We introduced this function to help transition from versioned IDs in
// providers to unversioned IDs in providers.
func (p Providers) GetFirstID(ids ...string) (*Provider, bool) {
	for _, id := range ids {
		if found, ok := p[id]; ok {
			return found, true
		}
	}
	return nil, false
}

// ^^

// Lookup a provider in this list. If you search via ProviderID we will
// try to find the exact provider. Otherwise we will try to find a matching
// connector type first and name second.
func (p Providers) Lookup(search ProviderLookup) *Provider {
	if search.ID != "" {
		for _, provider := range p {
			if provider.ID == search.ID {
				return provider
			}
		}
	}

	if search.ConnType != "" {
		for _, provider := range p {
			if slices.Contains(provider.ConnectionTypes, search.ConnType) {
				return provider
			}
			for i := range provider.Connectors {
				if slices.Contains(provider.Connectors[i].Aliases, search.ConnType) {
					return provider
				}
			}
		}
	}

	if search.ConnName != "" {
		for _, provider := range p {
			for i := range provider.Connectors {
				if provider.Connectors[i].Name == search.ConnName {
					return provider
				}
				if slices.Contains(provider.Connectors[i].Aliases, search.ConnName) {
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

type Provider struct {
	*plugin.Provider
	Schema    *resources.Schema
	Path      string
	HasBinary bool
}

var (
	defaultHttpTimeout         = 30 * time.Second
	defaultIdleConnTimeout     = 30 * time.Second
	defaultTLSHandshakeTimeout = 10 * time.Second
)

func httpClientWithRetry() (*http.Client, error) {
	var proxyFn func(*http.Request) (*url.URL, error)

	proxy, err := config.GetAPIProxy()
	if err != nil {
		log.Fatal().Err(err).Msg("could not parse proxy URL")
	}

	if proxy != nil {
		proxyFn = http.ProxyURL(proxy)
	}

	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 3
	retryClient.Logger = &ZerologAdapter{logger: log.Logger}
	retryClient.HTTPClient = &http.Client{
		Transport: &http.Transport{
			Proxy: proxyFn,
			DialContext: (&net.Dialer{
				Timeout:   defaultHttpTimeout,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       defaultIdleConnTimeout,
			TLSHandshakeTimeout:   defaultTLSHandshakeTimeout,
			ExpectContinueTimeout: 1 * time.Second,
		},
		Timeout: defaultHttpTimeout,
	}
	return retryClient.StandardClient(), nil
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
	Coordinator.SetProviders(res)
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
	if SystemPath == "" && HomePath == "" && CustomProviderPath == "" {
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

	// when the user provides a custom provider path, we always load it and we ignore the system and home path
	// we do not check for its existence, and instead create it on the fly when needed
	if CustomProviderPath != "" {
		cur, err := findProviders(CustomProviderPath)
		if err != nil {
			log.Warn().Str("path", CustomProviderPath).Err(err).Msg("failed to get providers from custom provider path")
		}
		all = append(all, cur...)
	}

	if sysOk && CustomProviderPath == "" {
		cur, err := findProviders(SystemPath)
		if err != nil {
			log.Warn().Str("path", SystemPath).Err(err).Msg("failed to get providers from system path")
		}
		all = append(all, cur...)
	}

	if homeOk && CustomProviderPath == "" {
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

type ProviderNotFoundError struct {
	lookup ProviderLookup
}

func (e *ProviderNotFoundError) Error() string {
	return "cannot find provider for " + e.lookup.String()
}

// EnsureProvider makes sure that a given provider exists and returns it.
// You can supply providers either via:
//  1. providerID, which universally identifies it, e.g. "go.mondoo.com/cnquery/v10/providers/os"
//  2. connectorName, which is what you see in the CLI e.g. "local", "ssh", ...
//  3. connectorType, which is how assets define the connector type when
//     they are moved between discovery and execution, e.g. "registry-image".
//
// If you disable autoUpdate, it will neither update NOR install missing providers.
//
// If you don't supply existing providers, it will look for alist of all
// active providers first.
func EnsureProvider(search ProviderLookup, autoUpdate bool, existing Providers) (*Provider, error) {
	if existing == nil {
		var err error
		existing, err = ListActive()
		if err != nil {
			return nil, err
		}
	}

	provider := existing.Lookup(search)
	if provider != nil {
		return provider, nil
	}

	if search.ID == mockProvider.ID || search.ConnName == "mock" || search.ConnType == "mock" {
		existing.Add(&mockProvider)
		return &mockProvider, nil
	}

	upstream := DefaultProviders.Lookup(search)
	if upstream == nil {
		// we can't find any provider for this connector in our default set
		// FIXME: This causes a panic in the CLI, we should handle this better
		return nil, &ProviderNotFoundError{lookup: search}
	}

	if !autoUpdate {
		return nil, errors.New("cannot find installed provider for " + search.String())
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
	client, err := httpClientWithRetry()
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

	return installed[0], nil
}

func LatestVersion(name string) (string, error) {
	client, err := httpClientWithRetry()
	if err != nil {
		return "", err
	}

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

// kept a tad bit higher to give I/O more time to complete
const osRetryDuration = 100 * time.Millisecond

// In the process of installing larger binaries, we will need time for
// antivirus software to scan it. This is currently set to retry for:
// 100ms (above) * 10 (=1sec) * 60 (=1min) * 3 (=3min)
const maxInstallBinaryRetries = 10 * 60 * 3

// The retries for config files (like JSON) are much shorter, since these
// files are considerably smaller:
// 100ms (above) * 10 (=1sec) * 20 (=20sec)
const maxInstallConfRetries = 10 * 20

// osRetry will try to re-run the given function as long as the resource is busy.
// This is helpful in e.g. Windows systems, which may get an antivirus tool
// check files while we create them (e.g. installing providers).
// It will look for common OS signals that the I/O is busy right now or that
// it asks the caller to run their call again later.
// It is retried every osRetryDuration.
// maxRetry has the maximum number of retries (or -1 for indefinite)
func osRetry(f func() error, maxRetry int) error {
	for maxRetry != 0 {
		err := f()
		if err == nil {
			return nil
		}

		if errno, ok := err.(syscall.Errno); ok && errno.Temporary() {
			time.Sleep(osRetryDuration)
		} else {
			return err
		}

		if maxRetry > 0 {
			maxRetry--
		}
	}
	return nil
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
		// We don't set a max retry, since we can indefinitely try to remove this
		err := osRetry(func() error {
			return os.RemoveAll(tmpdir)
		}, maxInstallConfRetries)
		if err != nil {
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
		if err = osRetry(func() error {
			return os.Rename(srcBin, dstBin)
		}, maxInstallBinaryRetries); err != nil {
			return nil, err
		}
		if err = os.Chmod(dstBin, 0o755); err != nil {
			return nil, err
		}

		srcMeta := filepath.Join(tmpdir, providerName)
		dstMeta := filepath.Join(dstPath, providerName)
		if err = osRetry(func() error {
			return os.Rename(srcMeta+".json", dstMeta+".json")
		}, maxInstallConfRetries); err != nil {
			return nil, err
		}
		if err = osRetry(func() error {
			return os.Rename(srcMeta+".resources.json", dstMeta+".resources.json")
		}, maxInstallConfRetries); err != nil {
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

	// we need to clear out the cache now, because we installed something new,
	// otherwise it will load old data
	CachedProviders = nil
	LastProviderInstall = time.Now().Unix()

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
		Path:      pdir,
		HasBinary: config.ProbeFile(bin),
	}, nil
}

func (p *Provider) LoadJSON() error {
	path := p.confJSONPath()
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

func (p *Provider) confJSONPath() string {
	return filepath.Join(p.Path, p.Name+".json")
}

func (p *Provider) binPath() string {
	name := p.Name
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(p.Path, name)
}

func MustLoadSchema(name string, data []byte) *resources.Schema {
	var res resources.Schema
	if err := json.Unmarshal(data, &res); err != nil {
		panic("failed to embed schema for " + name)
	}
	return &res
}

func MustLoadSchemaFromFile(name string, path string) *resources.Schema {
	raw, err := os.ReadFile(path)
	if err != nil {
		panic("cannot read schema file: " + path)
	}
	return MustLoadSchema(name, raw)
}

// ZerologAdapter adapts the zerolog logger to the LeveledLogger interface.
// Converts all retry logs to debug logs
type ZerologAdapter struct {
	logger zerolog.Logger
}

func (z *ZerologAdapter) Error(msg string, keysAndValues ...interface{}) {
	z.logger.Debug().Fields(convertToFields(keysAndValues...)).Msg(msg)
}

func (z *ZerologAdapter) Info(msg string, keysAndValues ...interface{}) {
	z.logger.Debug().Fields(convertToFields(keysAndValues...)).Msg(msg)
}

func (z *ZerologAdapter) Debug(msg string, keysAndValues ...interface{}) {
	z.logger.Debug().Fields(convertToFields(keysAndValues...)).Msg(msg)
}

func (z *ZerologAdapter) Warn(msg string, keysAndValues ...interface{}) {
	z.logger.Debug().Fields(convertToFields(keysAndValues...)).Msg(msg)
}

func convertToFields(keysAndValues ...interface{}) map[string]interface{} {
	fields := make(map[string]interface{})
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			fields[keysAndValues[i].(string)] = keysAndValues[i+1]
		}
	}
	return fields
}
