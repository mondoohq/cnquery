// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/resources"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/upstream/gql"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
	mondoogql "go.mondoo.com/mondoo-go"
)

type mqlVulnmgmtInternal struct {
	gqlClient *gql.MondooClient
}

func (v *mqlVulnmgmt) lastAssessment() (*time.Time, error) {
	mcc := v.MqlRuntime.Upstream
	if mcc == nil || mcc.ApiEndpoint == "" {
		return nil, resources.MissingUpstreamError{}
	}

	var mondooClient *gql.MondooClient
	var err error
	if v.gqlClient != nil {
		mondooClient = v.gqlClient
	} else {
		// get new gql client
		mondooClient, err = gql.NewClient(&mcc.UpstreamConfig, mcc.HttpClient)
		if err != nil {
			return nil, err
		}
		v.gqlClient = mondooClient
	}

	if v.MqlRuntime.Upstream.AssetMrn == "" {
		return nil, errors.New("no asset mrn available")
	}
	lastUpdate, err := mondooClient.LastAssessment(v.MqlRuntime.Upstream.AssetMrn)
	if err != nil {
		return nil, err
	}

	log.Debug().Str("time", lastUpdate).Msg("search for package last update")
	if lastUpdate == "" {
		return nil, errors.New("no update time available")
	}

	var lastUpdateTime *time.Time
	if lastUpdate != "" {
		parsedLastUpdateTime, err := time.Parse(time.RFC3339, lastUpdate)
		if err != nil {
			return nil, errors.New("could not parse last update time: " + lastUpdate)
		}
		lastUpdateTime = &parsedLastUpdateTime
	} else {
		lastUpdateTime = &llx.NeverFutureTime
	}

	return lastUpdateTime, nil
}

func (v *mqlVulnmgmt) cves() ([]interface{}, error) {
	// see command resource for reference
	// we ignore the return value because everything is set in populateData
	// `plugin.StateIsSet` is used to indicate that the data is available
	return nil, v.populateData()
}

func (v *mqlVulnmgmt) advisories() ([]interface{}, error) {
	// see command resource for reference
	// we ignore the return value because everything is set in populateData
	// `plugin.StateIsSet` is used to indicate that the data is available
	return nil, v.populateData()
}

func (v *mqlVulnmgmt) packages() ([]interface{}, error) {
	// see command resource for reference
	// we ignore the return value because everything is set in populateData
	// `plugin.StateIsSet` is used to indicate that the data is available
	return nil, v.populateData()
}

func (v *mqlVulnmgmt) stats() (*mqlAuditCvss, error) {
	// see command resource for reference
	// we ignore the return value because everything is set in populateData
	// `plugin.StateIsSet` is used to indicate that the data is available
	return nil, v.populateData()
}

func (v *mqlVulnmgmt) populateData() error {
	vulnReport, err := v.getReport()
	if err != nil {
		return err
	}

	mqlVulAdvisories := make([]interface{}, len(vulnReport.Advisories))
	for i, a := range vulnReport.Advisories {
		var parsedPublished *time.Time
		var parsedModified *time.Time
		var err error
		published, err := time.Parse(time.RFC3339, a.PublishedAt)
		if err != nil {
			log.Debug().Str("date", a.PublishedAt).Str("advisory", a.Id).Msg("could not parse published date")
		} else {
			parsedPublished = &published
		}
		modified, err := time.Parse(time.RFC3339, a.ModifiedAt)
		if err != nil {
			log.Debug().Str("date", a.ModifiedAt).Str("advisory", a.Id).Msg("could not parse modified date")
		} else {
			parsedModified = &modified
		}
		id := fmt.Sprintf("%d-%s", a.CvssScore.Value, a.CvssScore.Vector)
		cvssScore, err := CreateResource(v.MqlRuntime, "audit.cvss", map[string]*llx.RawData{
			"__id":   llx.StringData(id),
			"score":  llx.FloatData(float64(a.CvssScore.Value) / 10),
			"vector": llx.StringData(a.CvssScore.Vector),
		})
		if err != nil {
			return err
		}
		mqlVulnAdvisory, err := CreateResource(v.MqlRuntime, "vuln.advisory", map[string]*llx.RawData{
			"id":          llx.StringData(a.Id),
			"title":       llx.StringData(a.Title),
			"description": llx.StringData(a.Description),
			"published":   llx.TimeDataPtr(parsedPublished),
			"modified":    llx.TimeDataPtr(parsedModified),
			"worstScore":  llx.ResourceData(cvssScore, "audit.cvss"),
		})
		if err != nil {
			return err
		}
		mqlVulAdvisories[i] = mqlVulnAdvisory
	}

	mqlVulnCves := make([]interface{}, len(vulnReport.Cves))
	for i, c := range vulnReport.Cves {
		var parsedPublished *time.Time
		var parsedModified *time.Time
		var err error
		published, err := time.Parse(time.RFC3339, c.PublishedAt)
		if err != nil {
			log.Debug().Str("date", c.PublishedAt).Str("cve", c.Id).Msg("could not parse published date")
		} else {
			parsedPublished = &published
		}
		modified, err := time.Parse(time.RFC3339, c.ModifiedAt)
		if err != nil {
			log.Debug().Str("date", c.ModifiedAt).Str("cve", c.Id).Msg("could not parse modified date")
		} else {
			parsedModified = &modified
		}
		id := fmt.Sprintf("%d-%s", c.CvssScore.Value, c.CvssScore.Vector)
		cvssScore, err := CreateResource(v.MqlRuntime, "audit.cvss", map[string]*llx.RawData{
			"__id":   llx.StringData(id),
			"score":  llx.FloatData(float64(c.CvssScore.Value) / 10),
			"vector": llx.StringData(c.CvssScore.Vector),
		})
		if err != nil {
			return err
		}
		mqlVulnCve, err := CreateResource(v.MqlRuntime, "vuln.cve", map[string]*llx.RawData{
			"id":         llx.StringData(c.Id),
			"worstScore": llx.ResourceData(cvssScore, "audit.cvss"),
			"state":      llx.StringData(c.State),
			"summary":    llx.StringData(c.Summary),
			"published":  llx.TimeDataPtr(parsedPublished),
			"modified":   llx.TimeDataPtr(parsedModified),
		})
		if err != nil {
			return err
		}
		mqlVulnCves[i] = mqlVulnCve
	}

	mqlVulnPackages := make([]interface{}, len(vulnReport.Packages))
	for i, p := range vulnReport.Packages {
		mqlVulnPackage, err := CreateResource(v.MqlRuntime, "vuln.package", map[string]*llx.RawData{
			"name":      llx.StringData(p.Name),
			"version":   llx.StringData(p.Version),
			"available": llx.StringData(p.Available),
			"arch":      llx.StringData(p.Arch),
		})
		if err != nil {
			return err
		}
		mqlVulnPackages[i] = mqlVulnPackage
	}

	id := fmt.Sprintf("%d-%s", vulnReport.Stats.Score.Value, vulnReport.Stats.Score.Vector)
	res, err := CreateResource(v.MqlRuntime, "audit.cvss", map[string]*llx.RawData{
		"__id":   llx.StringData(id),
		"score":  llx.FloatData(float64(vulnReport.Stats.Score.Value) / 10),
		"vector": llx.StringData(vulnReport.Stats.Score.Vector),
	})
	if err != nil {
		return err
	}
	statsCvssScore := res.(*mqlAuditCvss)

	v.Advisories = plugin.TValue[[]interface{}]{Data: mqlVulAdvisories, State: plugin.StateIsSet}
	v.Cves = plugin.TValue[[]interface{}]{Data: mqlVulnCves, State: plugin.StateIsSet}
	v.Packages = plugin.TValue[[]interface{}]{Data: mqlVulnPackages, State: plugin.StateIsSet}
	v.Stats = plugin.TValue[*mqlAuditCvss]{Data: statsCvssScore, State: plugin.StateIsSet}

	return nil
}

func (v *mqlVulnmgmt) getReport() (*gql.VulnReport, error) {
	mcc := v.MqlRuntime.Upstream
	if mcc == nil || mcc.ApiEndpoint == "" {
		return nil, resources.MissingUpstreamError{}
	}

	var mondooClient *gql.MondooClient
	var err error
	if v.gqlClient != nil {
		mondooClient = v.gqlClient
	} else {
		// get new gql client
		mondooClient, err = gql.NewClient(&mcc.UpstreamConfig, mcc.HttpClient)
		if err != nil {
			return nil, err
		}
		v.gqlClient = mondooClient
	}

	if v.MqlRuntime.Upstream.AssetMrn == "" {
		log.Debug().Msg("no asset mrn available")
		return v.getIncognitoReport(mondooClient)
	}
	gqlVulnReport, err := mondooClient.GetVulnCompactReport(v.MqlRuntime.Upstream.AssetMrn)
	if err != nil {
		return nil, err
	}

	log.Debug().Interface("gqlReport", gqlVulnReport).Msg("search for asset vuln report")
	if gqlVulnReport == nil {
		return nil, errors.New("no vulnerability report available")
	}

	return gqlVulnReport, nil
}

func (v *mqlVulnmgmt) getIncognitoReport(mondooClient *gql.MondooClient) (*gql.VulnReport, error) {
	conn := v.MqlRuntime.Connection.(shared.Connection)
	platform := conn.Asset().Platform

	pkgsRes, err := CreateResource(v.MqlRuntime, "packages", nil)
	if err != nil {
		return nil, err
	}
	pkgs := pkgsRes.(*mqlPackages)
	pkgsList := pkgs.GetList().Data

	gqlPackages := make([]mondoogql.PackageInput, len(pkgsList))
	for i, p := range pkgsList {
		mqlPkg := p.(*mqlPackage)
		gqlPackages[i] = mondoogql.PackageInput{
			Name:    mondoogql.String(mqlPkg.Name.Data),
			Version: mondoogql.String(mqlPkg.Version.Data),
			Arch:    mondoogql.NewStringPtr(mondoogql.String(mqlPkg.Arch.Data)),
			Origin:  mondoogql.NewStringPtr(mondoogql.String(mqlPkg.Origin.Data)),
			Format:  mondoogql.NewStringPtr(mondoogql.String(mqlPkg.Format.Data)),
		}
	}

	family := []*mondoogql.String{}
	for _, f := range platform.Family {
		family = append(family, mondoogql.NewStringPtr(mondoogql.String(f)))
	}
	inputPlatform := mondoogql.PlatformInput{
		Name:    mondoogql.NewStringPtr(mondoogql.String(platform.Name)),
		Release: mondoogql.NewStringPtr(mondoogql.String(platform.Version)),
		Build:   mondoogql.NewStringPtr(mondoogql.String(platform.Build)),
		Family:  &family,
	}
	inputLabels := []*mondoogql.KeyValueInput{}
	for k := range platform.Labels {
		inputLabels = append(inputLabels, &mondoogql.KeyValueInput{
			Key:   mondoogql.String(k),
			Value: mondoogql.NewStringPtr(mondoogql.String(platform.Labels[k])),
		})
	}
	inputPlatform.Labels = &inputLabels
	gqlVulnReport, err := mondooClient.GetIncognitoVulnReport(inputPlatform, gqlPackages)
	if err != nil {
		return nil, err
	}

	log.Debug().Interface("gqlReport", gqlVulnReport).Msg("search for asset vuln report")
	if gqlVulnReport == nil {
		return nil, errors.New("no vulnerability report available")
	}

	return gqlVulnReport, nil
}

func (a *mqlVulnAdvisory) id() (string, error) {
	return a.Id.Data, a.Id.Error
}

func (c *mqlVulnCve) id() (string, error) {
	return c.Id.Data, c.Id.Error
}

func (p *mqlVulnPackage) id() (string, error) {
	id := p.Name.Data + "-" + p.Version.Data
	return id, p.Name.Error
}
