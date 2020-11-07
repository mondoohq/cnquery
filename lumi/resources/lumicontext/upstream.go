package lumicontext

import (
	"context"
	"errors"

	"go.mondoo.io/mondoo/falcon"
)

// mondoo cloud config so that resource scan talk upstream
type CloudConfig struct {
	AssetMrn    string // optional, not set in shell yet
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
