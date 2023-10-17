// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"time"

	"go.mondoo.com/cnquery/v9/llx"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/resources"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/upstream/mvd"
)

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
