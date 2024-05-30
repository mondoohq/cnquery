// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package generator

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.mondoo.com/cnquery/v11"
	"go.mondoo.com/cnquery/v11/cli/reporter"
	"go.mondoo.com/cnquery/v11/mrn"
	"go.mondoo.com/cnquery/v11/sbom"
)

var LABEL_KERNEL_RUNNING = "mondoo.com/os/kernel-running"

// NewBom creates a BOM from a cnquery report
func NewBom(report *reporter.Report) ([]*sbom.Sbom, error) {
	return GenerateBom(report)
}

// GenerateBom generates a BOM from a cnquery json report collection
func GenerateBom(r *reporter.Report) ([]*sbom.Sbom, error) {
	if r == nil {
		return nil, nil
	}

	generator := &sbom.Generator{
		Vendor:  "Mondoo, Inc.",
		Name:    "cnquery",
		Version: cnquery.Version,
		Url:     "https://mondoo.com",
	}
	now := time.Now().UTC().Format(time.RFC3339)

	boms := []*sbom.Sbom{}
	for mrn := range r.Assets {
		asset := r.Assets[mrn]

		bom := &sbom.Sbom{
			Generator: generator,
			Timestamp: now,
			Status:    sbom.Status_STATUS_SUCCEEDED,
		}

		bom.Asset = &sbom.Asset{
			Name:        asset.Name,
			PlatformIds: nil,
			Platform:    &sbom.Platform{},
			Labels:      map[string]string{},
			ExternalIds: []*sbom.ExternalID{},
			TraceId:     asset.TraceId,
		}

		bom.Packages = []*sbom.Package{}

		// extract os packages and python packages
		dataPoints := r.Data[mrn]
		if dataPoints == nil {
			bom.Status = sbom.Status_STATUS_FAILED
			bom.ErrorMessage = "no data points found"
			boms = append(boms, bom)
			continue
		}
		if dataPoints != nil {
			for k := range dataPoints.Values {
				dataValue := dataPoints.Values[k]
				jsondata, err := reporter.JsonValue(dataValue.Content)
				if err != nil {
					return nil, err
				}
				rb := BomFields{}
				err = json.Unmarshal(jsondata, &rb)
				if err != nil {
					return nil, err
				}
				if rb.Asset != nil {
					bom.Asset.Name = rb.Asset.Name
					bom.Asset.Platform.Name = rb.Asset.Platform
					bom.Asset.Platform.Version = rb.Asset.Version
					bom.Asset.Platform.Family = rb.Asset.Family
					bom.Asset.Platform.Arch = rb.Asset.Arch
					bom.Asset.Platform.Cpes = rb.Asset.CPEs
					bom.Asset.Platform.Labels = rb.Asset.Labels
					bom.Asset.PlatformIds = enrichPlatformIds(rb.Asset.IDs)
				}

				if bom.Asset == nil {
					bom.Asset = &sbom.Asset{}
				}
				if bom.Asset.Labels == nil {
					bom.Asset.Labels = map[string]string{}
				}

				// store version of running kernel
				for _, kernel := range rb.KernelInstalled {
					if kernel.Running {
						bom.Asset.Labels[LABEL_KERNEL_RUNNING] = kernel.Version
					}
				}

				if rb.Packages != nil {
					for _, pkg := range rb.Packages {
						bomPkg := &sbom.Package{
							Name:         pkg.Name,
							Version:      pkg.Version,
							Architecture: pkg.Arch,
							Origin:       pkg.Origin,
							Purl:         pkg.Purl,
							Cpes:         pkg.CPEs,
							Type:         pkg.Format,
						}

						for _, filepath := range pkg.FilePaths {
							bomPkg.EvidenceList = append(bomPkg.EvidenceList, &sbom.Evidence{
								Type:  sbom.EvidenceType_EVIDENCE_TYPE_FILE,
								Value: filepath,
							})
						}

						bom.Packages = append(bom.Packages, bomPkg)
					}
				}

				for _, pkg := range rb.PythonPackages {
					bomPkg := &sbom.Package{
						Name:    pkg.Name,
						Version: pkg.Version,
						Purl:    pkg.Purl,
						Cpes:    pkg.CPEs,
						Type:    "pypi",
					}

					// deprecated path, all files are now in the FilePaths field
					// TODO: update once the python resource returns multiple results
					if pkg.FilePath != "" {
						bomPkg.EvidenceList = append(bomPkg.EvidenceList, &sbom.Evidence{
							Type:  sbom.EvidenceType_EVIDENCE_TYPE_FILE,
							Value: pkg.FilePath,
						})
					}

					for _, filepath := range pkg.FilePaths {
						bomPkg.EvidenceList = append(bomPkg.EvidenceList, &sbom.Evidence{
							Type:  sbom.EvidenceType_EVIDENCE_TYPE_FILE,
							Value: filepath,
						})
					}

					bom.Packages = append(bom.Packages, bomPkg)
				}

				for _, pkg := range rb.NpmPackages {
					bomPkg := &sbom.Package{
						Name:    pkg.Name,
						Version: pkg.Version,
						Purl:    pkg.Purl,
						Cpes:    pkg.CPEs,
						Type:    "npm",
					}

					for _, filepath := range pkg.FilePaths {
						bomPkg.EvidenceList = append(bomPkg.EvidenceList, &sbom.Evidence{
							Type:  sbom.EvidenceType_EVIDENCE_TYPE_FILE,
							Value: filepath,
						})
					}

					bom.Packages = append(bom.Packages, bomPkg)
				}
			}
		}
		boms = append(boms, bom)
	}
	return boms, nil
}

// enrichPlatformIds adds the platform id based on cnquery ids
// - AWS EC2 instance ARN
func enrichPlatformIds(ids []string) []string {
	platformIds := []string{}
	for i := range ids {
		platformIds = append(platformIds, ids[i])

		// handle AWS EC2 instance platform identifier and generate AWS ARN as additional identifier
		// EC2 arns have the following format arn:aws:ec2:<REGION>:<ACCOUNT_ID>:instance/<instance-id>
		// //platformid.api.mondoo.app/runtime/aws/ec2/v1/accounts/12345678910/regions/us-east-1/instances/i-1234567890abcdef0
		if strings.HasPrefix(ids[i], "//platformid.api.mondoo.app/runtime/aws/ec2/v1") {
			ec2mrn, err := mrn.NewMRN(ids[i])
			if err != nil {
				continue
			}

			accountID, _ := ec2mrn.ResourceID("accounts")
			region, _ := ec2mrn.ResourceID("regions")
			instanceID, _ := ec2mrn.ResourceID("instances")

			if accountID != "" && region != "" && instanceID != "" {
				platformIds = append(platformIds, fmt.Sprintf("arn:aws:ec2:%s:%s:instance/%s", region, accountID, instanceID))
			}
		}
	}
	return platformIds
}
