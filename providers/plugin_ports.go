// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"os"
	"strconv"
	"strings"

	"github.com/cockroachdb/errors"
	"go.mondoo.com/mql/cli/config"
)

// Providers are hashicorp/go-plugin subprocesses. On Linux and macOS the host
// reaches them over a Unix socket, but go-plugin hard-codes loopback TCP on
// Windows and lets the client pick the port range. Its own default is
// 10000-25000, tried in ascending order, so the first provider a scan started
// (usually os) always took port 10000 and collided with anything else that
// owns it, and 10000 is a popular one (Webmin, plenty of Java stacks). The
// range reaches the subprocess as PLUGIN_MIN_PORT/PLUGIN_MAX_PORT appended
// after the host environment, so setting those on the host changed nothing
// either.
//
// The default below sits in the IANA dynamic range (49152-65535), which no
// service is supposed to claim as a fixed port. go-plugin skips ports that are
// busy or excluded (Hyper-V and WSL reserve blocks in that range), and each
// running provider takes one port, so the range needs room for as many
// providers as a scan runs at once.
const (
	// envPluginMinPort and envPluginMaxPort are go-plugin's own environment
	// variables. Honoring them on the host makes the values a user sees in a
	// provider process's environment actually mean something.
	envPluginMinPort = "PLUGIN_MIN_PORT"
	envPluginMaxPort = "PLUGIN_MAX_PORT"

	defaultPluginMinPort uint = 50000
	defaultPluginMaxPort uint = 65535
)

// pluginPortRange is the inclusive loopback TCP port range handed to a
// provider subprocess. Only Windows uses it; other platforms ignore it.
type pluginPortRange struct {
	Min, Max uint
}

// resolvePluginPortRange decides which ports a provider subprocess may listen on:
//
//  1. provider_port_range in mondoo.yml or MONDOO_PROVIDER_PORT_RANGE, as "min-max"
//  2. PLUGIN_MIN_PORT and PLUGIN_MAX_PORT in the host environment, both required
//  3. the default range
//
// A value that is present but malformed is an error rather than a silent fall
// back to the default: an operator who set it did so to move away from a
// collision, and keeping the old range would leave that collision in place
// unannounced.
func resolvePluginPortRange() (pluginPortRange, error) {
	return resolvePluginPortRangeFrom(config.GetProviderPortRange(), os.Getenv(envPluginMinPort), os.Getenv(envPluginMaxPort))
}

// resolvePluginPortRangeFrom is the pure core of resolvePluginPortRange, with
// the config value and the two environment variables passed in so it can be
// tested without touching the process environment.
func resolvePluginPortRangeFrom(configured, envMin, envMax string) (pluginPortRange, error) {
	if configured = strings.TrimSpace(configured); configured != "" {
		r, err := parsePluginPortRange(configured)
		if err != nil {
			return pluginPortRange{}, errors.Wrapf(err, "invalid %s %q", config.KeyProviderPortRange, configured)
		}
		return r, nil
	}

	envMin, envMax = strings.TrimSpace(envMin), strings.TrimSpace(envMax)
	if envMin != "" || envMax != "" {
		if envMin == "" || envMax == "" {
			return pluginPortRange{}, errors.Newf("%s and %s must be set together", envPluginMinPort, envPluginMaxPort)
		}
		r, err := newPluginPortRange(envMin, envMax)
		if err != nil {
			return pluginPortRange{}, errors.Wrapf(err, "invalid %s=%q %s=%q", envPluginMinPort, envMin, envPluginMaxPort, envMax)
		}
		return r, nil
	}

	return pluginPortRange{Min: defaultPluginMinPort, Max: defaultPluginMaxPort}, nil
}

// parsePluginPortRange parses the "min-max" form, for example "50000-50100".
func parsePluginPortRange(s string) (pluginPortRange, error) {
	minS, maxS, ok := strings.Cut(s, "-")
	if !ok {
		return pluginPortRange{}, errors.New("expected the form min-max, for example 50000-50100")
	}
	return newPluginPortRange(strings.TrimSpace(minS), strings.TrimSpace(maxS))
}

func newPluginPortRange(minS, maxS string) (pluginPortRange, error) {
	minPort, err := parsePort(minS)
	if err != nil {
		return pluginPortRange{}, err
	}
	maxPort, err := parsePort(maxS)
	if err != nil {
		return pluginPortRange{}, err
	}
	if minPort > maxPort {
		return pluginPortRange{}, errors.Newf("min port %d is greater than max port %d", minPort, maxPort)
	}
	return pluginPortRange{Min: minPort, Max: maxPort}, nil
}

// parsePort accepts 1-65535. Port 0 is rejected because go-plugin reads a
// 0-0 range as "unset" and falls back to its own 10000-25000 default, which
// is exactly the range this setting exists to move away from.
func parsePort(s string) (uint, error) {
	p, err := strconv.ParseUint(s, 10, 16)
	if err != nil {
		return 0, errors.Newf("%q is not a valid port", s)
	}
	if p == 0 {
		return 0, errors.New("port 0 is not allowed")
	}
	return uint(p), nil
}
