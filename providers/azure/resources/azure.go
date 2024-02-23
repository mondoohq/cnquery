// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

// TODO: we should look into restructuring resources for v11.
// we should be able to define the subscription as a property on the azure one, i.e.
//
//	azure {
//	  subscription() azure.subscription
//	}
//
// right now this isn't possible as the resource lookup gets confused between trying to directly create azure.subscription
// or create azure and then do azure.subscription()
type mqlAzureInternal struct {
	sub *mqlAzureSubscription
}
