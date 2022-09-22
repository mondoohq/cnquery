package health

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/ranger-rpc"
)

//go:generate protoc --proto_path=. --go_out=. --go_opt=paths=source_relative --rangerrpc_out=. health.proto

type Status struct {
	API struct {
		Endpoint  string `json:"endpoint,omitempty"`
		Status    string `json:"status,omitempty"`
		Timestamp string `json:"timestamp,omitempty"`
		Version   string `json:"version,omitempty"`
	} `json:"api,omitempty"`
	Features []string `json:"features,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

func CheckApiHealth(endpoint string) Status {
	status := Status{}
	status.API.Endpoint = endpoint

	sendTime := time.Now()
	healthClient, err := NewHealthClient(endpoint, ranger.DefaultHttpClient())
	if err != nil {
		log.Error().Err(err).Msg("could not run api health check")
	}
	healthResp, err := healthClient.Check(context.Background(), &HealthCheckRequest{})
	if err != nil {
		log.Error().Err(err).Msg("could not run api health check")
	} else {
		status.API.Status = healthResp.Status.String()
		status.API.Timestamp = healthResp.Time
		status.API.Version = healthResp.ApiVersion

		// do time check to make it easier to dected ssl/tls issues
		recievedResponseTime := time.Now()
		roundTripDuration := recievedResponseTime.Sub(sendTime)
		if roundTripDuration > time.Second*5 {
			status.Warnings = append(status.Warnings, "detected very long round-trip times: "+roundTripDuration.String())
		}

		upstream, err := time.Parse(time.RFC3339, healthResp.Time)
		if err != nil {
			status.Warnings = append(status.Warnings, "cannot run clock skew check")
		} else {
			diff := upstream.Sub(sendTime)
			if abs(diff) > time.Second*30 {
				status.Warnings = append(status.Warnings, "possible clock skew detected: "+diff.String())
			}
		}
	}
	return status
}

func abs(a time.Duration) time.Duration {
	if a >= 0 {
		return a
	}
	return -a
}
