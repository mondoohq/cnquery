// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"encoding/json"
	"fmt"
	"io"
	"runtime"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v11/providers/os/detector/windows"
	"go.mondoo.com/cnquery/v11/providers/os/registry"
	"go.mondoo.com/cnquery/v11/providers/os/resources/cpe"
	"go.mondoo.com/cnquery/v11/providers/os/resources/powershell"
)

// ProcessorArchitecture Enum
// https://docs.microsoft.com/en-us/uwp/api/windows.system.processorarchitecture
// https://docs.microsoft.com/en-us/dotnet/api/system.reflection.processorarchitecture?redirectedfrom=MSDN&view=netframework-4.8
// Microsoft.Windows.Appx.PackageManager.Commands.AppxPackage
// https://github.com/tpn/winsdk-10/blob/master/Include/10.0.10240.0/um/appxpackaging.idl#L60-L67
const (
	WinArchX86     = 0
	WinArchArm     = 5
	WinArchX64     = 9
	WinArchNeutral = 11
	WinArchArm64   = 12
	// The Arm64 processor architecture emulating the X86 architecture
	WinArchX86OnArm64 = 14
)

// https://docs.microsoft.com/en-us/previous-versions/windows/desktop/ff357803(v=vs.85)
var (
	wsusClassificationGUID = map[string]WSUSClassification{
		"5c9376ab-8ce6-464a-b136-22113dd69801 ": Application,
		"434de588-ed14-48f5-8eed-a15e09a991f6":  Connectors,
		"e6cf1350-c01b-414d-a61f-263d14d133b4":  CriticalUpdates,
		"e0789628-ce08-4437-be74-2495b842f43b":  DefinitionUpdates,
		"e140075d-8433-45c3-ad87-e72345b3607":   DeveloperKits,
		"b54e7d24-7add-428f-8b75-90a396fa584f ": FeaturePacks,
		"9511D615-35B2-47BB-927F-F73D8E9260BB":  Guidance,
		"0fa1201d-4330-4fa8-8ae9-b877473b6441":  SecurityUpdates,
		"68c5b0a3-d1a6-4553-ae49-01d3a7827828":  ServicePacks,
		"b4832bd8-e735-4761-8daf-37f882276dab":  Tools,
		"28bc880e-0592-4cbf-8f95-c79b17911d5f":  UpdateRollups,
		"cd5ffd1e-e932-4e3a-bf74-18bf0b1bbd83":  Updates,
		"ebfc1fc5-71a4-4f7b-9aca-3b9a503104a0":  Drivers,
		"8c3fcc84-7410-4a95-8b89-a166a0190486":  Defender,
	}

	appxArchitecture = map[int]string{
		WinArchNeutral:    "neutral",
		WinArchX86:        "x86",
		WinArchX64:        "x64",
		WinArchArm64:      "arm64",
		WinArchArm:        "arm",
		WinArchX86OnArm64: "x86onarm",
	}
)

type WSUSClassification int

const (
	Application WSUSClassification = iota
	Connectors
	CriticalUpdates
	DefinitionUpdates
	DeveloperKits
	FeaturePacks
	Guidance
	SecurityUpdates
	ServicePacks
	Tools
	UpdateRollups
	Updates
	Drivers
	Defender
)

const installedAppsScript = `
Get-ItemProperty (@(
  'HKLM:\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Uninstall\\*',
  'HKCU:\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Uninstall\\*',
  'HKLM:\\SOFTWARE\\Wow6432Node\\Microsoft\\Windows\\CurrentVersion\\Uninstall\\*',
  'HKCU:\\SOFTWARE\\Wow6432Node\\Microsoft\\Windows\\CurrentVersion\\Uninstall\\*'
) | Where-Object { Test-Path $_ }) |
Select-Object -Property DisplayName,DisplayVersion,Publisher,EstimatedSize,InstallSource,UninstallString | ConvertTo-Json -Compress
`

var (
	WINDOWS_QUERY_HOTFIXES      = `Get-HotFix | Select-Object -Property Status, Description, HotFixId, Caption, InstalledOn, InstalledBy | ConvertTo-Json`
	WINDOWS_QUERY_APPX_PACKAGES = `Get-AppxPackage -AllUsers | Select Name, PackageFullName, Architecture, Version, Publisher  | ConvertTo-Json`
)

type powershellWinAppxPackages struct {
	Name         string `json:"Name"`
	FullName     string `json:"PackageFullName"`
	Architecture int    `json:"Architecture"`
	Version      string `json:"Version"`
	Publisher    string `json:"Publisher"`
}

// Good read: https://www.wintips.org/view-installed-apps-and-packages-in-windows-10-8-1-8-from-powershell/
func ParseWindowsAppxPackages(input io.Reader) ([]Package, error) {
	data, err := io.ReadAll(input)
	if err != nil {
		return nil, err
	}

	var appxPackages []powershellWinAppxPackages

	// handle case where no packages are installed
	if len(data) == 0 {
		return []Package{}, nil
	}

	err = json.Unmarshal(data, &appxPackages)
	if err != nil {
		return nil, err
	}

	pkgs := make([]Package, len(appxPackages))
	for i := range appxPackages {
		arch, ok := appxArchitecture[appxPackages[i].Architecture]
		if !ok {
			log.Warn().Int("arch", appxPackages[i].Architecture).Msg("unknown architecture value for windows appx package")
			arch = "unknown"
		}

		cpeWfns := []string{}
		if appxPackages[i].Name != "" && appxPackages[i].Version != "" {
			cpeWfns, err = cpe.NewPackage2Cpe(appxPackages[i].Publisher, appxPackages[i].Name, appxPackages[i].Version, "", "")
			if err != nil {
				log.Debug().Err(err).Str("name", appxPackages[i].Name).Str("version", appxPackages[i].Version).Msg("could not create cpe for windows appx package")
			}
		} else {
			log.Debug().Msg("ignored package since information is missing")
		}

		pkgs[i] = Package{
			Name:    appxPackages[i].Name,
			Version: appxPackages[i].Version,
			Arch:    arch,
			Format:  "windows/appx",
			CPEs:    cpeWfns,
			Vendor:  appxPackages[i].Publisher,
		}
	}
	return pkgs, nil
}

type PowershellWinHotFix struct {
	Status      string `json:"Status"`
	Description string `json:"Description"`
	HotFixId    string `json:"HotFixId"`
	Caption     string `json:"Caption"`
	InstalledOn struct {
		Value    string `json:"value"`
		DateTime string `json:"DateTime"`
	} `json:"InstalledOn"`
	InstalledBy string `json:"InstalledBy"`
}

func (hf PowershellWinHotFix) InstalledOnTime() *time.Time {
	return powershell.PSJsonTimestamp(hf.InstalledOn.Value)
}

func ParseWindowsHotfixes(input io.Reader) ([]PowershellWinHotFix, error) {
	data, err := io.ReadAll(input)
	if err != nil {
		return nil, err
	}

	// for empty result set do not get the '{}', therefore lets abort here
	if len(data) == 0 {
		return []PowershellWinHotFix{}, nil
	}

	var powershellWinHotFixPkgs []PowershellWinHotFix
	err = json.Unmarshal(data, &powershellWinHotFixPkgs)
	if err != nil {
		return nil, err
	}

	return powershellWinHotFixPkgs, nil
}

func HotFixesToPackages(hotfixes []PowershellWinHotFix) []Package {
	pkgs := make([]Package, len(hotfixes))
	for i := range hotfixes {
		pkgs[i] = Package{
			Name:        hotfixes[i].HotFixId,
			Description: hotfixes[i].Description,
			Format:      "windows/hotfix",
		}
	}
	return pkgs
}

type WinPkgManager struct {
	conn     shared.Connection
	platform *inventory.Platform
}

func (w *WinPkgManager) Name() string {
	return "Windows Package Manager"
}

func (w *WinPkgManager) Format() string {
	return "win"
}

func (w *WinPkgManager) getLocalInstalledApps() ([]Package, error) {
	pkgs := []string{
		"HKLM\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Uninstall",
		"HKCU\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Uninstall",
		"HKLM\\SOFTWARE\\Wow6432Node\\Microsoft\\Windows\\CurrentVersion\\Uninstall",
		"HKCU\\SOFTWARE\\Wow6432Node\\Microsoft\\Windows\\CurrentVersion\\Uninstall",
	}
	packages := []Package{}
	for _, r := range pkgs {
		children, err := registry.GetNativeRegistryKeyChildren(r)
		if err != nil {
			continue
		}
		for _, c := range children {
			p, err := getPackageFromRegistryKey(c)
			if err != nil {
				return nil, err
			}
			if p == nil {
				continue
			}
			packages = append(packages, *p)
		}
	}
	return packages, nil
}

func (w *WinPkgManager) getInstalledApps() ([]Package, error) {
	if w.conn.Type() == shared.Type_Local && runtime.GOOS == "windows" {
		return w.getLocalInstalledApps()
	}

	if w.conn.Type() == shared.Type_FileSystem || w.conn.Type() == shared.Type_Device {
		return w.getFsInstalledApps()
	}

	cmd, err := w.conn.RunCommand(powershell.Encode(installedAppsScript))
	if err != nil {
		return nil, fmt.Errorf("could not read app package list")
	}

	if cmd.ExitStatus != 0 {
		stderr, err := io.ReadAll(cmd.Stderr)
		if err != nil {
			return nil, err
		}
		return nil, errors.New("failed to retrieve installed apps: " + string(stderr))
	}

	return ParseWindowsAppPackages(cmd.Stdout)
}

func (w *WinPkgManager) getFsInstalledApps() ([]Package, error) {
	rh := registry.NewRegistryHandler()
	defer func() {
		err := rh.UnloadSubkeys()
		if err != nil {
			log.Debug().Err(err).Msg("could not unload registry subkeys")
		}
	}()
	fi, err := w.conn.FileInfo(registry.SoftwareRegPath)
	if err != nil {
		log.Debug().Err(err).Msg("could not find SOFTWARE registry key file")
		return nil, err
	}
	err = rh.LoadSubkey(registry.Software, fi.Path)
	if err != nil {
		log.Debug().Err(err).Msg("could not load SOFTWARE registry key file")
		return nil, err
	}
	pkgs := []string{
		"Microsoft\\Windows\\CurrentVersion\\Uninstall",
		"Wow6432Node\\Microsoft\\Windows\\CurrentVersion\\Uninstall",
	}
	packages := []Package{}
	for _, r := range pkgs {
		children, err := rh.GetNativeRegistryKeyChildren(registry.Software, r)
		if err != nil {
			continue
		}
		for _, c := range children {
			p, err := getPackageFromRegistryKey(c)
			if err != nil {
				return nil, err
			}
			if p == nil {
				continue
			}
			packages = append(packages, *p)
		}
	}
	return packages, nil
}

func getPackageFromRegistryKey(key registry.RegistryKeyChild) (*Package, error) {
	items, err := registry.GetNativeRegistryKeyItems(key.Path + "\\" + key.Name)
	if err != nil {
		log.Debug().Err(err).Str("path", key.Path).Msg("could not read registry key children")
		return nil, err
	}
	return getPackageFromRegistryKeyItems(items), nil
}

func getPackageFromRegistryKeyItems(children []registry.RegistryKeyItem) *Package {
	var uninstallString string
	var displayName string
	var displayVersion string
	var publisher string

	for _, i := range children {
		switch i.Key {
		case "UninstallString":
			uninstallString = i.Value.String
		case "DisplayName":
			displayName = i.Value.String
		case "DisplayVersion":
			displayVersion = i.Value.String
		case "Publisher":
			publisher = i.Value.String
		}
	}

	if uninstallString == "" {
		return nil
	}

	pkg := &Package{
		Name:    displayName,
		Version: displayVersion,
		Format:  "windows/app",
		Vendor:  publisher,
	}

	if displayName != "" && displayVersion != "" {
		cpeWfns, err := cpe.NewPackage2Cpe(publisher, displayName, displayVersion, "", "")
		if err != nil {
			log.Debug().Err(err).Str("name", displayName).Str("version", displayVersion).Msg("could not create cpe for windows app package")
		} else {
			pkg.CPEs = cpeWfns
		}
	} else {
		log.Debug().Msg("ignored package since information is missing")
	}
	return pkg
}

// returns installed appx packages as well as hot fixes
func (w *WinPkgManager) List() ([]Package, error) {
	b, err := windows.Version(w.platform.Version)
	if err != nil {
		return nil, err
	}

	pkgs := []Package{}
	appPkgs, err := w.getInstalledApps()
	if err != nil {
		return nil, fmt.Errorf("could not read app package list")
	}
	pkgs = append(pkgs, appPkgs...)

	canRunCmd := w.conn.Capabilities().Has(shared.Capability_RunCommand)
	if !canRunCmd {
		log.Debug().Msg("cannot run command on windows, skipping appx package and hotfixes list")
		return pkgs, nil
	}

	if b.Build > 10240 {
		// only win 10+ are compatible with app x packages
		cmd, err := w.conn.RunCommand(powershell.Wrap(WINDOWS_QUERY_APPX_PACKAGES))
		if err != nil {
			return nil, fmt.Errorf("could not read appx package list")
		}
		appxPkgs, err := ParseWindowsAppxPackages(cmd.Stdout)
		if err != nil {
			return nil, fmt.Errorf("could not read appx package list")
		}
		pkgs = append(pkgs, appxPkgs...)
	}

	// hotfixes
	cmd, err := w.conn.RunCommand(powershell.Wrap(WINDOWS_QUERY_HOTFIXES))
	if err != nil {
		return nil, errors.Wrap(err, "could not fetch hotfixes")
	}
	hotfixes, err := ParseWindowsHotfixes(cmd.Stdout)
	if err != nil {
		return nil, errors.Wrapf(err, "could not parse hotfix results")
	}
	hotfixAsPkgs := HotFixesToPackages(hotfixes)

	pkgs = append(pkgs, hotfixAsPkgs...)
	return pkgs, nil
}

func ParseWindowsAppPackages(input io.Reader) ([]Package, error) {
	data, err := io.ReadAll(input)
	if err != nil {
		return nil, err
	}

	// for empty result set do not get the '{}', therefore lets abort here
	if len(data) == 0 {
		return []Package{}, nil
	}

	type powershellUninstallEntry struct {
		DisplayName     string `json:"DisplayName"`
		DisplayVersion  string `json:"DisplayVersion"`
		Publisher       string `json:"Publisher"`
		InstallSource   string `json:"InstallSource"`
		EstimatedSize   int    `json:"EstimatedSize"`
		UninstallString string `json:"UninstallString"`
	}

	var entries []powershellUninstallEntry
	err = json.Unmarshal(data, &entries)
	if err != nil {
		return nil, err
	}

	pkgs := []Package{}
	for i := range entries {
		entry := entries[i]
		if entry.UninstallString == "" {
			continue
		}
		cpeWfns := []string{}
		if entry.DisplayName != "" && entry.DisplayVersion != "" {
			cpeWfns, err = cpe.NewPackage2Cpe(entry.Publisher, entry.DisplayName, entry.DisplayVersion, "", "")
			if err != nil {
				log.Debug().Err(err).Str("name", entry.DisplayName).Str("version", entry.DisplayVersion).Msg("could not create cpe for windows app package")
			}
		} else {
			log.Debug().Msg("ignored package since information is missing")
		}
		pkgs = append(pkgs, Package{
			Name:    entry.DisplayName,
			Version: entry.DisplayVersion,
			Format:  "windows/app",
			CPEs:    cpeWfns,
			Vendor:  entry.Publisher,
		})
	}

	return pkgs, nil
}

func (win *WinPkgManager) Available() (map[string]PackageUpdate, error) {
	return map[string]PackageUpdate{}, nil
}

func (win *WinPkgManager) Files(name string, version string, arch string) ([]FileRecord, error) {
	// not yet implemented
	return nil, nil
}
