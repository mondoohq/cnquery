package packages

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi/resources/powershell"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/platform/winbuild"
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
	WINDOWS_QUERY_HOTFIXES       = `Get-HotFix | Select-Object -Property Status, Description, HotFixId, Caption, InstallDate, InstalledBy | ConvertTo-Json -AsArray`
	WINDOWS_QUERY_APPX_PACKAGES  = `Get-AppxPackage -AllUsers | Select Name, PackageFullName, Architecture, Version  | ConvertTo-Json -AsArray`
	WINDOWS_QUERY_WSUS_AVAILABLE = `
$ProgressPreference='SilentlyContinue';
$updateSession = new-object -com "Microsoft.Update.Session"
$searcher=$updateSession.CreateupdateSearcher().Search(("IsInstalled=0 and Type='Software'"))
$updates = $searcher.Updates | ForEach-Object {
	$update = $_
	$value = New-Object psobject -Property @{
		"UpdateID" =  $update.Identity.UpdateID;
		"Title" = $update.Title
		"MsrcSeverity" = $update.MsrcSeverity
		"RevisionNumber" =  $update.Identity.RevisionNumber;
		"CategoryIDs" = @($update.Categories | % { $_.CategoryID })
		"SecurityBulletinIDs" = $update.SecurityBulletinIDs
		"RebootRequired" = $update.RebootRequired
		"KBArticleIDs" = $update.KBArticleIDs
		"CveIDs" = @($update.CveIDs)
	}
	$value
}
@($updates) | ConvertTo-Json`
)

type powershellWinAppxPackages struct {
	Name         string `json:"Name"`
	FullName     string `json:"PackageFullName"`
	Architecture int    `json:"Architecture"`
	Version      string `json:"Version"`
}

// Good read: https://www.wintips.org/view-installed-apps-and-packages-in-windows-10-8-1-8-from-powershell/
func ParseWindowsAppxPackages(input io.Reader) ([]Package, error) {
	data, err := ioutil.ReadAll(input)
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

type powershellWinUpdate struct {
	UpdateID     string   `json:"UpdateID"`
	Title        string   `json:"Title"`
	CategoryIDs  []string `json:"CategoryIDs"`
	KBArticleIDs []string `json:"KBArticleIDs"`
}

func ParseWindowsUpdates(input io.Reader) ([]Package, error) {
	data, err := ioutil.ReadAll(input)
	if err != nil {
		return nil, err
	}

	// handle case where no packages are installed
	if len(data) == 0 {
		return []Package{}, nil
	}

	var powerShellUpdates []powershellWinUpdate
	err = json.Unmarshal(data, &powerShellUpdates)
	if err != nil {
		return nil, err
	}

	updates := make([]Package, len(powerShellUpdates))
	for i := range powerShellUpdates {
		if len(powerShellUpdates[i].KBArticleIDs) == 0 {
			log.Warn().Str("update", powerShellUpdates[i].UpdateID).Msg("ms update has no kb assigned")
			continue
		}

		// todo: we may want to make that decision server-side, since it does not require us to update the agent
		// therefore we need additional information to be transmitted via the packages eg. labels
		// important := false
		// for ci := range powerShellUpdates[i].CategoryIDs {
		// 	id := powerShellUpdates[i].CategoryIDs[ci]
		// 	classification := wsusClassificationGUID[strings.ToLower(id)]
		// 	if classification == CriticalUpdates || classification == SecurityUpdates || classification == UpdateRollups {
		// 		important = true
		// 	}
		// }

		updates[i] = Package{
			Name:        powerShellUpdates[i].KBArticleIDs[0],
			Version:     powerShellUpdates[i].UpdateID,
			Description: powerShellUpdates[i].Title,
			Format:      "windows/updates",
		}
	}
	return updates, nil
}

type powershellWinHotFix struct {
	Status      string `json:"Status"`
	Description string `json:"Description"`
	HotFixId    string `json:"HotFixId"`
	Caption     string `json:"Caption"`
	InstallDate string `json:"InstallDate"`
	InstalledBy string `json:"InstalledBy"`
}

func ParseWindowsHotfixes(input io.Reader) ([]Package, error) {
	data, err := ioutil.ReadAll(input)
	if err != nil {
		return nil, err
	}

	// for empty result set do not get the '{}', therefore lets abort here
	if len(data) == 0 {
		return []Package{}, nil
	}

	var powershellWinHotFixPkgs []powershellWinHotFix
	err = json.Unmarshal(data, &powershellWinHotFixPkgs)
	if err != nil {
		return nil, err
	}

	pkgs := make([]Package, len(powershellWinHotFixPkgs))
	for i := range powershellWinHotFixPkgs {
		pkgs[i] = Package{
			Name:        powershellWinHotFixPkgs[i].HotFixId,
			Description: powershellWinHotFixPkgs[i].Description,
			Format:      "windows/hotfix",
		}
	}
	return pkgs, nil
}

type WinPkgManager struct {
	motor *motor.Motor
}

func (win *WinPkgManager) Name() string {
	return "Windows Package Manager"
}

func (win *WinPkgManager) Format() string {
	return "win"
}

// returns installed appx packages as well as hot fixes
func (win *WinPkgManager) List() ([]Package, error) {

	pf, err := win.motor.Platform()
	if err != nil {
		return nil, err
	}

	b, err := winbuild.Version(pf.Release)

	pkgs := []Package{}

	// only win 10+ are compatible with app x packages
	if b.Build > 10240 {
		cmd, err := win.motor.Transport.RunCommand(powershell.Wrap(WINDOWS_QUERY_APPX_PACKAGES))
		if err != nil {
			return nil, fmt.Errorf("could not read package list")
		}
		appxPkgs, err := ParseWindowsAppxPackages(cmd.Stdout)
		if err != nil {
			return nil, fmt.Errorf("could not read appx package list")
		}
		pkgs = append(pkgs, appxPkgs...)
	}

	// try to read wsus updates
	wsusCmd, err := win.motor.Transport.RunCommand(powershell.Encode(WINDOWS_QUERY_WSUS_AVAILABLE))
	if err == nil {
		wsusUpdates, err := ParseWindowsUpdates(wsusCmd.Stdout)
		if err == nil {
			pkgs = append(pkgs, wsusUpdates...)
		} else {
			log.Warn().Err(err).Msg("could not parse wsus results")
		}
	} else {
		log.Warn().Err(err).Msg("could not fetch windows update services")
	}

	cmd, err := win.motor.Transport.RunCommand(powershell.Wrap(WINDOWS_QUERY_HOTFIXES))
	if err != nil {
		return nil, errors.Wrap(err, "could not fetch hotfixes")
	}
	hotfixes, err := ParseWindowsHotfixes(cmd.Stdout)
	if err != nil {
		return nil, errors.Wrapf(err, "could not parse hotfix results")
	}
	pkgs = append(pkgs, hotfixes...)

	return pkgs, nil
}

func (win *WinPkgManager) Available() (map[string]PackageUpdate, error) {
	return map[string]PackageUpdate{}, nil
}
