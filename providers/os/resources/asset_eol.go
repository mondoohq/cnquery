// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/upstream/mvd"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/resources"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
)

// FIXME: DEPRECATED, update in v10.0 vv
// This code moved to the core provider and is replaced by the code there
func initAssetEol(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	conn := runtime.Connection.(shared.Connection)
	platform := conn.Asset().Platform
	eolPlatform := mvd.NewMvdPlatform(platform)

	mcc := runtime.Upstream
	if mcc == nil || mcc.ApiEndpoint == "" {
		return nil, nil, resources.MissingUpstreamError{}
	}

	// get new mvd client
	scannerClient, err := mvd.NewAdvisoryScannerClient(mcc.ApiEndpoint, mcc.HttpClient, mcc.Plugins...)
	if err != nil {
		return nil, nil, err
	}

	eolInfo, err := scannerClient.IsEol(context.Background(), eolPlatform)
	if err != nil {
		return nil, nil, err
	}

	log.Debug().Str("name", eolPlatform.Name).Str("release", eolPlatform.Release).Str("title", eolPlatform.Title).Msg("search for eol information")
	if eolInfo == nil {
		return nil, nil, errors.New("no platform eol information available")
	}

	var eolDate *time.Time
	if eolInfo.EolDate != "" {
		parsedEolDate, err := time.Parse(time.RFC3339, eolInfo.EolDate)
		if err != nil {
			return nil, nil, errors.New("could not parse eol date: " + eolInfo.EolDate)
		}
		eolDate = &parsedEolDate
	} else {
		eolDate = &llx.NeverFutureTime
	}

	res := mqlAssetEol{
		MqlRuntime: runtime,
		DocsUrl:    plugin.TValue[string]{Data: eolInfo.DocsUrl, State: plugin.StateIsSet},
		ProductUrl: plugin.TValue[string]{Data: eolInfo.ProductUrl, State: plugin.StateIsSet},
		Date:       plugin.TValue[*time.Time]{Data: eolDate, State: plugin.StateIsSet},
	}

	return nil, &res, nil
}

// ^^

// FIXME: DEPRECATED, update in v10.0 vv
func initPlatformEol(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	res, cache, err := initAssetEol(runtime, args)
	if err != nil || res != nil || cache == nil {
		return res, nil, err
	}

	acache := cache.(*mqlAssetEol)
	cres := mqlPlatformEol{
		MqlRuntime: acache.MqlRuntime,
		DocsUrl:    acache.DocsUrl,
		ProductUrl: acache.ProductUrl,
		Date:       acache.Date,
	}

	return nil, &cres, nil
}

// ^^

// FIXME: DEPRECATED, update in v10.0 vv
func (s *mqlMondooEol) id() (string, error) {
	return "product:" + s.Product.Data + ":" + s.Version.Data, nil
}

func (s *mqlMondooEol) date() (*time.Time, error) {
	name := s.Product.Data
	version := s.Version.Data

	mcc := s.MqlRuntime.Upstream
	if mcc == nil || mcc.ApiEndpoint == "" {
		return nil, resources.MissingUpstreamError{}
	}

	// get new mvd client
	scannerClient, err := mvd.NewAdvisoryScannerClient(mcc.ApiEndpoint, mcc.HttpClient, mcc.Plugins...)
	if err != nil {
		return nil, err
	}

	platformEolInfo, err := scannerClient.IsEol(context.Background(), &mvd.Platform{
		Name:    name,
		Release: version,
	})
	if err != nil {
		return nil, err
	}

	if platformEolInfo == nil {
		return nil, errors.New("no platform eol information available")
	}

	var eolDate *time.Time

	if platformEolInfo.EolDate != "" {
		parsedEolDate, err := time.Parse(time.RFC3339, platformEolInfo.EolDate)
		if err != nil {
			return nil, errors.New("could not parse eol date: " + platformEolInfo.EolDate)
		}
		eolDate = &parsedEolDate
	} else {
		eolDate = &llx.NeverFutureTime
	}

	return eolDate, nil
}

// ^^
