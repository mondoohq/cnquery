// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strconv"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/types"
)

func (g *mqlGithubRepository) findings() ([]any, error) {
	if g.Name.Error != nil {
		return nil, g.Name.Error
	}
	repoName := g.Name.Data

	if g.Owner.Error != nil {
		return nil, g.Owner.Error
	}
	owner := g.Owner.Data

	if owner.Login.Error != nil {
		return nil, owner.Login.Error
	}
	ownerLogin := owner.Login.Data

	findings, err := dependabotAlertFindings(g, ownerLogin, repoName)
	if err != nil {
		return nil, err
	}

	return findings, nil
}

func dependabotAlertFindings(g *mqlGithubRepository, owner, repository string) ([]any, error) {
	alertsResult := g.GetDependabotAlerts()
	if alertsResult.Error != nil {
		return nil, alertsResult.Error
	}

	res := []any{}
	for _, a := range alertsResult.Data {
		alert, ok := a.(*mqlGithubDependabotAlert)
		if !ok {
			continue
		}

		alertID, err := alert.id()
		if err != nil {
			return nil, err
		}

		var summary, severityStr, description, cvssVector string
		var cvssScore float64
		if alert.SecurityAdvisory.Error == nil && alert.SecurityAdvisory.Data != nil {
			if advisory, ok := alert.SecurityAdvisory.Data.(map[string]any); ok {
				summary, _ = advisory["summary"].(string)
				severityStr, _ = advisory["severity"].(string)
				description, _ = advisory["description"].(string)
				if cvss, ok := advisory["cvss"].(map[string]any); ok {
					cvssScore, _ = cvss["score"].(float64)
					cvssVector, _ = cvss["vectorString"].(string)
				}
			}
		}

		// SecurityVulnerability provides package-specific severity and version range,
		// which take precedence over the advisory-level values.
		var vulnerableVersionRange, firstPatchedVersion string
		if alert.SecurityVulnerability.Error == nil && alert.SecurityVulnerability.Data != nil {
			if vuln, ok := alert.SecurityVulnerability.Data.(map[string]any); ok {
				if s, ok := vuln["severity"].(string); ok && s != "" {
					severityStr = s
				}
				vulnerableVersionRange, _ = vuln["vulnerableVersionRange"].(string)
				if fpv, ok := vuln["firstPatchedVersion"].(map[string]any); ok {
					firstPatchedVersion, _ = fpv["identifier"].(string)
				}
			}
		}

		// Shared source resource representing GitHub Dependabot
		source, err := CreateResource(g.MqlRuntime, ResourceFindingSource, map[string]*llx.RawData{
			"__id": llx.StringData(ResourceFindingSource + "/" + owner + "/" + repository + "/" + alertID),
			"name": llx.StringData("github-dependabot"),
			"url":  llx.StringData(alert.Url.Data),
		})
		if err != nil {
			return nil, err
		}

		severity, err := CreateResource(g.MqlRuntime, ResourceFindingSeverity, map[string]*llx.RawData{
			"__id":     llx.StringData(ResourceFindingSeverity + "/" + owner + "/" + repository + "/" + alertID),
			"source":   llx.ResourceData(source, ResourceFindingSource),
			"score":    llx.FloatData(cvssScore),
			"severity": llx.StringData(severityStr),
			"vector":   llx.StringData(cvssVector),
			"method":   llx.StringData(""),
			"rating":   llx.StringData(severityStr),
		})
		if err != nil {
			return nil, err
		}

		detail, err := CreateResource(g.MqlRuntime, ResourceFindingDetail, map[string]*llx.RawData{
			"__id":        llx.StringData(ResourceFindingDetail + "/" + owner + "/" + repository + "/" + alertID),
			"category":    llx.StringData("vulnerability"),
			"severity":    llx.ResourceData(severity, ResourceFindingSeverity),
			"confidence":  llx.StringData(""),
			"description": llx.StringData(description),
			"references":  llx.ArrayData([]any{}, types.Resource(ResourceFindingReference)),
			"properties":  llx.MapData(map[string]any{}, types.String),
		})
		if err != nil {
			return nil, err
		}

		affectsList := []any{}
		if alert.Dependency.Error == nil && alert.Dependency.Data != nil {
			if dep, ok := alert.Dependency.Data.(map[string]any); ok {
				componentID := ""
				identifiers := map[string]any{}
				if pkg, ok := dep["package"].(map[string]any); ok {
					if name, ok := pkg["name"].(string); ok {
						componentID = name
					}
					if ecosystem, ok := pkg["ecosystem"].(string); ok {
						identifiers["ecosystem"] = ecosystem
					}
				}
				if manifestPath, ok := dep["manifestPath"].(string); ok {
					identifiers["manifestPath"] = manifestPath
				}
				if vulnerableVersionRange != "" {
					identifiers["vulnerableVersionRange"] = vulnerableVersionRange
				}
				if firstPatchedVersion != "" {
					identifiers["firstPatchedVersion"] = firstPatchedVersion
				}

				component, err := CreateResource(g.MqlRuntime, ResourceFindingComponent, map[string]*llx.RawData{
					"__id":        llx.StringData(ResourceFindingComponent + "/" + owner + "/" + repository + "/" + alertID),
					"id":          llx.StringData(componentID),
					"identifiers": llx.MapData(identifiers, types.String),
					"properties":  llx.MapData(map[string]any{}, types.String),
					"file":        llx.NilData,
				})
				if err != nil {
					return nil, err
				}

				affects, err := CreateResource(g.MqlRuntime, ResourceFindingAffectedComponent, map[string]*llx.RawData{
					"__id":          llx.StringData(ResourceFindingAffectedComponent + "/" + owner + "/" + repository + "/" + alertID),
					"component":     llx.ResourceData(component, ResourceFindingComponent),
					"subComponents": llx.ArrayData([]any{}, types.Resource(ResourceFindingComponent)),
				})
				if err != nil {
					return nil, err
				}
				affectsList = []any{affects}
			}
		}

		id := ResourceFinding + "/" + owner + "/" + repository + "/" + alertID
		finding, err := CreateResource(g.MqlRuntime, ResourceFinding, map[string]*llx.RawData{
			"__id":               llx.StringData(id),
			"id":                 llx.StringData(""),
			"ref":                llx.StringData(strconv.FormatInt(alert.Number.Data, 10)),
			"mrn":                llx.StringData(""),
			"groupId":            llx.StringData(""),
			"summary":            llx.StringData(summary),
			"details":            llx.ResourceData(detail, ResourceFindingDetail),
			"firstSeenAt":        llx.TimeDataPtr(alert.CreatedAt.Data),
			"lastSeenAt":         llx.TimeDataPtr(alert.UpdatedAt.Data),
			"remediatedAt":       llx.TimeDataPtr(alert.FixedAt.Data),
			"status":             llx.StringData(alert.State.Data),
			"src":                llx.ResourceData(source, ResourceFindingSource),
			"affectedComponents": llx.ArrayData(affectsList, types.Resource(ResourceFindingAffectedComponent)),
			"evidences":          llx.ArrayData([]any{}, types.Resource(ResourceFindingEvidence)),
			"remediations":       llx.ArrayData([]any{}, types.Dict),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, finding)
	}

	return res, nil
}
