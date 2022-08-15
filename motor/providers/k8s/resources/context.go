package resources

import (
	"context"
)

type k8sDiscoveryCache struct{}

func GetDiscoveryCache(ctx context.Context) (*DiscoveryCache, bool) {
	c, ok := ctx.Value(k8sDiscoveryCache{}).(*DiscoveryCache)
	return c, ok
}

func SetDiscoveryCache(ctx context.Context, dCache *DiscoveryCache) context.Context {
	return context.WithValue(ctx, k8sDiscoveryCache{}, dCache)
}
