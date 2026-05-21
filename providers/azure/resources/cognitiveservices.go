// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cognitiveservices/armcognitiveservices/v3"
	"github.com/rs/zerolog/log"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"
)

func initAzureSubscriptionCognitiveServicesService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}

	conn, ok := runtime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not an Azure connection")
	}
	args["subscriptionId"] = llx.StringData(conn.SubId())

	return args, nil, nil
}

func (a *mqlAzureSubscriptionCognitiveServicesService) id() (string, error) {
	return "azure.subscription.cognitiveServicesService/" + a.SubscriptionId.Data, nil
}

func (a *mqlAzureSubscriptionCognitiveServicesServiceAccount) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionCognitiveServicesService) accounts() ([]any, error) {
	conn, ok := a.MqlRuntime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, errors.New("invalid connection provided, it is not an Azure connection")
	}

	ctx := context.Background()
	client, err := armcognitiveservices.NewAccountsClient(a.SubscriptionId.Data, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(nil)
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusForbidden {
				log.Warn().Err(err).Msg("could not list cognitive services accounts due to access denied")
				return res, nil
			}
			return nil, err
		}
		for _, account := range page.Value {
			if account == nil {
				continue
			}
			mqlAccount, err := cognitiveServicesAccountToMql(a.MqlRuntime, account)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAccount)
		}
	}
	return res, nil
}

func cognitiveServicesAccountToMql(runtime *plugin.Runtime, account *armcognitiveservices.Account) (*mqlAzureSubscriptionCognitiveServicesServiceAccount, error) {
	sku, err := convert.JsonToDict(account.SKU)
	if err != nil {
		return nil, err
	}
	identity, err := convert.JsonToDict(account.Identity)
	if err != nil {
		return nil, err
	}

	var publicNetworkAccess, customSubDomainName string
	var cmkKeySource, cmkKeyName, cmkKeyVaultUri string
	var endpoint, provisioningState string
	var disableLocalAuth, restrictOutboundNetworkAccess, storedCompletionsDisabled bool
	networkAcls := llx.NilData

	if p := account.Properties; p != nil {
		if p.PublicNetworkAccess != nil {
			publicNetworkAccess = string(*p.PublicNetworkAccess)
		}
		if p.DisableLocalAuth != nil {
			disableLocalAuth = *p.DisableLocalAuth
		}
		if p.RestrictOutboundNetworkAccess != nil {
			restrictOutboundNetworkAccess = *p.RestrictOutboundNetworkAccess
		}
		if p.CustomSubDomainName != nil {
			customSubDomainName = *p.CustomSubDomainName
		}
		if p.NetworkACLs != nil {
			d, err := convert.JsonToDict(p.NetworkACLs)
			if err != nil {
				return nil, err
			}
			networkAcls = llx.DictData(d)
		}
		if enc := p.Encryption; enc != nil {
			if enc.KeySource != nil {
				cmkKeySource = string(*enc.KeySource)
			}
			if enc.KeyVaultProperties != nil {
				if enc.KeyVaultProperties.KeyName != nil {
					cmkKeyName = *enc.KeyVaultProperties.KeyName
				}
				if enc.KeyVaultProperties.KeyVaultURI != nil {
					cmkKeyVaultUri = *enc.KeyVaultProperties.KeyVaultURI
				}
			}
		}
		if p.Endpoint != nil {
			endpoint = *p.Endpoint
		}
		if p.ProvisioningState != nil {
			provisioningState = string(*p.ProvisioningState)
		}
		if p.StoredCompletionsDisabled != nil {
			storedCompletionsDisabled = *p.StoredCompletionsDisabled
		}
	}

	res, err := CreateResource(runtime, "azure.subscription.cognitiveServicesService.account", map[string]*llx.RawData{
		"id":                            llx.StringDataPtr(account.ID),
		"name":                          llx.StringDataPtr(account.Name),
		"location":                      llx.StringDataPtr(account.Location),
		"kind":                          llx.StringDataPtr(account.Kind),
		"tags":                          llx.MapData(convert.PtrMapStrToInterface(account.Tags), types.String),
		"sku":                           llx.DictData(sku),
		"identity":                      llx.DictData(identity),
		"publicNetworkAccess":           llx.StringData(publicNetworkAccess),
		"disableLocalAuth":              llx.BoolData(disableLocalAuth),
		"restrictOutboundNetworkAccess": llx.BoolData(restrictOutboundNetworkAccess),
		"customSubDomainName":           llx.StringData(customSubDomainName),
		"networkAcls":                   networkAcls,
		"cmkKeySource":                  llx.StringData(cmkKeySource),
		"cmkKeyName":                    llx.StringData(cmkKeyName),
		"cmkKeyVaultUri":                llx.StringData(cmkKeyVaultUri),
		"endpoint":                      llx.StringData(endpoint),
		"provisioningState":             llx.StringData(provisioningState),
		"storedCompletionsDisabled":     llx.BoolData(storedCompletionsDisabled),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionCognitiveServicesServiceAccount), nil
}

func (a *mqlAzureSubscriptionCognitiveServicesServiceAccountRaiPolicy) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionCognitiveServicesServiceAccountRaiTopic) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionCognitiveServicesServiceAccount) raiPolicies() ([]any, error) {
	conn, ok := a.MqlRuntime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, errors.New("invalid connection provided, it is not an Azure connection")
	}

	parsed, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return nil, err
	}
	accountName := parsed.Path["accounts"]

	ctx := context.Background()
	client, err := armcognitiveservices.NewRaiPoliciesClient(parsed.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(parsed.ResourceGroup, accountName, nil)
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusForbidden {
				log.Warn().Err(err).Msg("could not list cognitive services RAI policies due to access denied")
				return res, nil
			}
			return nil, err
		}
		for _, pol := range page.Value {
			if pol == nil {
				continue
			}
			mqlPol, err := raiPolicyToMql(a.MqlRuntime, pol)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlPol)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionCognitiveServicesServiceAccount) raiTopics() ([]any, error) {
	conn, ok := a.MqlRuntime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, errors.New("invalid connection provided, it is not an Azure connection")
	}

	parsed, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return nil, err
	}
	accountName := parsed.Path["accounts"]

	ctx := context.Background()
	client, err := armcognitiveservices.NewRaiTopicsClient(parsed.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(parsed.ResourceGroup, accountName, nil)
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusForbidden {
				log.Warn().Err(err).Msg("could not list cognitive services RAI topics due to access denied")
				return res, nil
			}
			return nil, err
		}
		for _, t := range page.Value {
			if t == nil {
				continue
			}
			mqlTopic, err := raiTopicToMql(a.MqlRuntime, t)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlTopic)
		}
	}
	return res, nil
}

func raiPolicyToMql(runtime *plugin.Runtime, pol *armcognitiveservices.RaiPolicy) (*mqlAzureSubscriptionCognitiveServicesServiceAccountRaiPolicy, error) {
	var mode, policyType, basePolicyName string
	contentFilters := []any{}
	customBlocklists := []any{}
	topicRefs := []any{}

	if p := pol.Properties; p != nil {
		if p.Mode != nil {
			mode = string(*p.Mode)
		}
		if p.Type != nil {
			policyType = string(*p.Type)
		}
		if p.BasePolicyName != nil {
			basePolicyName = *p.BasePolicyName
		}
		for i, f := range p.ContentFilters {
			if f == nil {
				continue
			}
			cf, err := raiContentFilterToMql(runtime, pol, i, f)
			if err != nil {
				return nil, err
			}
			contentFilters = append(contentFilters, cf)
		}
		for _, b := range p.CustomBlocklists {
			if b == nil {
				continue
			}
			entry := map[string]any{}
			if b.BlocklistName != nil {
				entry["blocklistName"] = *b.BlocklistName
			}
			if b.Source != nil {
				entry["source"] = string(*b.Source)
			}
			if b.Blocking != nil {
				entry["blocking"] = *b.Blocking
			}
			customBlocklists = append(customBlocklists, entry)
		}
		for _, t := range p.CustomTopics {
			if t == nil {
				continue
			}
			ref, err := raiTopicRefToMql(runtime, pol, t)
			if err != nil {
				return nil, err
			}
			topicRefs = append(topicRefs, ref)
		}
	}

	res, err := CreateResource(runtime, "azure.subscription.cognitiveServicesService.account.raiPolicy", map[string]*llx.RawData{
		"id":               llx.StringDataPtr(pol.ID),
		"name":             llx.StringDataPtr(pol.Name),
		"mode":             llx.StringData(mode),
		"policyType":       llx.StringData(policyType),
		"basePolicyName":   llx.StringData(basePolicyName),
		"contentFilters":   llx.ArrayData(contentFilters, types.Resource("azure.subscription.cognitiveServicesService.account.raiPolicy.contentFilter")),
		"customBlocklists": llx.ArrayData(customBlocklists, types.Dict),
		"customTopics":     llx.ArrayData(topicRefs, types.Resource("azure.subscription.cognitiveServicesService.account.raiPolicy.topicRef")),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionCognitiveServicesServiceAccountRaiPolicy), nil
}

func raiContentFilterToMql(runtime *plugin.Runtime, pol *armcognitiveservices.RaiPolicy, idx int, f *armcognitiveservices.RaiPolicyContentFilter) (*mqlAzureSubscriptionCognitiveServicesServiceAccountRaiPolicyContentFilter, error) {
	policyId := ""
	if pol.ID != nil {
		policyId = *pol.ID
	}
	name := ""
	source := ""
	severityThreshold := ""
	var enabled, blocking bool
	if f.Name != nil {
		name = *f.Name
	}
	if f.Source != nil {
		source = string(*f.Source)
	}
	if f.SeverityThreshold != nil {
		severityThreshold = string(*f.SeverityThreshold)
	}
	if f.Enabled != nil {
		enabled = *f.Enabled
	}
	if f.Blocking != nil {
		blocking = *f.Blocking
	}
	id := fmt.Sprintf("%s/contentFilter/%s/%s", policyId, source, name)
	if name == "" && source == "" {
		id = fmt.Sprintf("%s/contentFilter/%d", policyId, idx)
	}
	res, err := CreateResource(runtime, "azure.subscription.cognitiveServicesService.account.raiPolicy.contentFilter", map[string]*llx.RawData{
		"__id":              llx.StringData(id),
		"name":              llx.StringData(name),
		"source":            llx.StringData(source),
		"enabled":           llx.BoolData(enabled),
		"blocking":          llx.BoolData(blocking),
		"severityThreshold": llx.StringData(severityThreshold),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionCognitiveServicesServiceAccountRaiPolicyContentFilter), nil
}

func raiTopicRefToMql(runtime *plugin.Runtime, pol *armcognitiveservices.RaiPolicy, t *armcognitiveservices.CustomTopicConfig) (*mqlAzureSubscriptionCognitiveServicesServiceAccountRaiPolicyTopicRef, error) {
	policyId := ""
	if pol.ID != nil {
		policyId = *pol.ID
	}
	topicName := ""
	source := ""
	var blocking bool
	if t.TopicName != nil {
		topicName = *t.TopicName
	}
	if t.Source != nil {
		source = string(*t.Source)
	}
	if t.Blocking != nil {
		blocking = *t.Blocking
	}
	id := fmt.Sprintf("%s/topicRef/%s/%s", policyId, source, topicName)
	res, err := CreateResource(runtime, "azure.subscription.cognitiveServicesService.account.raiPolicy.topicRef", map[string]*llx.RawData{
		"__id":      llx.StringData(id),
		"topicName": llx.StringData(topicName),
		"source":    llx.StringData(source),
		"blocking":  llx.BoolData(blocking),
	})
	if err != nil {
		return nil, err
	}
	ref := res.(*mqlAzureSubscriptionCognitiveServicesServiceAccountRaiPolicyTopicRef)
	if parsed, err := ParseResourceID(policyId); err == nil {
		ref.cacheTopicId = fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.CognitiveServices/accounts/%s/raiTopics/%s",
			parsed.SubscriptionID, parsed.ResourceGroup, parsed.Path["accounts"], topicName)
	}
	return ref, nil
}

type mqlAzureSubscriptionCognitiveServicesServiceAccountRaiPolicyTopicRefInternal struct {
	cacheTopicId string
}

func (a *mqlAzureSubscriptionCognitiveServicesServiceAccountRaiPolicyTopicRef) topic() (*mqlAzureSubscriptionCognitiveServicesServiceAccountRaiTopic, error) {
	if a.cacheTopicId == "" {
		a.Topic.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	mqlTopic, err := NewResource(a.MqlRuntime, "azure.subscription.cognitiveServicesService.account.raiTopic",
		map[string]*llx.RawData{"id": llx.StringData(a.cacheTopicId)})
	if err != nil {
		return nil, err
	}
	return mqlTopic.(*mqlAzureSubscriptionCognitiveServicesServiceAccountRaiTopic), nil
}

func initAzureSubscriptionCognitiveServicesServiceAccountRaiTopic(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}
	idArg, ok := args["id"]
	if !ok {
		return args, nil, nil
	}
	id := idArg.Value.(string)

	conn, ok := runtime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not an Azure connection")
	}
	parsed, err := ParseResourceID(id)
	if err != nil {
		return nil, nil, err
	}
	accountName := parsed.Path["accounts"]
	topicName := parsed.Path["raitopics"]
	if accountName == "" || topicName == "" {
		return args, nil, nil
	}

	client, err := armcognitiveservices.NewRaiTopicsClient(parsed.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, nil, err
	}
	resp, err := client.Get(context.Background(), parsed.ResourceGroup, accountName, topicName, nil)
	if err != nil {
		return nil, nil, err
	}
	mqlTopic, err := raiTopicToMql(runtime, &resp.RaiTopic)
	if err != nil {
		return nil, nil, err
	}
	return nil, mqlTopic, nil
}

func raiTopicToMql(runtime *plugin.Runtime, t *armcognitiveservices.RaiTopic) (*mqlAzureSubscriptionCognitiveServicesServiceAccountRaiTopic, error) {
	var topicName, topicId, description, status, failedReason, sampleBlobUrl string
	var createdAt, lastModifiedAt *time.Time

	if p := t.Properties; p != nil {
		if p.TopicName != nil {
			topicName = *p.TopicName
		}
		if p.TopicID != nil {
			topicId = *p.TopicID
		}
		if p.Description != nil {
			description = *p.Description
		}
		if p.Status != nil {
			status = *p.Status
		}
		if p.FailedReason != nil {
			failedReason = *p.FailedReason
		}
		if p.SampleBlobURL != nil {
			sampleBlobUrl = *p.SampleBlobURL
		}
		createdAt = p.CreatedAt
		lastModifiedAt = p.LastModifiedAt
	}

	res, err := CreateResource(runtime, "azure.subscription.cognitiveServicesService.account.raiTopic", map[string]*llx.RawData{
		"id":             llx.StringDataPtr(t.ID),
		"name":           llx.StringDataPtr(t.Name),
		"topicName":      llx.StringData(topicName),
		"topicId":        llx.StringData(topicId),
		"description":    llx.StringData(description),
		"status":         llx.StringData(status),
		"failedReason":   llx.StringData(failedReason),
		"sampleBlobUrl":  llx.StringData(sampleBlobUrl),
		"createdAt":      llx.TimeDataPtr(createdAt),
		"lastModifiedAt": llx.TimeDataPtr(lastModifiedAt),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionCognitiveServicesServiceAccountRaiTopic), nil
}
