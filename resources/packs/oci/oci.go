package oci

import (
	"context"
	"errors"
	"time"

	"go.mondoo.com/cnquery/motor/providers"
	oci_provider "go.mondoo.com/cnquery/motor/providers/oci"
	"go.mondoo.com/cnquery/resources/packs/core"
	"go.mondoo.com/cnquery/resources/packs/oci/info"
)

var Registry = info.Registry

func init() {
	Init(Registry)
}

func ociProvider(t providers.Instance) (*oci_provider.Provider, error) {
	provider, ok := t.(*oci_provider.Provider)
	if !ok {
		return nil, errors.New("oci resource is not supported on this provider")
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

func (o *mqlOci) id() (string, error) {
	return "oci", nil
}

func (o *mqlOci) GetRegions() ([]interface{}, error) {
	provider, err := ociProvider(o.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	regions, err := provider.GetRegions(context.Background())
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range regions {
		region := regions[i]
		mqlRegion, err := o.MotorRuntime.CreateResource("oci.region",
			"id", core.ToString(region.RegionKey),
			"name", core.ToString(region.RegionName),
			"isHomeRegion", core.ToBool(region.IsHomeRegion),
			"status", string(region.Status),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlRegion)
	}

	return res, nil
}

func (o *mqlOciRegion) id() (string, error) {
	id, err := o.Id()
	if err != nil {
		return "", err
	}
	return "oci.region/" + id, nil
}

func (o *mqlOci) GetCompartments() ([]interface{}, error) {
	provider, err := ociProvider(o.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	compartments, err := provider.GetCompartments(context.Background())
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range compartments {
		compartment := compartments[i]

		var created *time.Time
		if compartment.TimeCreated != nil {
			created = &compartment.TimeCreated.Time
		}

		mqlCompartment, err := o.MotorRuntime.CreateResource("oci.compartment",
			"id", core.ToString(compartment.Id),
			"name", core.ToString(compartment.Name),
			"description", core.ToString(compartment.Description),
			"created", created,
			"lifecycleState", string(compartment.LifecycleState),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlCompartment)
	}

	return res, nil
}

func (o *mqlOciCompartment) id() (string, error) {
	id, err := o.Id()
	if err != nil {
		return "", err
	}
	return "oci.compartment/" + id, nil
}
