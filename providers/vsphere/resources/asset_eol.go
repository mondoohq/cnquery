package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v9/llx"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/resources"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/upstream/mvd"
	"go.mondoo.com/cnquery/v9/providers/vsphere/connection"
	"time"
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

func initAssetEol(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	conn := runtime.Connection.(*connection.VsphereConnection)
	platform := conn.Asset().Platform
	eolPlatform := convertPlatform2VulnPlatform(platform)

	mcc := runtime.Upstream
	if mcc == nil || mcc.ApiEndpoint == "" {
		return nil, nil, resources.MissingUpstreamError{}
	}

	scannerClient, err := newAdvisoryScannerHttpClient(mcc.ApiEndpoint, mcc.Plugins, mcc.HttpClient)
	if err != nil {
		return nil, nil, err
	}

	data, _ := json.Marshal(eolPlatform)
	fmt.Println(string(data))

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
