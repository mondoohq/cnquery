package packages

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/platform/windows"
	"go.mondoo.com/cnquery/motor/providers/os"
	"go.mondoo.com/cnquery/motor/providers/os/powershell"
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

var (
	WINDOWS_QUERY_HOTFIXES      = `Get-HotFix | Select-Object -Property Status, Description, HotFixId, Caption, InstalledOn, InstalledBy | ConvertTo-Json`
	WINDOWS_QUERY_APPX_PACKAGES = `Get-AppxPackage -AllUsers | Select Name, PackageFullName, Architecture, Version  | ConvertTo-Json`
)

type powershellWinAppxPackages struct {
	Name         string `json:"Name"`
	FullName     string `json:"PackageFullName"`
	Architecture int    `json:"Architecture"`
	Version      string `json:"Version"`
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

		pkgs[i] = Package{
			Name:    appxPackages[i].Name,
			Version: appxPackages[i].Version,
			Arch:    arch,
			Format:  "windows/appx",
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
	provider os.OperatingSystemProvider
	platform *platform.Platform
}

func (w *WinPkgManager) Name() string {
	return "Windows Package Manager"
}

func (w *WinPkgManager) Format() string {
	return "win"
}

const installedAppsScript = `
Get-ItemProperty (@(
  'HKLM:\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Uninstall\\*',
  'HKCU:\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Uninstall\\*',
  'HKLM:\\SOFTWARE\\Wow6432Node\\Microsoft\\Windows\\CurrentVersion\\Uninstall\\*',
  'HKCU:\\SOFTWARE\\Wow6432Node\\Microsoft\\Windows\\CurrentVersion\\Uninstall\\*'
) | Where-Object { Test-Path $_ }) |
Select-Object -Property DisplayName,DisplayVersion,Publisher,EstimatedSize,InstallSource,UninstallString | ConvertTo-Json -Compress
`

// returns installed appx packages as well as hot fixes
func (w *WinPkgManager) List() ([]Package, error) {
	b, err := windows.Version(w.platform.Version)
	if err != nil {
		return nil, err
	}

	pkgs := []Package{}

	cmd, err := w.provider.RunCommand(powershell.Encode(installedAppsScript))
	if err != nil {
		return nil, fmt.Errorf("could not read app package list")
	}
	appPkgs, err := ParseWindowsAppPackages(cmd.Stdout)
	if err != nil {
		return nil, fmt.Errorf("could not read app package list")
	}
	pkgs = append(pkgs, appPkgs...)

	// only win 10+ are compatible with app x packages
	if b.Build > 10240 {
		cmd, err := w.provider.RunCommand(powershell.Wrap(WINDOWS_QUERY_APPX_PACKAGES))
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
	cmd, err = w.provider.RunCommand(powershell.Wrap(WINDOWS_QUERY_HOTFIXES))
	if err != nil {
		return nil, errors.Join(err, errors.New("could not fetch hotfixes"))
	}
	hotfixes, err := ParseWindowsHotfixes(cmd.Stdout)
	if err != nil {
		return nil, errors.Join(err, errors.New("could not parse hotfix results"))
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

	type pwershellUninstallEntry struct {
		DisplayName     string `json:"DisplayName"`
		DisplayVersion  string `json:"DisplayVersion"`
		Publisher       string `json:"Publisher"`
		InstallSource   string `json:"InstallSource"`
		EstimatedSize   int    `json:"EstimatedSize"`
		UninstallString string `json:"UninstallString"`
	}

	var entries []pwershellUninstallEntry
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
		pkgs = append(pkgs, Package{
			Name:    entry.DisplayName,
			Version: entry.DisplayVersion,
			Format:  "windows/app",
		})
	}

	return pkgs, nil
}

func (win *WinPkgManager) Available() (map[string]PackageUpdate, error) {
	return map[string]PackageUpdate{}, nil
}
