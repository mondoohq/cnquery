// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package sbom

import (
	"encoding/json"
	"errors"
	"os"
	"sigs.k8s.io/yaml"
)

type ReportCollectionJsonAsset struct {
	Mrn          string `json:"mrn"`
	Name         string `json:"name"`
	PlatformName string `json:"platform_name"`
}

type ReportCollectionJsonScore struct {
	Score  int    `json:"score"`
	Status string `json:"status"`
}

type ReportCollectionJson struct {
	Assets map[string]ReportCollectionJsonAsset            `json:"assets"`
	Data   map[string]map[string]json.RawMessage           `json:"data"`
	Scores map[string]map[string]ReportCollectionJsonScore `json:"scores"`
}

// Structures to parse the data from cnquery report
type BomAsset struct {
	Name     string            `json:"name,omitempty"`
	Platform string            `json:"platform,omitempty"`
	Version  string            `json:"version,omitempty"`
	Arch     string            `json:"arch,omitempty"`
	CPEs     []string          `json:"cpes.map,omitempty"`
	IDs      []string          `json:"ids,omitempty"`
	Labels   map[string]string `json:"labels,omitempty"`
}

type BomPackage struct {
	Name    string   `json:"name,omitempty"`
	Version string   `json:"version,omitempty"`
	Origin  string   `json:"origin,omitempty"`
	Arch    string   `json:"arch,omitempty"`
	Format  string   `json:"format,omitempty"`
	Purl    string   `json:"purl,omitempty"`
	CPEs    []string `json:"cpes.map,omitempty"`
	// used by python packages
	// deprecated: remove once python.packages uses files
	FilePath string `json:"file.path,omitempty"`
	// used by os packages
	FilePaths []string `json:"files.map,omitempty"`
}

type BomReport struct {
	Asset          *BomAsset    `json:"asset,omitempty"`
	Packages       []BomPackage `json:"packages.list,omitempty"`
	PythonPackages []BomPackage `json:"python.packages,omitempty"`
	NpmPackages    []BomPackage `json:"npm.packages.list,omitempty"`
}

func (b *BomReport) ToJSON() ([]byte, error) {
	return json.Marshal(b)
}

// AssetMrn returns the MRN of the asset if there is only one
func (r ReportCollectionJson) AssetMrn() (string, error) {
	if len(r.Assets) > 1 {
		return "", errors.New("report contains more than one asset")
	}

	if len(r.Assets) == 0 {
		return "", errors.New("report contains no assets")
	}

	for _, asset := range r.Assets {
		return asset.Mrn, nil
	}

	// should not happen
	return "", errors.New("report contains no assets")
}

// NewReportCollectionJsonFromSingleFile loads a cnspec report bundle from a single file
func NewReportCollectionJsonFromSingleFile(path string) (*ReportCollectionJson, error) {
	reportData, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return NewReportCollectionJson(reportData)
}

// NewReportCollectionJson creates a cnspec report from json contents
func NewReportCollectionJson(data []byte) (*ReportCollectionJson, error) {
	var res ReportCollectionJson
	err := yaml.Unmarshal(data, &res)
	return &res, err
}

func (p *ReportCollectionJson) ToYAML() ([]byte, error) {
	return yaml.Marshal(p)
}
