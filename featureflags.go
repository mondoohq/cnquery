//
// In this file we introduce feature flags. They help us activate optional
// features on requests.
//
// Features can only be activated, never deactivated. Features are efficiently encoded.
// They are introduced at a given version and destined to be removed at a later version.
// Please mark them accordingly. Feature flags are short term contextual information.
//
// Example usage:
//
// features := []Feature{ MassResolver, LiveQueries }
//
// features.IsActive( MassResolver )   // true
//

package cnquery

//go:generate go run golang.org/x/tools/cmd/stringer -type=Feature

import (
	"bytes"
	"context"
	"encoding/base64"
)

const (
	// For all features, use this format:
	// desc:   A description of this feature and what it does...
	// start:  vX.x  (the version when it ix introduced)
	// end:    vZ.0  (the version when this flag will be removed)

	// Feature flags:

	// MassQueries feature flag
	// desc:   Resolve similar queries the same way. If 100 assets have the same
	//         dependent queries and overrides, they create the same resolved
	//         plan. Cannot be used with old resolver at the same time for asset.
	// start:  v3.x, available at v4.x, default at v5.x
	// end:    v6.0
	MassQueries Feature = iota + 1

	// PiperCode feature flag
	// desc:   Allows MQL to use variable references across blocks. Fully changes
	//         the compiled code.
	// start:  v5.x
	// end:    v7.0
	PiperCode

	// BoolAssertions feature flag
	// desc:  Only boolean results are checked when evaluating a query for success
	//
	// start: v6.x
	// end:   v8.0
	BoolAssertions

	// K8sNodeDiscovery feature flag
	// desc:  Enables discovery of Kubernetes cluster nodes as individual assets
	//
	// start: v6.12
	// end:   unknown
	K8sNodeDiscovery
)

// FeaturesValue is a map from feature name to feature flag
var FeaturesValue = map[string]Feature{
	MassQueries.String():    MassQueries,
	PiperCode.String():      PiperCode,
	BoolAssertions.String(): BoolAssertions,
}

// DefaultFeatures are a set of default flags that are active
var DefaultFeatures = Features{
	byte(MassQueries),
	byte(PiperCode),
}

// Features is a collection of activated features
type Features []byte

// Feature is a simple feature flag
type Feature byte

// IsActive returns true if the given feature has been requested in this list
func (f Features) IsActive(feature Feature) bool {
	return bytes.IndexByte(f, byte(feature)) != -1
}

// Encode a set of features to base64
func (f Features) Encode() string {
	return base64.StdEncoding.EncodeToString(f)
}

// DecodeFeatures that were previously encoded
func DecodeFeatures(s string) (Features, error) {
	data, err := base64.StdEncoding.DecodeString(s)
	return Features(data), err
}

type featureContextID struct{}

// SetFeatures to a given context
func SetFeatures(ctx context.Context, fts Features) context.Context {
	return context.WithValue(ctx, featureContextID{}, fts)
}

// GetFeatures from a given context
func GetFeatures(ctx context.Context) Features {
	f, ok := ctx.Value(featureContextID{}).(Features)
	if !ok {
		// nothing stored, assume empty features
		return Features{}
	}
	return f
}
