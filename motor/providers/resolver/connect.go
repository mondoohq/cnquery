package resolver

import (
	"context"

	"errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/vault"
)

func EstablishConnection(ctx context.Context, tc *providers.Config, credsResolver vault.Resolver, insecure bool, record bool) (*motor.Motor, error) {
	log.Debug().Str("connection", tc.ToUrl()).Bool("insecure", insecure).Msg("establish connection to asset")
	// overwrite connection specific insecure with global insecure
	if insecure {
		tc.Insecure = insecure
	}

	if record {
		tc.Record = true
	}

	return NewMotorConnection(ctx, tc, credsResolver)
}

func OpenAssetConnection(ctx context.Context, assetInfo *asset.Asset, credsResolver vault.Resolver, record bool) (*motor.Motor, error) {
	if assetInfo == nil {
		return nil, errors.New("asset is not defined")
	}

	// connect to the platform
	if len(assetInfo.Connections) == 0 {
		return nil, errors.New("no connection provided for asset " + assetInfo.Name)
	}

	// TODO: we may want to allow multiple connection trials later
	pCfg := assetInfo.Connections[0]

	// use connection host as default
	if assetInfo.Name == "" {
		assetInfo.Name = pCfg.Host
	}

	// some transports have their own kind/runtime information already
	// NOTE: going forward we may want to enforce that assets have at least kind and runtime information
	if assetInfo.Platform != nil {
		pCfg.Kind = assetInfo.Platform.Kind
		pCfg.Runtime = assetInfo.Platform.Runtime
		if pCfg.Options == nil {
			pCfg.Options = map[string]string{}
		}
		// set platform name override to ensure we get the correct platform at policy execution time
		pCfg.Options["platform-override"] = assetInfo.Platform.Name
	}

	// parse reference id and restore options
	if len(assetInfo.PlatformIds) > 0 {
		pCfg.PlatformId = assetInfo.PlatformIds[0]
	}

	m, err := EstablishConnection(ctx, pCfg, credsResolver, pCfg.Insecure, record)
	if err != nil {
		return nil, err
	}

	m.SetAsset(assetInfo)

	return m, nil
}

func OpenAssetConnections(ctx context.Context, assetInfo *asset.Asset, credsResolver vault.Resolver, record bool) ([]*motor.Motor, error) {
	if assetInfo == nil {
		return nil, errors.New("asset is not defined")
	}

	// connect to the platform
	if len(assetInfo.Connections) == 0 {
		return nil, errors.New("no connection provided for asset " + assetInfo.Name)
	}

	// TODO: we may want to allow multiple connection trials later
	connections := []*motor.Motor{}
	for ci := range assetInfo.Connections {
		pCfg := assetInfo.Connections[ci]

		// use connection host as default
		if assetInfo.Name == "" {
			assetInfo.Name = pCfg.Host
		}

		// some transports have their own kind/runtime information already
		// NOTE: going forward we may want to enforce that assets have at least kind and runtime information
		if assetInfo.Platform != nil {
			pCfg.Kind = assetInfo.Platform.Kind
			pCfg.Runtime = assetInfo.Platform.Runtime
			if pCfg.Options == nil {
				pCfg.Options = map[string]string{}
			}
			// set platform name override to ensure we get the correct platform at policy execution time
			pCfg.Options["platform-override"] = assetInfo.Platform.Name
		}

		// parse reference id and restore options
		if len(assetInfo.PlatformIds) > 0 {
			pCfg.PlatformId = assetInfo.PlatformIds[0]
		}

		m, err := EstablishConnection(ctx, pCfg, credsResolver, pCfg.Insecure, record)
		if err != nil {
			return nil, err
		}

		m.SetAsset(assetInfo)
		connections = append(connections, m)
	}
	return connections, nil
}
