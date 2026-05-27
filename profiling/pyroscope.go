// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package profiling integrates with Grafana Pyroscope for continuous
// profiling. It is configured entirely via environment variables so the
// same code path works for the main mql binary and every provider
// subprocess.
package profiling

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/grafana/pyroscope-go"
	"github.com/rs/zerolog/log"
)

const (
	EnvEnabled              = "MONDOO_PYROSCOPE_ENABLED"
	EnvServerAddress        = "MONDOO_PYROSCOPE_SERVER_ADDRESS"
	EnvAuthToken            = "MONDOO_PYROSCOPE_AUTH_TOKEN"
	EnvBasicAuthUser        = "MONDOO_PYROSCOPE_BASIC_AUTH_USER"
	EnvBasicAuthPass        = "MONDOO_PYROSCOPE_BASIC_AUTH_PASSWORD"
	EnvTenantID             = "MONDOO_PYROSCOPE_TENANT_ID"
	EnvHTTPHeaders          = "MONDOO_PYROSCOPE_HTTP_HEADERS"
	EnvSampleRate           = "MONDOO_PYROSCOPE_SAMPLE_RATE"
	EnvUploadRateSecs       = "MONDOO_PYROSCOPE_UPLOAD_RATE_SECONDS"
	EnvTags                 = "MONDOO_PYROSCOPE_TAGS"
	EnvApplicationName      = "MONDOO_PYROSCOPE_APPLICATION_NAME"
	EnvMutexProfileFraction = "MONDOO_PYROSCOPE_MUTEX_PROFILE_FRACTION"
	EnvBlockProfileRateNs   = "MONDOO_PYROSCOPE_BLOCK_PROFILE_RATE_NS"
)

// Defaults for the runtime profilers that the Go runtime ships disabled.
// Tuned for "always on" use rather than the aggressive 1-in-5 / 5ns example
// values from pyroscope-go's README:
//
//   - 1-in-100 mutex events ≈ 1% sampling rate
//   - 10µs block threshold skips the avalanche of sub-µs channel/mutex
//     parks that dominate Go programs and would otherwise dwarf the
//     genuinely slow waits in the profile.
//
// Set the env vars to 0 to disable either; set to 1 to record everything.
const (
	defaultMutexProfileFraction = 100
	defaultBlockProfileRateNs   = 10_000
)

// Stopper is the subset of pyroscope.Profiler we expose so callers can
// shut profiling down on process exit. Returns nil when profiling is
// disabled, so callers can unconditionally defer Stop().
type Stopper interface {
	Stop() error
}

type noopStopper struct{}

func (noopStopper) Stop() error { return nil }

// Start initializes Pyroscope if MONDOO_PYROSCOPE_ENABLED is truthy.
//
// serviceName is used as the application name and is also stored as the
// "service" tag so the same Pyroscope instance can host both the main
// mql binary and every provider subprocess. extraTags are merged on top
// of the env-supplied tags (env wins) and is meant for caller-known
// metadata like the provider name.
//
// When profiling is disabled, Start returns a no-op Stopper and nil
// error — callers can `defer stopper.Stop()` unconditionally.
func Start(serviceName string, extraTags map[string]string) (Stopper, error) {
	if !isEnabled() {
		return noopStopper{}, nil
	}

	addr := strings.TrimSpace(os.Getenv(EnvServerAddress))
	if addr == "" {
		return noopStopper{}, errors.New(EnvServerAddress + " must be set when " + EnvEnabled + " is true")
	}

	appName := os.Getenv(EnvApplicationName)
	if appName == "" {
		appName = serviceName
	}

	tags := map[string]string{
		"service": serviceName,
	}
	if hn, err := os.Hostname(); err == nil && hn != "" {
		tags["hostname"] = hn
	}
	for k, v := range extraTags {
		if k == "" || v == "" {
			continue
		}
		tags[k] = v
	}
	for k, v := range parseTags(os.Getenv(EnvTags)) {
		tags[k] = v
	}

	headers := parseTags(os.Getenv(EnvHTTPHeaders))

	cfg := pyroscope.Config{
		ApplicationName:   appName,
		ServerAddress:     addr,
		AuthToken:         os.Getenv(EnvAuthToken),
		BasicAuthUser:     os.Getenv(EnvBasicAuthUser),
		BasicAuthPassword: os.Getenv(EnvBasicAuthPass),
		TenantID:          os.Getenv(EnvTenantID),
		HTTPHeaders:       headers,
		Tags:              tags,
		Logger:            zerologAdapter{},
		ProfileTypes: []pyroscope.ProfileType{
			pyroscope.ProfileCPU,
			pyroscope.ProfileAllocObjects,
			pyroscope.ProfileAllocSpace,
			pyroscope.ProfileInuseObjects,
			pyroscope.ProfileInuseSpace,
			pyroscope.ProfileGoroutines,
			pyroscope.ProfileMutexCount,
			pyroscope.ProfileMutexDuration,
			pyroscope.ProfileBlockCount,
			pyroscope.ProfileBlockDuration,
		},
	}

	if rate := os.Getenv(EnvSampleRate); rate != "" {
		n, err := strconv.ParseUint(rate, 10, 32)
		if err != nil || n == 0 {
			return noopStopper{}, fmt.Errorf("invalid %s=%q (want positive integer Hz)", EnvSampleRate, rate)
		}
		cfg.SampleRate = uint32(n)
	}

	if rate := os.Getenv(EnvUploadRateSecs); rate != "" {
		n, err := strconv.ParseUint(rate, 10, 32)
		if err != nil || n == 0 {
			return noopStopper{}, fmt.Errorf("invalid %s=%q (want positive integer seconds)", EnvUploadRateSecs, rate)
		}
		cfg.UploadRate = time.Duration(n) * time.Second
	}

	// Enable mutex and block profiles. Pyroscope can collect them, but the
	// Go runtime only records samples once these knobs are set. Defaults
	// are tuned to be safe for continuous use; override via env if you're
	// hunting a specific contention bug.
	mutexFrac, err := readIntEnv(EnvMutexProfileFraction, defaultMutexProfileFraction)
	if err != nil {
		return noopStopper{}, err
	}
	blockRate, err := readIntEnv(EnvBlockProfileRateNs, defaultBlockProfileRateNs)
	if err != nil {
		return noopStopper{}, err
	}
	runtime.SetMutexProfileFraction(mutexFrac)
	runtime.SetBlockProfileRate(blockRate)

	p, err := pyroscope.Start(cfg)
	if err != nil {
		return noopStopper{}, fmt.Errorf("start pyroscope: %w", err)
	}

	log.Info().
		Str("server", addr).
		Str("application", appName).
		Interface("tags", tags).
		Msg("Pyroscope profiling started")
	return p, nil
}

// readIntEnv returns the int value of the named env var, or def if unset.
// Empty string is treated as "use the default", not "0", so an operator who
// exports the var without a value doesn't silently disable a profile.
func readIntEnv(name string, def int) (int, error) {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("invalid %s=%q (want integer)", name, v)
	}
	return n, nil
}

func isEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(EnvEnabled)))
	switch v {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// parseTags accepts "k=v,k2=v2" (commas) and "k=v;k2=v2" (semicolons).
// Whitespace around keys and values is trimmed. Empty pairs are
// skipped silently so trailing separators don't break parsing.
func parseTags(s string) map[string]string {
	out := map[string]string{}
	if s == "" {
		return out
	}
	splitter := func(r rune) bool { return r == ',' || r == ';' }
	for _, pair := range strings.FieldsFunc(s, splitter) {
		eq := strings.IndexByte(pair, '=')
		if eq <= 0 {
			continue
		}
		k := strings.TrimSpace(pair[:eq])
		v := strings.TrimSpace(pair[eq+1:])
		if k == "" {
			continue
		}
		out[k] = v
	}
	return out
}

// zerologAdapter adapts pyroscope's Logger interface to our zerolog global
// logger. Pyroscope's Infof/Debugf are dropped because they're noisy
// (a multi-line config dump at startup, and upload-loop chatter); our own
// one-line "Pyroscope profiling started" log in Start() is enough to
// confirm the agent engaged. Errorf is downgraded to Warn since upload
// failures shouldn't look like application errors.
type zerologAdapter struct{}

func (zerologAdapter) Infof(format string, args ...any)  {}
func (zerologAdapter) Debugf(format string, args ...any) {}

func (zerologAdapter) Errorf(format string, args ...any) {
	log.Warn().Msgf("pyroscope: "+format, args...)
}
