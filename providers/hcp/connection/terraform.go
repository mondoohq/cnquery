// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/jsonapi"
)

const (
	// OptionTfeAddress overrides the HCP Terraform / Terraform Enterprise base
	// URL, for a self-hosted Terraform Enterprise installation.
	OptionTfeAddress = "tfe-address"
	// OptionTfeOrganization scopes the connection to a single HCP Terraform
	// organization instead of every organization the token can reach.
	OptionTfeOrganization = "tfe-organization"

	// CredentialTfeToken tags the HCP Terraform API token.
	CredentialTfeToken = "tfe-token"

	// DefaultTfeAddress is the managed HCP Terraform endpoint.
	DefaultTfeAddress = "https://app.terraform.io"

	// tfeAPIPath is the versioned API prefix every request is issued under.
	tfeAPIPath = "/api/v2"

	// tfePageSize is the page size requested from list endpoints. The API caps
	// it at 100.
	tfePageSize = 100

	// tfeMaxPages bounds a pagination walk. An endpoint that keeps advertising
	// a next page (a cursor it never honors) would otherwise loop forever and
	// multiply every record it returns.
	tfeMaxPages = 1000

	// tfeRequestTimeout bounds a single API request.
	tfeRequestTimeout = 60 * time.Second
)

// TfeClient talks to the HCP Terraform / Terraform Enterprise JSON:API. The
// HCP service principal that authenticates the rest of this provider is not
// accepted by this API, so it carries its own bearer token.
//
// Record *types* come from github.com/hashicorp/go-tfe (see
// TfeRecord.DecodeTyped); the transport stays here deliberately, because on
// every other axis go-tfe's client is weaker than this one:
//
//   - it does not paginate at all. Every List returns a single page and leaves
//     the walk to the caller, with no page cap and no stuck-cursor guard.
//   - its error classification discards the status code for anything that is
//     not a 401 or a 404, so a 403 arrives as a bare errors.New(body) and could
//     only be recognised by matching on message text.
//
// Both of those are load-bearing here: see List below and TfeError above.
type TfeClient struct {
	baseURL *url.URL
	token   string
	http    *http.Client
}

// NewTfeClient builds a client for the given installation address and API
// token. The address may be a bare host or a full URL, with or without the
// /api/v2 suffix.
func NewTfeClient(address, token string, httpClient *http.Client) (*TfeClient, error) {
	if token == "" {
		return nil, errors.New("HCP Terraform API token required: set --tfe-token or TFE_TOKEN")
	}
	if address == "" {
		address = DefaultTfeAddress
	}
	if !strings.Contains(address, "://") {
		address = "https://" + address
	}
	u, err := url.Parse(strings.TrimRight(address, "/"))
	if err != nil {
		return nil, errors.Join(fmt.Errorf("invalid HCP Terraform address %q", address), err)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("invalid HCP Terraform address %q: no host", address)
	}
	// Accept an address that already carries the API prefix so both forms work.
	if !strings.HasSuffix(u.Path, tfeAPIPath) {
		u.Path = strings.TrimRight(u.Path, "/") + tfeAPIPath
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: tfeRequestTimeout}
	}
	return &TfeClient{baseURL: u, token: token, http: httpClient}, nil
}

// BaseURL returns the resolved API base URL, including the version prefix.
func (c *TfeClient) BaseURL() string { return c.baseURL.String() }

// TfeRef is a JSON:API resource linkage: the id and type of a related record.
type TfeRef struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// TfeRelationship is one entry of a record's relationships object. Its data is
// either a single linkage, a list of linkages, or null, so it is kept raw and
// decoded by One or Many.
type TfeRelationship struct {
	Data json.RawMessage `json:"data"`
}

// One returns the single related record, or nil when the relationship is null,
// absent, or a to-many list.
func (r TfeRelationship) One() *TfeRef {
	if len(r.Data) == 0 {
		return nil
	}
	var ref TfeRef
	if err := json.Unmarshal(r.Data, &ref); err != nil {
		return nil
	}
	if ref.ID == "" {
		return nil
	}
	return &ref
}

// Many returns the related records of a to-many relationship. A to-one
// linkage is returned as a single-element slice, and a null or absent
// relationship as an empty slice, so callers never have to nil-check.
func (r TfeRelationship) Many() []TfeRef {
	out := []TfeRef{}
	if len(r.Data) == 0 {
		return out
	}
	var refs []TfeRef
	if err := json.Unmarshal(r.Data, &refs); err == nil {
		for _, ref := range refs {
			if ref.ID != "" {
				out = append(out, ref)
			}
		}
		return out
	}
	if one := r.One(); one != nil {
		out = append(out, *one)
	}
	return out
}

// TfeRecord is a single JSON:API resource object. Attributes stay raw so each
// caller decodes them into the struct that matches its endpoint.
type TfeRecord struct {
	ID            string                     `json:"id"`
	Type          string                     `json:"type"`
	Attributes    json.RawMessage            `json:"attributes"`
	Relationships map[string]TfeRelationship `json:"relationships"`
}

// DecodeTyped unmarshals the record into a go-tfe struct, which is where the
// attribute names live now: go-tfe's `jsonapi:"attr,…"` tags are maintained by
// the vendor and exercised continuously by terraform-provider-tfe, so they are
// a far better source of truth than tags written here from documentation.
//
// The record is re-wrapped as a single-resource JSON:API document because that
// is the shape jsonapi.UnmarshalPayload expects. Rebuilding it from the parsed
// fields rather than from the original bytes means a record assembled by hand
// (in a test, say) decodes exactly like one that came off the wire.
func (r TfeRecord) DecodeTyped(out any) error {
	type resourceObject struct {
		ID            string                     `json:"id"`
		Type          string                     `json:"type"`
		Attributes    json.RawMessage            `json:"attributes,omitempty"`
		Relationships map[string]TfeRelationship `json:"relationships,omitempty"`
	}
	doc := struct {
		Data resourceObject `json:"data"`
	}{
		Data: resourceObject{
			ID:            r.ID,
			Type:          r.Type,
			Attributes:    r.Attributes,
			Relationships: r.Relationships,
		},
	}
	body, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("hcp terraform: re-encoding %s record: %w", r.Type, err)
	}
	if err := jsonapi.UnmarshalPayload(bytes.NewReader(body), out); err != nil {
		return fmt.Errorf("hcp terraform: decoding %s record: %w", r.Type, err)
	}
	return nil
}

// Rel returns the named relationship, or the zero relationship when the record
// does not carry it.
func (r TfeRecord) Rel(name string) TfeRelationship {
	if r.Relationships == nil {
		return TfeRelationship{}
	}
	return r.Relationships[name]
}

// DecodeAttributes unmarshals the record's attributes into out. A record with
// no attributes object leaves out untouched rather than failing.
func (r TfeRecord) DecodeAttributes(out any) error {
	if len(r.Attributes) == 0 {
		return nil
	}
	if err := json.Unmarshal(r.Attributes, out); err != nil {
		return fmt.Errorf("hcp terraform: decoding %s attributes: %w", r.Type, err)
	}
	return nil
}

// tfeDocument is the envelope every JSON:API response arrives in.
type tfeDocument struct {
	Data json.RawMessage `json:"data"`
	Meta tfeMeta         `json:"meta"`
}

type tfeMeta struct {
	Pagination tfePagination `json:"pagination"`
}

type tfePagination struct {
	CurrentPage int  `json:"current-page"`
	NextPage    *int `json:"next-page"`
	TotalPages  int  `json:"total-pages"`
	TotalCount  int  `json:"total-count"`
}

// TfeError is an error response from the API, carrying the HTTP status code so
// callers can classify it. Only a response that actually reached the API
// produces one, which is what keeps a transport failure from being mistaken
// for a "not found".
//
// This is kept rather than adopting go-tfe's errors because go-tfe maps only
// 401 and 404 onto sentinels and drops every other status: a 403 becomes
// errors.New(<response body>), with no code to test. Classifying that would
// mean matching on message text, which is exactly what
// TestTfeErrorClassifiersRejectNonAPIErrors forbids.
type TfeError struct {
	StatusCode int
	Detail     string
}

func (e *TfeError) Error() string {
	if e.Detail == "" {
		return fmt.Sprintf("hcp terraform: API returned status %d", e.StatusCode)
	}
	return fmt.Sprintf("hcp terraform: API returned status %d: %s", e.StatusCode, e.Detail)
}

// tfeErrorBody is the JSON:API error document shape.
type tfeErrorBody struct {
	Errors []struct {
		Status string `json:"status"`
		Title  string `json:"title"`
		Detail string `json:"detail"`
	} `json:"errors"`
}

// IsTfeNotFound reports whether err is an API response saying the record does
// not exist. A transport error (connection refused, TLS failure, timeout) is
// never a not-found, so a network blip cannot silently degrade a field to null
// and let an audit pass on data that was never read.
func IsTfeNotFound(err error) bool {
	var te *TfeError
	if !errors.As(err, &te) {
		return false
	}
	return te.StatusCode == http.StatusNotFound
}

// IsTfeForbidden reports whether err is an API response refusing access. As
// with IsTfeNotFound, a transport error never matches.
func IsTfeForbidden(err error) bool {
	var te *TfeError
	if !errors.As(err, &te) {
		return false
	}
	return te.StatusCode == http.StatusUnauthorized || te.StatusCode == http.StatusForbidden
}

// IsTfeUnavailable reports whether err means the record or the feature simply
// is not there for this token, so the caller can degrade to an empty or null
// result instead of failing the whole scan.
func IsTfeUnavailable(err error) bool {
	return IsTfeNotFound(err) || IsTfeForbidden(err)
}

// get issues a single GET and decodes the JSON:API envelope.
func (c *TfeClient) get(ctx context.Context, path string, query url.Values) (*tfeDocument, error) {
	u := *c.baseURL
	u.Path = strings.TrimRight(u.Path, "/") + "/" + strings.TrimLeft(path, "/")
	if len(query) > 0 {
		u.RawQuery = query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/vnd.api+json")

	resp, err := c.http.Do(req)
	if err != nil {
		// A transport failure is returned as-is, never as a *TfeError, so the
		// classifiers above cannot mistake it for a 404.
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, &TfeError{StatusCode: resp.StatusCode, Detail: tfeErrorDetail(body)}
	}

	var doc tfeDocument
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("hcp terraform: decoding response from %s: %w", path, err)
	}
	return &doc, nil
}

// tfeErrorDetail renders the API's error document into a single line, falling
// back to the raw body when it is not a JSON:API error.
func tfeErrorDetail(body []byte) string {
	var parsed tfeErrorBody
	if err := json.Unmarshal(body, &parsed); err == nil && len(parsed.Errors) > 0 {
		parts := make([]string, 0, len(parsed.Errors))
		for _, e := range parsed.Errors {
			switch {
			case e.Title != "" && e.Detail != "":
				parts = append(parts, e.Title+": "+e.Detail)
			case e.Title != "":
				parts = append(parts, e.Title)
			case e.Detail != "":
				parts = append(parts, e.Detail)
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, "; ")
		}
	}
	detail := strings.TrimSpace(string(body))
	if len(detail) > 200 {
		detail = detail[:200]
	}
	return detail
}

// GetOne fetches a single record from an endpoint whose data is one object.
func (c *TfeClient) GetOne(ctx context.Context, path string, query url.Values) (*TfeRecord, error) {
	doc, err := c.get(ctx, path, query)
	if err != nil {
		return nil, err
	}
	if len(doc.Data) == 0 || string(doc.Data) == "null" {
		return nil, nil
	}
	var rec TfeRecord
	if err := json.Unmarshal(doc.Data, &rec); err != nil {
		return nil, fmt.Errorf("hcp terraform: decoding record from %s: %w", path, err)
	}
	return &rec, nil
}

// List walks every page of a collection endpoint and returns the records.
//
// The walk stops when the API reports no next page. It also stops when the
// API points back at the page just read or at an earlier one, and when a page
// comes back empty while still advertising a successor: both mean the endpoint
// is ignoring the cursor, and continuing would return the same records over
// and over.
//
// go-tfe supplies no equivalent: it has no pagination walk at all, so there is
// nothing to adopt here and nothing that would carry these two guards.
func (c *TfeClient) List(ctx context.Context, path string, query url.Values) ([]TfeRecord, error) {
	out := []TfeRecord{}
	page := 1

	for walked := 0; walked < tfeMaxPages; walked++ {
		q := url.Values{}
		for k, v := range query {
			q[k] = append([]string(nil), v...)
		}
		q.Set("page[number]", strconv.Itoa(page))
		q.Set("page[size]", strconv.Itoa(tfePageSize))

		doc, err := c.get(ctx, path, q)
		if err != nil {
			return nil, err
		}

		batch := []TfeRecord{}
		if len(doc.Data) > 0 && string(doc.Data) != "null" {
			if err := json.Unmarshal(doc.Data, &batch); err != nil {
				return nil, fmt.Errorf("hcp terraform: decoding records from %s: %w", path, err)
			}
		}
		out = append(out, batch...)

		next := doc.Meta.Pagination.NextPage
		if next == nil || *next <= page || len(batch) == 0 {
			return out, nil
		}
		page = *next
	}

	return nil, fmt.Errorf("hcp terraform: %s exceeded the %d page pagination limit", path, tfeMaxPages)
}
