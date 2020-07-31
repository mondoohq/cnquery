package awspolicy

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestS3BucketPolicies(t *testing.T) {

	files := []string{
		"./testdata/s3bucket_policy1.json",
		"./testdata/s3bucket_policy2.json",
		"./testdata/s3bucket_policy3.json",
		"./testdata/s3bucket_policy_array.json",
		"./testdata/s3bucket_multistatement.json",
		"./testdata/s3bucket_multielements.json",
		"./testdata/s3bucket_multiblock.json",
		"./testdata/s3bucket_2008_public.json",
		"./testdata/s3bucket_2008_iprestriction.json",
	}

	for _, f := range files {

		data, err := ioutil.ReadFile(f)
		require.NoError(t, err, f)

		var policy S3BucketPolicy
		err = json.Unmarshal(data, &policy)
		require.NoError(t, err, f)
	}

}
