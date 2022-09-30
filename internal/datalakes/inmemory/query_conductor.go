package inmemory

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/explorer"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/types"
)

type wrapResolved struct {
	*explorer.ResolvedPack
	filtersChecksum string
	assetMRN        string
}

func (db *Db) SetResolvedPack(mrn string, filtersChecksum string, resolved *explorer.ResolvedPack) error {
	v := wrapResolved{resolved, filtersChecksum, mrn}
	ok := db.cache.Set(dbIDresolvedPack+mrn, v, 1)
	if !ok {
		return errors.New("failed to save query '" + mrn + "' to cache")
	}
	return nil
}

func (db *Db) SetAssetResolvedPack(ctx context.Context, assetMrn string, resolved *explorer.ResolvedPack, version explorer.ResolvedVersion) error {
	x, ok := db.cache.Get(dbIDAsset + assetMrn)
	if !ok {
		return errors.New("cannot find asset '" + assetMrn + "'")
	}
	assetw := x.(wrapAsset)

	if assetw.ResolvedPack != nil && assetw.ResolvedPack.GraphExecutionChecksum == resolved.GraphExecutionChecksum && string(assetw.ResolvedVersion) == string(version) {
		log.Debug().
			Str("asset", assetMrn).
			Msg("resolverj.db> asset resolved query pack is already cached (and unchanged)")
		return nil
	}

	assetw.ResolvedPack = resolved
	assetw.ResolvedVersion = version

	var err error
	job := resolved.ExecutionJob
	for checksum, info := range job.Datapoints {
		err = db.initDataValue(ctx, assetMrn, checksum, types.Type(info.Type))
		if err != nil {
			log.Error().
				Err(err).
				Str("asset", assetMrn).
				Str("query checksum", checksum).
				Msg("resolver.db> failed to set asset resolved pack, failed to initialize data value")
			return errors.New("failed to create asset scoring job (failed to init data)")
		}
	}

	ok = db.cache.Set(dbIDAsset+assetMrn, assetw, 1)
	if !ok {
		return errors.New("failed to save resolved pack for asset '" + assetMrn + "'")
	}

	return nil
}

func (db *Db) GetResolvedPack(mrn string) (*explorer.ResolvedPack, error) {
	q, ok := db.cache.Get(dbIDresolvedPack + mrn)
	if !ok {
		return nil, errors.New("query '" + mrn + "' not found")
	}
	return (q.(wrapResolved)).ResolvedPack, nil
}

var errTypesDontMatch = errors.New("types don't match")

// UpdateData sets the list of data value for a given asset and returns a list of updated IDs
func (db *Db) UpdateData(ctx context.Context, assetMrn string, data map[string]*llx.Result) (map[string]types.Type, error) {
	resolved, err := db.GetResolvedPack(assetMrn)
	if err != nil {
		return nil, errors.New("cannot find collectorJob to store data: " + err.Error())
	}
	executionJob := resolved.ExecutionJob

	res := make(map[string]types.Type, len(data))
	var errList error
	for dpChecksum, val := range data {
		info, ok := executionJob.Datapoints[dpChecksum]
		if !ok {
			return nil, errors.New("cannot find this datapoint to store values: " + dpChecksum)
		}

		if val.Data != nil && !val.Data.IsNil() && val.Data.Type != "" &&
			val.Data.Type != info.Type && types.Type(info.Type) != types.Unset {
			log.Warn().
				Str("checksum", dpChecksum).
				Str("asset", assetMrn).
				Interface("data", val.Data).
				Str("expected", types.Type(info.Type).Label()).
				Str("received", types.Type(val.Data.Type).Label()).
				Msg("resolver.db> failed to store data, types don't match")

			errList = multierror.Append(errList, fmt.Errorf("failed to store data for %q, %w: expected %s, got %s",
				dpChecksum, errTypesDontMatch, types.Type(info.Type).Label(), types.Type(val.Data.Type).Label()))

			continue
		}

		err := db.setDatum(ctx, assetMrn, dpChecksum, val)
		if err != nil {
			errList = multierror.Append(errList, err)
			continue
		}

		// TODO: we don't know which data was updated and which wasn't yet, so
		// we currently always notify...
		res[dpChecksum] = types.Type(info.Type)
	}

	if errList != nil {
		return nil, errList
	}

	return res, nil
}

func (db *Db) setDatum(ctx context.Context, assetMrn string, checksum string, value *llx.Result) error {
	id := dbIDData + assetMrn + "\x00" + checksum
	ok := db.cache.Set(id, value, 1)
	if !ok {
		return errors.New("failed to save asset data for asset '" + assetMrn + "' and checksum '" + checksum + "'")
	}
	return nil
}

// GetReport retrieves all scores and data for a given asset
func (db *Db) GetReport(ctx context.Context, assetMrn string, packMrn string) (*explorer.Report, error) {
	x, ok := db.cache.Get(dbIDAsset + assetMrn)
	if !ok {
		return nil, errors.New("cannot find asset '" + assetMrn + "'")
	}
	assetw := x.(wrapAsset)
	resolvedPack := assetw.ResolvedPack

	data := map[string]*llx.Result{}
	for id := range resolvedPack.ExecutionJob.Datapoints {
		datum, ok := db.cache.Get(dbIDData + assetMrn + "\x00" + id)
		if !ok {
			continue
		}
		data[id] = datum.(*llx.Result)
	}

	return &explorer.Report{
		PackMrn:   packMrn,
		EntityMrn: assetMrn,
		Data:      data,
	}, nil
}

func (db *Db) initDataValue(ctx context.Context, assetMrn string, checksum string, typ types.Type) error {
	id := dbIDData + assetMrn + "\x00" + checksum
	_, ok := db.cache.Get(id)
	if ok {
		return nil
	}

	ok = db.cache.Set(id, nil, 1)
	if !ok {
		return errors.New("failed to initialize data value for asset '" + assetMrn + "' with checksum '" + checksum + "'")
	}
	return nil
}
