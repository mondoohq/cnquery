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

	// desc:   Resolve similar policies the same way. If 100 assets have the same
	//         dependent policies and overrides, they create the same resolved
	//         policy. Cannot be used with old resolver at the same time for asset.
	// start:  v3.x, available at v4.x, default at v5.x
	// end:    v6.0
	MassQueries Feature = iota + 1

	// desc:   Allows MQL to use variable references across blocks. Fully changes
	//         the compiled code.
	// start:  v5.x
	// end:    v7.0
	PiperCode

	// desc:  Only boolean results are checked when evaluating a query for success
	//
	// start: v6.x
	// end:   v8.0
	BoolAssertions
)

// map of feature name to byte
var FeaturesValue = map[string]Feature{
	MassQueries.String():    MassQueries,
	PiperCode.String():      PiperCode,
	BoolAssertions.String(): BoolAssertions,
}

var DefaultFeatures = Features{
	byte(MassQueries),
	byte(PiperCode),
}

// Features is a collection of activated features
type Features []byte

// A simple feature
type Feature byte

// IsActive returns true if the given feature has been requested in this list
func (f Features) IsActive(feature Feature) bool {
	return bytes.IndexByte(f, byte(feature)) != -1
}

func (f Features) Encode() string {
	return base64.StdEncoding.EncodeToString(f)
}

func DecodeFeatures(s string) (Features, error) {
	data, err := base64.StdEncoding.DecodeString(s)
	return Features(data), err
}

type FeatureContextId struct{}

func SetFeatures(ctx context.Context, fts Features) context.Context {
	return context.WithValue(ctx, FeatureContextId{}, fts)
}

func GetFeatures(ctx context.Context) Features {
	f, ok := ctx.Value(FeatureContextId{}).(Features)
	if !ok {
		// nothing stored, assume empty features
		return Features{}
	}
	return f
}
