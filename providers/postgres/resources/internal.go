// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

// These mql<Resource>Internal structs are embedded into the generated resource
// structs. They carry the parent context (owning role, containing database)
// that typed references and per-database queries need.

type mqlPostgresDatabaseInternal struct {
	cacheOwner string
}

type mqlPostgresSchemaInternal struct {
	cacheDatabase string
	cacheOwner    string
}

type mqlPostgresFunctionInternal struct {
	cacheDatabase string
	cacheOwner    string
}

type mqlPostgresTableInternal struct {
	cacheDatabase string
	cacheOwner    string
}

type mqlPostgresTablespaceInternal struct {
	cacheOwner string
}

type mqlPostgresExtensionInternal struct {
	cacheOwner string
}

type mqlPostgresForeignServerInternal struct {
	cacheOwner    string
	cacheDatabase string
}

type mqlPostgresPublicationInternal struct {
	cacheOwner string
}

type mqlPostgresSubscriptionInternal struct {
	cacheOwner string
}
