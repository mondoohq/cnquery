// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"

	abstractions "github.com/microsoft/kiota-abstractions-go"
	betamodels "github.com/microsoftgraph/msgraph-beta-sdk-go/models"
	"github.com/microsoftgraph/msgraph-beta-sdk-go/reports"
	"github.com/microsoftgraph/msgraph-sdk-go/devices"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/ms365/connection"
	"go.mondoo.com/mql/v13/types"
)

// see https://learn.microsoft.com/en-us/graph/api/resources/device?view=graph-rest-1.0
var deviceSelectFields = []string{
	"id", "displayName", "deviceId", "deviceCategory", "enrollmentProfileName", "enrollmentType",
	"isCompliant", "isManaged", "manufacturer", "isRooted", "mdmAppId", "model", "operatingSystem",
	"operatingSystemVersion", "physicalIds", "registrationDateTime", "systemLabels", "trustType",
}

func initMicrosoftDevices(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	args["__id"] = newListResourceIdFromArguments("microsoft.devices", args)
	resource, err := runtime.CreateResource(runtime, "microsoft.devices", args)
	if err != nil {
		return args, nil, err
	}

	return args, resource.(*mqlMicrosoftDevices), nil
}

// list fetches devices from Entra ID and allows the user provide a filter to retrieve
// a subset of devices
//
// Permissions: Device.Read.All
// see https://learn.microsoft.com/en-us/graph/api/device-list?view=graph-rest-1.0&tabs=http
func (a *mqlMicrosoftDevices) list() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}

	betaClient, err := conn.BetaGraphClient()
	if err != nil {
		return nil, err
	}

	// Index of devices are stored inside the top level resource `microsoft`, just like
	// MFA response. Here we create or get the resource to access those internals
	mainResource, err := CreateResource(a.MqlRuntime, "microsoft", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	microsoft := mainResource.(*mqlMicrosoft)

	// fetch device data
	ctx := context.Background()
	top := int32(999)
	opts := &devices.DevicesRequestBuilderGetRequestConfiguration{
		QueryParameters: &devices.DevicesRequestBuilderGetQueryParameters{
			Select: deviceSelectFields,
			Top:    &top,
		},
	}

	if a.Search.State == plugin.StateIsSet || a.Filter.State == plugin.StateIsSet {
		// search and filter requires this header
		headers := abstractions.NewRequestHeaders()
		headers.Add("ConsistencyLevel", "eventual")
		opts.Headers = headers

		if a.Search.State == plugin.StateIsSet {
			log.Debug().
				Str("search", a.Search.Data).
				Msg("microsoft.devices.list.search set")
			search, err := parseSearch(a.Search.Data)
			if err != nil {
				return nil, err
			}
			opts.QueryParameters.Search = &search
		}
		if a.Filter.State == plugin.StateIsSet {
			log.Debug().
				Str("filter", a.Filter.Data).
				Msg("microsoft.devices.list.filter set")
			opts.QueryParameters.Filter = &a.Filter.Data
			count := true
			opts.QueryParameters.Count = &count
		}
	}

	resp, err := graphClient.Devices().Get(ctx, opts)
	if err != nil {
		return nil, transformError(err)
	}
	devices, err := iterate[*models.Device](ctx,
		resp,
		graphClient.GetAdapter(),
		devices.CreateDeltaGetResponseFromDiscriminatorValue,
	)
	if err != nil {
		return nil, transformError(err)
	}

	detailsResp, err := betaClient.
		Reports().
		AuthenticationMethods().
		UserRegistrationDetails().
		Get(ctx,
			&reports.AuthenticationMethodsUserRegistrationDetailsRequestBuilderGetRequestConfiguration{
				QueryParameters: &reports.AuthenticationMethodsUserRegistrationDetailsRequestBuilderGetQueryParameters{
					Top: &top,
				},
			})
	// we do not want to fail the device fetching here, this likely means the tenant does not have the right license
	if err != nil {
		microsoft.mfaResp = mfaResp{err: err}
	} else {
		userRegistrationDetails, err := iterate[*betamodels.UserRegistrationDetails](ctx, detailsResp, betaClient.GetAdapter(), betamodels.CreateUserRegistrationDetailsCollectionResponseFromDiscriminatorValue)
		// we do not want to fail the device fetching here, this likely means the tenant does not have the right license
		if err != nil {
			microsoft.mfaResp = mfaResp{err: err}
		} else {
			mfaMap := map[string]bool{}
			for _, u := range userRegistrationDetails {
				if u.GetId() == nil || u.GetIsMfaRegistered() == nil {
					continue
				}
				mfaMap[*u.GetId()] = *u.GetIsMfaRegistered()
			}
			microsoft.mfaResp = mfaResp{mfaMap: mfaMap}
		}
	}

	// construct the result
	res := []any{}
	for _, u := range devices {
		graphDevice, err := newMqlMicrosoftDevice(a.MqlRuntime, u)
		if err != nil {
			return nil, err
		}
		// indexUser devices by id
		microsoft.indexDevice(graphDevice)
		res = append(res, graphDevice)
	}

	return res, nil
}

func initMicrosoftDevice(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	// we only look up the user if we have been supplied by id, displayName or userPrincipalName
	if len(args) > 1 {
		return args, nil, nil
	}

	rawId, okId := args["id"]
	rawDisplayName, okDisplayName := args["displayName"]

	if !okId && !okDisplayName {
		// required parameters are not set, we just pass-through the initialization arguments
		return args, nil, nil
	}

	var filter *string
	if okId {
		idFilter := fmt.Sprintf("id eq '%s'", rawId.Value.(string))
		filter = &idFilter
	} else if okDisplayName {
		displayNameFilter := fmt.Sprintf("displayName eq '%s'", rawDisplayName.Value.(string))
		filter = &displayNameFilter
	}
	if filter == nil {
		return nil, nil, errors.New("no filter found")
	}

	conn := runtime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, nil, err
	}

	ctx := context.Background()
	resp, err := graphClient.Devices().Get(ctx, &devices.DevicesRequestBuilderGetRequestConfiguration{
		QueryParameters: &devices.DevicesRequestBuilderGetQueryParameters{
			Filter: filter,
		},
	})
	if err != nil {
		return nil, nil, transformError(err)
	}

	val := resp.GetValue()
	if len(val) == 0 {
		return nil, nil, errors.New("device not found")
	}

	deviceId := val[0].GetId()
	if deviceId == nil {
		return nil, nil, errors.New("device id not found")
	}

	// fetch devices by id
	device, err := graphClient.Devices().ByDeviceId(*deviceId).Get(ctx, &devices.DeviceItemRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, nil, transformError(err)
	}
	mqlMsApp, err := newMqlMicrosoftDevice(runtime, device)
	if err != nil {
		return nil, nil, err
	}

	return nil, mqlMsApp, nil
}

func newMqlMicrosoftDevice(runtime *plugin.Runtime, u models.Deviceable) (*mqlMicrosoftDevice, error) {
	graphDevice, err := CreateResource(runtime, "microsoft.device",
		map[string]*llx.RawData{
			"__id":                   llx.StringDataPtr(u.GetId()),
			"id":                     llx.StringDataPtr(u.GetId()),
			"displayName":            llx.StringDataPtr(u.GetDisplayName()),
			"deviceId":               llx.StringDataPtr(u.GetDeviceId()),
			"deviceCategory":         llx.StringDataPtr(u.GetDeviceCategory()),
			"enrollmentProfileName":  llx.StringDataPtr(u.GetEnrollmentProfileName()),
			"enrollmentType":         llx.StringDataPtr(u.GetEnrollmentType()),
			"isCompliant":            llx.BoolDataPtr(u.GetIsCompliant()),
			"isManaged":              llx.BoolDataPtr(u.GetIsManaged()),
			"manufacturer":           llx.StringDataPtr(u.GetManufacturer()),
			"isRooted":               llx.BoolDataPtr(u.GetIsRooted()),
			"mdmAppId":               llx.StringDataPtr(u.GetMdmAppId()),
			"model":                  llx.StringDataPtr(u.GetModel()),
			"operatingSystem":        llx.StringDataPtr(u.GetOperatingSystem()),
			"operatingSystemVersion": llx.StringDataPtr(u.GetOperatingSystemVersion()),
			"physicalIds":            llx.ArrayData(convert.SliceAnyToInterface(u.GetPhysicalIds()), types.String),
			"registrationDateTime":   llx.TimeDataPtr(u.GetRegistrationDateTime()),
			"systemLabels":           llx.ArrayData(convert.SliceAnyToInterface(u.GetSystemLabels()), types.String),
			"trustType":              llx.StringDataPtr(u.GetTrustType()),
		})
	if err != nil {
		return nil, err
	}
	return graphDevice.(*mqlMicrosoftDevice), nil
}
