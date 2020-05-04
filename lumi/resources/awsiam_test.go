package resources_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go/aws/endpoints"
)

func TestResource_AwsIamCredentialReport(t *testing.T) {
	t.Run("run a aws iam credential report", func(t *testing.T) {
		res := testQuery(t, "aws.iam.credentialreport.length")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.True(t, res[0].Data.Value.(int64) == 2)
	})

	t.Run("ask details about an iam credential report entry", func(t *testing.T) {
		res := testQuery(t, "aws.iam.credentialreport[0]['user']")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.True(t, res[0].Data.Value.(string) == "<root_account>")
	})

	t.Run("use where for credential report", func(t *testing.T) {
		res := testQuery(t, "aws.iam.credentialreport.where( _['user'] == '<root_account>')")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.True(t, res[0].Data.Value.(string) == "<root_account>")
	})
}
