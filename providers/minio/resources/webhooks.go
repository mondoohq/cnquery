// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"strconv"
	"strings"

	madmin "github.com/minio/madmin-go/v4"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/minio/connection"
)

const (
	webhookTypeAudit  = "audit"
	webhookTypeLogger = "logger"

	// defaultTargetName is what MinIO calls the unnamed target of a
	// subsystem, both in its status reporting and in its own documentation.
	defaultTargetName = "_"

	subsysAuditWebhook  = "audit_webhook"
	subsysLoggerWebhook = "logger_webhook"
	subsysBrowser       = "browser"
	subsysAPI           = "api"
)

// webhookTarget is one configured log destination, assembled from the
// deployment's configuration and its reported reachability.
//
// The authentication token is deliberately absent. MinIO withholds it: a target
// with a token configured comes back with the auth_token key removed entirely
// rather than emptied, so a field derived from its presence would report
// "unauthenticated" on an authenticated target and the reverse whenever that
// behavior changes. Neither direction is worth shipping.
type webhookTarget struct {
	Type                 string
	Name                 string
	Endpoint             string
	Enabled              bool
	Status               string
	QueueSize            int64
	QueueDir             string
	ClientCertConfigured bool
	MaxRetry             int64
	HTTPTimeout          string
}

// webhookTargetsFromConfig maps a subsystem's configuration onto log
// destinations. Rows carrying no endpoint are dropped: MinIO always reports a
// default row for a subsystem, whether or not anything was configured on it,
// and a row with no endpoint is that placeholder rather than a target.
//
// The status map is keyed the way ServerInfo reports it, "<type>-<name>", and a
// target missing from it gets an empty status rather than a guess.
func webhookTargetsFromConfig(kind string, configs []madmin.SubsysConfig, status map[string]string) []webhookTarget {
	targets := make([]webhookTarget, 0, len(configs))
	for _, cfg := range configs {
		kv := configKV(cfg)
		endpoint := kv["endpoint"]
		if endpoint == "" {
			continue
		}

		name := cfg.Target
		if name == "" {
			name = defaultTargetName
		}

		targets = append(targets, webhookTarget{
			Type:     kind,
			Name:     name,
			Endpoint: endpoint,
			// MinIO omits the enable key on a target it is using and writes
			// enable=off on one it is not, so an absent key means enabled.
			Enabled:              !strings.EqualFold(kv["enable"], "off"),
			Status:               status[kind+"-"+name],
			QueueSize:            parseInt(kv["queue_size"]),
			QueueDir:             kv["queue_dir"],
			ClientCertConfigured: kv["client_cert"] != "",
			MaxRetry:             parseInt(kv["max_retry"]),
			HTTPTimeout:          kv["http_timeout"],
		})
	}
	return targets
}

// configKV flattens one configuration row into a map, unquoting values. MinIO
// returns a value containing spaces wrapped in double quotes and the parser
// keeps the quotes, so a value read straight out of it carries them.
func configKV(cfg madmin.SubsysConfig) map[string]string {
	out := make(map[string]string, len(cfg.KV))
	for _, kv := range cfg.KV {
		out[kv.Key] = unquoteConfigValue(kv.Value)
	}
	return out
}

// unquoteConfigValue strips the double quotes MinIO wraps a value in when it
// contains a space. A value that is not wrapped is returned unchanged, and a
// lone quote is left alone rather than being treated as an opening one.
func unquoteConfigValue(value string) string {
	if len(value) >= 2 && strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
		return value[1 : len(value)-1]
	}
	return value
}

func parseInt(value string) int64 {
	n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return 0
	}
	return n
}

func parseOnOff(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "on", "true", "yes", "enabled":
		return true
	default:
		return false
	}
}

// webhookStatus flattens the deployment's reported target reachability into a
// map keyed "<type>-<name>", which is how MinIO names the entries.
func webhookStatus[T ~map[string]madmin.Status](entries []T) map[string]string {
	out := map[string]string{}
	for _, entry := range entries {
		for name, status := range entry {
			out[name] = status.Status
		}
	}
	return out
}

// readConfigSubsys reads one configuration subsystem and parses it.
func readConfigSubsys(ctx context.Context, conn *connection.MinioConnection, subsys string) ([]madmin.SubsysConfig, error) {
	raw, err := conn.Admin().GetConfigKV(ctx, subsys)
	if err != nil {
		return nil, err
	}
	return madmin.ParseServerConfigOutput(string(raw))
}

func (a *mqlMinio) webhooks(kind string) ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.MinioConnection)
	ctx := context.Background()

	subsys := subsysAuditWebhook
	if kind == webhookTypeLogger {
		subsys = subsysLoggerWebhook
	}

	configs, err := readConfigSubsys(ctx, conn, subsys)
	if err != nil {
		return nil, err
	}

	info, err := conn.Admin().ServerInfo(ctx)
	if err != nil {
		return nil, err
	}
	status := webhookStatus(info.Services.Audit)
	if kind == webhookTypeLogger {
		status = webhookStatus(info.Services.Logger)
	}

	targets := webhookTargetsFromConfig(kind, configs, status)
	res := make([]any, 0, len(targets))
	for _, t := range targets {
		resource, err := CreateResource(a.MqlRuntime, "minio.webhook", map[string]*llx.RawData{
			"__id":                 llx.StringData("webhook/" + t.Type + "/" + t.Name),
			"type":                 llx.StringData(t.Type),
			"name":                 llx.StringData(t.Name),
			"endpoint":             llx.StringData(t.Endpoint),
			"enabled":              llx.BoolData(t.Enabled),
			"status":               llx.StringData(t.Status),
			"queueSize":            llx.IntData(t.QueueSize),
			"queueDir":             llx.StringData(t.QueueDir),
			"clientCertConfigured": llx.BoolData(t.ClientCertConfigured),
			"maxRetry":             llx.IntData(t.MaxRetry),
			"httpTimeout":          llx.StringData(t.HTTPTimeout),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, resource)
	}
	return res, nil
}

func (a *mqlMinio) auditWebhooks() ([]any, error) {
	return a.webhooks(webhookTypeAudit)
}

func (a *mqlMinio) loggerWebhooks() ([]any, error) {
	return a.webhooks(webhookTypeLogger)
}
