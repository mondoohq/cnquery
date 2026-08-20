// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package bwapi holds the Go type layer generated from Bitwarden's official
// Public API OpenAPI v3 specification. The spec is vendored at
// ../openapi/swagger.json and the generator config at ../openapi/config.yaml;
// models.gen.go is produced by oapi-codegen (models only, no HTTP client) and
// must not be edited by hand.
//
// Regenerate with `go generate ./...` from this directory, or run:
//
//	oapi-codegen -config providers/bitwarden/connection/openapi/config.yaml \
//	    providers/bitwarden/connection/openapi/swagger.json
//
// oapi-codegen must be on PATH (installed via `go install
// github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.8.0`).
//
// These types track the published spec so drift is visible in code review.
// The provider's request/response decoding still lives in
// ../client.go rather than here, because the public spec is lossy for read
// modeling: see the "Public API spec coverage" note in the provider README
// and the TestOpenAPISpecGaps tripwire in ../client_test.go.
package bwapi

//go:generate oapi-codegen -config ../openapi/config.yaml ../openapi/swagger.json
