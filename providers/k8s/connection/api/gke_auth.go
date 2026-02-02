// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package api

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2/google"
	"k8s.io/client-go/rest"
)

const (
	// GCP scope required for GKE cluster access
	gcpCloudPlatformScope = "https://www.googleapis.com/auth/cloud-platform"
)

// attemptGKEAuthFlow detects GKE clusters and obtains a bearer token using GCP credentials.
// This bypasses the need for gke-gcloud-auth-plugin to be installed.
func attemptGKEAuthFlow(config *rest.Config) error {
	// Auto-detect GKE by checking if the ExecProvider references gke-gcloud-auth-plugin
	if config.ExecProvider == nil {
		return nil
	}
	if !strings.Contains(config.ExecProvider.Command, "gke-gcloud-auth-plugin") {
		return nil
	}

	log.Debug().Msg("detected GKE cluster, attempting to get bearer token using GCP credentials")

	// Get GCP credentials using the default chain:
	// 1. GOOGLE_APPLICATION_CREDENTIALS env var
	// 2. gcloud CLI credentials (~/.config/gcloud/application_default_credentials.json)
	// 3. GCE metadata service (when running on GCP)
	ctx := context.Background()
	creds, err := google.FindDefaultCredentials(ctx, gcpCloudPlatformScope)
	if err != nil {
		return errors.Wrap(err, "failed to get GCP credentials for GKE authentication")
	}

	token, err := creds.TokenSource.Token()
	if err != nil {
		return errors.Wrap(err, "failed to get access token for GKE authentication")
	}

	config.BearerToken = token.AccessToken

	// Clear the exec provider since we've obtained the token directly,
	// bypassing the need for gke-gcloud-auth-plugin
	config.ExecProvider = nil

	log.Debug().Msg("successfully obtained GKE bearer token using GCP credentials")

	return nil
}
