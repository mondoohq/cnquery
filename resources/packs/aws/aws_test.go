package aws_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/resources/packs/aws"
)

func TestArnParsing(t *testing.T) {
	arn1 := "arn:aws:es:us-east-2:921877552404:domain/test"
	res, err := aws.GetRegionFromArn(arn1)
	require.NoError(t, err)
	assert.Equal(t, res, "us-east-2")

	arn2 := "arn:aws:elasticloadbalancing:eu-west-1:921877552404:loadbalancer/classic/testname"
	res, err = aws.GetRegionFromArn(arn2)
	require.NoError(t, err)
	assert.Equal(t, res, "eu-west-1")
}
