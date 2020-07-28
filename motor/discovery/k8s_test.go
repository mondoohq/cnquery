package discovery

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseK8SURI(t *testing.T) {
	config := ParseK8SContext("k8s://")
	assert.Equal(t, "", config.Context)
	assert.Equal(t, "", config.Namespace)

	config = ParseK8SContext("k8s://context/c1")
	assert.Equal(t, "c1", config.Context)
	assert.Equal(t, "", config.Namespace)

	config = ParseK8SContext("k8s://context/c1/namespace/n1")
	assert.Equal(t, "c1", config.Context)
	assert.Equal(t, "n1", config.Namespace)

	config = ParseK8SContext("k8s://namespace/n1")
	assert.Equal(t, "", config.Context)
	assert.Equal(t, "n1", config.Namespace)
}
