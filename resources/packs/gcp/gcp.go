package gcp

import (
	"errors"
	"time"

	"go.mondoo.com/cnquery/motor/providers"
	gcp_provider "go.mondoo.com/cnquery/motor/providers/google"
	"go.mondoo.com/cnquery/resources/packs/gcp/info"
)

var Registry = info.Registry

func init() {
	Init(Registry)
}

func gcpProvider(t providers.Instance) (*gcp_provider.Provider, error) {
	provider, ok := t.(*gcp_provider.Provider)
	if !ok {
		return nil, errors.New("gcp resource is not supported on this provider")
	}
	return provider, nil
}

// parseTime parses RFC 3389 timestamps "2019-06-12T21:14:13.190Z"
func parseTime(timestamp string) *time.Time {
	parsedCreated, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return nil
	}
	return &parsedCreated
}
