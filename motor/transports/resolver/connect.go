package resolver

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/transports"
)

func Connect(tc *transports.TransportConfig, idDetector string, insecure bool, record bool) (*motor.Motor, error) {
	log.Debug().Str("connection", tc.ToUrl()).Bool("insecure", insecure).Msg("establish connection to asset")
	// overwrite connection specific insecure with global insecure
	if insecure {
		tc.Insecure = insecure
	}

	if record {
		tc.Record = true
	}

	return New(tc, idDetector)
}

func ConnectAsset(assetObj *asset.Asset, record bool) (*motor.Motor, error) {
	// connect to the platform
	if len(assetObj.Connections) == 0 {
		return nil, errors.New("no connection provided for asset " + assetObj.Name)
	}

	// TODO: we may want to allow multiple connection trials later
	tc := assetObj.Connections[0]

	// some transports have their own kind/runtime information already
	// NOTE: going forward we may want to enforce that assets have at least kind and runtime information
	if assetObj.Platform != nil {
		tc.Kind = assetObj.Platform.Kind
		tc.Runtime = assetObj.Platform.Runtime
	}

	// parse reference id and restore options
	if len(assetObj.ReferenceIDs) > 0 {
		tc.Platformid = assetObj.ReferenceIDs[0]
	}

	return Connect(tc, "", tc.Insecure, record)
}
