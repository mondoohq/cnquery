package k8s

import (
	"encoding/base64"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdmissionProvider(t *testing.T) {
	manifestFile := "./resources/testdata/admission-review.json"
	data, err := os.ReadFile(manifestFile)
	require.NoError(t, err)

	transport, err := newAdmissionProvider(base64.StdEncoding.EncodeToString(data), "", "")
	require.NoError(t, err)
	require.NotNil(t, transport)
	res, err := transport.AdmissionReviews()
	require.NoError(t, err)
	assert.Len(t, res, 1)
	ns, err := transport.Namespaces()
	require.NoError(t, err)
	assert.Len(t, ns, 1)
	pods, err := transport.Pods(ns[0])
	require.NoError(t, err)
	assert.Len(t, pods, 1)
	assert.Equal(t, "Kubernetes Admission", transport.PlatformInfo().Title)
	assert.Equal(t, "k8s-admission", transport.PlatformInfo().Runtime)
	name, err := transport.Name()
	require.NoError(t, err)
	assert.Equal(t, "K8S Admission review "+res[0].Request.Name, name)
}
