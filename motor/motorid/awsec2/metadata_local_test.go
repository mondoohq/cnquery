package awsec2_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/defaults"
	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/motor/motorid/awsec2"
)

func fakeConfig() aws.Config {
	config := defaults.Config()
	config.Region = "mock-region"
	config.EndpointResolver = aws.ResolveWithEndpointURL("https://endpoint")
	config.Credentials = aws.StaticCredentialsProvider{
		Value: aws.Credentials{
			AccessKeyID: "AKID", SecretAccessKey: "SECRET", SessionToken: "SESSION",
			Source: "unit test credentials",
		},
	}
	return config
}

func initTestServer(path string, resp string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI != path {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		w.Write([]byte(resp))
	}))
}

func TestEC2RoleProviderInstanceIdentityLocal(t *testing.T) {
	instanceIdentityDocument, err := ioutil.ReadFile("./testdata/instance-identity-document.json")
	if err != nil {
		t.Fatal(err)
	}

	server := initTestServer(
		"/latest/dynamic/instance-identity/document",
		string(instanceIdentityDocument),
	)
	defer server.Close()

	cfg := fakeConfig()
	cfg.EndpointResolver = aws.ResolveWithEndpointURL(server.URL + "/latest")

	metadata := awsec2.NewLocal(cfg)
	mrn, err := metadata.InstanceID()
	assert.Nil(t, err)
	assert.Equal(t, "//platformid.api.mondoo.app/runtime/aws/ec2/v1/accounts/123456789012/regions/us-west-2/instances/i-1234567890abcdef0", mrn)
}
