package providers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"sigs.k8s.io/yaml"
)

func TestBackendParser(t *testing.T) {
	content := `
- ssh
- docker
- tar
-  tar 
`

	v := []TransportBackend{}
	yaml.Unmarshal([]byte(content), &v)

	assert.Equal(t, 4, len(v))
	assert.Equal(t, TransportBackend_CONNECTION_SSH, v[0])
	assert.Equal(t, TransportBackend_CONNECTION_DOCKER, v[1])
	assert.Equal(t, TransportBackend_CONNECTION_TAR, v[2])
	assert.Equal(t, TransportBackend_CONNECTION_TAR, v[3])
}
