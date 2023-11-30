package gql

import (
	"context"

	mondoogql "go.mondoo.com/mondoo-go"
)

// LastAssessment fetches the las update time of the packages query
// This is also the lst time the vuln report was updated
func (c *MondooClient) LastAssessment(mrn string) (string, error) {
	var m struct {
		AssetLastPackageUpdateTime struct {
			LastUpdated string
		} `graphql:"assetLastPackageUpdateTime(input: $input)"`
	}
	err := c.Query(context.Background(), &m, map[string]interface{}{"input": mondoogql.AssetLastPackageUpdateTimeInput{Mrn: mondoogql.String(mrn)}})
	if err != nil {
		return "", err
	}
	return m.AssetLastPackageUpdateTime.LastUpdated, nil
}

type VulnReport struct {
	AssetMrn   string
	Advisories []*Advisory
	Cves       []*Cve
	Packages   []*Package
}

type Cve struct {
	Id     string
	Source struct {
		Id   string
		Name string
		Url  string
	}
	Title       string
	Description string
	Summary     string
	PublishedAt string
	ModifiedAt  string
	Url         string
	CvssScore   struct {
		Id     string
		Value  int
		Type   int
		Vector string
		Source string
	}
	CvssScores []struct {
		Id     string
		Value  int
		Type   int
		Vector string
		Source string
	}
	Cwe   string
	State string
}

type Advisory struct {
	Id     string
	Source struct {
		Id   string
		Name string
		Url  string
	}
	Title       string
	Description string

	Cves []struct {
		Cve
	}
	CvssScore struct {
		Id     string
		Value  int
		Type   int
		Vector string
		Source string
	}
	Vendorscore int
	PublishedAt string
	ModifiedAt  string
}

type Package struct {
	Id      string
	Name    string
	Version string
	Arch    string
	Format  string

	Namespace   string
	Description string
	Status      string
	Available   string
	Origin      string

	Score struct {
		Id     string
		Value  int
		Type   int
		Vector string
		Source string
	}

	Advisories []struct {
		Advisory
	}
	Cves []struct {
		Cve
	}
}

// GetVulnReport fetches the vuln report for a given asset
func (c *MondooClient) GetVulnReport(mrn string) (*VulnReport, error) {
	var m struct {
		AssetVulnerabilityReportResponse struct {
			AssetVulnerabilityCompactReport struct {
				AssetMrn   string
				Advisories []struct {
					Advisory
				}
				Cves []struct {
					Cve
				}
				Packages []struct {
					Package
				}
			} `graphql:"... on AssetVulnerabilityCompactReport"`
		} `graphql:"assetVulnerabilityCompactReport(input: $input)"`
	}
	err := c.Query(context.Background(), &m, map[string]interface{}{"input": mondoogql.AssetVulnerabilityReportInput{AssetMrn: mondoogql.String(mrn)}})
	if err != nil {
		return nil, err
	}

	gqlVulnReport := &VulnReport{
		AssetMrn:   m.AssetVulnerabilityReportResponse.AssetVulnerabilityCompactReport.AssetMrn,
		Advisories: make([]*Advisory, len(m.AssetVulnerabilityReportResponse.AssetVulnerabilityCompactReport.Advisories)),
		Cves:       make([]*Cve, len(m.AssetVulnerabilityReportResponse.AssetVulnerabilityCompactReport.Cves)),
		Packages:   make([]*Package, len(m.AssetVulnerabilityReportResponse.AssetVulnerabilityCompactReport.Packages)),
	}

	for i, a := range m.AssetVulnerabilityReportResponse.AssetVulnerabilityCompactReport.Advisories {
		gqlVulnReport.Advisories[i] = &a.Advisory
	}

	for i, c := range m.AssetVulnerabilityReportResponse.AssetVulnerabilityCompactReport.Cves {
		gqlVulnReport.Cves[i] = &c.Cve
	}

	for i, p := range m.AssetVulnerabilityReportResponse.AssetVulnerabilityCompactReport.Packages {
		gqlVulnReport.Packages[i] = &p.Package
	}

	return gqlVulnReport, nil
}

// GetIncognitoVulnReport fetches the vuln report for an anonymous asset
// This is a special case were we don't have an MRN, like in cnspec shell
func (c *MondooClient) GetIncognitoVulnReport(platform mondoogql.PlatformInput, pkgs []mondoogql.PackageInput) (*VulnReport, error) {
	var m struct {
		AssetVulnerabilityReportResponse struct {
			AssetIncognitoVulnerabilityReport struct {
				Advisories []struct {
					Advisory
				}
				Cves []struct {
					Cve
				}
				Packages []struct {
					Package
				}
			} `graphql:"... on AssetIncognitoVulnerabilityReport"`
		} `graphql:"analyseIncognitoAssetVulnerabilities(input: $input)"`
	}
	gqlInput := mondoogql.AnalyseIncognitoAssetInput{
		Platform: platform,
		Packages: pkgs,
	}

	err := c.Query(context.Background(), &m, map[string]interface{}{"input": gqlInput})
	if err != nil {
		return nil, err
	}

	gqlVulnReport := &VulnReport{
		Advisories: make([]*Advisory, len(m.AssetVulnerabilityReportResponse.AssetIncognitoVulnerabilityReport.Advisories)),
		Cves:       make([]*Cve, len(m.AssetVulnerabilityReportResponse.AssetIncognitoVulnerabilityReport.Cves)),
		Packages:   make([]*Package, len(m.AssetVulnerabilityReportResponse.AssetIncognitoVulnerabilityReport.Packages)),
	}

	for i, a := range m.AssetVulnerabilityReportResponse.AssetIncognitoVulnerabilityReport.Advisories {
		gqlVulnReport.Advisories[i] = &a.Advisory
	}

	for i, c := range m.AssetVulnerabilityReportResponse.AssetIncognitoVulnerabilityReport.Cves {
		gqlVulnReport.Cves[i] = &c.Cve
	}

	for i, p := range m.AssetVulnerabilityReportResponse.AssetIncognitoVulnerabilityReport.Packages {
		gqlVulnReport.Packages[i] = &p.Package
	}

	return gqlVulnReport, nil
}
