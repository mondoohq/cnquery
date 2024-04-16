// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package health

import (
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/cli/config"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/upstream"
	"go.mondoo.com/ranger-rpc"
	"runtime/debug"
)

//go:generate protoc --proto_path=. --go_out=. --go_opt=paths=source_relative --rangerrpc_out=. errors.proto

type PanicReportFn func(product, version, build string, r any, stacktrace []byte)

func ReportPanic(product, version, build string, reporters ...PanicReportFn) {
	if r := recover(); r != nil {
		sendPanic(product, version, build, r, debug.Stack())

		// call additional reporters
		for _, reporter := range reporters {
			reporter(product, version, build, r, debug.Stack())
		}

		// output error to console
		panic(r)
	}
}

// sendPanic sends a panic to the mondoo platform for further analysis if the
// service account is configured.
// This function does not return an error as it is not critical to send the panic to the platform.
func sendPanic(product, version, build string, r any, stacktrace []byte) {
	// 1. read config
	opts, err := config.Read()
	if err != nil {
		log.Error().Err(err).Msg("failed to read config")
		return
	}

	serviceAccount := opts.GetServiceCredential()
	if serviceAccount == nil {
		log.Error().Msg("no service account configured")
		return
	}

	// 2. create local support bundle
	event := &SendErrorReq{
		ServiceAccountMrn: opts.ServiceAccountMrn,
		AgentMrn:          opts.AgentMrn,
		Product: &ProductInfo{
			Name:    product,
			Version: version,
			Build:   build,
		},
		Error: &ErrorInfo{
			Message:    "panic: " + fmt.Sprintf("%v", r),
			Stacktrace: string(stacktrace),
		},
	}

	// 3. send error to mondoo platform
	proxy, err := config.GetAPIProxy()
	if err != nil {
		log.Error().Err(err).Msg("failed to parse proxy setting")
		return
	}
	httpClient := ranger.NewHttpClient(ranger.WithProxy(proxy))

	plugins := []ranger.ClientPlugin{}
	certAuth, err := upstream.NewServiceAccountRangerPlugin(serviceAccount)
	if err != nil {
		return
	}
	plugins = append(plugins, certAuth)

	cl, err := NewErrorReportingClient(serviceAccount.ApiEndpoint, httpClient, plugins...)
	if err != nil {
		log.Error().Err(err).Msg("failed to create error reporting client")
		return
	}

	_, err = cl.SendError(context.Background(), event)
	if err != nil {
		log.Error().Err(err).Msg("failed to send error to mondoo platform")
		return
	}

	log.Info().Msg("reported panic to Mondoo platform")
}
