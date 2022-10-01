package explorer

import "context"

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
	// DeleteQueryPack removes a given pack
	// Note: the MRN has to be valid
	DeleteQueryPack(ctx context.Context, mrn string) error
	// List all packs for a given owner
	// Note: Owner MRN is required
	ListQueryPacks(ctx context.Context, ownerMrn string, name string) ([]*QueryPack, error)
	// GetQueryPackFilters retrieves the list of asset filters for a pack (fast)
	GetQueryPackFilters(ctx context.Context, mrn string) ([]*Mquery, error)
}
