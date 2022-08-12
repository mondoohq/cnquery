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

	v := []ProviderType{}
	yaml.Unmarshal([]byte(content), &v)

	assert.Equal(t, 4, len(v))
	assert.Equal(t, ProviderType_SSH, v[0])
	assert.Equal(t, ProviderType_DOCKER, v[1])
	assert.Equal(t, ProviderType_TAR, v[2])
	assert.Equal(t, ProviderType_TAR, v[3])
}
