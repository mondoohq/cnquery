// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v9/llx"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/resources"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/upstream/gql"
	"go.mondoo.com/cnquery/v9/providers/vsphere/connection"
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
	return nil, v.populateData()
}

func (v *mqlVulnmgmt) advisories() ([]interface{}, error) {
	return nil, v.populateData()
}

func (v *mqlVulnmgmt) packages() ([]interface{}, error) {
	return nil, v.populateData()
}

func (v *mqlVulnmgmt) stats() (*mqlAuditCvss, error) {
	return nil, v.populateData()
}

func (v *mqlVulnmgmt) populateData() error {
	vulnReport, err := v.getReport()
	if err != nil {
		return err
	}

	mqlVulAdvisories := make([]interface{}, len(vulnReport.Advisories))
	for i, a := range vulnReport.Advisories {
		parsedPublished, err := time.Parse(time.RFC3339, a.PublishedAt)
		if err != nil {
			return err
		}
		parsedModified, err := time.Parse(time.RFC3339, a.ModifiedAt)
		if err != nil {
			return err
		}
		cvssScore, err := CreateResource(v.MqlRuntime, "audit.cvss", map[string]*llx.RawData{
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
			"published":   llx.TimeData(parsedPublished),
			"modified":    llx.TimeData(parsedModified),
			"worstScore":  llx.ResourceData(cvssScore, "audit.cvss"),
		})
		if err != nil {
			return err
		}
		mqlVulAdvisories[i] = mqlVulnAdvisory
	}

	mqlVulnCves := make([]interface{}, len(vulnReport.Cves))
	for i, c := range vulnReport.Cves {
		parsedPublished, err := time.Parse(time.RFC3339, c.PublishedAt)
		if err != nil {
			return err
		}
		parsedModified, err := time.Parse(time.RFC3339, c.ModifiedAt)
		if err != nil {
			return err
		}
		cvssScore, err := CreateResource(v.MqlRuntime, "audit.cvss", map[string]*llx.RawData{
			"score":  llx.FloatData(float64(c.CvssScore.Value) / 10),
			"vector": llx.StringData(c.CvssScore.Vector),
		})
		if err != nil {
			return err
		}
		mqlVulnCve, err := CreateResource(v.MqlRuntime, "vuln.cve", map[string]*llx.RawData{
			"id":         llx.StringData(c.Id),
			"worstScore": llx.ResourceData(cvssScore, "audit.cvss"),
			"published":  llx.TimeData(parsedPublished),
			"modified":   llx.TimeData(parsedModified),
		})
		if err != nil {
			return err
		}
		mqlVulnCves[i] = mqlVulnCve
	}

	res, err := CreateResource(v.MqlRuntime, "audit.cvss", map[string]*llx.RawData{
		"score":  llx.FloatData(float64(vulnReport.Stats.Score.Value) / 10),
		"vector": llx.StringData(vulnReport.Stats.Score.Vector),
	})
	if err != nil {
		return err
	}
	statsCvssScore := res.(*mqlAuditCvss)

	v.Advisories = plugin.TValue[[]interface{}]{Data: mqlVulAdvisories, State: plugin.StateIsSet}
	v.Cves = plugin.TValue[[]interface{}]{Data: mqlVulnCves, State: plugin.StateIsSet}
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
	conn := v.MqlRuntime.Connection.(*connection.VsphereConnection)
	platform := conn.Asset().Platform

	gqlVulnReport, err := mondooClient.GetIncognitoVulnReport(mondoogql.PlatformInput{
		Name:    mondoogql.NewStringPtr(mondoogql.String(platform.Name)),
		Release: mondoogql.NewStringPtr(mondoogql.String(platform.Version)),
	}, []mondoogql.PackageInput{})
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
