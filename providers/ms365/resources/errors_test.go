// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"fmt"
	"net"
	"testing"

	betaodataerrors "github.com/microsoftgraph/msgraph-beta-sdk-go/models/odataerrors"
	"github.com/microsoftgraph/msgraph-sdk-go/models/odataerrors"
	"github.com/stretchr/testify/assert"
)

func odataErrWithCode(code string) *odataerrors.ODataError {
	payload := odataerrors.NewMainError()
	payload.SetCode(&code)
	err := odataerrors.NewODataError()
	err.SetErrorEscaped(payload)
	return err
}

func betaODataErrWithCode(code string) *betaodataerrors.ODataError {
	payload := betaodataerrors.NewMainError()
	payload.SetCode(&code)
	err := betaodataerrors.NewODataError()
	err.SetErrorEscaped(payload)
	return err
}

func TestGraphErrorCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"nil error", nil, ""},
		{"plain error", errors.New("boom"), ""},
		{"v1 odata error", odataErrWithCode("Request_ResourceNotFound"), "Request_ResourceNotFound"},
		{"beta odata error", betaODataErrWithCode("Request_ResourceNotFound"), "Request_ResourceNotFound"},
		{"other v1 code", odataErrWithCode("Authorization_RequestDenied"), "Authorization_RequestDenied"},
		{"wrapped odata error", fmt.Errorf("fetching assignments: %w", odataErrWithCode("Request_ResourceNotFound")), "Request_ResourceNotFound"},
		{"odata error with no payload", odataerrors.NewODataError(), ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, graphErrorCode(tc.err))
		})
	}
}

// The classifier gates a retry, so it must not fire on a transport failure --
// retrying without $expand would silently drop the principal detail from every
// assignment on what is really a network problem.
func TestIsResourceNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"v1 resource not found", odataErrWithCode("Request_ResourceNotFound"), true},
		{"beta resource not found", betaODataErrWithCode("Request_ResourceNotFound"), true},
		{"wrapped resource not found", fmt.Errorf("wrapped: %w", odataErrWithCode("Request_ResourceNotFound")), true},
		{"permission denied is not a missing resource", odataErrWithCode("Authorization_RequestDenied"), false},
		{"throttling is not a missing resource", odataErrWithCode("TooManyRequests"), false},
		{"transport error is not a missing resource", &net.DNSError{Err: "no such host"}, false},
		{"plain error", errors.New("Request_ResourceNotFound"), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isResourceNotFound(tc.err))
		})
	}
}
