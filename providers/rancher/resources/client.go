// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/rancher/connection"
)

// API paths this provider reads. They are collected here so that the set of
// endpoints the provider touches is readable in one place.
const (
	pathSettings                   = "/v3/settings"
	pathClusters                   = "/v3/clusters"
	pathProjects                   = "/v3/projects"
	pathGlobalRoles                = "/v3/globalRoles"
	pathGlobalRoleBindings         = "/v3/globalRoleBindings"
	pathRoleTemplates              = "/v3/roleTemplates"
	pathClusterRoleTemplateBinding = "/v3/clusterRoleTemplateBindings"
	pathProjectRoleTemplateBinding = "/v3/projectRoleTemplateBindings"
	pathClusterTemplates           = "/v3/clusterTemplates"
	pathClusterTemplateRevisions   = "/v3/clusterTemplateRevisions"
	pathPodSecurityTemplates       = "/v3/podSecurityAdmissionConfigurationTemplates"
	pathAuthConfigs                = "/v3/authConfigs"
	pathTokens                     = "/v3/tokens"
	pathUsers                      = "/v3/users"
)

func rancherConnection(runtime *plugin.Runtime) (*connection.RancherConnection, error) {
	if runtime == nil {
		return nil, errors.New("no rancher connection")
	}
	conn, ok := runtime.Connection.(*connection.RancherConnection)
	if !ok || conn == nil {
		return nil, errors.New("no rancher connection")
	}
	return conn, nil
}

func rancherClient(runtime *plugin.Runtime) (*connection.Client, error) {
	conn, err := rancherConnection(runtime)
	if err != nil {
		return nil, err
	}
	return conn.Client(), nil
}

// listRecords fetches a collection and decodes every record into T.
//
// A record that does not decode is skipped rather than failing the whole
// listing: one malformed entry must not blank out every other cluster or
// binding on the server. The count of skipped records is reported in the error
// only when nothing at all decoded, since a listing that is entirely
// undecodable is a schema mismatch rather than a bad record.
func listRecords[T any](runtime *plugin.Runtime, path string) ([]T, error) {
	client, err := rancherClient(runtime)
	if err != nil {
		return nil, err
	}

	raw, err := client.ListCached(context.Background(), path)
	if err != nil {
		return nil, err
	}

	records := make([]T, 0, len(raw))
	skipped := 0
	for _, entry := range raw {
		var record T
		if err := json.Unmarshal(entry, &record); err != nil {
			skipped++
			continue
		}
		records = append(records, record)
	}

	if skipped > 0 && len(records) == 0 {
		return nil, fmt.Errorf("rancher api %s: none of the %d records could be read", path, skipped)
	}
	return records, nil
}

// listOptionalRecords is listRecords for an endpoint that a Rancher release may
// not serve at all, such as cluster templates on 2.12 and newer.
//
// Only a 404 is turned into an empty result. A 403 says the token may not read
// the endpoint, and a transport failure says nothing at all; reporting either
// as "the feature is not present" would let an unreadable server pass a check
// written against what the feature governs.
func listOptionalRecords[T any](runtime *plugin.Runtime, path string) ([]T, error) {
	records, err := listRecords[T](runtime, path)
	if err != nil && connection.IsNotFound(err) {
		return []T{}, nil
	}
	return records, err
}

// endpointExists reports whether the server serves a collection at all.
//
// Rancher removes whole features between minor releases and stops registering
// their schema, so the endpoint answers 404 rather than an empty collection.
// That distinction matters: an empty collection means the feature exists and
// nothing uses it, while a missing endpoint means the feature is gone and any
// setting that used to govern it now governs nothing.
//
// Only a 404 answers false. A refusal or a transport failure is not an answer,
// and is returned as the error it is.
func endpointExists(runtime *plugin.Runtime, path string) (bool, error) {
	client, err := rancherClient(runtime)
	if err != nil {
		return false, err
	}
	if _, err := client.ListCached(context.Background(), path); err != nil {
		if connection.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ServerVersion reads the version the server reports about itself. It is used
// during asset detection, before any resource exists.
func ServerVersion(ctx context.Context, client *connection.Client) (string, error) {
	if client == nil {
		return "", errors.New("no rancher client")
	}
	var record settingRecord
	if err := client.Get(ctx, pathSettings+"/server-version", &record); err != nil {
		return "", err
	}
	if record.Value != "" {
		return record.Value, nil
	}
	return record.Default, nil
}
