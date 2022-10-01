package inmemory

import (
	"context"
	"errors"
	"strings"

	"go.mondoo.com/cnquery/explorer"
)

type wrapQuery struct {
	*explorer.Mquery
}

type wrapQueryPack struct {
	*explorer.QueryPack
	filters []*explorer.Mquery
}

// QueryExists checks if the given MRN exists
func (db *Db) QueryExists(ctx context.Context, mrn string) (bool, error) {
	_, ok := db.cache.Get(dbIDQuery + mrn)
	return ok, nil
}

// GetQuery retrieves a given query
func (db *Db) GetQuery(ctx context.Context, mrn string) (*explorer.Mquery, error) {
	q, ok := db.cache.Get(dbIDQuery + mrn)
	if !ok {
		return nil, errors.New("query '" + mrn + "' not found")
	}
	return (q.(wrapQuery)).Mquery, nil
}

// SetQuery stores a given query
// Note: the query must be defined, it cannot be nil
func (db *Db) SetQuery(ctx context.Context, mrn string, mquery *explorer.Mquery) error {
	v := wrapQuery{mquery}
	ok := db.cache.Set(dbIDQuery+mrn, v, 1)
	if !ok {
		return errors.New("failed to save query '" + mrn + "' to cache")
	}
	return nil
}

// SetQueryPack stores a given pack in the datalake
func (db *Db) SetQueryPack(ctx context.Context, obj *explorer.QueryPack, filters []*explorer.Mquery) error {
	_, err := db.setQueryPack(ctx, obj, filters)
	return err
}

// GetQueryPack retrieves the pack
func (db *Db) GetQueryPack(ctx context.Context, mrn string) (*explorer.QueryPack, error) {
	q, ok := db.cache.Get(dbIDQueryPack + mrn)
	if !ok {
		return nil, errors.New("query pack '" + mrn + "' not found")
	}
	return (q.(wrapQueryPack)).QueryPack, nil
}

// GetQueryPackFilters retrieves the query pack filters
func (db *Db) GetQueryPackFilters(ctx context.Context, mrn string) ([]*explorer.Mquery, error) {
	q, ok := db.cache.Get(dbIDQueryPack + mrn)
	if !ok {
		return nil, errors.New("query pack '" + mrn + "' not found")
	}
	return (q.(wrapQueryPack)).filters, nil
}

func (db *Db) setQueryPack(ctx context.Context, in *explorer.QueryPack, filters []*explorer.Mquery) (wrapQueryPack, error) {
	var err error

	for i := range filters {
		filter := filters[i]
		if err = db.SetQuery(ctx, filter.Mrn, filter); err != nil {
			return wrapQueryPack{}, err
		}
	}

	obj := wrapQueryPack{
		QueryPack: in,
		filters:   filters,
	}

	ok := db.cache.Set(dbIDQueryPack+obj.Mrn, obj, 2)
	if !ok {
		return wrapQueryPack{}, errors.New("failed to save query pack '" + in.Mrn + "' to cache")
	}

	list, err := db.listQueryPacks()
	if err != nil {
		return wrapQueryPack{}, err
	}

	list[in.Mrn] = struct{}{}
	ok = db.cache.Set(dbIDListQueryPacks, list, 0)
	if !ok {
		return wrapQueryPack{}, errors.New("failed to update query pack list cache")
	}

	return obj, nil
}

// DeleteQueryPack removes a given mrn
// Note: the MRN has to be valid
func (db *Db) DeleteQueryPack(ctx context.Context, mrn string) error {
	_, ok := db.cache.Get(dbIDQueryPack + mrn)
	if !ok {
		return nil
	}

	errors := strings.Builder{}

	// list update
	list, err := db.listQueryPacks()
	if err != nil {
		return err
	}

	delete(list, mrn)
	ok = db.cache.Set(dbIDListQueryPacks, list, 0)
	if !ok {
		errors.WriteString("failed to update query packs list cache")
	}

	db.cache.Del(dbIDQueryPack + mrn)

	return nil
}

// ListQueryPacks for a given owner
// Note: Owner MRN is required
func (db *Db) ListQueryPacks(ctx context.Context, ownerMrn string, name string) ([]*explorer.QueryPack, error) {
	mrns, err := db.listQueryPacks()
	if err != nil {
		return nil, err
	}

	res := []*explorer.QueryPack{}
	for k := range mrns {
		obj, err := db.GetQueryPack(ctx, k)
		if err != nil {
			return nil, err
		}

		if obj.OwnerMrn != ownerMrn {
			continue
		}

		res = append(res, obj)
	}

	return res, nil
}

func (db *Db) listQueryPacks() (map[string]struct{}, error) {
	x, ok := db.cache.Get(dbIDListQueryPacks)
	if ok {
		return x.(map[string]struct{}), nil
	}

	nu := map[string]struct{}{}
	ok = db.cache.Set(dbIDListQueryPacks, nu, 0)
	if !ok {
		return nil, errors.New("failed to initialize query packs list cache")
	}
	return nu, nil
}
