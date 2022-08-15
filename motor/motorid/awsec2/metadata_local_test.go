package awsec2_test

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/motor/motorid/awsec2"
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
	instanceIdentityDocument, err := ioutil.ReadFile("./testdata/instance-identity-document.json")
	if err != nil {
		t.Fatal(err)
	}

	cfg := fakeConfig()
	cfg.HTTPClient = smithyhttp.ClientDoFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Header:     http.Header{},
			Body:       ioutil.NopCloser(bytes.NewReader(instanceIdentityDocument)),
		}, nil
	})

	metadata := awsec2.NewLocal(cfg)
	ident, err := metadata.Identify()
	assert.Nil(t, err)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/aws/ec2/v1/accounts/123456789012/regions/us-west-2/instances/i-1234567890abcdef0", ident.InstanceID)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/aws/accounts/123456789012", ident.AccountID)
}
