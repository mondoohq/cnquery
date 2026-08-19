// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azqueue"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/azure/connection"
)

// queueSignedIdentifiersToDicts converts queue stored access policies into the
// same dict shape the table resource already reports, so a query that spans
// queues and tables reads one set of keys.
//
// Times are emitted as RFC3339 strings rather than as time values because the
// surrounding value is a dict, whose contents have to stay JSON-native.
func queueSignedIdentifiersToDicts(identifiers []*azqueue.SignedIdentifier) []any {
	res := []any{}
	for _, si := range identifiers {
		if si == nil {
			continue
		}
		entry := map[string]any{}
		if si.ID != nil {
			entry["id"] = *si.ID
		}
		if si.AccessPolicy != nil {
			if si.AccessPolicy.Permission != nil {
				entry["permission"] = *si.AccessPolicy.Permission
			}
			if si.AccessPolicy.Start != nil {
				entry["startTime"] = si.AccessPolicy.Start.Format(time.RFC3339)
			}
			if si.AccessPolicy.Expiry != nil {
				entry["expiryTime"] = si.AccessPolicy.Expiry.Format(time.RFC3339)
			}
		}
		res = append(res, entry)
	}
	return res
}

// isQueueDataPlaneUnreadable reports whether the error says the queue's access
// policy could not be read at all, as opposed to saying there are no policies
// on it. Reading the policies goes through the queue service, so a principal
// holding only Azure Resource Manager rights, or one calling into an account
// whose firewall excludes it, is refused here even though the queue itself
// listed fine.
func isQueueDataPlaneUnreadable(err error) bool {
	var respErr *azcore.ResponseError
	if !errors.As(err, &respErr) {
		return false
	}
	switch respErr.StatusCode {
	case http.StatusForbidden, http.StatusUnauthorized, http.StatusNotFound:
		return true
	}
	return false
}

// signedIdentifiers returns the queue's stored access policies.
//
// This is a per-queue call against the queue service, so it stays a computed
// field rather than being fetched while the queues are listed: an account with
// many queues would otherwise pay for all of them to answer a query that never
// mentions this field.
func (a *mqlAzureSubscriptionStorageServiceAccountQueue) signedIdentifiers() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)

	_, accountName, err := storageAccountResourceGroup(a.Id.Data)
	if err != nil {
		return nil, err
	}

	queueURL, err := azureStorageDataPlaneURL(accountName, "queue", a.Name.Data)
	if err != nil {
		return nil, err
	}

	client, err := azqueue.NewQueueClient(queueURL, conn.Token(), &azqueue.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	resp, err := client.GetAccessPolicy(context.Background(), nil)
	if err != nil {
		// An unreadable policy is not an absent one. Reporting the empty list
		// here would assert that the queue has no stored access policy, which
		// is exactly the finding a policy audit acts on.
		if isQueueDataPlaneUnreadable(err) {
			a.SignedIdentifiers.State = plugin.StateIsNull | plugin.StateIsSet
			return nil, nil
		}
		return nil, err
	}

	return queueSignedIdentifiersToDicts(resp.SignedIdentifiers), nil
}
