// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package sdk

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/okta/okta-sdk-golang/v2/okta/query"
)

// Policy wrapper over okta.Policy until okta sdk is complete
type PolicyWrapper struct {
	// reuse existing struct from okta sdk
	okta.Policy

	Settings *PolicySettings `json:"settings,omitempty"`
}

// MarshalJSON has to handle the custom embedded struct that is based on okta.Policy
// ideas from https://jhall.io/posts/go-json-tricks-embedded-marshaler/
func (a *PolicyWrapper) MarshalJSON() ([]byte, error) {
	policyJSON, err := a.Policy.MarshalJSON()
	if err != nil {
		return nil, err
	}

	var settingsJSON []byte
	if a.Settings != nil {
		result := *a.Settings
		settingsJSON, err = json.Marshal(&result)
		if err != nil {
			return nil, err
		}

		separator := ","
		if string(policyJSON) == "{}" {
			separator = ""
		}
		settingsJSON = []byte(fmt.Sprintf("%s\"settings\":%s}", separator, settingsJSON))
	}

	var _json string
	if len(settingsJSON) > 0 {
		_json = fmt.Sprintf("%s%s", policyJSON[:len(policyJSON)-1], settingsJSON)
	} else {
		_json = string(policyJSON)
	}
	return []byte(_json), nil
}

func (a *PolicyWrapper) UnmarshalJSON(data []byte) error {
	type tmp PolicyWrapper
	result := &struct {
		*tmp
	}{
		tmp: (*tmp)(a),
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}

	// Since the original okta policy does not support settings, we need to handle that ourselves
	settings := struct {
		Settings PolicySettings `json:"settings,omitempty"`
	}{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return err
	}
	a.Settings = &settings.Settings
	return nil
}

type PolicySettings struct {
	Authenticators []*PolicyAuthenticator                 `json:"authenticators,omitempty"`
	Delegation     *okta.PasswordPolicyDelegationSettings `json:"delegation,omitempty"`
	Factors        *PolicyFactorsSettings                 `json:"factors,omitempty"`
	Password       *okta.PasswordPolicyPasswordSettings   `json:"password,omitempty"`
	Recovery       *okta.PasswordPolicyRecoverySettings   `json:"recovery,omitempty"`
	Type           string                                 `json:"type,omitempty"`
}

type PolicyFactorsSettings struct {
	Duo          *PolicyFactor `json:"duo,omitempty"`
	FidoU2f      *PolicyFactor `json:"fido_u2f,omitempty"`
	FidoWebauthn *PolicyFactor `json:"fido_webauthn,omitempty"`
	Hotp         *PolicyFactor `json:"hotp,omitempty"`
	GoogleOtp    *PolicyFactor `json:"google_otp,omitempty"`
	OktaCall     *PolicyFactor `json:"okta_call,omitempty"`
	OktaOtp      *PolicyFactor `json:"okta_otp,omitempty"`
	OktaPassword *PolicyFactor `json:"okta_password,omitempty"`
	OktaPush     *PolicyFactor `json:"okta_push,omitempty"`
	OktaQuestion *PolicyFactor `json:"okta_question,omitempty"`
	OktaSms      *PolicyFactor `json:"okta_sms,omitempty"`
	OktaEmail    *PolicyFactor `json:"okta_email,omitempty"`
	RsaToken     *PolicyFactor `json:"rsa_token,omitempty"`
	SymantecVip  *PolicyFactor `json:"symantec_vip,omitempty"`
	YubikeyToken *PolicyFactor `json:"yubikey_token,omitempty"`
}

type PolicyFactor struct {
	Consent *Consent `json:"consent,omitempty"`
	Enroll  *Enroll  `json:"enroll,omitempty"`
}

type PolicyAuthenticator struct {
	Key    string  `json:"key,omitempty"`
	Enroll *Enroll `json:"enroll,omitempty"`
}

type Consent struct {
	Terms *Terms `json:"terms,omitempty"`
	Type  string `json:"type,omitempty"`
}

type Terms struct {
	Format string `json:"format,omitempty"`
	Value  string `json:"value,omitempty"`
}

type Enroll struct {
	Self string `json:"self,omitempty"`
}

// Retrieve all policies with the specified type
func (m *ApiExtension) ListPolicies(ctx context.Context, qp *query.Params) ([]*PolicyWrapper, *okta.Response, error) {
	url := fmt.Sprintf("/api/v1/policies")
	if qp != nil {
		url = url + qp.String()
	}

	rq := m.RequestExecutor
	req, err := rq.WithAccept("application/json").WithContentType("application/json").NewRequest("GET", url, nil)
	if err != nil {
		return nil, nil, err
	}

	var policies []PolicyWrapper
	resp, err := rq.Do(ctx, req, &policies)
	if err != nil {
		return nil, resp, err
	}

	policiesPtr := make([]*PolicyWrapper, len(policies))
	for i := range policies {
		policiesPtr[i] = &policies[i]
	}
	return policiesPtr, resp, nil
}
