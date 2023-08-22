// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package explorer

import (
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	spb "google.golang.org/genproto/googleapis/rpc/status"
)

// cnquery codes start at 100 to avoid conflicts with GRPC and Ranger RPC codes
// see https://github.com/mondoohq/ranger-rpc/blob/main/codes/codes.go
type ErrorStatusCode uint32

const (
	Unknown ErrorStatusCode = iota + 100
	// NotApplicable is returned when a query or asset is not applicable to the current scan
	NotApplicable
	// NoQueries is returned when no queries are found in the bundle for the asset
	NoQueries
)

func NewErrorStatusCodeFromString(s string) ErrorStatusCode {
	switch s {
	case "NotApplicable":
		return NotApplicable
	case "NoQueries":
		return NoQueries
	}
	return Unknown
}

func (e ErrorStatusCode) String() string {
	switch e {
	case NotApplicable:
		return "NotApplicable"
	case NoQueries:
		return "NoQueries"
	default:
		return "Unknown"
	}
}

// cnquery codes start at 100 to avoid conflicts with GRPC and Ranger RPC codes
type ErrorCategory uint32

const (
	ErrorCategoryError ErrorCategory = iota
	ErrorCategoryWarning
	ErrorCategoryInformational
)

func (e ErrorCategory) String() string {
	switch e {
	case ErrorCategoryInformational:
		return "info"
	case ErrorCategoryWarning:
		return "warning"
	case ErrorCategoryError:
		return "error"
	default:
		return "error"
	}
}

func (e ErrorStatusCode) Category() ErrorCategory {
	switch e {
	case NotApplicable:
		return ErrorCategoryInformational
	case NoQueries:
		return ErrorCategoryWarning
	default:
		return ErrorCategoryError
	}
}

func NewErrorStatus(err error) *ErrorStatus {
	s, ok := status.FromError(err)
	if !ok {
		// not status error - just add it as a generic error
		return &ErrorStatus{
			Code:    int32(codes.Unknown),
			Message: err.Error(),
		}
	}

	// if it's a status error, add it as a status error with details if possible
	p := s.Proto()
	if p != nil {
		return &ErrorStatus{
			Code:    p.GetCode(),
			Message: p.GetMessage(),
			Details: p.Details,
		}
	}
	return &ErrorStatus{
		Code:    int32(s.Code()),
		Message: s.Message(),
	}
}

func (e *ErrorStatus) StatusProto() *spb.Status {
	return &spb.Status{
		Code:    e.Code,
		Message: e.Message,
		Details: e.Details,
	}
}

func (e *ErrorStatus) ErrorCode() ErrorStatusCode {
	s := status.FromProto(e.StatusProto())
	for _, detail := range s.Details() {
		switch v := detail.(type) {
		case *errdetails.ErrorInfo:
			code, ok := v.Metadata["errorCode"]
			if ok {
				return NewErrorStatusCodeFromString(code)
			}
		}
	}

	return Unknown
}
