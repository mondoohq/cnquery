package awsparameterstore

// import (
// 	"context"
// 	"os"
// 	"testing"

// 	"github.com/aws/aws-sdk-go-v2/aws/external"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/require"
// 	"go.mondoo.io/mondoo/motor/vault"
// )

// func TestAwsParameterStore(t *testing.T) {

// 	os.Setenv("AWS_PROFILE", "mondoo-dev")
// 	os.Setenv("AWS_REGION", "us-east-1")
// 	cfg, err := external.LoadDefaultAWSConfig()
// 	require.NoError(t, err)
// 	v := New(cfg)
// 	ctx := context.Background()

// 	cred := &vault.Credential{
// 		Key: vault.Mrn2secretKey("//platformid.api.mondoo.app/runtime/aws/ec2/v1/accounts/675173580680/regions/eu-west-1/instances/i-0e11b0762369fbefa"),
// 		Fields: map[string]string{
// 			"key":  "value1",
// 			"key2": "value2",
// 		},
// 	}

// 	id, err := v.Set(ctx, cred)
// 	require.NoError(t, err)

// 	newCred, err := v.Get(ctx, id)
// 	require.NoError(t, err)
// 	assert.Equal(t, cred, newCred)
// }
