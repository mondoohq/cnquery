package platform

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"strconv"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/transports/fsutil"
)

type WmicOS struct {
	Node                                      string
	BootDevice                                string
	BuildNumber                               string
	BuildType                                 string
	Caption                                   string
	CodeSet                                   string
	CountryCode                               string
	CreationClassName                         string
	CSCreationClassName                       string
	CSDVersion                                string
	CSName                                    string
	CurrentTimeZone                           string
	DataExecutionPrevention_32BitApplications string
	DataExecutionPrevention_Available         string
	DataExecutionPrevention_Drivers           string
	DataExecutionPrevention_SupportPolicy     string
	Debug                                     string
	Description                               string
	Distributed                               string
	EncryptionLevel                           string
	ForegroundApplicationBoost                string
	FreePhysicalMemory                        string
	FreeSpaceInPagingFiles                    string
	FreeVirtualMemory                         string
	InstallDate                               string
	LargeSystemCache                          string
	LastBootUpTime                            string
	LocalDateTime                             string
	Locale                                    string
	Manufacturer                              string
	MaxNumberOfProcesses                      string
	MaxProcessMemorySize                      string
	MUILanguages                              string
	Name                                      string
	NumberOfLicensedUsers                     string
	NumberOfProcesses                         string
	NumberOfUsers                             string
	OperatingSystemSKU                        string
	Organization                              string
	OSArchitecture                            string
	OSLanguage                                string
	OSProductSuite                            string
	OSType                                    string
	OtherTypeDescription                      string
	PAEEnabled                                string
	PlusProductID                             string
	PlusVersionNumber                         string
	PortableOperatingSystem                   string
	Primary                                   string
	ProductType                               string
	RegisteredUser                            string
	SerialNumber                              string
	ServicePackMajorVersion                   string
	ServicePackMinorVersion                   string
	SizeStoredInPagingFiles                   string
	Status                                    string
	SuiteMask                                 string
	SystemDevice                              string
	SystemDirectory                           string
	SystemDrive                               string
	TotalSwapSpaceSize                        string
	TotalVirtualMemorySize                    string
	TotalVisibleMemorySize                    string
	Version                                   string
	WindowsDirectory                          string
}

func ParseWinWmicOS(csvData io.Reader) (*WmicOS, error) {
	reader := csv.NewReader(fsutil.NewLineFeedReader(csvData))
	os := []*WmicOS{}
	header := map[string]int{}

	i := -1 // to ensure counting starts at 0
	for {
		i++
		line, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Error().Err(err).Msg("could not read from wmic stream")
		}

		// store header index
		if i == 0 {
			for j, item := range line {
				header[item] = j
			}
			continue
		}

		os = append(os, &WmicOS{
			Node:           line[header["Node"]],
			Name:           line[header["Name"]],
			Caption:        line[header["Caption"]],
			Manufacturer:   line[header["Manufacturer"]],
			OSArchitecture: line[header["OSArchitecture"]],
			Version:        line[header["Version"]],
			BuildNumber:    line[header["BuildNumber"]],
			Description:    line[header["Description"]],
			OSType:         line[header["OSType"]],

			// 1 = Desktop OS
			// 2 = Server OS – Domain Controller
			// 3 = Server OS – Not a Domain Controller
			ProductType: line[header["ProductType"]],
		})

	}

	if len(os) == 1 {
		return os[0], nil
	} else {
		return nil, errors.New("could not parse wmic, retrieved unexpected amount of rows " + strconv.Itoa(len(os)))
	}
}

type WindowsCurrentVersion struct {
	CurrentBuild string `json:"CurrentBuild"`
	EditionID    string `json:"EditionID"`
	ReleaseId    string `json:"ReleaseId"`
	// Update Build Revision
	UBR int `json:"UBR"`
}

func ParseWinRegistryCurrentVersion(r io.Reader) (*WindowsCurrentVersion, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var winCurrentVersion WindowsCurrentVersion
	err = json.Unmarshal(data, &winCurrentVersion)
	if err != nil {
		return nil, err
	}

	return &winCurrentVersion, nil
}
