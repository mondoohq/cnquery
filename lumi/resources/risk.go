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
func getAdvisoryReport(r *lumi.Runtime) (*scanner.VulnReport, error) {
	// get platform information
	obj, err := r.CreateResource("platform")
	if err != nil {
		return nil, err
	}

	platform := obj.(Platform)

	name, _ := platform.Name()
	release, _ := platform.Release()
	arch, _ := platform.Arch()

	// check if the data is cached
	// NOTE: we cache it in the platform resource, so that platform.advisories, platform.cves and
	// platform.exploits can all share the results
	cachedReport, ok := platform.LumiResource().Cache.Load("_report")
	if ok {
		report := cachedReport.Data.(*scanner.VulnReport)
		return report, nil
	}

	// get new advisory report
	spaceMrn, scannerClient, err := getScannerClient(r.Motor)
	if err != nil {
		return nil, err
	}

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

	obj, err = r.CreateResource("packages")
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

	platform.LumiResource().Cache.Store("_report", &lumi.CacheEntry{Data: report})

	return report, nil
}

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

func (a *lumiPlatformAdvisories) GetCvss() (interface{}, error) {
	report, err := getAdvisoryReport(a.Runtime)
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
	report, err := getAdvisoryReport(a.Runtime)
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
	report, err := getAdvisoryReport(a.Runtime)
	if err != nil {
		return nil, err
	}

	dict, err := jsonToDict(report.Stats.Advisories)
	if err != nil {
		return nil, err
	}

	return dict, nil
}

func (c *lumiRiskCve) id() (string, error) {
	return c.Mrn()
}

func (a *lumiPlatformCves) id() (string, error) {
	return "platform.cves", nil
}

func (a *lumiPlatformCves) GetList() ([]interface{}, error) {
	report, err := getAdvisoryReport(a.Runtime)
	if err != nil {
		return nil, err
	}

	cveList := report.Cves()

	lumiCves := make([]interface{}, len(cveList))
	for i := range cveList {
		cve := cveList[i]

		cvssScore, err := a.Runtime.CreateResource("cvss",
			"score", float64(cve.Score)/10,
			"vector", "", // TODO: we need to extend the report to include the vector in the report
			"source", "", // TODO: we need to extend the report to include the source in the report
		)
		if err != nil {
			return nil, err
		}

		var published *time.Time
		parsedTime, err := time.Parse(time.RFC3339, cve.Published)
		if err == nil {
			published = &parsedTime
		}

		var modified *time.Time
		parsedTime, err = time.Parse(time.RFC3339, cve.Modified)
		if err == nil {
			modified = &parsedTime
		}

		lumiCve, err := a.Runtime.CreateResource("risk.cve",
			"id", cve.ID,
			"mrn", cve.Mrn,
			"state", cve.State.String(),
			"summary", cve.Summary,
			"unscored", cve.Unscored,
			"published", published,
			"modified", modified,
			"worstScore", cvssScore,
		)
		if err != nil {
			return nil, err
		}

		lumiCves[i] = lumiCve
	}

	return lumiCves, nil
}

func (a *lumiPlatformCves) GetCvss() (interface{}, error) {
	report, err := getAdvisoryReport(a.Runtime)
	if err != nil {
		return nil, err
	}

	// TODO: we need to distingush between advisory, cve and exploit cvss
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

func (a *lumiPlatformCves) GetStats() (interface{}, error) {
	report, err := getAdvisoryReport(a.Runtime)
	if err != nil {
		return nil, err
	}

	dict, err := jsonToDict(report.Stats.Cves)
	if err != nil {
		return nil, err
	}

	return dict, nil
}

func (c *lumiRiskExploit) id() (string, error) {
	return c.Mrn()
}

func (a *lumiPlatformExploits) id() (string, error) {
	return "platform.exploits", nil
}

func (a *lumiPlatformExploits) GetList() ([]interface{}, error) {
	report, err := getAdvisoryReport(a.Runtime)
	if err != nil {
		return nil, err
	}

	lumiExploits := make([]interface{}, len(report.Exploits))
	for i := range report.Exploits {
		exploit := report.Exploits[i]

		cvssScore, err := a.Runtime.CreateResource("cvss",
			"score", float64(exploit.Score)/10,
			"vector", "", // TODO: we need to extend the report to include the vector in the report
			"source", "", // TODO: we need to extend the report to include the source in the report
		)
		if err != nil {
			return nil, err
		}

		var modified *time.Time
		parsedTime, err := time.Parse(time.RFC3339, exploit.Modified)
		if err == nil {
			modified = &parsedTime
		}

		lumiExploit, err := a.Runtime.CreateResource("risk.exploit",
			"id", exploit.ID,
			"mrn", exploit.Mrn,
			"modified", modified,
			"worstScore", cvssScore,
		)
		if err != nil {
			return nil, err
		}

		lumiExploits[i] = lumiExploit
	}

	return lumiExploits, nil
}

func (a *lumiPlatformExploits) GetCvss() (interface{}, error) {
	report, err := getAdvisoryReport(a.Runtime)
	if err != nil {
		return nil, err
	}

	// TODO: this needs to be the exploit worst score
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

func (a *lumiPlatformExploits) GetStats() (interface{}, error) {
	report, err := getAdvisoryReport(a.Runtime)
	if err != nil {
		return nil, err
	}

	dict, err := jsonToDict(report.Stats.Exploits)
	if err != nil {
		return nil, err
	}

	return dict, nil
}
