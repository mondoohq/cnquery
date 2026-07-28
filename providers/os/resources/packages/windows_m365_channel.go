// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers/os/registry"
	"go.mondoo.com/mql/v13/providers/os/resources/purl"
)

// Microsoft 365 Apps (Click-to-Run) update channel collection.
//
// M365 Apps has the same shape as an RPM modular package: one package name with
// several concurrently-valid version tracks (Current / Monthly Enterprise /
// Semi-Annual Enterprise channels) whose builds must not be compared against
// each other's fixed-version ranges. Microsoft can ship the same marketing
// Version number on several channels in the same month at different build
// revisions (July 2026's Version 2606 shipped as 20131.20154 / .20152 / .20150),
// and without a channel signal a matcher has to collapse them.
//
// We attach the channel the same way rpm_packages.go attaches modularity: as a
// purl qualifier (`?channel=current`), never by rewriting the display name. A
// package without the qualifier keeps matching the plain, unqualified purl, so
// an asset whose registry read fails or is denied simply degrades to the
// version-only match instead of dropping out.
//
// The qualifier is collected here first; the vulnerability matching side
// consumes it separately, and until it does the extra qualifier is inert.
const m365ChannelQualifier = "channel"

// Click-to-Run writes its configuration to the 64-bit view on 64-bit Windows,
// and to the Wow6432Node view when 32-bit Office is installed on a 64-bit OS.
// Probe both, in that order.
var (
	// full paths, used against the native registry API and PowerShell
	officeC2RConfigKeys = []string{
		"HKLM\\SOFTWARE\\Microsoft\\Office\\ClickToRun\\Configuration",
		"HKLM\\SOFTWARE\\WOW6432Node\\Microsoft\\Office\\ClickToRun\\Configuration",
	}
	// hive-relative paths, used when a SOFTWARE hive is mounted from a filesystem
	officeC2RConfigHivePaths = []string{
		"Microsoft\\Office\\ClickToRun\\Configuration",
		"WOW6432Node\\Microsoft\\Office\\ClickToRun\\Configuration",
	}
)

// m365ChannelValueNames are the ClickToRun\Configuration values that can carry
// the update channel, in the order we trust them for the build that is actually
// INSTALLED:
//
//   - CDNBaseUrl is written when Office installs from a channel, so it describes
//     the provenance of the bits on disk. That is what a fixed-build comparison
//     needs. It is also the value Configuration Manager's own channel detection
//     reads.
//   - UpdateChannel / UpdateUrl / UnmanagedUpdateURL describe the ASSIGNED
//     channel, i.e. where the next update will come from. During a channel
//     switch they move ahead of the installed build (Microsoft: "the client
//     device's user interface will display the updated channel only after
//     installing a build from the new channel"), so they are a fallback for
//     installs that have no CDNBaseUrl rather than a preferred source.
//
// Either way the value can lag reality during a transition, so consumers must
// treat the qualifier as a hint that scopes a match group, not as ground truth.
//
// Ref: https://learn.microsoft.com/microsoft-365-apps/updates/change-update-channels
var m365ChannelValueNames = []string{
	"CDNBaseUrl",
	"UpdateChannel",
	"UpdateUrl",
	"UnmanagedUpdateURL",
}

// m365ChannelByGUID maps Microsoft's stable per-channel audience GUIDs to the
// normalized channel token used in the purl qualifier. The same GUIDs key the
// server-side channel list (officeCdnServicedChannels), so both sides agree on
// what a channel is without exchanging display names.
//
// Ref: https://learn.microsoft.com/intune/configmgr/sum/deploy-use/manage-office-365-proplus-updates#update-channels-for-microsoft-365-apps
var m365ChannelByGUID = map[string]string{
	"492350f6-3a01-4f97-b9c0-c7c6ddf67d60": "current",
	"64256afe-f5d9-4f86-8936-8840a6a4f5be": "current-preview",
	"55336b82-a18d-4dd6-b5f6-9e5095c314a6": "monthly-enterprise",
	"7ffbc6bf-bc32-4f92-8982-f9dd17fd3114": "semi-annual-enterprise",
	"b8f9b850-328d-4355-9145-c59439a0c4cf": "semi-annual-enterprise-preview",
	"5440fd1f-7ecb-4221-8110-145efaa6372f": "beta",
}

// m365ChannelByName maps the channel names the Office Deployment Tool and the
// Update Channel group policy accept to the same normalized tokens. The
// ClickToRun values we read normally hold a CDN URL, but a policy-driven install
// can leave a bare channel name behind, so accept both forms.
//
// Ref: https://learn.microsoft.com/microsoft-365-apps/deploy/office-deployment-tool-configuration-options#updates-element
var m365ChannelByName = map[string]string{
	"current":              "current",
	"monthly":              "current",
	"currentpreview":       "current-preview",
	"firstreleasecurrent":  "current-preview",
	"insiderslow":          "current-preview",
	"monthlyenterprise":    "monthly-enterprise",
	"semiannual":           "semi-annual-enterprise",
	"deferred":             "semi-annual-enterprise",
	"broad":                "semi-annual-enterprise",
	"semiannualpreview":    "semi-annual-enterprise-preview",
	"firstreleasedeferred": "semi-annual-enterprise-preview",
	"targeted":             "semi-annual-enterprise-preview",
	"betachannel":          "beta",
	"insiderfast":          "beta",
	"perpetualvl2019":      "perpetual-vl-2019",
	"perpetualvl2021":      "perpetual-vl-2021",
	"perpetualvl2024":      "perpetual-vl-2024",
}

// m365AppsNameRegExp matches the Click-to-Run subscription SKUs, the ones that
// have a per-device update channel. MSI-installed Office (Office 2016 and
// earlier, and the volume-licensed Professional Plus SKUs) has none, so it is
// deliberately not matched: an unqualified purl is the correct output there.
var m365AppsNameRegExp = regexp.MustCompile(`(?i)^Microsoft (365 Apps\b|365 - |Office 365 (ProPlus|Business)\b)`)

// isM365AppsPackage reports whether a Windows app display name is a
// Click-to-Run Microsoft 365 Apps SKU.
func isM365AppsPackage(name string) bool {
	return m365AppsNameRegExp.MatchString(name)
}

// normalizeM365Channel turns a raw ClickToRun configuration value into the
// normalized channel token. It accepts a CDN URL
// ("http://officecdn.microsoft.com/pr/<guid>"), a bare audience GUID, and the
// ODT/group-policy channel names.
//
// An unrecognized value returns the empty string on purpose. A channel Microsoft
// introduces after this build ships would otherwise be stamped as an opaque GUID
// that no consumer can interpret; falling back to no qualifier keeps the package
// on the plain-purl match path, which is the documented degradation tier. The
// same is true for a value that isn't a channel at all, e.g. an UpdateUrl
// pointing at an internal file share.
func normalizeM365Channel(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "/")
	if value == "" {
		return ""
	}

	// a CDN URL carries the audience GUID in its last path segment
	guid := value
	if idx := strings.LastIndex(guid, "/"); idx >= 0 {
		guid = guid[idx+1:]
	}
	if channel, ok := m365ChannelByGUID[strings.ToLower(guid)]; ok {
		return channel
	}

	if channel, ok := m365ChannelByName[strings.ToLower(strings.ReplaceAll(value, " ", ""))]; ok {
		return channel
	}

	return ""
}

// m365ChannelFromValues resolves the update channel by asking `read` for each
// candidate ClickToRun value in priority order. `read` returns the raw registry
// value, or the empty string when it is absent.
func m365ChannelFromValues(read func(valueName string) string) string {
	for _, name := range m365ChannelValueNames {
		if channel := normalizeM365Channel(read(name)); channel != "" {
			return channel
		}
	}
	return ""
}

// m365ChannelFromRegistryItems resolves the update channel from the values of an
// already-read ClickToRun\Configuration key.
func m365ChannelFromRegistryItems(items []registry.RegistryKeyItem) string {
	return m365ChannelFromValues(func(valueName string) string {
		for i := range items {
			if strings.EqualFold(items[i].Key, valueName) {
				return items[i].Value.String
			}
		}
		return ""
	})
}

// applyM365ChannelQualifier stamps the `channel` purl qualifier onto every
// Click-to-Run Microsoft 365 Apps package in pkgs.
//
// `resolve` is only called when there is something to stamp, so assets without
// Office never pay for the registry probe. When it yields no channel the
// packages are left untouched and keep their plain purl — the fallback tier for
// agents that cannot read the ClickToRun configuration.
func applyM365ChannelQualifier(pkgs []Package, platform *inventory.Platform, resolve func() string) {
	targets := []int{}
	for i := range pkgs {
		if isM365AppsPackage(pkgs[i].Name) {
			targets = append(targets, i)
		}
	}
	if len(targets) == 0 {
		return
	}

	channel := resolve()
	if channel == "" {
		log.Debug().Msg("could not determine the Microsoft 365 Apps update channel, falling back to the unqualified package url")
		return
	}

	for _, i := range targets {
		pkgs[i].PUrl = purl.NewPackageURL(
			platform, purl.TypeWindows, pkgs[i].Name, pkgs[i].Version,
			purl.WithQualifiers(map[string]string{m365ChannelQualifier: channel}),
		).String()
		log.Debug().Str("package", pkgs[i].Name).Str("channel", channel).Msg("detected Microsoft 365 Apps update channel")
	}
}

// m365ChannelFromNativeRegistry reads the Click-to-Run update channel via the
// native registry API. Only valid on a local Windows host.
func (w *WinPkgManager) m365ChannelFromNativeRegistry() string {
	for _, path := range officeC2RConfigKeys {
		items, err := registry.GetNativeRegistryKeyItems(path)
		if err != nil {
			log.Debug().Err(err).Str("path", path).Msg("could not read the ClickToRun configuration")
			continue
		}
		if channel := m365ChannelFromRegistryItems(items); channel != "" {
			return channel
		}
	}
	return ""
}

// m365ChannelFromPowershell reads the Click-to-Run update channel over the
// active connection, so it works for remote connections such as SSH and WinRM.
func (w *WinPkgManager) m365ChannelFromPowershell() string {
	for _, path := range officeC2RConfigKeys {
		items, err := w.readRegistryItems(path)
		if err != nil {
			log.Debug().Err(err).Str("path", path).Msg("could not read the ClickToRun configuration")
			continue
		}
		if channel := m365ChannelFromRegistryItems(items); channel != "" {
			return channel
		}
	}
	return ""
}

// m365ChannelFromHive reads the Click-to-Run update channel from a mounted
// SOFTWARE hive. The handler must already have that hive loaded.
func (w *WinPkgManager) m365ChannelFromHive(rh *registry.RegistryHandler) string {
	for _, path := range officeC2RConfigHivePaths {
		channel := m365ChannelFromValues(func(valueName string) string {
			item, err := rh.GetRegistryItemValue(registry.Software, path, valueName)
			if err != nil {
				return ""
			}
			return item.Value.String
		})
		if channel != "" {
			return channel
		}
	}
	return ""
}
