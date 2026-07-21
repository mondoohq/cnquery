// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// recordingRT is a stub http.RoundTripper that records the request it received
// and returns a canned response/error.
type recordingRT struct {
	resp *http.Response
	err  error
	got  *http.Request
}

func (r *recordingRT) RoundTrip(req *http.Request) (*http.Response, error) {
	r.got = req
	return r.resp, r.err
}

// The trace transport sits in front of every REST GCP call, so its contract is
// strict: delegate the request unchanged and return the base response/error
// verbatim. A regression here would corrupt or hide every REST API result.
func TestApiTraceTransport_PassesResponseThrough(t *testing.T) {
	want := &http.Response{StatusCode: http.StatusOK}
	base := &recordingRT{resp: want}
	tr := newApiTraceTransport(base)

	req, err := http.NewRequest(http.MethodGet, "https://compute.googleapis.com/x", nil)
	require.NoError(t, err)

	resp, err := tr.RoundTrip(req)
	require.NoError(t, err)
	require.Same(t, want, resp, "response must be returned unchanged")
	require.Same(t, req, base.got, "request must be delegated unchanged")
}

func TestApiTraceTransport_PassesErrorThroughWithNilResponse(t *testing.T) {
	wantErr := errors.New("transport failure")
	base := &recordingRT{err: wantErr} // resp is nil

	req, err := http.NewRequest(http.MethodGet, "https://compute.googleapis.com/x", nil)
	require.NoError(t, err)

	// Must not panic when the base returns a nil response (the status=0 path).
	resp, err := newApiTraceTransport(base).RoundTrip(req)
	require.Nil(t, resp)
	require.Equal(t, wantErr, err)
}

func TestNewApiTraceTransport_NilBaseFallsBackToDefault(t *testing.T) {
	tr := newApiTraceTransport(nil).(*apiTraceTransport)
	require.Equal(t, http.DefaultTransport, tr.base)
}

// The unary interceptor wraps every gRPC GCP call; it must invoke the real
// invoker and propagate its error unchanged.
func TestLoggingUnaryInterceptor_InvokesAndPropagatesError(t *testing.T) {
	// grpc.NewClient is lazy and does not dial, so this never touches the network.
	cc, err := grpc.NewClient("passthrough:///unit-test", grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer cc.Close()

	wantErr := errors.New("rpc failure")
	called := false
	invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		called = true
		return wantErr
	}

	got := loggingUnaryInterceptor(context.Background(), "/pkg.Service/Method", "req", "reply", cc, invoker)
	require.True(t, called, "interceptor must call the underlying invoker")
	require.Equal(t, wantErr, got, "interceptor must propagate the invoker error")
}
