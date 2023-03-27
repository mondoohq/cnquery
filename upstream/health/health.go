package health

import (
	"context"
	"time"

	"go.mondoo.com/cnquery/shared/rangerclient"
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

func CheckApiHealth(endpoint, proxy string) (Status, error) {
	status := Status{}
	status.API.Endpoint = endpoint

	sendTime := time.Now()

	rangerClient, err := rangerclient.NewRangerClient(&rangerclient.RangerClientOpts{
		Proxy: proxy,
	})
	if err != nil {
		return status, err
	}

	healthClient, err := NewHealthClient(endpoint, rangerClient)
	if err != nil {
		return status, err
	}
	healthResp, err := healthClient.Check(context.Background(), &HealthCheckRequest{})
	if err != nil {
		return status, err
	} else {
		status.API.Status = healthResp.Status.String()
		status.API.Timestamp = healthResp.Time
		status.API.Version = healthResp.ApiVersion

		// do time check to make it easier to detect ssl/tls issues
		receivedResponseTime := time.Now()
		roundTripDuration := receivedResponseTime.Sub(sendTime)
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
	return status, nil
}

func abs(a time.Duration) time.Duration {
	if a >= 0 {
		return a
	}
	return -a
}
