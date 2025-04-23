// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package metrics

import (
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

// This setting matches what we have inside 'prometheus.yml' config
const prometheusDefaultAddr = ":2112"

// Start starts a metrics server only when this app is run on debug mode. It has basic metric
// collectors that captures things like number of goroutines or cpu and memory utilization.
func Start() {
	if os.Getenv("DEBUG") != "1" {
		return // not in debug mode
	}

	// Create a custom Prometheus registry to avoid global conflicts
	registry := prometheus.NewRegistry()

	// Register standard Go and process collectors
	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	// Register custom metrics here, if needed

	// Serve metrics using the custom registry
	http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	log.Info().Str("addr", prometheusDefaultAddr).Msg("Starting prometheus metrics server")
	if err := http.ListenAndServe(prometheusDefaultAddr, nil); err != nil {
		log.Fatal().Err(err).Msg("Error starting HTTP server")
	}
}
