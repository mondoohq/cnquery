// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/logger"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/resources"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/upstream/gql"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/upstream/mvd"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/upstream/mvd/cvss"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
)

// TODO: generalize this kind of function
func getKernelVersion(kernel *mqlKernel) string {
	raw := kernel.GetInfo()
	if raw.Error != nil {
		return ""
	}

	info, ok := raw.Data.(map[string]interface{})
	if !ok {
		return ""
	}

	val, ok := info["version"]
	if !ok {
		return ""
	}

	return val.(string)
}

func fetchVulnReport(runtime *plugin.Runtime) (interface{}, error) {
	mcc := runtime.Upstream
	if mcc == nil || mcc.ApiEndpoint == "" {
		return nil, resources.MissingUpstreamError{}
	}

	// get new mvd client
	scannerClient, err := mvd.NewAdvisoryScannerClient(mcc.ApiEndpoint, mcc.HttpClient, mcc.Plugins...)
	if err != nil {
		return nil, err
	}

	conn := runtime.Connection.(shared.Connection)
	apiPackages := []*mvd.Package{}
	kernelVersion := ""

	// collect packages if the platform supports gathering files
	if conn.Capabilities().Has(shared.Capability_File) {
		obj, err := CreateResource(runtime, "packages", map[string]*llx.RawData{})
		if err != nil {
			return nil, err
		}
		packages := obj.(*mqlPackages)

		r := packages.GetList()
		if r.Error != nil {
			return nil, r.Error
		}

		for i := range r.Data {
			mqlPkg := r.Data[i]
			pkg := mqlPkg.(*mqlPackage)

			apiPackages = append(apiPackages, &mvd.Package{
				Name:    pkg.Name.Data,
				Version: pkg.Version.Data,
				Arch:    pkg.Arch.Data,
				Format:  pkg.Format.Data,
				Origin:  pkg.Origin.Data,
			})
		}

		// determine the kernel version if possible (just needed for linux at this point)
		// therefore we ignore the error because its not important, worst case the user sees to many advisories
		objKernel, err := CreateResource(runtime, "kernel", map[string]*llx.RawData{})
		if err == nil {
			kernelVersion = getKernelVersion(objKernel.(*mqlKernel))
		}
	}

	scanjob := &mvd.AnalyseAssetRequest{
		Platform:      mvd.NewMvdPlatform(conn.Asset().Platform),
		Packages:      apiPackages,
		KernelVersion: kernelVersion,
	}
	logger.DebugDumpYAML("vuln-scan-job", scanjob)

	log.Debug().Bool("incognito", mcc.Incognito).Msg("run advisory scan")
	report, err := scannerClient.AnalyseAsset(context.Background(), scanjob)
	if err != nil {
		return nil, err
	}

	return convert.JsonToDict(report)
}

func (p *mqlPlatform) vulnerabilityReport() (interface{}, error) {
	return fetchVulnReport(p.MqlRuntime)
}

// fetches the vulnerability report and returns the full report
func (p *mqlAsset) vulnerabilityReport() (interface{}, error) {
	return fetchVulnReport(p.MqlRuntime)
}

func getAdvisoryReport(runtime *plugin.Runtime) (*mvd.VulnReport, error) {
	mcc := runtime.Upstream
	if mcc == nil || mcc.ApiEndpoint == "" {
		return nil, resources.MissingUpstreamError{}
	}

	// get new gql client
	mondooClient, err := gql.NewClient(&mcc.UpstreamConfig, mcc.HttpClient)
	if err != nil {
		return nil, err
	}

	gqlVulnReport, err := mondooClient.GetVulnCompactReport(runtime.Upstream.AssetMrn)
	if err != nil {
		return nil, err
	}

	log.Debug().Interface("gqlReport", gqlVulnReport).Msg("search for asset vuln report")
	if gqlVulnReport == nil {
		return nil, errors.New("no vulnerability report available")
	}

	vulnReport := gql.ConvertToMvdVulnReport(gqlVulnReport)

	return vulnReport, nil
}

func (a *mqlPlatformAdvisories) id() (string, error) {
	return "platform.advisories", nil
}

func (a *mqlPlatformAdvisories) cvss() (*mqlAuditCvss, error) {
	report, err := getAdvisoryReport(a.MqlRuntime)
	if err != nil {
		return nil, err
	}

	obj, err := CreateResource(a.MqlRuntime, "audit.cvss", map[string]*llx.RawData{
		"score":  llx.FloatData(float64(report.Stats.Score) / 10),
		"vector": llx.StringData(""), // TODO: we need to extend the report to include the vector in the report
	})
	if err != nil {
		return nil, err
	}

	return obj.(*mqlAuditCvss), nil
}

func (a *mqlPlatformAdvisories) list() ([]interface{}, error) {
	report, err := getAdvisoryReport(a.MqlRuntime)
	if err != nil {
		return nil, err
	}

	mqlAdvisories := make([]interface{}, len(report.Advisories))
	for i := range report.Advisories {
		advisory := report.Advisories[i]

		var worstScore *cvss.Cvss
		if advisory.WorstScore != nil {
			worstScore = advisory.WorstScore
		} else {
			worstScore = &cvss.Cvss{Score: 0.0, Vector: ""}
		}

		cvssScore, err := CreateResource(a.MqlRuntime, "audit.cvss", map[string]*llx.RawData{
			"score":  llx.FloatData(float64(worstScore.Score)),
			"vector": llx.StringData(worstScore.Vector),
		})
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

		mqlAdvisory, err := CreateResource(a.MqlRuntime, "audit.advisory", map[string]*llx.RawData{
			"id":          llx.StringData(advisory.ID),
			"mrn":         llx.StringData(advisory.Mrn),
			"title":       llx.StringData(advisory.Title),
			"description": llx.StringData(advisory.Description),
			"published":   llx.TimeData(*published),
			"modified":    llx.TimeData(*modified),
			"worstScore":  llx.ResourceData(cvssScore, "audit.cvss"),
		})
		if err != nil {
			return nil, err
		}

		mqlAdvisories[i] = mqlAdvisory
	}

	return mqlAdvisories, nil
}

func (a *mqlPlatformAdvisories) stats() (interface{}, error) {
	report, err := getAdvisoryReport(a.MqlRuntime)
	if err != nil {
		return nil, err
	}

	dict, err := convert.JsonToDict(report.Stats.Advisories)
	if err != nil {
		return nil, err
	}

	return dict, nil
}

func (a *mqlPlatformCves) id() (string, error) {
	return "platform.cves", nil
}

func (a *mqlPlatformCves) list() ([]interface{}, error) {
	report, err := getAdvisoryReport(a.MqlRuntime)
	if err != nil {
		return nil, err
	}

	cveList := report.Cves()

	mqlCves := make([]interface{}, len(cveList))
	for i := range cveList {
		cve := cveList[i]

		var worstScore *cvss.Cvss
		if cve.WorstScore != nil {
			worstScore = cve.WorstScore
		} else {
			worstScore = &cvss.Cvss{Score: 0.0, Vector: ""}
		}

		cvssScore, err := CreateResource(a.MqlRuntime, "audit.cvss", map[string]*llx.RawData{
			"score":  llx.FloatData(float64(worstScore.Score)),
			"vector": llx.StringData(worstScore.Vector),
		})
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

		mqlCve, err := CreateResource(a.MqlRuntime, "audit.cve", map[string]*llx.RawData{
			"id":         llx.StringData(cve.ID),
			"mrn":        llx.StringData(cve.Mrn),
			"state":      llx.StringData(cve.State.String()),
			"summary":    llx.StringData(cve.Summary),
			"unscored":   llx.BoolData(cve.Unscored),
			"published":  llx.TimeDataPtr(published),
			"modified":   llx.TimeDataPtr(modified),
			"worstScore": llx.ResourceData(cvssScore, "audit.cvss"),
		})
		if err != nil {
			return nil, err
		}

		mqlCves[i] = mqlCve
	}

	return mqlCves, nil
}

func (a *mqlPlatformCves) cvss() (*mqlAuditCvss, error) {
	report, err := getAdvisoryReport(a.MqlRuntime)
	if err != nil {
		return nil, err
	}

	score := float32(0.0)
	for i := range report.Advisories {
		advisory := report.Advisories[i]
		for j := range advisory.Cves {
			cve := advisory.Cves[j]
			if cve.WorstScore != nil && cve.WorstScore.Score > score {
				score = cve.WorstScore.Score
			}
		}
	}

	obj, err := CreateResource(a.MqlRuntime, "audit.cvss", map[string]*llx.RawData{
		"score":  llx.FloatData(float64(int(score*10)) / 10),
		"vector": llx.StringData(""),
	})
	if err != nil {
		return nil, err
	}

	return obj.(*mqlAuditCvss), nil
}

func (a *mqlPlatformCves) stats() (interface{}, error) {
	report, err := getAdvisoryReport(a.MqlRuntime)
	if err != nil {
		return nil, err
	}

	dict, err := convert.JsonToDict(report.Stats.Cves)
	if err != nil {
		return nil, err
	}

	return dict, nil
}
