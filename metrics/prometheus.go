// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

const prometheusAddr = ":2112"

func Start() {
	// Create a custom Prometheus registry to avoid global conflicts
	registry := prometheus.NewRegistry()

	// Register standard Go and process collectors
	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	log.Info().Str("addr", prometheusAddr).Msg("Starting prometheus metrics server")
	if err := http.ListenAndServe(prometheusAddr, nil); err != nil {
		log.Fatal().Err(err).Msg("Error starting HTTP server")
	}
}
