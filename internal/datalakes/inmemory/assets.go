package inmemory

import (
	"context"
	"errors"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/explorer"
)

type wrapAsset struct {
	mrn             string
	ResolvedPack    *explorer.ResolvedPack
	ResolvedVersion explorer.ResolvedVersion
	Bundle          *explorer.Bundle
}

// EnsureAsset makes sure an asset exists
func (db *Db) EnsureAsset(ctx context.Context, mrn string) error {
	_, _, err := db.ensureAssetObject(ctx, mrn)
	return err
}

func (db *Db) ensureAssetObject(ctx context.Context, mrn string) (wrapAsset, bool, error) {
	log.Debug().Str("mrn", mrn).Msg("assets> ensure asset")

	x, ok := db.cache.Get(dbIDAsset + mrn)
	if ok {
		return x.(wrapAsset), false, nil
	}

	assetw := wrapAsset{
		mrn: mrn,
		Bundle: &explorer.Bundle{
			OwnerMrn: mrn,
		},
	}
	ok = db.cache.Set(dbIDAsset+mrn, assetw, 1)
	if !ok {
		return wrapAsset{}, false, errors.New("failed to create asset '" + mrn + "'")
	}

	return assetw, true, nil
}
