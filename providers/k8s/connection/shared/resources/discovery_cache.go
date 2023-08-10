package resources

import (
	"sync"

	"k8s.io/client-go/rest"
)

type DiscoveryCache struct {
	mx    sync.Mutex
	cache map[string]*Discovery
}

func NewDiscoveryCache() *DiscoveryCache {
	return &DiscoveryCache{
		cache: make(map[string]*Discovery),
	}
}

func (d *DiscoveryCache) Get(config *rest.Config) (*Discovery, error) {
	d.mx.Lock()
	defer d.mx.Unlock()
	if d.cache[config.Host] != nil {
		return d.cache[config.Host], nil
	}

	discovery, err := NewDiscovery(config)
	if err != nil {
		return nil, err
	}

	d.cache[config.Host] = discovery

	return discovery, nil
}
