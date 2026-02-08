// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

// In this file we introduce feature flags.
// - Please configure any new feature-flags in features.yaml
// - To generate, use go generate. See the call to go:generate below
// - To learn more about the generator, look at ./utils/featureflags/main.go header
//
// Example usage:
//
// features := []Feature{ MassResolver, LiveQueries }
// features.IsActive( MassResolver )   // true

package mql

//go:generate go run utils/featureflags/main.go features.yaml -type=Feature -out=features.go
//go:generate go run golang.org/x/tools/cmd/stringer -type=Feature

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"strings"
)

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

// String returns a list of features into human-readable form
func (f Features) String() string {
	all := make([]string, len(f))
	for i, cur := range f {
		all[i] = Feature(cur).String()
	}
	return strings.Join(all, ", ")
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

func WithFeature(ctx context.Context, feature Feature) context.Context {
	existingFeatures := GetFeatures(ctx)
	if existingFeatures.IsActive(feature) {
		return ctx
	}
	// clone existing features
	features := make(Features, len(existingFeatures)+1)
	copy(features, existingFeatures)
	features[len(existingFeatures)] = byte(feature)
	return SetFeatures(ctx, features)
}

func IsFeatureActive(ctx context.Context, f Feature) bool {
	features := GetFeatures(ctx)
	return features.IsActive(f)
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

// InitFeatures initialized everything using the default features
// and can turn individual features on and off based on the
// strings that are provided. To turn a feature on just use its
// name. To turn it off use the "no" prefix in front of its name.
// Feature names are case-sensitive
func InitFeatures(features ...string) (Features, error) {
	bitSet := make([]bool, MAX_FEATURES)

	for _, f := range DefaultFeatures {
		if !bitSet[f] {
			bitSet[f] = true
		}
	}

	var failing []string
	for _, name := range features {
		flag, ok := FeaturesValue[name]
		if ok {
			bitSet[byte(flag)] = true
			continue
		}

		rest, found := strings.CutPrefix(name, "no")
		if found {
			flag, ok = FeaturesValue[rest]
			if ok {
				bitSet[byte(flag)] = false
				continue
			}
		}

		failing = append(failing, name)
	}

	flags := []byte{}
	for i := 1; i < int(MAX_FEATURES); i++ {
		if bitSet[i] {
			flags = append(flags, byte(i))
		}
	}

	var err error
	if len(failing) != 0 {
		err = errors.New("Failed to parse feature-flags: " + strings.Join(failing, ", "))
	}

	return Features(flags), err
}
