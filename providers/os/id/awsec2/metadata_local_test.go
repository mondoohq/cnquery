// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package awsec2

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/stretchr/testify/assert"
)

func fakeConfig() aws.Config {
	conf := aws.Config{}
	conf.Region = "mock-region"
	localResolverFn := func(service, region string) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: "https://endpoint",
		}, nil
	}
	conf.EndpointResolver = aws.EndpointResolverFunc(localResolverFn)
	conf.Credentials = credentials.StaticCredentialsProvider{
		Value: aws.Credentials{
			AccessKeyID: "AKID", SecretAccessKey: "SECRET", SessionToken: "SESSION",
			Source: "unit test credentials",
		},
	}
	return conf
}

func TestEC2RoleProviderInstanceIdentityLocal(t *testing.T) {
	instanceIdentityDocument, err := os.ReadFile("./testdata/instance-identity-document.json")
	if err != nil {
		t.Fatal(err)
	}

	cfg := fakeConfig()
	cfg.HTTPClient = smithyhttp.ClientDoFunc(func(r *http.Request) (*http.Response, error) {
		url := r.URL.String()
		if strings.Contains(url, "latest/api/token") {
			return &http.Response{
				StatusCode: 200,
				Header:     http.Header{},
				Body:       io.NopCloser(bytes.NewBufferString("mock-token")),
			}, nil
		}
		if strings.Contains(url, "tags/instance/Name") {
			return &http.Response{
				StatusCode: 200,
				Header:     http.Header{},
				Body:       io.NopCloser(bytes.NewBufferString("ec2-name")),
			}, nil
		}
		return &http.Response{
			StatusCode: 200,
			Header:     http.Header{},
			Body:       io.NopCloser(bytes.NewReader(instanceIdentityDocument)),
		}, nil
	})

	metadata := NewLocal(cfg)
	ident, err := metadata.Identify()
	assert.Nil(t, err)
	assert.Equal(t, "ec2-name", ident.InstanceName)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/aws/ec2/v1/accounts/123456789012/regions/us-west-2/instances/i-1234567890abcdef0", ident.InstanceID)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/aws/accounts/123456789012", ident.AccountID)
}

func TestEC2RoleProviderInstanceIdentityLocalDisabledTagsService(t *testing.T) {
	instanceIdentityDocument, err := os.ReadFile("./testdata/instance-identity-document.json")
	if err != nil {
		t.Fatal(err)
	}

	cfg := fakeConfig()
	cfg.HTTPClient = smithyhttp.ClientDoFunc(func(r *http.Request) (*http.Response, error) {
		url := r.URL.String()
		if strings.Contains(url, "latest/api/token") {
			return &http.Response{
				StatusCode: 200,
				Header:     http.Header{},
				Body:       io.NopCloser(bytes.NewBufferString("mock-token")),
			}, nil
		}
		if strings.Contains(url, "tags/instance/Name") {
			return &http.Response{
				StatusCode: 404,
				Header:     http.Header{},
				Body:       io.NopCloser(bytes.NewBufferString("not enabled")),
			}, nil
		}
		return &http.Response{
			StatusCode: 200,
			Header:     http.Header{},
			Body:       io.NopCloser(bytes.NewReader(instanceIdentityDocument)),
		}, nil
	})

	metadata := NewLocal(cfg)
	ident, err := metadata.Identify()
	assert.Nil(t, err)
	assert.Equal(t, "i-1234567890abcdef0", ident.InstanceName)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/aws/ec2/v1/accounts/123456789012/regions/us-west-2/instances/i-1234567890abcdef0", ident.InstanceID)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/aws/accounts/123456789012", ident.AccountID)
}

func TestEC2RoleProviderInstanceRawMetadataLocal(t *testing.T) {
	cfg := fakeConfig()
	cfg.HTTPClient = smithyhttp.ClientDoFunc(func(r *http.Request) (*http.Response, error) {
		url := r.URL.String()
		if strings.HasSuffix(url, "latest/api/token") {
			return &http.Response{
				StatusCode: 200,
				Header:     http.Header{},
				Body:       io.NopCloser(bytes.NewBufferString("mock-token")),
			}, nil
		}
		// Root metadata listing
		if strings.HasSuffix(url, "/latest/meta-data") || strings.HasSuffix(url, "/latest/meta-data/") {
			return &http.Response{
				StatusCode: 200,
				Header:     http.Header{},
				Body:       io.NopCloser(bytes.NewBufferString("ami-id\ninstance-id\ninstance-type\nnetwork/\n")),
			}, nil
		}

		// Simple metadata values
		if strings.HasSuffix(url, "/latest/meta-data/instance-id") {
			return &http.Response{
				StatusCode: 200,
				Header:     http.Header{},
				Body:       io.NopCloser(bytes.NewBufferString("i-0abcd1234efgh5678")),
			}, nil
		}
		if strings.HasSuffix(url, "/latest/meta-data/ami-id") {
			return &http.Response{
				StatusCode: 200,
				Header:     http.Header{},
				Body:       io.NopCloser(bytes.NewBufferString("ami-12345678")),
			}, nil
		}
		if strings.HasSuffix(url, "/latest/meta-data/instance-type") {
			return &http.Response{
				StatusCode: 200,
				Header:     http.Header{},
				Body:       io.NopCloser(bytes.NewBufferString("t2.micro")),
			}, nil
		}

		// Nested metadata
		if strings.HasSuffix(url, "/latest/meta-data/network/") {
			return &http.Response{
				StatusCode: 200,
				Header:     http.Header{},
				Body:       io.NopCloser(bytes.NewBufferString("interfaces/")),
			}, nil
		}
		if strings.HasSuffix(url, "/latest/meta-data/network/interfaces/") {
			return &http.Response{
				StatusCode: 200,
				Header:     http.Header{},
				Body:       io.NopCloser(bytes.NewBufferString("macs/")),
			}, nil
		}
		if strings.HasSuffix(url, "/latest/meta-data/network/interfaces/macs/") {
			return &http.Response{
				StatusCode: 200,
				Header:     http.Header{},
				Body:       io.NopCloser(bytes.NewBufferString("12:34:56:78:9a:bc/")),
			}, nil
		}
		if strings.HasSuffix(url, "/latest/meta-data/network/interfaces/macs/12:34:56:78:9a:bc/") {
			return &http.Response{
				StatusCode: 200,
				Header:     http.Header{},
				Body:       io.NopCloser(bytes.NewBufferString("device-number\nlocal-ipv4s\n")),
			}, nil
		}
		if strings.HasSuffix(url, "/latest/meta-data/network/interfaces/macs/12:34:56:78:9a:bc/device-number") {
			return &http.Response{
				StatusCode: 200,
				Header:     http.Header{},
				Body:       io.NopCloser(bytes.NewBufferString("0")),
			}, nil
		}
		if strings.HasSuffix(url, "/latest/meta-data/network/interfaces/macs/12:34:56:78:9a:bc/local-ipv4s") {
			return &http.Response{
				StatusCode: 200,
				Header:     http.Header{},
				Body:       io.NopCloser(bytes.NewBufferString("192.168.1.100")),
			}, nil
		}

		return &http.Response{
			StatusCode: 404,
			Header:     http.Header{},
			Body:       io.NopCloser(bytes.NewBufferString("unexpected call")),
		}, nil
	})

	metadata := NewLocal(cfg)
	raw, err := metadata.RawMetadata()
	assert.Nil(t, err)
	// Convert to JSON for readability
	jsonData, _ := json.MarshalIndent(raw, "", "  ")
	expected := `{
  "ami-id": "ami-12345678",
  "instance-id": "i-0abcd1234efgh5678",
  "instance-type": "t2.micro",
  "network": {
    "interfaces": {
      "macs": {
        "12:34:56:78:9a:bc": {
          "device-number": 0,
          "local-ipv4s": "192.168.1.100"
        }
      }
    }
  }
}`

	// Compare actual vs expected JSON output
	assert.JSONEq(t, expected, string(jsonData))
}
