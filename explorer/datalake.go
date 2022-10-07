package explorer

import (
	"context"

	llx "go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/types"
)

// DataLake provides a shared database access layer
type DataLake interface {
	// GetQuery retrieves a given query
	GetQuery(ctx context.Context, mrn string) (*Mquery, error)
	// SetQuery stores a given query
	// Note: the query must be defined, it cannot be nil
	SetQuery(ctx context.Context, mrn string, query *Mquery) error

	// SetQueryPack stores a given pack in the data lake
	SetQueryPack(ctx context.Context, querypack *QueryPack, filters []*Mquery) error
	// GetQueryPack retrieves and if necessary updates the pack
	GetQueryPack(ctx context.Context, mrn string) (*QueryPack, error)
	// GetBundle retrieves and if necessary updates the pack. Used for assets,
	// which have multiple query packs associated with them.
	GetBundle(ctx context.Context, mrn string) (*Bundle, error)
	// DeleteQueryPack removes a given pack
	// Note: the MRN has to be valid
	DeleteQueryPack(ctx context.Context, mrn string) error
	// List all packs for a given owner
	// Note: Owner MRN is required
	ListQueryPacks(ctx context.Context, ownerMrn string, name string) ([]*QueryPack, error)
	// GetQueryPackFilters retrieves the list of asset filters for a pack (fast)
	GetQueryPackFilters(ctx context.Context, mrn string) ([]*Mquery, error)

	// MutateBundle runs the given mutation on a bundle, typically an asset.
	// If it cannot find the owner, it will create it.
	MutateBundle(ctx context.Context, mutation *BundleMutationDelta, createIfMissing bool) (*Bundle, error)

	// EnsureAsset makes sure an asset with mrn exists
	EnsureAsset(ctx context.Context, mrn string) error

	// SetResolvedPack stores a resolved pack
	SetResolvedPack(mrn string, filtersChecksum string, resolved *ResolvedPack) error
	// SetAssetResolvedPack stores the resolved pack for a given asset
	SetAssetResolvedPack(ctx context.Context, assetMrn string, resolved *ResolvedPack, version ResolvedVersion) error
	// UpdateData sets the list of data value for a given asset and returns a list of updated IDs
	UpdateData(ctx context.Context, assetMrn string, data map[string]*llx.Result) (map[string]types.Type, error)
	// GetReport retrieves all scores and data for a given asset
	GetReport(ctx context.Context, assetMrn string, packMrn string) (*Report, error)
}
