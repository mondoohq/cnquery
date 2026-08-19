// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustParse(t *testing.T, layout, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(layout, value)
	require.NoError(t, err)
	return parsed
}

// -----------------------------------------------------------------------------
// timestamps
// -----------------------------------------------------------------------------

func TestNrTimeDecoding(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    *time.Time
		wantErr bool
	}{
		// An absent timestamp has to stay null. Decoding it to the zero time
		// would report 1 January year 1 as a real date, and to a zero epoch
		// would report 1 January 1970, either of which an age check reads as a
		// very old record.
		{name: "null", json: `null`},
		{name: "empty string", json: `""`},
		{name: "zero epoch", json: `0`},

		{
			name: "rfc3339",
			json: `"2024-03-01T10:20:30Z"`,
			want: ptrTime(mustParse(t, time.RFC3339, "2024-03-01T10:20:30Z")),
		},
		{
			name: "rfc3339 with offset",
			json: `"2024-03-01T10:20:30+02:00"`,
			want: ptrTime(mustParse(t, time.RFC3339, "2024-03-01T10:20:30+02:00")),
		},
		{
			name: "rfc3339 with milliseconds",
			json: `"2024-03-01T10:20:30.123Z"`,
			want: ptrTime(mustParse(t, time.RFC3339Nano, "2024-03-01T10:20:30.123Z")),
		},
		{
			name: "naive datetime",
			json: `"2024-03-01T10:20:30"`,
			want: ptrTime(mustParse(t, "2006-01-02T15:04:05", "2024-03-01T10:20:30")),
		},
		{
			name: "date only",
			json: `"2024-03-01"`,
			want: ptrTime(mustParse(t, "2006-01-02", "2024-03-01")),
		},
		{
			// API keys report their creation time as epoch seconds.
			name: "epoch seconds",
			json: `1709288430`,
			want: ptrTime(time.Unix(1709288430, 0).UTC()),
		},
		{
			// Anything at or beyond the seconds cutoff is milliseconds.
			name: "epoch milliseconds",
			json: `1709288430000`,
			want: ptrTime(time.UnixMilli(1709288430000).UTC()),
		},
		{
			name: "epoch seconds inside a string",
			json: `"1709288430"`,
			want: ptrTime(time.Unix(1709288430, 0).UTC()),
		},

		// A shape nobody recognizes is a failure, not a null. Reporting "no
		// timestamp" for every record would make every key look newly created
		// and every rotation check pass.
		{name: "garbage string", json: `"march the first"`, wantErr: true},
		{name: "object", json: `{"seconds":1}`, wantErr: true},
		{name: "boolean", json: `true`, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var wrapper struct {
				At nrTime `json:"at"`
			}
			err := json.Unmarshal([]byte(`{"at":`+tc.json+`}`), &wrapper)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			got := wrapper.At.Time()
			if tc.want == nil {
				assert.Nil(t, got, "an absent timestamp must stay null")
				assert.True(t, wrapper.At.IsZero())
				return
			}
			require.NotNil(t, got)
			assert.True(t, tc.want.Equal(*got), "want %s, got %s", tc.want, got)
		})
	}
}

func TestNrTimeAbsentFieldStaysNull(t *testing.T) {
	var wrapper struct {
		At nrTime `json:"at"`
	}
	require.NoError(t, json.Unmarshal([]byte(`{}`), &wrapper))
	assert.Nil(t, wrapper.At.Time())
	assert.True(t, wrapper.At.IsZero())
}

func ptrTime(t time.Time) *time.Time { return &t }

func TestIsAllDigits(t *testing.T) {
	assert.True(t, isAllDigits("12345"))
	assert.True(t, isAllDigits("-12345"))
	assert.False(t, isAllDigits(""))
	assert.False(t, isAllDigits("-"))
	assert.False(t, isAllDigits("12a45"))
	assert.False(t, isAllDigits("2024-03-01"))
}

// -----------------------------------------------------------------------------
// record decoding
// -----------------------------------------------------------------------------

// userPayload mirrors a NerdGraph user record as the documented query returns
// it. Every security-relevant tag on the struct is pinned against it, because a
// mistyped tag decodes to a zero value rather than to an error.
const userPayload = `{
  "id": "1001",
  "name": "Ada Lovelace",
  "email": "ada@example.com",
  "timeZone": "Europe/London",
  "lastActive": "2024-03-01T10:20:30Z",
  "emailVerificationState": "Verified",
  "type": {"id": "0", "displayName": "Full platform"},
  "pendingUpgradeRequest": {
    "id": "77",
    "message": "need admin",
    "requestedUserType": {"id": "1", "displayName": "Full platform"}
  },
  "groups": {
    "nextCursor": "",
    "totalCount": 2,
    "groups": [
      {"id": "g1", "displayName": "Admins"},
      {"id": "g2", "displayName": "Engineers"}
    ]
  }
}`

func TestUserDecode(t *testing.T) {
	var user apiUser
	require.NoError(t, json.Unmarshal([]byte(userPayload), &user))

	assert.Equal(t, "1001", user.ID)
	assert.Equal(t, "Ada Lovelace", user.Name)
	assert.Equal(t, "ada@example.com", user.Email)
	assert.Equal(t, "Europe/London", user.TimeZone)
	assert.Equal(t, "Verified", user.EmailVerificationState)
	assert.Equal(t, "Full platform", user.Type.DisplayName)
	require.NotNil(t, user.LastActive.Time())
	assert.Equal(t, 2024, user.LastActive.Time().Year())
	assert.Equal(t, 2, user.Groups.TotalCount)
	require.Len(t, user.Groups.Groups, 2)
	assert.Equal(t, "g1", user.Groups.Groups[0].ID)
	assert.Equal(t, "Admins", user.Groups.Groups[0].DisplayName)

	assert.True(t, hasPendingUpgrade(user))
	assert.Equal(t, "Full platform", requestedUserType(user))
}

func TestUserWithoutOptionalValues(t *testing.T) {
	var user apiUser
	require.NoError(t, json.Unmarshal([]byte(`{
		"id": "1002",
		"email": "never@example.com",
		"lastActive": null,
		"pendingUpgradeRequest": null,
		"groups": {"groups": []}
	}`), &user))

	// A user that has never signed in must report no last-active time at all,
	// not the epoch, or a dormancy check would treat them as active in 1970.
	assert.Nil(t, user.LastActive.Time())
	assert.False(t, hasPendingUpgrade(user))
	assert.Equal(t, "", requestedUserType(user))
	assert.Empty(t, user.Groups.Groups)
}

// New Relic answers a user with no upgrade request with an empty object rather
// than null, so the presence of the field cannot stand in for a real request.
func TestUserEmptyPendingUpgradeObject(t *testing.T) {
	var user apiUser
	require.NoError(t, json.Unmarshal([]byte(`{"id":"1","pendingUpgradeRequest":{}}`), &user))
	assert.False(t, hasPendingUpgrade(user))
	assert.Equal(t, "", requestedUserType(user))
}

func TestAuthDomainDecode(t *testing.T) {
	var page apiAuthDomainsPage
	require.NoError(t, json.Unmarshal([]byte(`{
		"nextCursor": "cursor-2",
		"totalCount": 3,
		"authenticationDomains": [
			{"id": "d1", "name": "SSO domain", "provisioningType": "SCIM", "users": {"nextCursor": "", "totalCount": 1, "users": [`+userPayload+`]}},
			{"id": "d2", "name": "Manual domain", "provisioningType": "MANUAL", "users": {"users": []}}
		]
	}`), &page))

	assert.Equal(t, "cursor-2", page.NextCursor)
	require.Len(t, page.AuthenticationDomains, 2)

	assert.Equal(t, "SCIM", page.AuthenticationDomains[0].ProvisioningType)
	assert.True(t, isScimProvisioning(page.AuthenticationDomains[0].ProvisioningType))
	require.Len(t, page.AuthenticationDomains[0].Users.Users, 1)

	assert.Equal(t, "MANUAL", page.AuthenticationDomains[1].ProvisioningType)
	assert.False(t, isScimProvisioning(page.AuthenticationDomains[1].ProvisioningType))
}

func TestAdminAuthDomainDecode(t *testing.T) {
	var page apiAdminAuthDomainsPage
	require.NoError(t, json.Unmarshal([]byte(`{
		"nextCursor": "",
		"items": [
			{"id": "d1", "name": "SSO domain", "authenticationType": "SAML_SSO", "provisioningType": "SCIM", "organizationId": "org-1"},
			{"id": "d2", "name": "Password domain", "authenticationType": "PASSWORD", "provisioningType": "MANUAL", "organizationId": "org-1"}
		]
	}`), &page))

	require.Len(t, page.Items, 2)
	// A mistyped tag here makes ssoEnabled read false on a domain that enforces
	// single sign-on, which is the exact reading an audit is meant to catch.
	assert.Equal(t, "SAML_SSO", page.Items[0].AuthenticationType)
	assert.True(t, isSSOAuthentication(page.Items[0].AuthenticationType))
	assert.Equal(t, "PASSWORD", page.Items[1].AuthenticationType)
	assert.False(t, isSSOAuthentication(page.Items[1].AuthenticationType))
	assert.True(t, isPasswordAuthentication(page.Items[1].AuthenticationType))
}

func TestAPIKeyDecode(t *testing.T) {
	var page apiKeySearchPage
	require.NoError(t, json.Unmarshal([]byte(`{
		"nextCursor": "",
		"count": 2,
		"keys": [
			{"id": "k1", "name": "prod license", "notes": "infra", "type": "INGEST", "ingestType": "LICENSE", "createdAt": 1709288430, "accountId": 1234567},
			{"id": "k2", "name": "ada's key", "type": "USER", "createdAt": 0, "accountId": 1234567, "userId": 1001}
		]
	}`), &page))

	require.Len(t, page.Keys, 2)

	ingest := page.Keys[0]
	assert.Equal(t, "INGEST", ingest.Type)
	assert.Equal(t, "LICENSE", ingest.IngestType)
	assert.Equal(t, 1234567, ingest.AccountID)
	assert.Equal(t, 0, ingest.UserID, "an ingest key has no owner")
	require.NotNil(t, ingest.CreatedAt.Time())
	assert.Equal(t, int64(1709288430), ingest.CreatedAt.Time().Unix())

	user := page.Keys[1]
	assert.Equal(t, "USER", user.Type)
	assert.Equal(t, "", user.IngestType, "a user key has no ingest subtype")
	assert.Equal(t, 1001, user.UserID)
	// The original account keys predate key management and carry no creation
	// time. Reporting 1970 would make them the oldest credentials in the estate
	// and drown a rotation report in false findings.
	assert.Nil(t, user.CreatedAt.Time())
}

func TestAlertConditionDecode(t *testing.T) {
	var page apiAlertConditionsPage
	require.NoError(t, json.Unmarshal([]byte(`{
		"nextCursor": "",
		"totalCount": 1,
		"nrqlConditions": [{
			"id": "c1",
			"name": "High error rate",
			"description": "watches errors",
			"enabled": false,
			"type": "STATIC",
			"policyId": "p1",
			"runbookUrl": "https://runbook.example.com",
			"violationTimeLimitSeconds": 86400,
			"nrql": {"query": "SELECT count(*) FROM TransactionError"},
			"terms": [{"operator":"ABOVE","priority":"CRITICAL","threshold":5.5,"thresholdDuration":600,"thresholdOccurrences":"ALL"}]
		}]
	}`), &page))

	require.Len(t, page.NrqlConditions, 1)
	condition := page.NrqlConditions[0]
	// A disabled condition still appears under its policy while watching
	// nothing, so this tag decides whether alerting coverage is real.
	assert.False(t, condition.Enabled)
	assert.Equal(t, "p1", condition.PolicyID)
	assert.Equal(t, "SELECT count(*) FROM TransactionError", condition.Nrql.Query)
	assert.Equal(t, 86400, condition.ViolationTimeLimitSeconds)
	require.Len(t, condition.Terms, 1)
	assert.Equal(t, 5.5, condition.Terms[0].Threshold)
	assert.Equal(t, "CRITICAL", condition.Terms[0].Priority)
}

func TestAlertTermsDict(t *testing.T) {
	terms := alertTermsDict([]apiAlertTerm{{
		Operator:             "BELOW",
		Priority:             "WARNING",
		Threshold:            1.25,
		ThresholdDuration:    300,
		ThresholdOccurrences: "AT_LEAST_ONCE",
	}})

	require.Len(t, terms, 1)
	assert.Equal(t, map[string]any{
		"operator":             "BELOW",
		"priority":             "WARNING",
		"threshold":            1.25,
		"thresholdDuration":    int64(300),
		"thresholdOccurrences": "AT_LEAST_ONCE",
	}, terms[0])

	assert.Empty(t, alertTermsDict(nil))
}

func TestNotificationDestinationDecode(t *testing.T) {
	var page apiNotificationDestinationsPage
	require.NoError(t, json.Unmarshal([]byte(`{
		"nextCursor": "",
		"totalCount": 2,
		"entities": [
			{"id":"d1","name":"ops webhook","type":"WEBHOOK","active":true,"status":"DEFAULT","isUserAuthenticated":true,"createdAt":"2024-01-02T03:04:05Z","updatedAt":"2024-02-02T03:04:05Z","lastSent":"2024-03-02T03:04:05Z","accountId":1234567},
			{"id":"d2","name":"never used","type":"EMAIL","active":false,"status":"DRAFT","isUserAuthenticated":false,"createdAt":"2020-01-02T03:04:05Z","updatedAt":"2020-01-02T03:04:05Z","lastSent":null,"accountId":1234567}
		]
	}`), &page))

	require.Len(t, page.Entities, 2)
	assert.True(t, page.Entities[0].Active)
	assert.True(t, page.Entities[0].IsUserAuthenticated)
	require.NotNil(t, page.Entities[0].LastSent.Time())

	assert.False(t, page.Entities[1].Active)
	// A destination that has never delivered anything must report no last-sent
	// time, not the epoch, or "configured but never proven" is unfindable.
	assert.Nil(t, page.Entities[1].LastSent.Time())
	assert.Equal(t, "DRAFT", page.Entities[1].Status)
}

func TestNotificationChannelDecode(t *testing.T) {
	var page apiNotificationChannelsPage
	require.NoError(t, json.Unmarshal([]byte(`{
		"entities": [{"id":"c1","name":"pager","type":"PAGERDUTY_ACCOUNT_INTEGRATION","product":"IINT","active":true,"status":"DEFAULT","createdAt":"2024-01-02T03:04:05Z","updatedAt":"2024-01-03T03:04:05Z","destinationId":"d1","accountId":1234567}]
	}`), &page))

	require.Len(t, page.Entities, 1)
	assert.Equal(t, "d1", page.Entities[0].DestinationID)
	assert.Equal(t, "IINT", page.Entities[0].Product)
	assert.True(t, page.Entities[0].Active)
}

func TestDropRuleDecode(t *testing.T) {
	var list apiDropRulesList
	require.NoError(t, json.Unmarshal([]byte(`{
		"error": null,
		"rules": [{
			"id": "r1",
			"action": "DROP_DATA",
			"nrql": "SELECT * FROM Log WHERE service = 'auth'",
			"description": "noise",
			"source": "NerdGraph",
			"createdAt": "2024-01-02T03:04:05Z",
			"accountId": 1234567,
			"createdBy": 1001,
			"creator": {"id": 1001, "name": "Ada Lovelace", "email": "ada@example.com"}
		}]
	}`), &list))

	require.Len(t, list.Rules, 1)
	rule := list.Rules[0]
	assert.Equal(t, "DROP_DATA", rule.Action)
	assert.Equal(t, "SELECT * FROM Log WHERE service = 'auth'", rule.Nrql)
	assert.Equal(t, "1001", dropRuleCreatorID(rule))
	assert.False(t, list.Error.isSet())
}

func TestDropRuleCreatorID(t *testing.T) {
	assert.Equal(t, "1001", dropRuleCreatorID(apiDropRule{Creator: &apiUserRef{ID: 1001}, CreatedBy: 1001}))
	// A rule registered by a system carries no creator object but still reports
	// the numeric id, so the fallback keeps the attribution.
	assert.Equal(t, "1001", dropRuleCreatorID(apiDropRule{CreatedBy: 1001}))
	assert.Equal(t, "1001", dropRuleCreatorID(apiDropRule{Creator: &apiUserRef{ID: 0}, CreatedBy: 1001}))
	assert.Equal(t, "", dropRuleCreatorID(apiDropRule{}))
}

func TestRetentionRuleDecode(t *testing.T) {
	var resp struct {
		Rules []apiRetentionRule `json:"eventRetentionRules"`
	}
	require.NoError(t, json.Unmarshal([]byte(`{
		"eventRetentionRules": [
			{"id":"1","namespace":"Log","retentionInDays":30,"createdAt":"2024-01-02T03:04:05Z","createdById":"1001","deletedAt":null,"deletedById":null},
			{"id":"2","namespace":"Log","retentionInDays":8,"createdAt":"2023-01-02T03:04:05Z","createdById":"1001","deletedAt":"2024-01-01T00:00:00Z","deletedById":"1002"}
		]
	}`), &resp))

	require.Len(t, resp.Rules, 2)

	// New Relic returns deleted rules alongside live ones. A deleted rule's
	// window is no longer applied, so counting it as in force would report an
	// account as retaining data it has already aged out.
	assert.True(t, isRetentionRuleActive(resp.Rules[0]))
	assert.Nil(t, resp.Rules[0].DeletedAt.Time())
	assert.Equal(t, 30, resp.Rules[0].RetentionInDays)

	assert.False(t, isRetentionRuleActive(resp.Rules[1]))
	require.NotNil(t, resp.Rules[1].DeletedAt.Time())
	assert.Equal(t, "1002", resp.Rules[1].DeletedByID)
}

func TestGrantedRoleDecode(t *testing.T) {
	var page apiGrantedRolesPage
	require.NoError(t, json.Unmarshal([]byte(`{
		"nextCursor": "",
		"totalCount": 2,
		"roles": [
			{"id":"a1","name":"admin","displayName":"Administrator","type":"STANDARD","roleId":99999,"accountId":1234567,"organizationId":null,"groupId":"g1"},
			{"id":"a2","name":"org_admin","displayName":"Organization admin","type":"CUSTOM","roleId":88888,"accountId":0,"organizationId":"org-1","groupId":null}
		]
	}`), &page))

	require.Len(t, page.Roles, 2)
	assert.False(t, isOrganizationWideGrant(page.Roles[0]))
	assert.True(t, isOrganizationWideGrant(page.Roles[1]))
	assert.Equal(t, 99999, page.Roles[0].RoleID)
}

// -----------------------------------------------------------------------------
// derived predicates
// -----------------------------------------------------------------------------

func TestIsSSOAuthentication(t *testing.T) {
	tests := map[string]bool{
		"SAML_SSO":   true,
		"OIDC_SSO":   true,
		"HEROKU_SSO": true,
		"PASSWORD":   false,
		"DISABLED":   false,
		"":           false,
		// A login method this provider has never seen must not be reported as
		// single sign-on. Failing the assertion prompts a look; passing it
		// would silently bless an unknown method.
		"SOMETHING_NEW": false,
		"saml_sso":      false,
	}
	for in, want := range tests {
		t.Run(in, func(t *testing.T) {
			assert.Equal(t, want, isSSOAuthentication(in))
		})
	}
}

func TestIsPasswordAuthentication(t *testing.T) {
	assert.True(t, isPasswordAuthentication("PASSWORD"))
	assert.False(t, isPasswordAuthentication("SAML_SSO"))
	assert.False(t, isPasswordAuthentication("DISABLED"))
	assert.False(t, isPasswordAuthentication(""))
	assert.False(t, isPasswordAuthentication("password"))
}

func TestIsScimProvisioning(t *testing.T) {
	assert.True(t, isScimProvisioning("SCIM"))
	assert.False(t, isScimProvisioning("MANUAL"))
	assert.False(t, isScimProvisioning("DISABLED"))
	assert.False(t, isScimProvisioning(""))
	assert.False(t, isScimProvisioning("scim"))
	assert.False(t, isScimProvisioning("SCIM_V2"), "an unrecognized method is not claimed to be automated")
}

func TestIsOrganizationWideGrant(t *testing.T) {
	assert.True(t, isOrganizationWideGrant(apiGrantedRole{OrganizationID: "org-1"}))
	assert.False(t, isOrganizationWideGrant(apiGrantedRole{AccountID: 1234567, OrganizationID: "org-1"}))
	assert.False(t, isOrganizationWideGrant(apiGrantedRole{AccountID: 1234567}))
	// Neither piece of evidence is present, so the broadest possible reading is
	// not asserted. Claiming it would raise a finding on every such grant.
	assert.False(t, isOrganizationWideGrant(apiGrantedRole{}))
}

func TestIsRetentionRuleActive(t *testing.T) {
	var deleted nrTime
	require.NoError(t, deleted.UnmarshalJSON([]byte(`"2024-01-01T00:00:00Z"`)))

	assert.True(t, isRetentionRuleActive(apiRetentionRule{}))
	assert.False(t, isRetentionRuleActive(apiRetentionRule{DeletedAt: deleted}))
}

// -----------------------------------------------------------------------------
// identity keys
// -----------------------------------------------------------------------------

// A cache key that misses an identity dimension makes the second record return
// the first one's data, which reports fewer records than exist and attributes
// the survivor's values to all of them.
func TestAccessGrantIDCarriesEveryDimension(t *testing.T) {
	admin := apiGrantedRole{ID: "1", Name: "admin", RoleID: 99999, AccountID: 1234567}
	groupA := apiGroup{ID: "gA"}
	groupB := apiGroup{ID: "gB"}

	assert.NotEqual(t, accessGrantID(groupA, admin), accessGrantID(groupB, admin),
		"the same role granted to two groups is two grants")

	otherAccount := admin
	otherAccount.AccountID = 7654321
	assert.NotEqual(t, accessGrantID(groupA, admin), accessGrantID(groupA, otherAccount),
		"the same role granted over two accounts is two grants")

	otherRole := admin
	otherRole.RoleID = 88888
	assert.NotEqual(t, accessGrantID(groupA, admin), accessGrantID(groupA, otherRole),
		"two roles granted to one group are two grants")

	orgWide := apiGrantedRole{ID: "1", Name: "admin", RoleID: 99999, OrganizationID: "org-1"}
	assert.NotEqual(t, accessGrantID(groupA, admin), accessGrantID(groupA, orgWide),
		"an account grant and an organization grant are two grants")
	assert.Contains(t, accessGrantID(groupA, orgWide), "organization/org-1")

	assert.Equal(t, accessGrantID(groupA, admin), accessGrantID(groupA, admin), "the key is stable")
}

// Ingest keys and user keys are numbered in separate spaces, so a key ID alone
// can collide across the two types.
func TestAPIKeyIDSeparatesKeyTypes(t *testing.T) {
	ingest := apiKey{ID: "42", Type: "INGEST", IngestType: "LICENSE"}
	user := apiKey{ID: "42", Type: "USER"}

	assert.NotEqual(t, apiKeyID(ingest), apiKeyID(user))
	assert.Equal(t, "apiKey/INGEST/42", apiKeyID(ingest))
	assert.Equal(t, "apiKey/USER/42", apiKeyID(user))
}

// -----------------------------------------------------------------------------
// secret handling
// -----------------------------------------------------------------------------

// The keystring is the one secret a New Relic key carries. The guarantee that
// it never reaches a user rests on it never being asked for, so the query text
// is asserted directly.
func TestAPIKeysQueryNeverAsksForTheKeystring(t *testing.T) {
	for _, forbidden := range []string{"\nkey\n", " key\n", "keyString", "\n        key"} {
		assert.NotContains(t, apiKeysQuery, forbidden,
			"the API key query must not select the keystring")
	}
	assert.Contains(t, apiKeysQuery, "ingestType")
	assert.Contains(t, apiKeysQuery, "createdAt")
}

// A destination's credentials live in its auth block and its properties, and a
// webhook URL routinely carries a token in its path. None of the three is
// requested.
func TestNotificationQueriesNeverAskForCredentials(t *testing.T) {
	for _, forbidden := range []string{"auth ", "auth{", "auth {", "properties", "secureUrl"} {
		assert.NotContains(t, notificationDestinationsQuery, forbidden)
		assert.NotContains(t, notificationChannelsQuery, forbidden)
	}
}

// The re-marshal sweep: feed every record type a payload that carries secret
// material in the fields New Relic would return it in, then prove none of it
// survives into the decoded struct. A struct tag added by accident later would
// fail this.
func TestNoSecretMaterialSurvivesDecoding(t *testing.T) {
	const secret = "NRAK-SUPER-SECRET-KEYSTRING"

	cases := []struct {
		name    string
		payload string
		target  func() any
	}{
		{
			name: "api key",
			payload: `{"keys":[{
				"id":"k1","name":"prod","type":"INGEST","ingestType":"LICENSE","createdAt":1709288430,
				"key":"` + secret + `","keyString":"` + secret + `"
			}]}`,
			target: func() any { return &apiKeySearchPage{} },
		},
		{
			name: "notification destination",
			payload: `{"entities":[{
				"id":"d1","name":"webhook","type":"WEBHOOK","active":true,"status":"DEFAULT",
				"auth":{"authType":"TOKEN","token":"` + secret + `"},
				"secureUrl":{"prefix":"https://hooks.example.com/` + secret + `"},
				"properties":[{"key":"url","value":"https://hooks.example.com/` + secret + `"}]
			}]}`,
			target: func() any { return &apiNotificationDestinationsPage{} },
		},
		{
			name: "notification channel",
			payload: `{"entities":[{
				"id":"c1","name":"pager","type":"WEBHOOK","product":"IINT","active":true,
				"properties":[{"key":"payload","value":"` + secret + `"}]
			}]}`,
			target: func() any { return &apiNotificationChannelsPage{} },
		},
		{
			name: "user",
			payload: `{"id":"1","email":"ada@example.com","password":"` + secret + `",
				"apiKey":"` + secret + `"}`,
			target: func() any { return &apiUser{} },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			target := tc.target()
			require.NoError(t, json.Unmarshal([]byte(tc.payload), target))

			roundTripped, err := json.Marshal(target)
			require.NoError(t, err)
			assert.NotContains(t, string(roundTripped), secret,
				"no field of %s may carry key material", tc.name)
		})
	}
}
