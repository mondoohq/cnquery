package lumicontext

import (
	"context"
	"errors"

	"go.mondoo.io/mondoo/falcon"
)

// mondoo cloud config so that resource scan talk upstream
type CloudConfig struct {
	SpaceMrn    string
	Collector   string
	ApiEndpoint string
	Plugins     []falcon.ClientPlugin
	Incognito   bool
}

type contextCloudConfigType struct{}

var contextCloudConfigKey = &contextCloudConfigType{}

// WithCloudConfig puts the cloud config into the current context.
func WithCloudConfig(ctx context.Context, mcc *CloudConfig) context.Context {
	return context.WithValue(ctx, contextCloudConfigKey, mcc)
}

// CloudConfigFromContext returns the cloud config from the context.
// An error is returned if there is no cloud config in the
// current context.
func CloudConfigFromContext(ctx context.Context) (*CloudConfig, error) {
	v := ctx.Value(contextCloudConfigKey)
	if v == nil {
		return nil, errors.New("no cloud config in context")
	}
	return v.(*CloudConfig), nil
}

type contextAssetMrnType struct{}

var contextAssetMrnKey = &contextAssetMrnType{}

// WithAssetMrn puts the assetMrn the current context.
func WithAssetMrn(ctx context.Context, assetMrn string) context.Context {
	return context.WithValue(ctx, contextAssetMrnKey, assetMrn)
}

// AssetMrnFromContext returns the asset mrn from the context.
// An empty string is returned if there is no asset mrn in the
// current context.
func AssetMrnFromContext(ctx context.Context) string {
	v := ctx.Value(contextAssetMrnKey)
	if v == nil {
		return ""
	}
	return v.(string)
}
