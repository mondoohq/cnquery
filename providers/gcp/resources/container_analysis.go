// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"

	containeranalysis "cloud.google.com/go/containeranalysis/apiv1"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/gcp/connection"
	"go.mondoo.com/mql/v13/types"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	grafeaspb "google.golang.org/genproto/googleapis/grafeas/v1"
)

type mqlGcpProjectContainerAnalysisServiceInternal struct {
	serviceEnabled bool
}

func (g *mqlGcpProject) containerAnalysis() (*mqlGcpProjectContainerAnalysisService, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	projectId := g.Id.Data

	res, err := CreateResource(g.MqlRuntime, "gcp.project.containerAnalysisService", map[string]*llx.RawData{
		"projectId": llx.StringData(projectId),
	})
	if err != nil {
		return nil, err
	}

	serviceEnabled, err := g.isServiceEnabled(service_containeranalysis)
	if err != nil {
		return nil, err
	}

	svc := res.(*mqlGcpProjectContainerAnalysisService)
	svc.serviceEnabled = serviceEnabled
	if !serviceEnabled {
		log.Debug().Str("service", service_containeranalysis).Msg("gcp service is not enabled, skipping")
	}

	return svc, nil
}

func initGcpProjectContainerAnalysisService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}

	conn, ok := runtime.Connection.(*connection.GcpConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not a GCP connection")
	}

	projectId := conn.ResourceID()
	args["projectId"] = llx.StringData(projectId)

	return args, nil, nil
}

func (g *mqlGcpProjectContainerAnalysisService) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	return fmt.Sprintf("%s/gcp.project.containerAnalysisService", g.ProjectId.Data), nil
}

func (g *mqlGcpProjectContainerAnalysisServiceOccurrence) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectContainerAnalysisService) occurrences() ([]any, error) {
	if !g.serviceEnabled {
		return nil, nil
	}

	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(containeranalysis.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := containeranalysis.NewClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	grafeasClient := client.GetGrafeasClient()

	it := grafeasClient.ListOccurrences(ctx, &grafeaspb.ListOccurrencesRequest{
		Parent:   fmt.Sprintf("projects/%s", projectId),
		PageSize: 1000,
	})

	var res []any
	for {
		occ, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			if isGRPCSkippable(err) {
				log.Warn().Err(err).Msg("could not list Container Analysis occurrences")
				return nil, nil
			}
			return nil, err
		}

		vulnerability, err := protoToDict(occ.GetVulnerability())
		if err != nil {
			return nil, err
		}

		var (
			vulnSeverity, vulnEffectiveSeverity       string
			vulnCvssScore                             float64
			vulnFixAvailable                          bool
			vulnShortDescription, vulnLongDescription string
			vulnPackageIssues                         []any
		)
		if v := occ.GetVulnerability(); v != nil {
			// Leave severity empty when unspecified so audits can filter on
			// `vulnerabilitySeverity != ""` without matching the proto zero value.
			if v.Severity != grafeaspb.Severity_SEVERITY_UNSPECIFIED {
				vulnSeverity = v.Severity.String()
			}
			if v.EffectiveSeverity != grafeaspb.Severity_SEVERITY_UNSPECIFIED {
				vulnEffectiveSeverity = v.EffectiveSeverity.String()
			}
			vulnCvssScore = float64(v.CvssScore)
			vulnFixAvailable = v.FixAvailable
			vulnShortDescription = v.ShortDescription
			vulnLongDescription = v.LongDescription
			for _, pi := range v.PackageIssue {
				d, err := protoToDict(pi)
				if err != nil {
					return nil, err
				}
				vulnPackageIssues = append(vulnPackageIssues, d)
			}
		}

		build, err := protoToDict(occ.GetBuild())
		if err != nil {
			return nil, err
		}

		image, err := protoToDict(occ.GetImage())
		if err != nil {
			return nil, err
		}

		packageInfo, err := protoToDict(occ.GetPackage())
		if err != nil {
			return nil, err
		}

		deployment, err := protoToDict(occ.GetDeployment())
		if err != nil {
			return nil, err
		}

		discovery, err := protoToDict(occ.GetDiscovery())
		if err != nil {
			return nil, err
		}

		attestation, err := protoToDict(occ.GetAttestation())
		if err != nil {
			return nil, err
		}

		mqlOcc, err := CreateResource(g.MqlRuntime, "gcp.project.containerAnalysisService.occurrence", map[string]*llx.RawData{
			"name":                           llx.StringData(occ.Name),
			"resourceUri":                    llx.StringData(occ.ResourceUri),
			"noteName":                       llx.StringData(occ.NoteName),
			"kind":                           llx.StringData(occ.Kind.String()),
			"remediation":                    llx.StringData(occ.Remediation),
			"vulnerability":                  llx.DictData(vulnerability),
			"vulnerabilitySeverity":          llx.StringData(vulnSeverity),
			"vulnerabilityEffectiveSeverity": llx.StringData(vulnEffectiveSeverity),
			"vulnerabilityCvssScore":         llx.FloatData(vulnCvssScore),
			"vulnerabilityFixAvailable":      llx.BoolData(vulnFixAvailable),
			"vulnerabilityShortDescription":  llx.StringData(vulnShortDescription),
			"vulnerabilityLongDescription":   llx.StringData(vulnLongDescription),
			"vulnerabilityPackageIssues":     llx.ArrayData(vulnPackageIssues, types.Dict),
			"build":                          llx.DictData(build),
			"image":                          llx.DictData(image),
			"packageInfo":                    llx.DictData(packageInfo),
			"deployment":                     llx.DictData(deployment),
			"discovery":                      llx.DictData(discovery),
			"attestation":                    llx.DictData(attestation),
			"created":                        llx.TimeDataPtr(timestampAsTimePtr(occ.CreateTime)),
			"updated":                        llx.TimeDataPtr(timestampAsTimePtr(occ.UpdateTime)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlOcc)
	}

	return res, nil
}
