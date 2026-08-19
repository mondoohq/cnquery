// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"net/http"
	"strings"

	consulapi "github.com/hashicorp/consul/api"
)

// aclDisabledMessage is the body a Consul agent returns from every ACL endpoint
// when the ACL system is switched off.
const aclDisabledMessage = "ACL support disabled"

// isACLSystemDisabled reports whether the agent answered that its ACL system is
// switched off, which is the one answer that may be turned into an empty
// inventory. The endpoints do not exist on such an agent, so reporting no
// tokens is a fact about the agent rather than a failure to read it.
//
// A 403 is deliberately excluded, and it is the status that matters here: a
// Consul agent answers 403 both for a token it does not recognize ("ACL not
// found") and for one lacking acl:read ("Permission denied"). Neither says
// anything about what is behind the endpoint, so reporting "none" would turn a
// missing permission into a clean audit pass.
//
// The classifier matches on the structured status and body the client attaches
// to the error, never on the stringified error, so a transport failure cannot
// be mistaken for a definitive answer. The body is required as well as the
// status because a bare 401 from something other than the ACL subsystem, a
// proxy in front of the agent for instance, must fail loudly rather than read
// as an absent feature.
func isACLSystemDisabled(err error) bool {
	if err == nil {
		return false
	}

	var statusErr consulapi.StatusError
	if !errors.As(err, &statusErr) {
		return false
	}
	if statusErr.Code != http.StatusUnauthorized {
		return false
	}
	return strings.Contains(statusErr.Body, aclDisabledMessage)
}

// isNotFound reports whether the agent answered that the thing asked for does
// not exist. Only a 404 counts: a permission failure is not an absence.
func isNotFound(err error) bool {
	if err == nil {
		return false
	}

	var statusErr consulapi.StatusError
	if !errors.As(err, &statusErr) {
		return false
	}
	return statusErr.Code == http.StatusNotFound
}
