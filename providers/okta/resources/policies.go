// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/okta/okta-sdk-golang/v2/okta/query"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers/okta/connection"
	"go.mondoo.com/cnquery/v10/providers/okta/resources/sdk"
)

// https://developer.okta.com/docs/reference/api/policy/#policy-object
type PolicyType string

const (
	OKTA_SIGN_ON               PolicyType = "OKTA_SIGN_ON"
	PASSWORD                              = "PASSWORD"
	MFA_ENROLL                            = "MFA_ENROLL"
	OAUTH_AUTHORIZATION_POLICY            = "OAUTH_AUTHORIZATION_POLICY"
	IDP_DISCOVERY                         = "IDP_DISCOVERY"
	ACCESS_POLICY                         = "ACCESS_POLICY"
	PROFILE_ENROLLMENT                    = "PROFILE_ENROLLMENT"
)

func (o *mqlOktaPolicies) id() (string, error) {
	return "okta.policies", nil
}

func listPolicies(runtime *plugin.Runtime, policyType PolicyType) ([]interface{}, error) {
	conn := runtime.Connection.(*connection.OktaConnection)
	client := conn.Client()

	ctx := context.Background()
	apiSupplement := &sdk.ApiExtension{
		RequestExecutor: client.CloneRequestExecutor(),
	}

	respList, resp, err := apiSupplement.ListPolicies(
		ctx,
		query.NewQueryParams(
			query.WithLimit(queryLimit),
			query.WithType(string(policyType)),
		),
	)
	// handle case where no policy exists
	if err != nil && resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	// handle special case where the policy type does not exist
	if err != nil && resp.StatusCode == http.StatusBadRequest && strings.Contains(strings.ToLower(err.Error()), "invalid policy type") {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	if len(respList) == 0 {
		return nil, nil
	}

	list := []interface{}{}
	appendEntry := func(datalist ...*sdk.PolicyWrapper) error {
		for i := range datalist {
			r, err := newMqlOktaPolicy(runtime, datalist[i])
			if err != nil {
				return err
			}
			list = append(list, r)
		}
		return nil
	}

	for i := range respList {
		err = appendEntry(respList[i])
		if err != nil {
			return nil, err
		}

	}

	// TODO: pagination not working properly for that call, need to chat with Okta
	//for resp.HasNextPage() {
	//	var slice []*okta.Policy
	//	resp, err = resp.Next(ctx, &slice)
	//	if err != nil {
	//		return nil, err
	//	}
	//	//	//	err = appendEntry(slice...)
	//	//	//	if err != nil {
	//	//	//		return nil, err
	//	//	//	}
	//}
	return list, nil
}

func (o *mqlOktaPolicies) password() ([]interface{}, error) {
	return listPolicies(o.MqlRuntime, PASSWORD)
}

func (o *mqlOktaPolicies) mfaEnroll() ([]interface{}, error) {
	return listPolicies(o.MqlRuntime, MFA_ENROLL)
}

func (o *mqlOktaPolicies) signOn() ([]interface{}, error) {
	return listPolicies(o.MqlRuntime, OKTA_SIGN_ON)
}

func (o *mqlOktaPolicies) oauthAuthorizationPolicy() ([]interface{}, error) {
	return listPolicies(o.MqlRuntime, OAUTH_AUTHORIZATION_POLICY)
}

func (o *mqlOktaPolicies) idpDiscovery() ([]interface{}, error) {
	return listPolicies(o.MqlRuntime, IDP_DISCOVERY)
}

func (o *mqlOktaPolicies) accessPolicy() ([]interface{}, error) {
	return listPolicies(o.MqlRuntime, ACCESS_POLICY)
}

func (o *mqlOktaPolicies) profileEnrollment() ([]interface{}, error) {
	return listPolicies(o.MqlRuntime, PROFILE_ENROLLMENT)
}

func newMqlOktaPolicy(runtime *plugin.Runtime, entry *sdk.PolicyWrapper) (interface{}, error) {
	conditions, err := convert.JsonToDict(entry.Conditions)
	if err != nil {
		return nil, err
	}

	system := false
	if entry.System != nil {
		system = *entry.System
	}

	settings, err := convert.JsonToDict(entry.Settings)
	if err != nil {
		return nil, err
	}

	return CreateResource(runtime, "okta.policy", map[string]*llx.RawData{
		"id":          llx.StringData(entry.Id),
		"name":        llx.StringData(entry.Name),
		"description": llx.StringData(entry.Description),
		"priority":    llx.IntData(entry.Priority),
		"status":      llx.StringData(entry.Status),
		"system":      llx.BoolData(system),
		"type":        llx.StringData(entry.Type),
		"conditions":  llx.DictData(conditions),
		"settings":    llx.DictData(settings),
		"created":     llx.TimeDataPtr(entry.Created),
		"lastUpdated": llx.TimeDataPtr(entry.LastUpdated),
	})
}

func (o *mqlOktaPolicy) id() (string, error) {
	return "okta.policy/" + o.Id.Data, o.Id.Error
}

func (o mqlOktaPolicy) rules() ([]interface{}, error) {
	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()

	ctx := context.Background()
	if o.Id.Error != nil {
		return nil, o.Id.Error
	}

	if o.Type.Data == ACCESS_POLICY {
		return getAccessPolicyRules(ctx, o.MqlRuntime, o.Id.Data, conn.OrganizationID(), conn.Token())
	}

	rules, resp, err := client.Policy.ListPolicyRules(ctx, o.Id.Data)
	if err != nil {
		return nil, err
	}

	if len(rules) == 0 {
		return nil, nil
	}

	list := []interface{}{}
	appendEntry := func(datalist []*okta.PolicyRule) error {
		for i := range datalist {
			r, err := newMqlOktaPolicyRule(o.MqlRuntime, datalist[i])
			if err != nil {
				return err
			}
			list = append(list, r)
		}
		return nil
	}

	err = appendEntry(rules)
	if err != nil {
		return nil, err
	}

	for resp.HasNextPage() {
		var rules []*okta.PolicyRule
		resp, err = resp.Next(ctx, &rules)
		if err != nil {
			return nil, err
		}
		err = appendEntry(rules)
		if err != nil {
			return nil, err
		}
	}
	return list, nil
}

func getAccessPolicyRules(ctx context.Context, runtime *plugin.Runtime, policyId, host, token string) ([]interface{}, error) {
	rules, err := fetchAccessPolicyRules(ctx, policyId, host, token)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	for _, entry := range rules {
		actions, err := convert.JsonToDict(entry.Actions)
		if err != nil {
			return nil, err
		}

		conditions, err := convert.JsonToDict(entry.Conditions)
		if err != nil {
			return nil, err
		}

		system := false
		if entry.System != nil {
			system = *entry.System
		}

		mqlRule, err := CreateResource(runtime, "okta.policyRule", map[string]*llx.RawData{
			"id":          llx.StringData(entry.Id),
			"name":        llx.StringData(entry.Name),
			"priority":    llx.IntData(entry.Priority),
			"status":      llx.StringData(entry.Status),
			"system":      llx.BoolData(system),
			"type":        llx.StringData(entry.Type),
			"actions":     llx.DictData(actions),
			"conditions":  llx.DictData(conditions),
			"created":     llx.TimeDataPtr(entry.Created),
			"lastUpdated": llx.TimeDataPtr(entry.LastUpdated),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlRule)
	}
	return res, nil
}

func newMqlOktaPolicyRule(runtime *plugin.Runtime, entry *okta.PolicyRule) (interface{}, error) {
	actions, err := convert.JsonToDict(entry.Actions)
	if err != nil {
		return nil, err
	}

	conditions, err := convert.JsonToDict(entry.Conditions)
	if err != nil {
		return nil, err
	}

	system := false
	if entry.System != nil {
		system = *entry.System
	}

	return CreateResource(runtime, "okta.policyRule", map[string]*llx.RawData{
		"id":          llx.StringData(entry.Id),
		"name":        llx.StringData(entry.Name),
		"priority":    llx.IntData(entry.Priority),
		"status":      llx.StringData(entry.Status),
		"system":      llx.BoolData(system),
		"type":        llx.StringData(entry.Type),
		"actions":     llx.DictData(actions),
		"conditions":  llx.DictData(conditions),
		"created":     llx.TimeDataPtr(entry.Created),
		"lastUpdated": llx.TimeDataPtr(entry.LastUpdated),
	})
}

func (o *mqlOktaPolicyRule) id() (string, error) {
	return "okta.policyRule/" + o.Id.Data, o.Id.Error
}

// see https://github.com/okta/okta-sdk-golang/issues/286 for context. okta's sdk doesnt letch you fetch
// type-specific rules which differ between the different policies. as such, we fetch those manually until the sdk allows us to
func fetchAccessPolicyRules(ctx context.Context, policyid, host, token string) ([]okta.AccessPolicyRule, error) {
	urlPath := fmt.Sprintf("https://%s/api/v1/policies/%s/rules?limit=50", host, policyid)
	client := http.Client{}
	req, err := http.NewRequest("GET", urlPath, nil)
	if err != nil {
		return []okta.AccessPolicyRule{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("SSWS %s", token))
	resp, err := client.Do(req)
	if err != nil {
		return []okta.AccessPolicyRule{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return []okta.AccessPolicyRule{}, errors.New("failed to fetch access policy rules from " + urlPath + ": " + resp.Status)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return []okta.AccessPolicyRule{}, err
	}
	result := []okta.AccessPolicyRule{}
	err = json.Unmarshal(raw, &result)
	return result, err
}
