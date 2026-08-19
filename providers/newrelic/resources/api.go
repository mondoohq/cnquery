// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/newrelic/newrelic-client-go/v2/pkg/accounts"
	"github.com/newrelic/newrelic-client-go/v2/pkg/alerts"
	"github.com/newrelic/newrelic-client-go/v2/pkg/authorizationmanagement"
	"github.com/newrelic/newrelic-client-go/v2/pkg/customeradministration"
	"github.com/newrelic/newrelic-client-go/v2/pkg/nrqldroprules"
	"github.com/newrelic/newrelic-client-go/v2/pkg/usermanagement"
	"go.mondoo.com/mql/v13/providers/newrelic/connection"
)

// maxPages bounds every cursor walk. New Relic returns at most 500 records per
// page, so the cap is well beyond any real organization and exists purely so a
// server that keeps handing out cursors cannot spin forever.
const maxPages = 1000

// walkPages drives a cursor-paginated NerdGraph collection. fetch is called
// once per page with the cursor to request and returns the cursor for the next
// page, or the empty string when the collection is exhausted.
//
// A cursor that has already been requested ends the walk with an error rather
// than a shorter list. A server that ignores the cursor argument answers with
// the same page and the same nextCursor forever, and the two failure modes that
// follow are both silent: stopping early under-reports the collection, and
// continuing multiplies every record until the page cap. Neither can be told
// from a correct answer by looking at the result, so the walk refuses to
// produce one.
func walkPages(what string, fetch func(cursor string) (string, error)) error {
	seen := map[string]struct{}{}
	cursor := ""

	for page := 0; page < maxPages; page++ {
		next, err := fetch(cursor)
		if err != nil {
			return err
		}
		if next == "" {
			return nil
		}
		if _, repeated := seen[next]; repeated {
			return fmt.Errorf("the New Relic API kept returning the same page cursor while listing %s, so the list cannot be completed", what)
		}
		seen[next] = struct{}{}
		cursor = next
	}

	return fmt.Errorf("listing %s did not finish within %d pages", what, maxPages)
}

// cursorVars builds the variable map for a paginated query. An empty cursor is
// left out entirely so the first page asks for the collection's own start
// rather than for the page after an empty cursor.
func cursorVars(cursor string, extra map[string]any) map[string]any {
	vars := make(map[string]any, len(extra)+1)
	for k, v := range extra {
		vars[k] = v
	}
	if cursor != "" {
		vars["cursor"] = cursor
	}
	return vars
}

// -----------------------------------------------------------------------------
// wire types
// -----------------------------------------------------------------------------
//
// Most records below decode straight into the types the official
// `newrelic/newrelic-client-go` client generates from the NerdGraph schema, so
// the struct tags are vendor-maintained and continuously exercised by
// terraform-provider-newrelic rather than hand-written here.
//
// Three groups deliberately keep local types. The reasons are recorded in
// TESTING-TODO.md under "Why three collections keep local types", and each is
// pinned by a test, because every one of them is the kind of thing a later
// reader would "fix" back:
//
//   - API keys. The SDK's key structs carry a `Key` field and its only read
//     query selects the keystring four times (pkg/apiaccess/keys.go:220). This
//     provider's schema states that the keystring is never requested, and that
//     guarantee is only worth anything while it is true of the query.
//   - Notification destinations and channels. The SDK's destination struct
//     carries `Auth`, `Properties` and `SecureURL`, and its query selects all
//     three. `AiNotificationsProperty.Value` is not masked and routinely holds a
//     webhook URL path.
//   - Event retention rules. The SDK does not model them at all: neither
//     `RetentionInDays` nor `eventRetentionRule` appears anywhere in it, and
//     pkg/datamanagement covers account limits only. There is nothing to adopt.
//
// Where a record needs a timestamp, the SDK type is embedded and the timestamp
// field is shadowed by nrTime. The SDK's `nrtime.DateTime` is a bare unparsed
// string and its epoch fields are plain int64s that cannot express "absent", so
// adopting them would report a key with no creation time as 1970-01-01. Go's
// encoding/json gives the shallower field precedence, so the outer nrTime wins
// for both directions and null stays null.

type apiAccount = accounts.AccountOutline

// apiOrganization stays local: the SDK's organization.Organization is a
// stitched-fields root carrying every namespace hung off an organization, and
// this provider reads two scalars from it.
type apiOrganization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// apiUser is the SDK's user record with the last-active timestamp decoded and
// the enclosing domain carried down. New Relic returns users nested under their
// authentication domain and never repeats the domain on the user, but the user
// resource has to carry it to resolve the reference back.
type apiUser struct {
	usermanagement.UserManagementUser

	// LastActive shadows the embedded nrtime.DateTime, which is an unparsed
	// string. A user who has never signed in must report no time at all rather
	// than a zero date a dormancy check would read as real.
	LastActive nrTime `json:"lastActive"`

	domainID string `json:"-"`
}

type apiUsersPage struct {
	NextCursor string    `json:"nextCursor"`
	TotalCount int       `json:"totalCount"`
	Users      []apiUser `json:"users"`
}

type apiAuthDomain struct {
	ID               string       `json:"id"`
	Name             string       `json:"name"`
	ProvisioningType string       `json:"provisioningType"`
	Users            apiUsersPage `json:"users"`
}

type apiAuthDomainsPage struct {
	NextCursor            string          `json:"nextCursor"`
	TotalCount            int             `json:"totalCount"`
	AuthenticationDomains []apiAuthDomain `json:"authenticationDomains"`
}

// apiAdminAuthDomain is the authentication domain as customerAdministration
// reports it. It is the only place the login method is exposed, which is why it
// is read separately from the user-management view.
type apiAdminAuthDomain = customeradministration.OrganizationAuthenticationDomain

type apiAdminAuthDomainsPage = customeradministration.OrganizationAuthenticationDomainCollection

// apiGrantedRole is one access grant: a role bound to a group over an account
// or over the organization.
type apiGrantedRole = authorizationmanagement.AuthorizationManagementGrantedRole

type apiGrantedRolesPage = authorizationmanagement.AuthorizationManagementGrantedRoleSearch

// apiGroup carries the authentication domain the group was listed under, which
// the group record itself does not repeat. The SDK models no read type for the
// domain/group/grant nesting, so the containers around the adopted grant type
// stay local.
type apiGroup struct {
	ID          string              `json:"id"`
	DisplayName string              `json:"displayName"`
	Roles       apiGrantedRolesPage `json:"roles"`

	domainID string `json:"-"`
}

type apiGroupsPage struct {
	NextCursor string     `json:"nextCursor"`
	TotalCount int        `json:"totalCount"`
	Groups     []apiGroup `json:"groups"`
}

type apiGrantAuthDomain struct {
	ID     string        `json:"id"`
	Name   string        `json:"name"`
	Groups apiGroupsPage `json:"groups"`
}

type apiGrantAuthDomainsPage struct {
	NextCursor            string               `json:"nextCursor"`
	TotalCount            int                  `json:"totalCount"`
	AuthenticationDomains []apiGrantAuthDomain `json:"authenticationDomains"`
}

type apiRole = authorizationmanagement.AuthorizationManagementRole

type apiRolesPage = authorizationmanagement.AuthorizationManagementRoleSearch

// apiKey is a user key or an ingest key.
//
// This is deliberately NOT apiaccess.APIAccessKey / APIAccessIngestKey /
// APIAccessUserKey: all three carry a `Key` field holding the keystring, and
// the SDK's only read query selects it. The keystring is the one secret a New
// Relic key carries, and not asking for it is the only guarantee that cannot be
// undone by a later change to the mapping code. The SDK's search is also
// unpaginated, so adopting it would silently truncate the list.
type apiKey struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Notes      string `json:"notes"`
	Type       string `json:"type"`
	IngestType string `json:"ingestType"`
	CreatedAt  nrTime `json:"createdAt"`
	AccountID  int    `json:"accountId"`
	UserID     int    `json:"userId"`
}

type apiKeySearchPage struct {
	NextCursor string   `json:"nextCursor"`
	Count      int      `json:"count"`
	Keys       []apiKey `json:"keys"`
}

// apiAlertPolicy is the SDK's policy record. The account ID is not part of it,
// so it is carried alongside where the mapping needs it.
type apiAlertPolicy = alerts.AlertsPolicy

type apiAlertPoliciesPage = alerts.AlertsPoliciesSearchResultSet

type apiAlertTerm = alerts.NrqlConditionTerm

type apiAlertCondition = alerts.NrqlAlertCondition

type apiAlertConditionsPage struct {
	NextCursor     string              `json:"nextCursor"`
	TotalCount     int                 `json:"totalCount"`
	NrqlConditions []apiAlertCondition `json:"nrqlConditions"`
}

// apiNotificationsError is the in-band error a notifications collection returns
// instead of a GraphQL error. A page carrying one is not an empty page.
type apiNotificationsError struct {
	Description string `json:"description"`
	Details     string `json:"details"`
	Type        string `json:"type"`
}

func (e *apiNotificationsError) isSet() bool {
	if e == nil {
		return false
	}
	return e.Description != "" || e.Details != "" || e.Type != ""
}

func (e *apiNotificationsError) asError(what string) error {
	if !e.isSet() {
		return nil
	}
	parts := make([]string, 0, 3)
	for _, s := range []string{e.Type, e.Description, e.Details} {
		if s != "" {
			parts = append(parts, s)
		}
	}
	return fmt.Errorf("the New Relic API could not list %s: %s", what, strings.Join(parts, ": "))
}

// apiNotificationDestination is deliberately NOT
// notifications.AiNotificationsDestination. That struct carries Auth,
// Properties and SecureURL, and the SDK's destinations query selects all three.
// A destination's auth block holds the token a webhook URL is signed with, and
// AiNotificationsProperty.Value is not masked, so a webhook URL carrying a
// token in its path would arrive in full. None of the three is modelled here
// and none is requested.
type apiNotificationDestination struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Type                string `json:"type"`
	Active              bool   `json:"active"`
	Status              string `json:"status"`
	IsUserAuthenticated bool   `json:"isUserAuthenticated"`
	CreatedAt           nrTime `json:"createdAt"`
	UpdatedAt           nrTime `json:"updatedAt"`
	LastSent            nrTime `json:"lastSent"`
	AccountID           int    `json:"accountId"`
}

type apiNotificationDestinationsPage struct {
	NextCursor string                       `json:"nextCursor"`
	TotalCount int                          `json:"totalCount"`
	Error      *apiNotificationsError       `json:"error"`
	Entities   []apiNotificationDestination `json:"entities"`
}

// apiNotificationChannel is deliberately NOT
// notifications.AiNotificationsChannel, which carries Properties. A channel's
// properties hold its payload template, which is where a token pasted into a
// custom payload would sit.
type apiNotificationChannel struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	Product       string `json:"product"`
	Active        bool   `json:"active"`
	Status        string `json:"status"`
	CreatedAt     nrTime `json:"createdAt"`
	UpdatedAt     nrTime `json:"updatedAt"`
	DestinationID string `json:"destinationId"`
	AccountID     int    `json:"accountId"`
}

type apiNotificationChannelsPage struct {
	NextCursor string                   `json:"nextCursor"`
	TotalCount int                      `json:"totalCount"`
	Error      *apiNotificationsError   `json:"error"`
	Entities   []apiNotificationChannel `json:"entities"`
}

type apiUserRef = nrqldroprules.UserReference

// apiDropRule is the SDK's drop rule with the creation timestamp decoded. The
// SDK's CreatedAt is an unparsed nrtime.DateTime string.
type apiDropRule struct {
	nrqldroprules.NRQLDropRulesDropRule

	CreatedAt nrTime `json:"createdAt"`
}

// apiDropRulesError is the in-band error the drop rule list returns instead of
// a GraphQL error, with the same consequence: a page carrying one reports
// nothing, and reporting nothing as "no drop rules" would pass an audit on data
// that was never read.
type apiDropRulesError = nrqldroprules.NRQLDropRulesError

// dropRulesErrorIsSet reports whether the in-band error carries anything. The
// SDK models the error as a value rather than a pointer, so its presence in the
// response cannot stand in for it being set.
func dropRulesErrorIsSet(e *apiDropRulesError) bool {
	if e == nil {
		return false
	}
	return e.Description != "" || e.Reason != ""
}

type apiDropRulesList struct {
	Error *apiDropRulesError `json:"error"`
	Rules []apiDropRule      `json:"rules"`
}

// apiRetentionRule stays local because the SDK does not model event retention
// rules at all. Neither RetentionInDays nor eventRetentionRule appears anywhere
// in newrelic-client-go, and pkg/datamanagement covers account limits only.
type apiRetentionRule struct {
	ID              string `json:"id"`
	Namespace       string `json:"namespace"`
	RetentionInDays int    `json:"retentionInDays"`
	CreatedAt       nrTime `json:"createdAt"`
	CreatedByID     string `json:"createdById"`
	DeletedAt       nrTime `json:"deletedAt"`
	DeletedByID     string `json:"deletedById"`
}

// -----------------------------------------------------------------------------
// queries
// -----------------------------------------------------------------------------

const userFields = `
  id
  name
  email
  timeZone
  lastActive
  emailVerificationState
  type { id displayName }
  pendingUpgradeRequest { id message requestedUserType { id displayName } }
  groups { nextCursor totalCount groups { id displayName } }
`

const accountsQuery = `query { actor { accounts { id name } } }`

const organizationQuery = `query { actor { organization { id name } } }`

const authDomainsWithUsersQuery = `query($cursor: String) {
  actor { organization { userManagement {
    authenticationDomains(cursor: $cursor) {
      nextCursor
      totalCount
      authenticationDomains {
        id
        name
        provisioningType
        users { nextCursor totalCount users {` + userFields + `} }
      }
    }
  } } }
}`

const domainUsersPageQuery = `query($domainIds: [ID!], $cursor: String) {
  actor { organization { userManagement {
    authenticationDomains(id: $domainIds) {
      authenticationDomains {
        id
        users(cursor: $cursor) { nextCursor totalCount users {` + userFields + `} }
      }
    }
  } } }
}`

const adminAuthDomainsQuery = `query($cursor: String) {
  customerAdministration {
    authenticationDomains(filter: {}, cursor: $cursor) {
      nextCursor
      items { id name authenticationType provisioningType organizationId }
    }
  }
}`

const grantedRoleFields = `
  id
  name
  displayName
  type
  roleId
  accountId
  organizationId
  groupId
`

const groupsWithGrantsQuery = `query($cursor: String) {
  actor { organization { authorizationManagement {
    authenticationDomains(cursor: $cursor) {
      nextCursor
      totalCount
      authenticationDomains {
        id
        name
        groups {
          nextCursor
          totalCount
          groups {
            id
            displayName
            roles { nextCursor totalCount roles {` + grantedRoleFields + `} }
          }
        }
      }
    }
  } } }
}`

const domainGroupsPageQuery = `query($domainIds: [ID!], $cursor: String) {
  actor { organization { authorizationManagement {
    authenticationDomains(id: $domainIds) {
      authenticationDomains {
        id
        groups(cursor: $cursor) {
          nextCursor
          totalCount
          groups {
            id
            displayName
            roles { nextCursor totalCount roles {` + grantedRoleFields + `} }
          }
        }
      }
    }
  } } }
}`

const rolesQuery = `query($cursor: String) {
  actor { organization { authorizationManagement {
    roles(cursor: $cursor) {
      nextCursor
      totalCount
      roles { id name displayName type scope }
    }
  } } }
}`

// apiKeysQuery never selects the `key` field. The keystring is the one piece of
// a New Relic key that is a secret, and not asking for it is the only guarantee
// that cannot be undone by a later change to the mapping code.
const apiKeysQuery = `query($accountId: Int!, $cursor: String) {
  actor { apiAccess {
    keySearch(query: { types: [INGEST, USER], scope: { accountIds: [$accountId] } }, cursor: $cursor) {
      nextCursor
      count
      keys {
        id
        name
        notes
        type
        createdAt
        ... on ApiAccessIngestKey { ingestType accountId }
        ... on ApiAccessUserKey { accountId userId }
      }
    }
  } }
}`

const alertPoliciesQuery = `query($accountId: Int!, $cursor: String) {
  actor { account(id: $accountId) { alerts {
    policiesSearch(cursor: $cursor) {
      nextCursor
      totalCount
      policies { id name incidentPreference accountId }
    }
  } } }
}`

const alertConditionsQuery = `query($accountId: Int!, $cursor: String) {
  actor { account(id: $accountId) { alerts {
    nrqlConditionsSearch(cursor: $cursor) {
      nextCursor
      totalCount
      nrqlConditions {
        id
        name
        description
        enabled
        type
        policyId
        runbookUrl
        violationTimeLimitSeconds
        nrql { query }
        terms { operator priority threshold thresholdDuration thresholdOccurrences }
      }
    }
  } } }
}`

// notificationDestinationsQuery reads no credential material. A destination's
// auth block and its properties carry the token a webhook URL is signed with,
// so neither is requested.
const notificationDestinationsQuery = `query($accountId: Int!, $cursor: String) {
  actor { account(id: $accountId) { aiNotifications {
    destinations(cursor: $cursor) {
      nextCursor
      totalCount
      error { description details type }
      entities {
        id
        name
        type
        active
        status
        isUserAuthenticated
        createdAt
        updatedAt
        lastSent
        accountId
      }
    }
  } } }
}`

const notificationChannelsQuery = `query($accountId: Int!, $cursor: String) {
  actor { account(id: $accountId) { aiNotifications {
    channels(cursor: $cursor) {
      nextCursor
      totalCount
      error { description details type }
      entities {
        id
        name
        type
        product
        active
        status
        createdAt
        updatedAt
        destinationId
        accountId
      }
    }
  } } }
}`

const dropRulesQuery = `query($accountId: Int!) {
  actor { account(id: $accountId) { nrqlDropRules { list {
    error { description reason }
    rules {
      id
      action
      nrql
      description
      source
      createdAt
      accountId
      createdBy
      creator { id name email }
    }
  } } } }
}`

const retentionRulesQuery = `query($accountId: Int!) {
  actor { account(id: $accountId) { dataManagement {
    eventRetentionRules {
      id
      namespace
      retentionInDays
      createdAt
      createdById
      deletedAt
      deletedById
    }
  } } }
}`

// -----------------------------------------------------------------------------
// fetches
// -----------------------------------------------------------------------------

func fetchAccounts(ctx context.Context, client *connection.Client) ([]apiAccount, error) {
	var resp struct {
		Actor struct {
			Accounts []apiAccount `json:"accounts"`
		} `json:"actor"`
	}
	if err := client.Query(ctx, accountsQuery, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Actor.Accounts, nil
}

func fetchOrganization(ctx context.Context, client *connection.Client) (*apiOrganization, error) {
	var resp struct {
		Actor struct {
			Organization apiOrganization `json:"organization"`
		} `json:"actor"`
	}
	if err := client.Query(ctx, organizationQuery, nil, &resp); err != nil {
		return nil, err
	}
	if resp.Actor.Organization.ID == "" {
		return nil, errors.New("the New Relic API returned no organization for the supplied key")
	}
	return &resp.Actor.Organization, nil
}

// fetchAuthDomainsWithUsers reads every authentication domain in the
// organization along with its users. Users arrive nested inside their domain,
// so one walk over the domains carries most of the user list, and only a domain
// with more users than fit on a page needs follow-up calls.
func fetchAuthDomainsWithUsers(ctx context.Context, client *connection.Client) ([]apiAuthDomain, error) {
	var domains []apiAuthDomain

	err := walkPages("authentication domains", func(cursor string) (string, error) {
		var resp struct {
			Actor struct {
				Organization struct {
					UserManagement struct {
						AuthenticationDomains apiAuthDomainsPage `json:"authenticationDomains"`
					} `json:"userManagement"`
				} `json:"organization"`
			} `json:"actor"`
		}
		if err := client.Query(ctx, authDomainsWithUsersQuery, cursorVars(cursor, nil), &resp); err != nil {
			return "", err
		}
		page := resp.Actor.Organization.UserManagement.AuthenticationDomains
		domains = append(domains, page.AuthenticationDomains...)
		return page.NextCursor, nil
	})
	if err != nil {
		return nil, err
	}

	for i := range domains {
		domain := &domains[i]
		if err := completeDomainUsers(ctx, client, domain); err != nil {
			return nil, err
		}
		for j := range domain.Users.Users {
			domain.Users.Users[j].domainID = domain.ID
			// A user's group list that does not fit on one page cannot be
			// completed through a documented follow-up query. Truncating it
			// would drop memberships and make an over-privileged account look
			// narrower than it is, so it is reported instead.
			if domain.Users.Users[j].Groups.NextCursor != "" {
				return nil, fmt.Errorf("user %q belongs to more groups than one page returns, and this provider cannot page them", domain.Users.Users[j].ID)
			}
		}
	}

	return domains, nil
}

// completeDomainUsers keeps requesting user pages for one domain until the
// domain's user list is exhausted.
func completeDomainUsers(ctx context.Context, client *connection.Client, domain *apiAuthDomain) error {
	cursor := domain.Users.NextCursor
	if cursor == "" {
		return nil
	}

	seen := map[string]struct{}{cursor: {}}
	for page := 0; page < maxPages; page++ {
		var resp struct {
			Actor struct {
				Organization struct {
					UserManagement struct {
						AuthenticationDomains struct {
							AuthenticationDomains []apiAuthDomain `json:"authenticationDomains"`
						} `json:"authenticationDomains"`
					} `json:"userManagement"`
				} `json:"organization"`
			} `json:"actor"`
		}
		vars := cursorVars(cursor, map[string]any{"domainIds": []string{domain.ID}})
		if err := client.Query(ctx, domainUsersPageQuery, vars, &resp); err != nil {
			return err
		}

		found := resp.Actor.Organization.UserManagement.AuthenticationDomains.AuthenticationDomains
		if len(found) == 0 {
			// The domain answered on the first page but not on this one. That
			// is not an empty page, it is a filter that stopped matching, and
			// treating it as the end of the list would drop every remaining
			// user in the domain.
			return fmt.Errorf("the New Relic API stopped returning authentication domain %q while paging its users", domain.ID)
		}

		next := found[0].Users.NextCursor
		domain.Users.Users = append(domain.Users.Users, found[0].Users.Users...)
		if next == "" {
			return nil
		}
		if _, repeated := seen[next]; repeated {
			return fmt.Errorf("the New Relic API kept returning the same page cursor while listing the users of authentication domain %q", domain.ID)
		}
		seen[next] = struct{}{}
		cursor = next
	}

	return fmt.Errorf("listing the users of authentication domain %q did not finish within %d pages", domain.ID, maxPages)
}

// fetchAdminAuthDomains reads the authentication domains from the customer
// administration view, which is the only place the login method is exposed.
func fetchAdminAuthDomains(ctx context.Context, client *connection.Client) ([]apiAdminAuthDomain, error) {
	var items []apiAdminAuthDomain

	err := walkPages("authentication domain login methods", func(cursor string) (string, error) {
		var resp struct {
			CustomerAdministration struct {
				AuthenticationDomains apiAdminAuthDomainsPage `json:"authenticationDomains"`
			} `json:"customerAdministration"`
		}
		if err := client.Query(ctx, adminAuthDomainsQuery, cursorVars(cursor, nil), &resp); err != nil {
			return "", err
		}
		page := resp.CustomerAdministration.AuthenticationDomains
		items = append(items, page.Items...)
		return page.NextCursor, nil
	})
	if err != nil {
		return nil, err
	}
	return items, nil
}

// fetchGroupsWithGrants reads every group in the organization along with the
// access grants it holds.
func fetchGroupsWithGrants(ctx context.Context, client *connection.Client) ([]apiGroup, error) {
	var domains []apiGrantAuthDomain

	err := walkPages("groups", func(cursor string) (string, error) {
		var resp struct {
			Actor struct {
				Organization struct {
					AuthorizationManagement struct {
						AuthenticationDomains apiGrantAuthDomainsPage `json:"authenticationDomains"`
					} `json:"authorizationManagement"`
				} `json:"organization"`
			} `json:"actor"`
		}
		if err := client.Query(ctx, groupsWithGrantsQuery, cursorVars(cursor, nil), &resp); err != nil {
			return "", err
		}
		page := resp.Actor.Organization.AuthorizationManagement.AuthenticationDomains
		domains = append(domains, page.AuthenticationDomains...)
		return page.NextCursor, nil
	})
	if err != nil {
		return nil, err
	}

	var groups []apiGroup
	for i := range domains {
		domain := &domains[i]
		if err := completeDomainGroups(ctx, client, domain); err != nil {
			return nil, err
		}
		for j := range domain.Groups.Groups {
			group := domain.Groups.Groups[j]
			group.domainID = domain.ID
			// An access grant list that does not fit on one page cannot be
			// completed through a documented follow-up query, so it is reported
			// rather than silently truncated. Truncating it would drop grants
			// and make an over-privileged group look narrower than it is.
			if group.Roles.NextCursor != "" {
				return nil, fmt.Errorf("group %q holds more access grants than one page returns, and this provider cannot page them", group.ID)
			}
			groups = append(groups, group)
		}
	}

	return groups, nil
}

// completeDomainGroups keeps requesting group pages for one domain until the
// domain's group list is exhausted.
func completeDomainGroups(ctx context.Context, client *connection.Client, domain *apiGrantAuthDomain) error {
	cursor := domain.Groups.NextCursor
	if cursor == "" {
		return nil
	}

	seen := map[string]struct{}{cursor: {}}
	for page := 0; page < maxPages; page++ {
		var resp struct {
			Actor struct {
				Organization struct {
					AuthorizationManagement struct {
						AuthenticationDomains struct {
							AuthenticationDomains []apiGrantAuthDomain `json:"authenticationDomains"`
						} `json:"authenticationDomains"`
					} `json:"authorizationManagement"`
				} `json:"organization"`
			} `json:"actor"`
		}
		vars := cursorVars(cursor, map[string]any{"domainIds": []string{domain.ID}})
		if err := client.Query(ctx, domainGroupsPageQuery, vars, &resp); err != nil {
			return err
		}

		found := resp.Actor.Organization.AuthorizationManagement.AuthenticationDomains.AuthenticationDomains
		if len(found) == 0 {
			return fmt.Errorf("the New Relic API stopped returning authentication domain %q while paging its groups", domain.ID)
		}

		next := found[0].Groups.NextCursor
		domain.Groups.Groups = append(domain.Groups.Groups, found[0].Groups.Groups...)
		if next == "" {
			return nil
		}
		if _, repeated := seen[next]; repeated {
			return fmt.Errorf("the New Relic API kept returning the same page cursor while listing the groups of authentication domain %q", domain.ID)
		}
		seen[next] = struct{}{}
		cursor = next
	}

	return fmt.Errorf("listing the groups of authentication domain %q did not finish within %d pages", domain.ID, maxPages)
}

func fetchRoles(ctx context.Context, client *connection.Client) ([]apiRole, error) {
	var roles []apiRole

	err := walkPages("roles", func(cursor string) (string, error) {
		var resp struct {
			Actor struct {
				Organization struct {
					AuthorizationManagement struct {
						Roles apiRolesPage `json:"roles"`
					} `json:"authorizationManagement"`
				} `json:"organization"`
			} `json:"actor"`
		}
		if err := client.Query(ctx, rolesQuery, cursorVars(cursor, nil), &resp); err != nil {
			return "", err
		}
		page := resp.Actor.Organization.AuthorizationManagement.Roles
		roles = append(roles, page.Roles...)
		return page.NextCursor, nil
	})
	if err != nil {
		return nil, err
	}
	return roles, nil
}

func fetchAPIKeys(ctx context.Context, client *connection.Client, accountID int) ([]apiKey, error) {
	var keys []apiKey

	err := walkPages("API keys", func(cursor string) (string, error) {
		var resp struct {
			Actor struct {
				APIAccess struct {
					KeySearch apiKeySearchPage `json:"keySearch"`
				} `json:"apiAccess"`
			} `json:"actor"`
		}
		vars := cursorVars(cursor, map[string]any{"accountId": accountID})
		if err := client.Query(ctx, apiKeysQuery, vars, &resp); err != nil {
			return "", err
		}
		page := resp.Actor.APIAccess.KeySearch
		keys = append(keys, page.Keys...)
		return page.NextCursor, nil
	})
	if err != nil {
		return nil, err
	}
	return keys, nil
}

func fetchAlertPolicies(ctx context.Context, client *connection.Client, accountID int) ([]apiAlertPolicy, error) {
	var policies []apiAlertPolicy

	err := walkPages("alert policies", func(cursor string) (string, error) {
		var resp struct {
			Actor struct {
				Account struct {
					Alerts struct {
						PoliciesSearch apiAlertPoliciesPage `json:"policiesSearch"`
					} `json:"alerts"`
				} `json:"account"`
			} `json:"actor"`
		}
		vars := cursorVars(cursor, map[string]any{"accountId": accountID})
		if err := client.Query(ctx, alertPoliciesQuery, vars, &resp); err != nil {
			return "", err
		}
		page := resp.Actor.Account.Alerts.PoliciesSearch
		policies = append(policies, page.Policies...)
		return page.NextCursor, nil
	})
	if err != nil {
		return nil, err
	}
	return policies, nil
}

func fetchAlertConditions(ctx context.Context, client *connection.Client, accountID int) ([]apiAlertCondition, error) {
	var conditions []apiAlertCondition

	err := walkPages("alert conditions", func(cursor string) (string, error) {
		var resp struct {
			Actor struct {
				Account struct {
					Alerts struct {
						NrqlConditionsSearch apiAlertConditionsPage `json:"nrqlConditionsSearch"`
					} `json:"alerts"`
				} `json:"account"`
			} `json:"actor"`
		}
		vars := cursorVars(cursor, map[string]any{"accountId": accountID})
		if err := client.Query(ctx, alertConditionsQuery, vars, &resp); err != nil {
			return "", err
		}
		page := resp.Actor.Account.Alerts.NrqlConditionsSearch
		conditions = append(conditions, page.NrqlConditions...)
		return page.NextCursor, nil
	})
	if err != nil {
		return nil, err
	}
	return conditions, nil
}

func fetchNotificationDestinations(ctx context.Context, client *connection.Client, accountID int) ([]apiNotificationDestination, error) {
	var destinations []apiNotificationDestination

	err := walkPages("notification destinations", func(cursor string) (string, error) {
		var resp struct {
			Actor struct {
				Account struct {
					AiNotifications struct {
						Destinations apiNotificationDestinationsPage `json:"destinations"`
					} `json:"aiNotifications"`
				} `json:"account"`
			} `json:"actor"`
		}
		vars := cursorVars(cursor, map[string]any{"accountId": accountID})
		if err := client.Query(ctx, notificationDestinationsQuery, vars, &resp); err != nil {
			return "", err
		}
		page := resp.Actor.Account.AiNotifications.Destinations
		if err := page.Error.asError("notification destinations"); err != nil {
			return "", err
		}
		destinations = append(destinations, page.Entities...)
		return page.NextCursor, nil
	})
	if err != nil {
		return nil, err
	}
	return destinations, nil
}

func fetchNotificationChannels(ctx context.Context, client *connection.Client, accountID int) ([]apiNotificationChannel, error) {
	var channels []apiNotificationChannel

	err := walkPages("notification channels", func(cursor string) (string, error) {
		var resp struct {
			Actor struct {
				Account struct {
					AiNotifications struct {
						Channels apiNotificationChannelsPage `json:"channels"`
					} `json:"aiNotifications"`
				} `json:"account"`
			} `json:"actor"`
		}
		vars := cursorVars(cursor, map[string]any{"accountId": accountID})
		if err := client.Query(ctx, notificationChannelsQuery, vars, &resp); err != nil {
			return "", err
		}
		page := resp.Actor.Account.AiNotifications.Channels
		if err := page.Error.asError("notification channels"); err != nil {
			return "", err
		}
		channels = append(channels, page.Entities...)
		return page.NextCursor, nil
	})
	if err != nil {
		return nil, err
	}
	return channels, nil
}

func fetchDropRules(ctx context.Context, client *connection.Client, accountID int) ([]apiDropRule, error) {
	var resp struct {
		Actor struct {
			Account struct {
				NrqlDropRules struct {
					List apiDropRulesList `json:"list"`
				} `json:"nrqlDropRules"`
			} `json:"account"`
		} `json:"actor"`
	}
	vars := map[string]any{"accountId": accountID}
	if err := client.Query(ctx, dropRulesQuery, vars, &resp); err != nil {
		return nil, err
	}

	list := resp.Actor.Account.NrqlDropRules.List
	if dropRulesErrorIsSet(list.Error) {
		// The list came back empty because the API refused it, not because the
		// account has no drop rules. Reporting an empty list would say that
		// nothing is being discarded, which is the opposite of unknown.
		reason := string(list.Error.Reason)
		if reason == "" {
			reason = list.Error.Description
		} else if list.Error.Description != "" {
			reason += ": " + list.Error.Description
		}
		return nil, fmt.Errorf("the New Relic API could not list drop rules: %s", reason)
	}
	return list.Rules, nil
}

func fetchRetentionRules(ctx context.Context, client *connection.Client, accountID int) ([]apiRetentionRule, error) {
	var resp struct {
		Actor struct {
			Account struct {
				DataManagement struct {
					EventRetentionRules []apiRetentionRule `json:"eventRetentionRules"`
				} `json:"dataManagement"`
			} `json:"account"`
		} `json:"actor"`
	}
	vars := map[string]any{"accountId": accountID}
	if err := client.Query(ctx, retentionRulesQuery, vars, &resp); err != nil {
		return nil, err
	}
	return resp.Actor.Account.DataManagement.EventRetentionRules, nil
}
