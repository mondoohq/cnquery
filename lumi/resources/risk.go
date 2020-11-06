package resources

import (
	"context"
	"errors"
	"strconv"
	"time"

	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/nexus/assets"
	"go.mondoo.io/mondoo/nexus/scanner"
	"go.mondoo.io/mondoo/nexus/scanner/scannerclient"
	"go.mondoo.io/mondoo/vadvisor/api"
)

func (c *lumiCvss) id() (string, error) {
	// TODO: use c.Vector() once we have the data available
	score, _ := c.Score()
	return "cvss/" + strconv.FormatFloat(score, 'f', 2, 64), nil
}

func (c *lumiRiskAdvisory) id() (string, error) {
	return c.Mrn()
}

func (a *lumiPlatformAdvisories) id() (string, error) {
	return "platform.advisories", nil
}

func getScannerClient(m *motor.Motor) (string, scannerclient.Client, error) {
	mcc := m.CloudConfig()

	if mcc == nil {
		return "", nil, errors.New("mondoo upstream configuration is missing")
	}

	// start scanner client
	cl, err := scannerclient.New(mcc.Collector, mcc.ApiEndpoint, mcc.Plugins, false, mcc.Incognito)
	return mcc.SpaceMrn, cl, err
}

// fetches the vulnerability report and caches it
func (a *lumiPlatformAdvisories) getAdvisoryReport() (*scanner.VulnReport, error) {
	// check if the data is cached
	r, ok := a.LumiResource().Cache.Load("_report")
	if ok {
		report := r.Data.(*scanner.VulnReport)
		return report, nil
	}

	// get new advisory report
	spaceMrn, scannerClient, err := getScannerClient(a.Runtime.Motor)
	if err != nil {
		return nil, err
	}

	// get platform information
	obj, err := a.Runtime.CreateResource("platform")
	if err != nil {
		return nil, err
	}

	platform := obj.(Platform)

	name, _ := platform.Name()
	release, _ := platform.Release()
	arch, _ := platform.Arch()

	// TODO: get the asset and basis platfrom via a new asset resource so that we can also send the mrn
	asset := &assets.Asset{
		Mrn:      "", // TODO: get the asset mrn from motor
		SpaceMrn: spaceMrn,
		Platform: &assets.Platform{
			Name:    name,
			Release: release,
			Arch:    arch,
		},
	}

	apiPackages := []*api.Package{}

	obj, err = a.Runtime.CreateResource("packages")
	if err != nil {
		return nil, err
	}
	packages := obj.(Packages)

	lumiPkgs, err := packages.List()
	if err != nil {
		return nil, err
	}

	for i := range lumiPkgs {
		lumiPkg := lumiPkgs[i]
		pkg := lumiPkg.(Package)
		name, _ := pkg.Name()
		version, _ := pkg.Version()
		arch, _ := pkg.Arch()
		format, _ := pkg.Format()

		apiPackages = append(apiPackages, &api.Package{
			Name:    name,
			Version: version,
			Arch:    arch,
			Format:  format,
		})
	}

	reportBinder, err := scannerClient.AnalysePlatform(context.Background(), &scanner.AssetVulnMetadataList{
		Metadata: []*scanner.AssetVulnMetadata{
			&scanner.AssetVulnMetadata{
				Asset:    asset,
				Packages: apiPackages,
				// TODO: remove the incognito here to ensure the data is stored per asset mrn upstream
				Incognito: true,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	reports := reportBinder.GetReports()

	if len(reports) > 1 {
		return nil, errors.New("vulnerability report contains too many reports")
	}

	var report *scanner.VulnReport
	for i := range reports {
		report = reports[i]
	}

	a.Cache.Store("_report", &lumi.CacheEntry{Data: report})

	return report, nil
}

func (a *lumiPlatformAdvisories) GetCvss() (interface{}, error) {
	report, err := a.getAdvisoryReport()
	if err != nil {
		return nil, err
	}

	obj, err := a.Runtime.CreateResource("cvss",
		"score", float64(report.Stats.Score)/10,
		"vector", "", // TODO: we need to extend the report to include the vector in the report
		"source", "", // TODO: we need to extend the report to include the source in the report
	)
	if err != nil {
		return nil, err
	}

	return obj, nil
}

func (a *lumiPlatformAdvisories) GetList() ([]interface{}, error) {
	report, err := a.getAdvisoryReport()
	if err != nil {
		return nil, err
	}

	lumiAdvisories := make([]interface{}, len(report.Advisories))
	for i := range report.Advisories {
		advisory := report.Advisories[i]

		cvssScore, err := a.Runtime.CreateResource("cvss",
			"score", float64(advisory.Score)/10,
			"vector", "", // TODO: we need to extend the report to include the vector in the report
			"source", "", // TODO: we need to extend the report to include the source in the report
		)
		if err != nil {
			return nil, err
		}

		var published *time.Time
		parsedTime, err := time.Parse(time.RFC3339, advisory.Published)
		if err == nil {
			published = &parsedTime
		}

		var modified *time.Time
		parsedTime, err = time.Parse(time.RFC3339, advisory.Modified)
		if err == nil {
			modified = &parsedTime
		}

		lumiAdvisory, err := a.Runtime.CreateResource("risk.advisory",
			"id", advisory.ID,
			"mrn", advisory.Mrn,
			"title", advisory.Title,
			"description", advisory.Description,
			"published", published,
			"modified", modified,
			"worstScore", cvssScore,
		)
		if err != nil {
			return nil, err
		}

		lumiAdvisories[i] = lumiAdvisory
	}

	return lumiAdvisories, nil
}

func (a *lumiPlatformAdvisories) GetStats() (interface{}, error) {
	report, err := a.getAdvisoryReport()
	if err != nil {
		return nil, err
	}

	dict, err := jsonToDict(report.Stats.Advisories)
	if err != nil {
		return nil, err
	}

	return dict, nil
}
