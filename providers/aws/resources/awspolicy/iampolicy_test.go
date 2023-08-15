package awspolicy

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIamPolicies(t *testing.T) {
	files := []string{
		"./testdata/iam_policy1.json",
		"./testdata/iam_policy2.json",
	}

	for _, f := range files {
		data, err := os.ReadFile(f)
		require.NoError(t, err, f)

		var policy IamPolicyDocument
		err = json.Unmarshal(data, &policy)
		require.NoError(t, err, f)
	}
}
