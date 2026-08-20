// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"go.mondoo.com/mql/providers/aws/connection"
)

// isRootAccessNotConfigured reports whether the error means centralized root
// access has simply never been turned on, rather than that the read failed.
// IAM answers OrganizationNotFoundException when the calling account is not in
// an organization at all, and OrganizationNotInAllFeaturesModeException when
// the organization exists but runs in consolidated-billing mode, where the
// feature cannot be enabled. Both are "no root features are enabled", which is
// an answer about the organization, so they resolve to an empty list.
func isRootAccessNotConfigured(err error) bool {
	var noOrg *iamtypes.OrganizationNotFoundException
	var notAllFeatures *iamtypes.OrganizationNotInAllFeaturesModeException
	var svcAccessNotEnabled *iamtypes.ServiceAccessNotEnabledException
	return errors.As(err, &noOrg) ||
		errors.As(err, &notAllFeatures) ||
		errors.As(err, &svcAccessNotEnabled)
}

// enabledRootFeatures reports which centralized root access features the
// organization has turned on. Empty means none, which is the state CIS AWS
// Foundations 2.1.1 flags.
func (a *mqlAwsOrganization) enabledRootFeatures() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	client := conn.Iam("") // IAM is global, like every other aws.iam.* call

	resp, err := client.ListOrganizationsFeatures(context.Background(), &iam.ListOrganizationsFeaturesInput{})
	if err != nil {
		if isRootAccessNotConfigured(err) {
			return []any{}, nil
		}
		// A member account cannot read this. That leaves the answer unknown
		// rather than empty, so it stays an error: an empty list here would
		// read as "centralized root access is off", which is the finding.
		return nil, err
	}
	if resp == nil {
		return []any{}, nil
	}

	res := make([]any, 0, len(resp.EnabledFeatures))
	for _, feature := range resp.EnabledFeatures {
		res = append(res, string(feature))
	}
	return res, nil
}

// trustedAccessServicePrincipals lists the services allowed to act across the
// organization's accounts. This is the prerequisite for any service to be
// administered organization-wide, including IAM for centralized root access.
func (a *mqlAwsOrganization) trustedAccessServicePrincipals() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	client := conn.Organizations("")
	ctx := context.Background()

	res := []any{}
	paginator := organizations.NewListAWSServiceAccessForOrganizationPaginator(client,
		&organizations.ListAWSServiceAccessForOrganizationInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			// Consolidated-billing organizations cannot enable trusted access
			// for anything, so "no principals" is the correct answer there.
			if isPolicyTypeUnavailable(err) {
				return res, nil
			}
			return nil, err
		}
		for _, principal := range page.EnabledServicePrincipals {
			if principal.ServicePrincipal == nil {
				continue
			}
			res = append(res, *principal.ServicePrincipal)
		}
	}
	return res, nil
}
