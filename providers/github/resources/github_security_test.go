// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/go-github/v88/github"
	"github.com/stretchr/testify/assert"
)

func ghErrorResponse(status int) error {
	return &github.ErrorResponse{
		Response: &http.Response{StatusCode: status},
	}
}

func TestIsAccessDeniedOrNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil error", err: nil, want: false},
		{name: "404 not found", err: ghErrorResponse(http.StatusNotFound), want: true},
		{name: "403 forbidden", err: ghErrorResponse(http.StatusForbidden), want: true},
		{name: "500 server error", err: ghErrorResponse(http.StatusInternalServerError), want: false},
		{name: "200 ok", err: ghErrorResponse(http.StatusOK), want: false},
		{name: "plain error", err: errors.New("boom"), want: false},
		{name: "no available registrations fallback", err: errors.New("no available registrations"), want: true},
		{name: "wrapped 404", err: errors.Join(errors.New("ctx"), ghErrorResponse(http.StatusNotFound)), want: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isAccessDeniedOrNotFound(tc.err))
		})
	}
}

type samlErr = struct {
	Type       string         `json:"type"`
	Message    string         `json:"message"`
	Extensions map[string]any `json:"extensions"`
}

func TestIsSamlScopeOrPermissionError(t *testing.T) {
	tests := []struct {
		name string
		errs []samlErr
		want bool
	}{
		{name: "no errors", errs: nil, want: false},
		{name: "unrelated error", errs: []samlErr{{Type: "NOT_FOUND", Message: "missing"}}, want: false},
		{name: "type INSUFFICIENT_SCOPES", errs: []samlErr{{Type: "INSUFFICIENT_SCOPES"}}, want: true},
		{name: "type lowercase forbidden", errs: []samlErr{{Type: "forbidden"}}, want: true},
		{name: "type unauthorized", errs: []samlErr{{Type: "UNAUTHORIZED"}}, want: true},
		{name: "extensions code", errs: []samlErr{{Extensions: map[string]any{"code": "insufficient_scopes"}}}, want: true},
		{name: "message contains scope", errs: []samlErr{{Message: "Your token is missing the required scope"}}, want: true},
		{name: "message must have admin", errs: []samlErr{{Message: "You must have admin access"}}, want: true},
		{name: "first benign, second matches", errs: []samlErr{{Type: "OTHER"}, {Type: "FORBIDDEN"}}, want: true},
		{name: "non-string extension code ignored", errs: []samlErr{{Extensions: map[string]any{"code": 42}}}, want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isSamlScopeOrPermissionError(tc.errs))
		})
	}
}

func TestKeyAgeInDays(t *testing.T) {
	tests := []struct {
		name      string
		createdAt *github.Timestamp
		want      int64
	}{
		{name: "nil timestamp", createdAt: nil, want: -1},
		{name: "zero timestamp", createdAt: &github.Timestamp{}, want: -1},
		{name: "two days old", createdAt: &github.Timestamp{Time: time.Now().Add(-48 * time.Hour)}, want: 2},
		{name: "ten days old", createdAt: &github.Timestamp{Time: time.Now().Add(-10 * 24 * time.Hour)}, want: 10},
		{name: "younger than a day", createdAt: &github.Timestamp{Time: time.Now().Add(-1 * time.Hour)}, want: 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, keyAgeInDays(tc.createdAt))
		})
	}
}

func TestIsCodeownersCommentLine(t *testing.T) {
	tests := []struct {
		name string
		line string
		want bool
	}{
		{name: "empty line", line: "", want: false},
		{name: "leading hash", line: "# comment", want: true},
		{name: "indented hash with spaces", line: "   # comment", want: true},
		{name: "indented hash with tab", line: "\t# comment", want: true},
		{name: "rule line", line: "*.go @team", want: false},
		{name: "inline hash is not a comment", line: "path/to/#special @owner", want: false},
		{name: "whitespace only", line: "   ", want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isCodeownersCommentLine(tc.line))
		})
	}
}

func TestParseCodeowners(t *testing.T) {
	content := "# top comment\n" +
		"\n" +
		"*.go      @org/go-team @alice\n" +
		"   # indented comment\n" +
		"/docs/    @org/docs-team\n" +
		"path/to/#special   @owner\r\n" +
		"   \n" +
		"*.js\n"

	rules := parseCodeowners(content)

	assert.Equal(t, []codeownersRule{
		{pattern: "*.go", owners: []string{"@org/go-team", "@alice"}, lineNumber: 3},
		{pattern: "/docs/", owners: []string{"@org/docs-team"}, lineNumber: 5},
		{pattern: "path/to/#special", owners: []string{"@owner"}, lineNumber: 6},
		{pattern: "*.js", owners: []string{}, lineNumber: 8},
	}, rules)
}

func TestParseCodeowners_Empty(t *testing.T) {
	assert.Empty(t, parseCodeowners(""))
	assert.Empty(t, parseCodeowners("# only a comment\n\n   \n"))
}
