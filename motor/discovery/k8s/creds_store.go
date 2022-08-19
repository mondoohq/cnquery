package k8s

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/providers/k8s"
	v1 "k8s.io/api/core/v1"
)

type cacheEntry struct {
	secret *v1.Secret
	err    error
}

type credsStore struct {
	provider k8s.KubernetesProvider
	cache    map[string]cacheEntry
}

func NewCredsStore(p k8s.KubernetesProvider) *credsStore {
	return &credsStore{
		provider: p,
		cache:    make(map[string]cacheEntry),
	}
}

// Get retrieves the secret with the provided namespace and name. The value is retrieved
// once and is cached. All consecutive calls will retrieve the cached value. Note that the
// implementation is not thread-safe.
func (c *credsStore) Get(namespace, name string) (*v1.Secret, error) {
	key := credsStoreKey(namespace, name)
	if s, ok := c.cache[key]; ok {
		return s.secret, s.err
	}

	s, err := c.provider.Secret(namespace, name)
	// We log the warning here to make sure we don't log the same message for every pod that uses
	// the same pull secret.
	if err != nil {
		log.Warn().Msgf(
			"cannot read image pull secret %s/%s from cluster. Image pulling might now work", namespace, name)
	}
	c.cache[key] = cacheEntry{secret: s, err: err}
	return s, err
}

func credsStoreKey(namespace, name string) string {
	return fmt.Sprintf("%s:%s", namespace, name)
}
