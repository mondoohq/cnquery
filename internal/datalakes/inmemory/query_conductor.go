// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package inmemory

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/explorer"
	"go.mondoo.com/cnquery/v11/explorer/resources"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/types"
	"go.mondoo.com/cnquery/v11/utils/multierr"
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

// GetResources retrieves previously stored resources about an asset
func (db *Db) GetResources(ctx context.Context, assetMrn string, req []*resources.ResourceDataReq) ([]*llx.ResourceRecording, error) {
	res := make([]*llx.ResourceRecording, len(req))
	for i := range req {
		rr := req[i]
		raw, ok := db.cache.Get(dbIDData + assetMrn + "\x00" + rr.Resource + "\x00" + rr.Id)
		if !ok {
			return nil, errors.New("cannot find resource " + rr.Resource + " id=" + rr.Id + " on " + assetMrn)
		}
		res[i] = raw.(*llx.ResourceRecording)
	}
	return res, nil
}

// UpdateData sets the list of data value for a given asset and returns a list of updated IDs
func (db *Db) UpdateData(ctx context.Context, assetMrn string, data map[string]*llx.Result) (map[string]types.Type, error) {
	resolved, err := db.GetResolvedPack(assetMrn)
	if err != nil {
		return nil, errors.New("cannot find collectorJob to store data: " + err.Error())
	}
	executionJob := resolved.ExecutionJob

	res := make(map[string]types.Type, len(data))
	var errs multierr.Errors
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

			errs.Add(fmt.Errorf("failed to store data for %q, %w: expected %s, got %s",
				dpChecksum, errTypesDontMatch, types.Type(info.Type).Label(), types.Type(val.Data.Type).Label()))

			continue
		}

		err := db.setDatum(ctx, assetMrn, dpChecksum, val)
		if err != nil {
			errs.Add(err)
			continue
		}

		// TODO: we don't know which data was updated and which wasn't yet, so
		// we currently always notify...
		res[dpChecksum] = types.Type(info.Type)
	}

	if !errs.IsEmpty() {
		return nil, errs.Deduplicate()
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
		if datum == nil {
			data[id] = &llx.Result{
				Data:   llx.NilPrimitive,
				CodeId: id,
			}
		} else {
			data[id] = datum.(*llx.Result)
		}
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

// SetProps will override properties for a given entity (asset, space, org)
func (db *Db) SetProps(ctx context.Context, req *explorer.PropsReq) error {
	x, ok := db.cache.Get(dbIDAsset + req.EntityMrn)
	if !ok {
		return errors.New("failed to find entity " + req.EntityMrn)
	}
	asset := x.(wrapAsset)

	if asset.Bundle == nil {
		return errors.New("found an asset without a bundle configured in the DB")
	}

	allProps := make(map[string]*explorer.Property, len(asset.Bundle.Props))
	for i := range asset.Bundle.Props {
		cur := asset.Bundle.Props[i]
		if cur.Mrn != "" {
			allProps[cur.Mrn] = cur
		}
		if cur.Uid != "" {
			allProps[cur.Uid] = cur
		}
	}

	for i := range req.Props {
		cur := req.Props[i]
		id := cur.Mrn
		if id == "" {
			id = cur.Uid
		}
		if id == "" {
			return errors.New("cannot set property without MRN: " + cur.Mql)
		}

		if cur.Mql == "" {
			delete(allProps, id)
		}
		allProps[id] = cur
	}

	asset.Bundle.Props = []*explorer.Property{}
	for k, v := range allProps {
		// since props can be in the list with both UIDs and MRNs, in the case
		// where a property sets both we want to ignore one entry to avoid duplicates
		if v.Mrn != "" && v.Uid != "" && k == v.Uid {
			continue
		}
		asset.Bundle.Props = append(asset.Bundle.Props, v)
	}

	db.cache.Set(dbIDAsset+req.EntityMrn, asset, 1)

	return nil
}
