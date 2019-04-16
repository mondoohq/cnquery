package packages

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/rs/zerolog/log"
)

// base64 encoding for long powershell script
func EncodePowershell(cmd string) string {

	// powershall uses two bytes chars :-(
	withSpaceCmd := ""
	for _, b := range []byte(cmd) {
		withSpaceCmd += string(b) + "\x00"
	}

	// encode the command as base64
	input := []uint8(withSpaceCmd)
	return fmt.Sprintf("powershell.exe -EncodedCommand %s", base64.StdEncoding.EncodeToString(input))
}

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
		11: "noarch",
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

var WINDOWS_QUERY_APPX_PACKAGES = `Get-AppxPackage -AllUsers | Select Name, PackageFullName, Architecture, Version  | ConvertTo-Json`

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
		}
	}
	return pkgs, nil
}

var WINDOWS_QUERY_WSUS_AVAILABLE = `
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
@($updates) | ConvertTo-Json
	`

type powershellWinUpdate struct {
	UpdateID    string   `json:"UpdateID"`
	Title       string   `json:"Title"`
	CategoryIDs []string `json:"CategoryIDs"`
}

func ParseWindowsUpdates(input io.Reader) ([]OperatingSystemUpdate, error) {
	data, err := ioutil.ReadAll(input)
	if err != nil {
		return nil, err
	}

	var powerShellUpdates []powershellWinUpdate
	err = json.Unmarshal(data, &powerShellUpdates)
	if err != nil {
		return nil, err
	}

	updates := make([]OperatingSystemUpdate, len(powerShellUpdates))
	for i := range powerShellUpdates {
		updates[i] = OperatingSystemUpdate{
			Name:        powerShellUpdates[i].UpdateID,
			Description: powerShellUpdates[i].Title,
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
		}
	}
	return pkgs, nil
}
