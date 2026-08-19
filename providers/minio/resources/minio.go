// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"strings"
	"sync"

	madmin "github.com/minio/madmin-go/v4"
	"go.mondoo.com/mql/v13/providers/minio/connection"
)

// mqlMinioInternal caches the deployment description. Most of the deployment
// fields come out of one ServerInfo call, so fetching it per field would cost
// one admin request per field read.
type mqlMinioInternal struct {
	lock          sync.Mutex
	serverInfo    *madmin.InfoMessage
	serverInfoErr error
	fetchedInfo   bool

	configLock sync.Mutex
	configs    map[string][]madmin.SubsysConfig
}

func (r *mqlMinio) id() (string, error) {
	return "minio", nil
}

// info returns the deployment description, fetching it once.
func (r *mqlMinio) info() (*madmin.InfoMessage, error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.fetchedInfo {
		return r.serverInfo, r.serverInfoErr
	}

	conn := r.MqlRuntime.Connection.(*connection.MinioConnection)
	info, err := conn.Admin().ServerInfo(context.Background())
	r.fetchedInfo = true
	if err != nil {
		r.serverInfoErr = err
		return nil, err
	}
	r.serverInfo = &info
	return r.serverInfo, nil
}

// config returns one parsed configuration subsystem, fetching it once.
func (r *mqlMinio) config(subsys string) ([]madmin.SubsysConfig, error) {
	r.configLock.Lock()
	defer r.configLock.Unlock()
	if cached, ok := r.configs[subsys]; ok {
		return cached, nil
	}

	conn := r.MqlRuntime.Connection.(*connection.MinioConnection)
	parsed, err := readConfigSubsys(context.Background(), conn, subsys)
	if err != nil {
		return nil, err
	}
	if r.configs == nil {
		r.configs = map[string][]madmin.SubsysConfig{}
	}
	r.configs[subsys] = parsed
	return parsed, nil
}

// configValue reads one key from the default (unnamed) row of a subsystem.
func (r *mqlMinio) configValue(subsys, key string) (string, error) {
	rows, err := r.config(subsys)
	if err != nil {
		return "", err
	}
	for _, row := range rows {
		if row.Target != "" {
			continue
		}
		if v, ok := configKV(row)[key]; ok {
			return v, nil
		}
	}
	return "", nil
}

func (r *mqlMinio) version() (string, error) {
	info, err := r.info()
	if err != nil {
		return "", err
	}
	if len(info.Servers) == 0 {
		return "", nil
	}
	return info.Servers[0].Version, nil
}

// deploymentMode reports standalone for a deployment served by one process and
// distributed for one spanning several. A single server with many drives is
// still standalone: it loses availability whenever that one server does.
func (r *mqlMinio) deploymentMode() (string, error) {
	info, err := r.info()
	if err != nil {
		return "", err
	}
	if len(info.Servers) > 1 {
		return "distributed", nil
	}
	return "standalone", nil
}

func (r *mqlMinio) region() (string, error) {
	info, err := r.info()
	if err != nil {
		return "", err
	}
	return info.Region, nil
}

func (r *mqlMinio) deploymentId() (string, error) {
	info, err := r.info()
	if err != nil {
		return "", err
	}
	return info.DeploymentID, nil
}

func (r *mqlMinio) endpoint() (string, error) {
	conn := r.MqlRuntime.Connection.(*connection.MinioConnection)
	return conn.Host(), nil
}

func (r *mqlMinio) tlsEnabled() (bool, error) {
	conn := r.MqlRuntime.Connection.(*connection.MinioConnection)
	return conn.Secure(), nil
}

func (r *mqlMinio) backendType() (string, error) {
	info, err := r.info()
	if err != nil {
		return "", err
	}
	return string(info.Backend.Type), nil
}

func (r *mqlMinio) onlineDrives() (int64, error) {
	info, err := r.info()
	if err != nil {
		return 0, err
	}
	return int64(info.Backend.OnlineDisks), nil
}

func (r *mqlMinio) offlineDrives() (int64, error) {
	info, err := r.info()
	if err != nil {
		return 0, err
	}
	return int64(info.Backend.OfflineDisks), nil
}

// kmsConfigured reports whether a key management service is attached. MinIO
// refuses to configure any bucket encryption until one is, so a deployment
// reporting false cannot encrypt objects at rest with server-managed keys.
func (r *mqlMinio) kmsConfigured() (bool, error) {
	info, err := r.info()
	if err != nil {
		return false, err
	}
	for _, kms := range info.Services.KMSStatus {
		if kms.Status != "" || kms.Endpoint != "" {
			return true, nil
		}
	}
	return false, nil
}

func (r *mqlMinio) consoleHstsSeconds() (int64, error) {
	value, err := r.configValue(subsysBrowser, "hsts_seconds")
	if err != nil {
		return 0, err
	}
	return parseInt(value), nil
}

func (r *mqlMinio) consoleHstsIncludeSubdomains() (bool, error) {
	value, err := r.configValue(subsysBrowser, "hsts_include_subdomains")
	if err != nil {
		return false, err
	}
	return parseOnOff(value), nil
}

func (r *mqlMinio) consoleHstsPreload() (bool, error) {
	value, err := r.configValue(subsysBrowser, "hsts_preload")
	if err != nil {
		return false, err
	}
	return parseOnOff(value), nil
}

// corsAllowOrigin reports the origins the S3 API accepts cross-origin browser
// requests from. MinIO stores them as one comma-separated value.
func (r *mqlMinio) corsAllowOrigin() ([]any, error) {
	value, err := r.configValue(subsysAPI, "cors_allow_origin")
	if err != nil {
		return nil, err
	}
	return splitCommaList(value), nil
}

func splitCommaList(value string) []any {
	out := []any{}
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}
