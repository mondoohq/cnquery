// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"golang.org/x/oauth2"
	"google.golang.org/api/idtoken"
)

type TokenSourceProvider func(ctx context.Context) (oauth2.TokenSource, error)

func AwsWithIdentityProvider(opts map[string]string) (*aws.Config, error) {
	if opts[IdentityRoleArnKey] == "" || opts[GcpInstanceIdentityKey] == "" {
		return nil, errors.New("missing information for identity")
	}
	tokenSourceProvider := GcpInstanceIdentity(opts[GcpInstanceIdentityKey])
	config := &Config{
		AssumeRole: &AssumeRoleConfig{
			RoleArn:             opts[IdentityRoleArnKey],
			TokenSourceProvider: tokenSourceProvider,
		},
	}
	a, err := AwsConfigProvider(config)
	if err != nil {
		return nil, err
	}
	cfg := a.Load()
	return &cfg, nil
}

// GcpInstanceIdentity returns a token source provider which generates
// OIDC identity tokens using the gcp instance metadata endpoint
func GcpInstanceIdentity(audience string) TokenSourceProvider {
	return func(ctx context.Context) (oauth2.TokenSource, error) {
		return idtoken.NewTokenSource(ctx, audience)
	}
}

// WebIdentityRoleCredentialsProvider returns an aws.CredentialsProvider which uses the
// a token source created from the given token source provider. The returned credentials
// provider exchanges this identity for aws credentials using sts
func WebIdentityRoleCredentialsProvider(ctx context.Context, defaultRegion string, roleARN string,
	tokenSourceProvider TokenSourceProvider,
) (aws.CredentialsProvider, error) {
	tokenSource, err := tokenSourceProvider(ctx)
	if err != nil {
		return nil, err
	}

	awsConfig := aws.NewConfig().Copy()
	awsConfig.Region = defaultRegion
	stsClient := sts.NewFromConfig(awsConfig)

	tokenRetriever := &identityTokenRetriever{
		tokenSource: tokenSource,
	}

	provider := stscreds.NewWebIdentityRoleProvider(stsClient, roleARN, tokenRetriever)

	return aws.NewCredentialsCache(provider), nil
}

type configProvider struct {
	config aws.Config
}

type loadOptions struct {
	region string
}
type LoadOpt func(*loadOptions)

type AssumeRoleConfig struct {
	RoleArn             string
	TokenSourceProvider TokenSourceProvider
}

type Config struct {
	DefaultRegion string
	AssumeRole    *AssumeRoleConfig
}
type ConfigProvider interface {
	Load(...LoadOpt) aws.Config
}

type configProviderOpts struct {
	credsProvider aws.CredentialsProvider
	region        string
}
type ConfigProviderOpt func(*configProviderOpts)

func WithCredentialsProvider(credsProvider aws.CredentialsProvider) ConfigProviderOpt {
	return func(p *configProviderOpts) {
		p.credsProvider = credsProvider
	}
}

func WithDefaultRegion(region string) ConfigProviderOpt {
	return func(p *configProviderOpts) {
		p.region = region
	}
}

// NewConfigProvider returns a config provider with the given options.
// For example, you can create a config that gets aws credentials using
// gcp instance credentials:
//
//	NewConfigProvider(ctx, WithCredentialsProvider(
//	  WebIdentityRoleCredentialsProvider(GcpInstanceIdentity("service_account_id"))
//	))
//
// or use the default credential lookup chain provided by aws libraries:
//
//	NewConfigProvider(ctx)
func NewConfigProvider(ctx context.Context, opts ...ConfigProviderOpt) (ConfigProvider, error) {
	popts := &configProviderOpts{}
	for _, o := range opts {
		o(popts)
	}

	c, err := config.LoadDefaultConfig(ctx,
		config.WithDefaultRegion(popts.region),
		config.WithCredentialsProvider(popts.credsProvider))
	if err != nil {
		return nil, err
	}
	return &configProvider{
		config: c,
	}, nil
}

func (p *configProvider) Load(opts ...LoadOpt) aws.Config {
	loadOpts := &loadOptions{}
	for _, o := range opts {
		o(loadOpts)
	}
	if loadOpts.region == "" {
		return p.config
	}
	c := p.config.Copy()
	c.Region = loadOpts.region
	return c
}

type identityTokenRetriever struct {
	tokenSource oauth2.TokenSource
}

func (p *identityTokenRetriever) GetIdentityToken() ([]byte, error) {
	token, err := p.tokenSource.Token()
	if err != nil {
		return nil, err
	}
	return []byte(token.AccessToken), nil
}

func AwsConfigProvider(config *Config) (ConfigProvider, error) {
	configProviderOpts := []ConfigProviderOpt{}
	if config != nil {
		if config.DefaultRegion != "" {
			configProviderOpts = append(configProviderOpts, WithDefaultRegion(config.DefaultRegion))
		}
		if config.AssumeRole != nil {
			o, err := WebIdentityRoleCredentialsProvider(
				context.Background(),
				config.DefaultRegion,
				config.AssumeRole.RoleArn,
				config.AssumeRole.TokenSourceProvider,
			)
			if err != nil {
				return nil, err
			}
			configProviderOpts = append(configProviderOpts, WithCredentialsProvider(o))
		}
	}
	awsConfigProvider, err := NewConfigProvider(context.Background(), configProviderOpts...)
	if err != nil {
		return nil, err
	}
	return awsConfigProvider, err
}
