package okta

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
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	"go.mondoo.com/cnquery/resources/packs/okta/sdk"
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

func listPolicies(runtime *resources.Runtime, policyType PolicyType) ([]interface{}, error) {
	op, err := oktaProvider(runtime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	client := op.Client()

	apiSupplement := &sdk.SkdExtension{
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

func (o *mqlOktaPolicies) GetPassword() (interface{}, error) {
	return listPolicies(o.MotorRuntime, PASSWORD)
}

func (o *mqlOktaPolicies) GetMfaEnroll() (interface{}, error) {
	return listPolicies(o.MotorRuntime, MFA_ENROLL)
}

func (o *mqlOktaPolicies) GetSignOn() (interface{}, error) {
	return listPolicies(o.MotorRuntime, OKTA_SIGN_ON)
}

func (o *mqlOktaPolicies) GetOauthAuthorizationPolicy() (interface{}, error) {
	return listPolicies(o.MotorRuntime, OAUTH_AUTHORIZATION_POLICY)
}

func (o *mqlOktaPolicies) GetIdpDiscovery() (interface{}, error) {
	return listPolicies(o.MotorRuntime, IDP_DISCOVERY)
}

func (o *mqlOktaPolicies) GetAccessPolicy() ([]interface{}, error) {
	return listPolicies(o.MotorRuntime, ACCESS_POLICY)
}

func (o *mqlOktaPolicies) GetProfileEnrollment() ([]interface{}, error) {
	return listPolicies(o.MotorRuntime, PROFILE_ENROLLMENT)
}

func newMqlOktaPolicy(runtime *resources.Runtime, entry *sdk.PolicyWrapper) (interface{}, error) {
	conditions, err := core.JsonToDict(entry.Conditions)
	if err != nil {
		return nil, err
	}

	system := false
	if entry.System != nil {
		system = *entry.System
	}

	settings, err := core.JsonToDict(entry.Settings)
	if err != nil {
		return nil, err
	}

	return runtime.CreateResource("okta.policy",
		"id", entry.Id,
		"name", entry.Name,
		"description", entry.Description,
		"priority", entry.Priority,
		"status", entry.Status,
		"system", system,
		"type", entry.Type,
		"conditions", conditions,
		"settings", settings,
		"created", entry.Created,
		"lastUpdated", entry.LastUpdated,
	)
}

func (o *mqlOktaPolicy) id() (string, error) {
	id, err := o.Id()
	if err != nil {
		return "", err
	}
	return "okta.policy/" + id, nil
}

func (o mqlOktaPolicy) GetRules() ([]interface{}, error) {
	op, err := oktaProvider(o.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	client := op.Client()

	policyId, err := o.Id()
	if err != nil {
		return nil, err
	}

	policyType, err := o.Type()
	if err != nil {
		return nil, err
	}

	if policyType == ACCESS_POLICY {
		return getAccessPolicyRules(ctx, o.MotorRuntime, policyId, op.OrganizationID(), op.Token())
	}

	rules, resp, err := client.Policy.ListPolicyRules(ctx, policyId)
	if err != nil {
		return nil, err
	}

	if len(rules) == 0 {
		return nil, nil
	}

	list := []interface{}{}
	appendEntry := func(datalist []*okta.PolicyRule) error {
		for i := range datalist {
			r, err := newMqlOktaPolicyRule(o.MotorRuntime, datalist[i])
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

func getAccessPolicyRules(ctx context.Context, runtime *resources.Runtime, policyId, host, token string) ([]interface{}, error) {
	rules, err := fetchAccessPolicyRules(ctx, policyId, host, token)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	for _, entry := range rules {
		actions, err := core.JsonToDict(entry.Actions)
		if err != nil {
			return nil, err
		}

		conditions, err := core.JsonToDict(entry.Conditions)
		if err != nil {
			return nil, err
		}

		system := false
		if entry.System != nil {
			system = *entry.System
		}

		mqlRule, err := runtime.CreateResource("okta.policyRule",
			"id", entry.Id,
			"name", entry.Name,
			"priority", entry.Priority,
			"status", entry.Status,
			"system", system,
			"type", entry.Type,
			"actions", actions,
			"conditions", conditions,
			"created", entry.Created,
			"lastUpdated", entry.LastUpdated,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlRule)
	}
	return res, nil
}

func newMqlOktaPolicyRule(runtime *resources.Runtime, entry *okta.PolicyRule) (interface{}, error) {
	actions, err := core.JsonToDict(entry.Actions)
	if err != nil {
		return nil, err
	}

	conditions, err := core.JsonToDict(entry.Conditions)
	if err != nil {
		return nil, err
	}

	system := false
	if entry.System != nil {
		system = *entry.System
	}

	return runtime.CreateResource("okta.policyRule",
		"id", entry.Id,
		"name", entry.Name,
		"priority", entry.Priority,
		"status", entry.Status,
		"system", system,
		"type", entry.Type,
		"actions", actions,
		"conditions", conditions,
		"created", entry.Created,
		"lastUpdated", entry.LastUpdated,
	)
}

func (o *mqlOktaPolicyRule) id() (string, error) {
	id, err := o.Id()
	if err != nil {
		return "", err
	}
	return "okta.policyRule/" + id, nil
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
