package resources

import (
	"context"
	"errors"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers-sdk/v1/resources"
	"go.mondoo.com/cnquery/providers/os/connection/shared"
	"go.mondoo.com/cnquery/upstream/mvd"
)

// convertPlatform2VulnPlatform converts the motor platform.Platform to the
// platform object we use for vulnerability data
// TODO: we need to harmonize the platform objects
func convertPlatform2VulnPlatform(pf *inventory.Platform) *mvd.Platform {
	if pf == nil {
		return nil
	}
	return &mvd.Platform{
		Name:    pf.Name,
		Release: pf.Version,
		Build:   pf.Build,
		Arch:    pf.Arch,
		Title:   pf.Title,
		Labels:  pf.Labels,
	}
}

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

func initAssetEol(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	conn := runtime.Connection.(shared.Connection)
	platform := conn.Asset().Platform
	eolPlatform := convertPlatform2VulnPlatform(platform)

	mcc := runtime.Upstream
	if mcc == nil || mcc.ApiEndpoint == "" {
		return nil, nil, resources.MissingUpstreamError{}
	}

	// get new advisory report
	// start scanner client

	scannerClient, err := newAdvisoryScannerHttpClient(mcc.ApiEndpoint, mcc.Plugins, mcc.HttpClient)
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
