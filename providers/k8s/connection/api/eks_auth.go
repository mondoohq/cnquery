// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package api

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/rs/zerolog/log"
	"k8s.io/client-go/rest"
)

const (
	// eksTokenPrefix is the required prefix for EKS bearer tokens
	eksTokenPrefix = "k8s-aws-v1."

	// clusterIDHeader is the EKS-specific header for cluster identification
	clusterIDHeader = "x-k8s-aws-id"

	// eksTokenExpiry is the presigned URL expiry, matching aws eks get-token (60 seconds).
	// EKS requires the X-Amz-Expires parameter in the presigned URL.
	eksTokenExpiry = 60 * time.Second
)

// attemptEKSAuthFlow detects EKS clusters and obtains a bearer token using AWS credentials.
// This bypasses the need for the aws CLI to be installed, similar to how attemptGKEAuthFlow
// bypasses the need for gke-gcloud-auth-plugin.
func attemptEKSAuthFlow(config *rest.Config) error {
	if config.ExecProvider == nil {
		return nil
	}

	// Detect EKS by checking if the exec command is "aws" with "eks" and "get-token" args
	if config.ExecProvider.Command != "aws" {
		return nil
	}

	hasEKS := false
	hasGetToken := false
	var clusterName, region string
	args := config.ExecProvider.Args
	for i, arg := range args {
		switch {
		case arg == "eks":
			hasEKS = true
		case arg == "get-token":
			hasGetToken = true
		case arg == "--cluster-name" && i+1 < len(args):
			clusterName = args[i+1]
		case strings.HasPrefix(arg, "--cluster-name="):
			clusterName = strings.TrimPrefix(arg, "--cluster-name=")
		case arg == "--region" && i+1 < len(args):
			region = args[i+1]
		case strings.HasPrefix(arg, "--region="):
			region = strings.TrimPrefix(arg, "--region=")
		}
	}

	if !hasEKS || !hasGetToken {
		return nil
	}

	if clusterName == "" {
		return fmt.Errorf("could not determine EKS cluster name from exec provider args")
	}

	log.Debug().Str("cluster", clusterName).Str("region", region).
		Msg("detected EKS cluster, attempting to get bearer token using AWS credentials")

	token, err := getEKSToken(context.Background(), clusterName, region)
	if err != nil {
		// Fall through and let the exec provider attempt to run (e.g. aws CLI).
		// This mirrors the GKE auth flow behavior of not blocking when credentials
		// are misconfigured.
		log.Warn().Err(err).Str("cluster", clusterName).
			Msg("failed to get EKS bearer token using AWS credentials, falling back to exec provider")
		return nil
	}

	config.BearerToken = token

	// Clear the exec provider since we've obtained the token directly,
	// bypassing the need for the aws CLI
	config.ExecProvider = nil

	log.Debug().Str("cluster", clusterName).Msg("successfully obtained EKS bearer token using AWS credentials")

	return nil
}

// getEKSToken generates a bearer token for EKS authentication.
// EKS tokens are base64-encoded presigned STS GetCallerIdentity URLs with a cluster ID header.
// This is the same mechanism used by `aws eks get-token` and `aws-iam-authenticator`.
func getEKSToken(ctx context.Context, clusterName string, region string) (string, error) {
	var opts []func(*awsconfig.LoadOptions) error
	if region != "" {
		opts = append(opts, awsconfig.WithRegion(region))
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return "", fmt.Errorf("loading AWS config: %w", err)
	}

	stsClient := sts.NewFromConfig(cfg)

	// Create a presigned GetCallerIdentity request with the cluster ID header.
	// The x-k8s-aws-id header binds the token to a specific cluster.
	presignClient := sts.NewPresignClient(stsClient, func(po *sts.PresignOptions) {
		po.ClientOptions = append(po.ClientOptions, func(o *sts.Options) {
			o.APIOptions = append(o.APIOptions, addClusterIDHeader(clusterName))
		})
	})

	// Set X-Amz-Expires via the presigner rather than a Build middleware, so the
	// V4 signer controls how the parameter appears in the final signed URL.
	presignedReq, err := presignClient.PresignGetCallerIdentity(ctx, &sts.GetCallerIdentityInput{}, func(po *sts.PresignOptions) {
		po.Presigner = &eksPresigner{inner: po.Presigner, expires: eksTokenExpiry}
	})
	if err != nil {
		return "", fmt.Errorf("presigning GetCallerIdentity: %w", err)
	}

	// The EKS token is the presigned URL, base64url-encoded with the "k8s-aws-v1." prefix.
	// No padding is used (RawURLEncoding), matching the aws-iam-authenticator format.
	token := eksTokenPrefix + base64.RawURLEncoding.EncodeToString([]byte(presignedReq.URL))

	return token, nil
}

// addClusterIDHeader returns a middleware function that adds the x-k8s-aws-id header
// to the presigned STS request. The header is required by the EKS API server to
// validate that the token was generated for the correct cluster.
func addClusterIDHeader(clusterName string) func(*middleware.Stack) error {
	return func(stack *middleware.Stack) error {
		return stack.Build.Add(middleware.BuildMiddlewareFunc("EKSClusterID",
			func(ctx context.Context, input middleware.BuildInput, next middleware.BuildHandler) (middleware.BuildOutput, middleware.Metadata, error) {
				if req, ok := input.Request.(*smithyhttp.Request); ok {
					req.Header.Set(clusterIDHeader, clusterName)
				}
				return next.HandleBuild(ctx, input)
			},
		), middleware.After)
	}
}

// eksPresigner wraps an STS presigner to add X-Amz-Expires to presigned requests.
// The X-Amz-Expires parameter is required by the EKS token verifier; without it
// the presigned URL is rejected with 401 Unauthorized.
// Setting it at the presigner level ensures the V4 signer includes it in the
// canonical request rather than relying on Build middleware ordering.
type eksPresigner struct {
	inner   sts.HTTPPresignerV4
	expires time.Duration
}

func (p *eksPresigner) PresignHTTP(
	ctx context.Context, credentials aws.Credentials, r *http.Request,
	payloadHash string, service string, region string, signingTime time.Time,
	optFns ...func(*v4.SignerOptions),
) (string, http.Header, error) {
	q := r.URL.Query()
	q.Set("X-Amz-Expires", strconv.FormatInt(int64(p.expires/time.Second), 10))
	r.URL.RawQuery = q.Encode()
	return p.inner.PresignHTTP(ctx, credentials, r, payloadHash, service, region, signingTime, optFns...)
}
