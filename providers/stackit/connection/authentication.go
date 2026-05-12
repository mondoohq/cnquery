// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

// CLI flag names. The matching env vars below mirror the ones the STACKIT SDK
// already recognizes natively, so users with a working SDK setup don't have to
// pass anything extra.
const (
	OptionProjectID             = "project-id"
	OptionRegion                = "region"
	OptionEndpoint              = "endpoint"
	OptionToken                 = "token"
	OptionServiceAccountKey     = "service-account-key"
	OptionServiceAccountKeyPath = "service-account-key-path"
	OptionPrivateKey            = "private-key"
	OptionPrivateKeyPath        = "private-key-path"

	ProjectIDEnvVar             = "STACKIT_PROJECT_ID"
	RegionEnvVar                = "STACKIT_REGION"
	EndpointEnvVar              = "STACKIT_ENDPOINT"
	TokenEnvVar                 = "STACKIT_SERVICE_ACCOUNT_TOKEN"
	ServiceAccountKeyEnvVar     = "STACKIT_SERVICE_ACCOUNT_KEY"
	ServiceAccountKeyPathEnvVar = "STACKIT_SERVICE_ACCOUNT_KEY_PATH"
	PrivateKeyEnvVar            = "STACKIT_PRIVATE_KEY"
	PrivateKeyPathEnvVar        = "STACKIT_PRIVATE_KEY_PATH"
)
