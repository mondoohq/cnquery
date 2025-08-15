// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v11/providers/os/detector/windows"
	"go.mondoo.com/cnquery/v11/providers/os/registry"
	"go.mondoo.com/cnquery/v11/providers/os/resources/cpe"
	"go.mondoo.com/cnquery/v11/providers/os/resources/powershell"
	"go.mondoo.com/cnquery/v11/providers/os/resources/purl"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
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
		"5c9376ab-8ce6-464a-b136-22113dd69801": Application,
		"434de588-ed14-48f5-8eed-a15e09a991f6": Connectors,
		"e6cf1350-c01b-414d-a61f-263d14d133b4": CriticalUpdates,
		"e0789628-ce08-4437-be74-2495b842f43b": DefinitionUpdates,
		"e140075d-8433-45c3-ad87-e72345b3607":  DeveloperKits,
		"b54e7d24-7add-428f-8b75-90a396fa584f": FeaturePacks,
		"9511D615-35B2-47BB-927F-F73D8E9260BB": Guidance,
		"0fa1201d-4330-4fa8-8ae9-b877473b6441": SecurityUpdates,
		"68c5b0a3-d1a6-4553-ae49-01d3a7827828": ServicePacks,
		"b4832bd8-e735-4761-8daf-37f882276dab": Tools,
		"28bc880e-0592-4cbf-8f95-c79b17911d5f": UpdateRollups,
		"cd5ffd1e-e932-4e3a-bf74-18bf0b1bbd83": Updates,
		"ebfc1fc5-71a4-4f7b-9aca-3b9a503104a0": Drivers,
		"8c3fcc84-7410-4a95-8b89-a166a0190486": Defender,
	}

	appxArchitecture = map[int]string{
		WinArchNeutral:    "neutral",
		WinArchX86:        "x86",
		WinArchX64:        "x64",
		WinArchArm64:      "arm64",
		WinArchArm:        "arm",
		WinArchX86OnArm64: "x86onarm",
	}

	sqlGDRUpdateRegExp = regexp.MustCompile(`^GDR \d+ .+ SQL Server \d+ \(KB\d+\)`)
	sqlHotfixRegExp    = regexp.MustCompile(`^Hotfix .+ SQL Server`)
	// Find the database engine package and use version as a reference for the update
	msSqlServiceRegexp = regexp.MustCompile(`^SQL Server \d+ Database Engine Services$`)
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
Select-Object -Property DisplayName,DisplayVersion,Publisher,EstimatedSize,InstallSource,UninstallString,InstallLocation | ConvertTo-Json -Compress
`

var (
	WINDOWS_QUERY_HOTFIXES      = `Get-HotFix | Select-Object -Property Status, Description, HotFixId, Caption, InstalledOn, InstalledBy | ConvertTo-Json`
	WINDOWS_QUERY_APPX_PACKAGES = `Get-AppxPackage -AllUsers | Select Name, PackageFullName, Architecture, Version, Publisher, InstallLocation | ConvertTo-Json`
)

type winAppxPackages struct {
	Name            string `json:"Name"`
	FullName        string `json:"PackageFullName"`
	Architecture    int    `json:"Architecture"`
	Version         string `json:"Version"`
	Publisher       string `json:"Publisher"`
	InstallLocation string `json:"InstallLocation"`
	// can directly set it to the architecture string, the pwsh script returns it as int (Architecture)
	arch string `json:"-"`
}

func (p winAppxPackages) toPackage(platform *inventory.Platform) Package {
	if p.arch == "" {
		arch, ok := appxArchitecture[p.Architecture]
		if !ok {
			log.Warn().Int("arch", p.Architecture).Msg("unknown architecture value for windows appx package")
			arch = "unknown"
		}
		p.arch = arch
	}

	pkg := createPackage(p.Name, p.Version, "windows/appx", p.arch, p.Publisher, p.InstallLocation, platform)

	return *pkg
}

// Good read: https://www.wintips.org/view-installed-apps-and-packages-in-windows-10-8-1-8-from-powershell/
func ParseWindowsAppxPackages(platform *inventory.Platform, input io.Reader) ([]Package, error) {
	data, err := io.ReadAll(input)
	if err != nil {
		return nil, err
	}

	var appxPackages []winAppxPackages

	// handle case where no packages are installed
	if len(data) == 0 {
		return []Package{}, nil
	}

	err = json.Unmarshal(data, &appxPackages)
	if err != nil {
		return nil, err
	}

	pkgs := make([]Package, len(appxPackages))
	for i, p := range appxPackages {
		if p.Name == "" {
			continue
		}
		pkgs[i] = p.toPackage(platform)
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
			p, err := getPackageFromRegistryKey(c, w.platform)
			if err != nil {
				return nil, err
			}
			if p == nil {
				continue
			}
			packages = append(packages, *p)
		}
	}

	// These are the .NET Framework packages
	// They do not show up in the general apps or features list, so we need to discover them separately
	dotNetFramework, err := w.getDotNetFramework()
	if err != nil {
		log.Debug().Err(err).Msg("could not get .NET Framework packages from registry")
	} else {
		packages = append(packages, dotNetFramework...)
	}
	return packages, nil
}

// getDotNetFramework returns the .NET Framework package
func (w *WinPkgManager) getDotNetFramework() ([]Package, error) {
	// https://learn.microsoft.com/en-us/dotnet/framework/install/how-to-determine-which-versions-are-installed#net-framework-45-and-later-versions
	dotNet45plus := "HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\NET Framework Setup\\NDP\\v4\\Full"
	// https://learn.microsoft.com/en-us/dotnet/framework/install/how-to-determine-which-versions-are-installed#use-registry-editor-older-framework-versions
	dotNet35 := "HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\NET Framework Setup\\NDP\\v3.5"

	return getDotNetFrameworkPackageFromRegistryKeys(dotNet45plus, dotNet35, w.platform)
}

// getDotNetFrameworkFs returns the .NET Framework package discovered on the filesystem
func (w *WinPkgManager) getDotNetFrameworkFs() ([]Package, error) {
	// https://learn.microsoft.com/en-us/dotnet/framework/install/how-to-determine-which-versions-are-installed#net-framework-45-and-later-versions
	dotNet45plus := "Microsoft\\NET Framework Setup\\NDP\\v4\\Full"
	// https://learn.microsoft.com/en-us/dotnet/framework/install/how-to-determine-which-versions-are-installed#use-registry-editor-older-framework-versions
	dotNet35 := "Microsoft\\NET Framework Setup\\NDP\\v3.5"

	return getDotNetFrameworkPackageFromRegistryKeys(dotNet45plus, dotNet35, w.platform)
}

// getDotNetFrameworkPackageFromRegistryKeys returns the .NET Framework package from the registry keys
func getDotNetFrameworkPackageFromRegistryKeys(dotNet45plus, dotNet35 string, platform *inventory.Platform) ([]Package, error) {
	items, err := registry.GetNativeRegistryKeyItems(dotNet45plus)
	if err != nil && status.Code(err) != codes.NotFound {
		return nil, err
	}

	if len(items) == 0 {
		items, err = registry.GetNativeRegistryKeyItems(dotNet35)
		if err != nil && status.Code(err) != codes.NotFound {
			return nil, err
		}
	}

	if len(items) == 0 {
		return nil, nil
	}

	p := getDotNetFrameworkPackageFromRegistryKeyItems(items, platform)
	if p == nil {
		return nil, nil
	}
	return []Package{*p}, nil
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

	return ParseWindowsAppPackages(w.platform, cmd.Stdout)
}

func (w *WinPkgManager) getAppxPackages() ([]Package, error) {
	canRunCmd := w.conn.Capabilities().Has(shared.Capability_RunCommand)
	// we always prefer to use the powershell command to get the appx packages, fallback to filesystem if not possible
	if !canRunCmd && (w.conn.Type() == shared.Type_FileSystem || w.conn.Type() == shared.Type_Device) {
		return w.getFsAppxPackages()
	}

	b, err := windows.Version(w.platform.Version)
	if err != nil {
		return nil, err
	}

	// only win 10+ are compatible with app x packages
	if b.Build > 10240 {
		return w.getPwshAppxPackages()
	}

	return []Package{}, nil
}

func (w *WinPkgManager) getPwshAppxPackages() ([]Package, error) {
	cmd, err := w.conn.RunCommand(powershell.Wrap(WINDOWS_QUERY_APPX_PACKAGES))
	if err != nil {
		return nil, fmt.Errorf("could not read appx package list")
	}
	return ParseWindowsAppxPackages(w.platform, cmd.Stdout)
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
			p, err := getPackageFromRegistryKey(c, w.platform)
			if err != nil {
				return nil, err
			}
			if p == nil {
				continue
			}
			packages = append(packages, *p)
		}
	}

	// These are the .NET Framework packages
	// They do not show up in the general apps or features list, so we need to discover them separately
	dotNetFramework, err := w.getDotNetFrameworkFs()
	if err != nil {
		log.Debug().Err(err).Msg("could not get .NET Framework packages from filesystem")
	} else {
		packages = append(packages, dotNetFramework...)
	}
	msSqlHotfixes := findMsSqlHotfixes(packages)
	if len(msSqlHotfixes) > 0 {
		packages = updateMsSqlPackages(packages, msSqlHotfixes[len(msSqlHotfixes)-1])
	}

	return packages, nil
}

func (w *WinPkgManager) getFsAppxPackages() ([]Package, error) {
	if !w.conn.Capabilities().Has(shared.Capability_FindFile) {
		return nil, errors.New("find file is not supported for your platform")
	}
	fs := w.conn.FileSystem()
	fsSearch, ok := fs.(shared.FileSearch)
	if !ok {
		return nil, errors.New("find file is not supported for your platform")
	}

	paths := map[string]int{
		filepath.Join("Windows", "SystemApps"):        1,
		filepath.Join("Program Files", "WindowsApps"): 1,
		"Windows": 1,
	}
	appxPaths := map[string]struct{}{}
	for p, depth := range paths {
		res, err := fsSearch.Find(p, regexp.MustCompile(".*/[Aa]ppx[Mm]anifest.xml"), "f", nil, &depth)
		if err != nil {
			continue
		}
		for _, r := range res {
			appxPaths[r] = struct{}{}
		}
	}
	log.Debug().Int("amount", len(appxPaths)).Msg("found appx manifest files")

	pkgs := []Package{}
	afs := &afero.Afero{Fs: fs}
	for p := range appxPaths {
		res, err := afs.ReadFile(p)
		if err != nil {
			log.Debug().Err(err).Str("path", p).Msg("could not read appx manifest")
			continue
		}
		winAppxPkg, err := parseAppxManifest(res)
		if err != nil {
			log.Debug().Err(err).Str("path", p).Msg("could not parse appx manifest")
			continue
		}
		if winAppxPkg.Name == "" {
			continue
		}
		pkg := winAppxPkg.toPackage(w.platform)
		pkgs = append(pkgs, pkg)

	}
	return pkgs, nil
}

func parseAppxManifest(input []byte) (winAppxPackages, error) {
	manifest := &AppxManifest{}
	err := xml.Unmarshal(input, manifest)
	if err != nil {
		return winAppxPackages{}, err
	}
	pkg := winAppxPackages{
		Name:      manifest.Identity.Name,
		Version:   manifest.Identity.Version,
		Publisher: manifest.Identity.Publisher,
		arch:      manifest.Identity.ProcessorArchitecture,
	}
	return pkg, nil
}

func getPackageFromRegistryKey(key registry.RegistryKeyChild, platform *inventory.Platform) (*Package, error) {
	items, err := registry.GetNativeRegistryKeyItems(key.Path + "\\" + key.Name)
	if err != nil {
		log.Debug().Err(err).Str("path", key.Path).Msg("could not read registry key children")
		return nil, err
	}
	return getPackageFromRegistryKeyItems(items, platform), nil
}

func getPackageFromRegistryKeyItems(children []registry.RegistryKeyItem, platform *inventory.Platform) *Package {
	var uninstallString string
	var displayName string
	var displayVersion string
	var publisher string
	var installLocation string

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
		case "InstallLocation":
			installLocation = i.Value.String
		}
	}

	if uninstallString == "" {
		return nil
	}

	// TODO: We need to figure out why we have empty displayNames.
	// this is common in windows but we need to verify it is a windows
	// issue and not a cnquery issue.
	if displayName == "" {
		log.Debug().Msg("ignored package since display name is missing")
		return nil
	}

	pkg := createPackage(displayName, displayVersion, "windows/app", platform.Arch, publisher, installLocation, platform)

	return pkg
}

// getDotNetFrameworkPackageFromRegistryKeyItems returns the .NET Framework package from the registry key items
func getDotNetFrameworkPackageFromRegistryKeyItems(items []registry.RegistryKeyItem, platform *inventory.Platform) *Package {
	var version string
	var installLocation string

	for _, i := range items {
		switch i.Key {
		case "Version":
			version = i.Value.String
		case "InstallLocation":
			installLocation = i.Value.String
		}
	}

	if version == "" {
		return nil
	}

	pkg := createPackage("Microsoft .NET Framework", version, "windows/app", platform.Arch, "Microsoft", installLocation, platform)

	return pkg
}

// returns installed appx packages as well as hot fixes
func (w *WinPkgManager) List() ([]Package, error) {
	pkgs := []Package{}
	appPkgs, err := w.getInstalledApps()
	if err != nil {
		return nil, errors.Wrap(err, "could not read app package list")
	}
	pkgs = append(pkgs, appPkgs...)

	appxPackages, err := w.getAppxPackages()
	if err != nil {
		return nil, errors.Wrap(err, "could not read appx package list")
	}
	pkgs = append(pkgs, appxPackages...)

	canRunCmd := w.conn.Capabilities().Has(shared.Capability_RunCommand)
	if !canRunCmd {
		log.Debug().Msg("cannot run command on windows, skipping hotfixes list")
		return pkgs, nil
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

	msSqlHotfixes := findMsSqlHotfixes(appPkgs)
	msSqlGdrPackages := findMsSqlGdrUpdates(appPkgs)
	// MS only allows GDR or Hotfixes/CU, no need to check which one takes precedence
	if len(msSqlGdrPackages) > 0 {
		pkgs = updateMsSqlPackages(pkgs, msSqlGdrPackages[len(msSqlGdrPackages)-1])
	}
	if len(msSqlHotfixes) > 0 {
		pkgs = updateMsSqlPackages(pkgs, msSqlHotfixes[len(msSqlHotfixes)-1])
	}

	pkgs = append(pkgs, hotfixAsPkgs...)
	return pkgs, nil
}

func ParseWindowsAppPackages(platform *inventory.Platform, input io.Reader) ([]Package, error) {
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
		InstallLocation string `json:"InstallLocation"`
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

		// TODO: We need to figure out why we have empty displayNames.
		// this is common in windows but we need to verify it is a windows
		// issue and not a cnquery issue.
		if entry.DisplayName == "" {
			continue
		}
		pkg := createPackage(entry.DisplayName, entry.DisplayVersion, "windows/app", platform.Arch, entry.Publisher, entry.InstallLocation, platform)
		pkgs = append(pkgs, *pkg)
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

// findMsSqlHotfixes returns a list of hotfixes that are related to Microsoft SQL Server
// The list is sorted by the hotfix id
func findMsSqlHotfixes(packages []Package) []Package {
	sqlHotfixes := []Package{}
	for _, p := range packages {
		if sqlHotfixRegExp.MatchString(p.Name) {
			sqlHotfixes = append(sqlHotfixes, p)
		}
	}
	slices.SortFunc(sqlHotfixes, func(a, b Package) int {
		return strings.Compare(a.Version, b.Version)
	})
	return sqlHotfixes
}

// findMsSqlGdrUpdates returns a list of GDR updates that are related to Microsoft SQL Server
// The list is sorted by the GDR update id
func findMsSqlGdrUpdates(packages []Package) []Package {
	sqlGdrUpdates := []Package{}
	for _, p := range packages {
		if sqlGDRUpdateRegExp.MatchString(p.Name) {
			sqlGdrUpdates = append(sqlGdrUpdates, p)
		}
	}
	slices.SortFunc(sqlGdrUpdates, func(a, b Package) int {
		return strings.Compare(a.Version, b.Version)
	})
	return sqlGdrUpdates
}

// updateMsSqlPackages updates the version of the SQL Server packages to the latest hotfix version
func updateMsSqlPackages(pkgs []Package, latestMsSqlUpdate Package) []Package {
	currentVersion := ""
	for _, pkg := range pkgs {
		if msSqlServiceRegexp.MatchString(pkg.Name) {
			currentVersion = pkg.Version
			break
		}
	}
	log.Debug().Str("currentVersion", currentVersion).Msg("Updating SQL Server packages")

	// Find other SQL Server packages and update them to the latest hotfix version
	for i, pkg := range pkgs {
		if strings.Contains(pkg.Name, "SQL Server") && pkg.Version == currentVersion {
			pkgs[i].Version = latestMsSqlUpdate.Version
			log.Debug().Str("package", pkg.Name).Str("version", latestMsSqlUpdate.Version).Msg("Updated SQL Server package")
			pkgs[i].PUrl = strings.Replace(pkgs[i].PUrl, currentVersion, latestMsSqlUpdate.Version, 1)
		}
	}
	return pkgs
}

// createPackage creates a new package with the given parameters
func createPackage(name, version, format, arch, publisher, installLocation string, platform *inventory.Platform) *Package {
	purlType := purl.TypeWindows
	if format == "windows/appx" {
		purlType = purl.TypeAppx
	}

	pkg := &Package{
		Name:    name,
		Version: version,
		Format:  format,
		Arch:    arch,
		Vendor:  publisher,
		PUrl: purl.NewPackageURL(
			platform, purlType, name, version,
		).String(),
	}
	if installLocation != "" {
		pkg.Files = []FileRecord{
			{
				Path: installLocation,
			},
		}
		pkg.FilesAvailable = PkgFilesIncluded
	}

	if version != "" {
		cpeWfns, err := cpe.NewPackage2Cpe(publisher, name, version, "", "")
		if err != nil {
			log.Debug().Err(err).Str("name", name).Str("version", version).Msg("could not create cpe for windows app package")
		} else {
			pkg.CPEs = cpeWfns
		}
	}

	return pkg
}
