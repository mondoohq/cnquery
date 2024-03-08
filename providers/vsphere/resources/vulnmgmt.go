// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/resources"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/upstream/mvd"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers/vsphere/connection"
)

type mqlVulnmgmtInternal struct{}

func getVulnMgmtClient(runtime *plugin.Runtime) (*mvd.VulnMgmtClient, error) {
	mcc := runtime.Upstream
	if mcc == nil || mcc.ApiEndpoint == "" {
		return nil, resources.MissingUpstreamError{}
	}

	// get new vulnMgmtClient client
	return mvd.NewVulnMgmtClient(mcc.ApiEndpoint, mcc.HttpClient, mcc.Plugins...)
}

func (v *mqlVulnmgmt) cves() ([]interface{}, error) {
	runtime := v.MqlRuntime
	vulnMgmtClient, err := getVulnMgmtClient(runtime)
	if err != nil {
		return nil, err
	}

	conn, ok := runtime.Connection.(*connection.VsphereConnection)
	if !ok {
		return nil, errors.New("no connection available")
	}

	var mrn string
	if conn.Asset() != nil {
		mrn = conn.Asset().Mrn
	}

	// TODO: handle incognito mode
	ctx := context.Background()
	resp, err := vulnMgmtClient.ListVulnerabilities(ctx, &mvd.ListVulnerabilitiesRequest{
		Mrn: mrn,
	})
	// TODO: handle case where no report is available
	if err != nil {
		return nil, err
	}

	list := make([]interface{}, len(resp.Cves))
	for i, c := range resp.Cves {
		var parsedPublished *time.Time
		var parsedModified *time.Time
		var err error
		published, err := time.Parse(time.RFC3339, c.Published)
		if err != nil {
			log.Debug().Str("date", c.Published).Str("cve", c.ID).Msg("could not parse published date")
		} else {
			parsedPublished = &published
		}
		modified, err := time.Parse(time.RFC3339, c.Modified)
		if err != nil {
			log.Debug().Str("date", c.Modified).Str("cve", c.ID).Msg("could not parse modified date")
		} else {
			parsedModified = &modified
		}

		cvssScore, err := newMqlVulnCvss(v.MqlRuntime, float64(c.WorstScore.Score), c.WorstScore.Vector)
		if err != nil {
			return nil, err
		}

		mqlVulnCve, err := newMqlVulnCve(v.MqlRuntime, c.ID, c.Summary, c.State.String(), parsedPublished, parsedModified, cvssScore)
		if err != nil {
			return nil, err
		}
		list[i] = mqlVulnCve
	}

	return list, nil
}

func (v *mqlVulnmgmt) advisories() ([]interface{}, error) {
	runtime := v.MqlRuntime
	vulnMgmtClient, err := getVulnMgmtClient(runtime)
	if err != nil {
		return nil, err
	}

	conn, ok := runtime.Connection.(*connection.VsphereConnection)
	if !ok {
		return nil, errors.New("no connection available")
	}

	var mrn string
	if conn.Asset() != nil {
		mrn = conn.Asset().Mrn
	}

	// TODO: handle incognito mode
	ctx := context.Background()
	resp, err := vulnMgmtClient.ListAdvisories(ctx, &mvd.ListAdvisoriesRequest{
		Mrn: mrn,
	})
	// TODO: handle case where no report is available
	if err != nil {
		return nil, err
	}

	list := make([]interface{}, len(resp.Advisories))
	for i, a := range resp.Advisories {
		var parsedPublished *time.Time
		var parsedModified *time.Time
		var err error
		published, err := time.Parse(time.RFC3339, a.Published)
		if err != nil {
			log.Debug().Str("date", a.Published).Str("advisory", a.ID).Msg("could not parse published date")
		} else {
			parsedPublished = &published
		}
		modified, err := time.Parse(time.RFC3339, a.Modified)
		if err != nil {
			log.Debug().Str("date", a.Modified).Str("advisory", a.ID).Msg("could not parse modified date")
		} else {
			parsedModified = &modified
		}

		cvssScore, err := newMqlVulnCvss(v.MqlRuntime, float64(a.WorstScore.Score), a.WorstScore.Vector)
		if err != nil {
			return nil, err
		}

		mqlVulnAdvisory, err := newMqlVulnAdvisory(v.MqlRuntime, a.ID, a.Title, a.Description, parsedPublished, parsedModified, cvssScore)
		if err != nil {
			return nil, err
		}
		list[i] = mqlVulnAdvisory
	}

	return list, nil
}

func (v *mqlVulnmgmt) packages() ([]interface{}, error) {
	runtime := v.MqlRuntime
	vulnMgmtClient, err := getVulnMgmtClient(runtime)
	if err != nil {
		return nil, err
	}

	conn, ok := runtime.Connection.(*connection.VsphereConnection)
	if !ok {
		return nil, errors.New("no connection available")
	}

	var mrn string
	if conn.Asset() != nil {
		mrn = conn.Asset().Mrn
	}

	// TODO: handle incognito mode
	ctx := context.Background()
	resp, err := vulnMgmtClient.ListVulnerablePackages(ctx, &mvd.ListVulnerablePackagesRequest{
		Mrn: mrn,
	})
	// TODO: handle case where no report is available
	if err != nil {
		return nil, err
	}

	list := make([]interface{}, len(resp.Packages))
	for i, p := range resp.Packages {
		mqlVulnPackage, err := newMqlVulnPackage(v.MqlRuntime, p.Name, p.Version, p.Available, p.Arch)
		if err != nil {
			return nil, err
		}
		list[i] = mqlVulnPackage
	}

	return list, nil
}

func (v *mqlVulnmgmt) summary() (interface{}, error) {
	return nil, errors.New("not implemented")
}

func (v *mqlVulnmgmt) analyze() (string, error) {
	runtime := v.MqlRuntime
	vulnMgmtClient, err := getVulnMgmtClient(runtime)
	if err != nil {
		return "", err
	}

	conn, ok := runtime.Connection.(*connection.VsphereConnection)
	if !ok {
		return "", errors.New("no connection available")
	}

	req := &mvd.AnalyseRequest{}

	// add platform to request
	platform := conn.Asset().Platform

	req.Platform = &mvd.Platform{
		Name:    platform.Name,
		Release: platform.Version,
		Build:   platform.Build,
		// Family:  platform.Family,
		Labels: platform.Labels,
	}

	resp, err := vulnMgmtClient.Analyse(context.Background(), req)
	if err != nil {
		return "", err
	}
	return resp.ReportMrn, nil
}

func newMqlVulnCvss(runtime *plugin.Runtime, score float64, vector string) (*mqlAuditCvss, error) {
	res, err := CreateResource(runtime, "audit.cvss", map[string]*llx.RawData{
		"score":  llx.FloatData(score / 10),
		"vector": llx.StringData(vector),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAuditCvss), nil
}

func newMqlVulnAdvisory(runtime *plugin.Runtime, id, title, description string, published, modified *time.Time, cvssScore *mqlAuditCvss) (*mqlVulnAdvisory, error) {
	res, err := CreateResource(runtime, "vuln.advisory", map[string]*llx.RawData{
		"id":          llx.StringData(id),
		"title":       llx.StringData(title),
		"description": llx.StringData(description),
		"published":   llx.TimeDataPtr(published),
		"modified":    llx.TimeDataPtr(modified),
		"worstScore":  llx.ResourceData(cvssScore, "audit.cvss"),
	})
	if err != nil {
		return nil, err
	}

	return res.(*mqlVulnAdvisory), nil
}

func (a *mqlVulnAdvisory) id() (string, error) {
	return a.Id.Data, a.Id.Error
}

func newMqlVulnCve(runtime *plugin.Runtime, id, summary, state string, published, modified *time.Time, cvssScore *mqlAuditCvss) (*mqlVulnCve, error) {
	res, err := CreateResource(runtime, "vuln.cve", map[string]*llx.RawData{
		"id":         llx.StringData(id),
		"summary":    llx.StringData(summary),
		"state":      llx.StringData(state),
		"published":  llx.TimeDataPtr(published),
		"modified":   llx.TimeDataPtr(modified),
		"worstScore": llx.ResourceData(cvssScore, "audit.cvss"),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlVulnCve), nil
}

func (c *mqlVulnCve) id() (string, error) {
	return c.Id.Data, c.Id.Error
}

func newMqlVulnPackage(runtime *plugin.Runtime, name, version, available, arch string) (*mqlVulnPackage, error) {
	res, err := CreateResource(runtime, "vuln.package", map[string]*llx.RawData{
		"name":      llx.StringData(name),
		"version":   llx.StringData(version),
		"available": llx.StringData(available),
		"arch":      llx.StringData(arch),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlVulnPackage), nil
}

func (p *mqlVulnPackage) id() (string, error) {
	id := p.Name.Data + "-" + p.Version.Data
	return id, p.Name.Error
}
