// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	network "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v10"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
)

// effectiveRouteTable resolves the routes actually applied to the network
// interface, merging system, user-defined, and BGP-learned routes. This is a
// long-running operation and is only available when the NIC is attached to a
// running VM; when it is not, Azure returns a 4xx that we surface as an empty
// list rather than an error.
func (a *mqlAzureSubscriptionNetworkServiceInterface) effectiveRouteTable() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	// Bound the long-poll so a stuck operation doesn't hang the interfaces query.
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resourceID, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return nil, err
	}
	nicName, err := resourceID.Component("networkInterfaces")
	if err != nil {
		return nil, err
	}

	client, err := network.NewInterfacesClient(resourceID.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	poller, err := client.BeginGetEffectiveRouteTable(ctx, resourceID.ResourceGroup, nicName, nil)
	if err != nil {
		return effectiveRouteTableErr(err, nicName)
	}
	resp, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return effectiveRouteTableErr(err, nicName)
	}

	res := []any{}
	for _, route := range resp.Value {
		if route == nil {
			continue
		}
		dict, err := convert.JsonToDict(route)
		if err != nil {
			return nil, err
		}
		res = append(res, dict)
	}
	return res, nil
}

// effectiveRouteTableErr treats a 4xx (typically a NIC not attached to a
// running VM, or missing permissions) as "no effective routes available"
// rather than a hard error, so one such NIC doesn't fail the whole query.
func effectiveRouteTableErr(err error, nicName string) ([]any, error) {
	var respErr *azcore.ResponseError
	if errors.As(err, &respErr) && respErr.StatusCode >= 400 && respErr.StatusCode < 500 {
		log.Warn().Str("nic", nicName).Int("status", respErr.StatusCode).Msg("effective route table unavailable for NIC")
		return []any{}, nil
	}
	return nil, err
}
