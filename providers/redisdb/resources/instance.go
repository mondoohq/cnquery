// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/types"
)

// initRedisdbInstance fetches server identity from INFO and the security-
// relevant configuration from CONFIG GET, populating the instance's fields. The
// config map is cached so the config sub-resource resolves without a re-fetch.
func initRedisdbInstance(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 3 {
		return args, nil, nil
	}

	conn := redisdbConnection(runtime)
	client, err := conn.Client()
	if err != nil {
		return nil, nil, err
	}
	ctx := redisdbContext()

	infoText, err := client.Info(ctx, "server").Result()
	if err != nil {
		return nil, nil, err
	}
	info := parseInfo(infoText)

	// CONFIG GET is required to assess posture; a permission error here means
	// the connecting credential cannot audit the server.
	cfg, err := client.ConfigGet(ctx, "*").Result()
	if err != nil {
		return nil, nil, err
	}

	isValkey := info["valkey_version"] != "" || info["server_name"] == "valkey"
	// Redis reports the server mode as redis_mode; Valkey uses server_mode.
	mode := info["redis_mode"]
	if mode == "" {
		mode = info["server_mode"]
	}
	bind := strings.Fields(cfg["bind"])
	bindList := make([]any, 0, len(bind))
	for _, b := range bind {
		bindList = append(bindList, b)
	}

	args["__id"] = llx.StringData(conn.ServerID())
	args["version"] = llx.StringData(info["redis_version"])
	args["isValkey"] = llx.BoolData(isValkey)
	args["valkeyVersion"] = llx.StringData(info["valkey_version"])
	args["mode"] = llx.StringData(mode)
	args["os"] = llx.StringData(info["os"])
	args["runId"] = llx.StringData(info["run_id"])
	args["port"] = llx.IntData(atoiOr(info["tcp_port"], atoiOr(cfg["port"], 0)))
	args["protectedMode"] = llx.BoolData(cfg["protected-mode"] == "yes")
	args["bind"] = llx.ArrayData(bindList, types.String)
	args["bindsAllInterfaces"] = llx.BoolData(bindsAll(bind))
	args["requirepassSet"] = llx.BoolData(cfg["requirepass"] != "")
	tlsPort := atoiOr(cfg["tls-port"], 0)
	args["tlsPort"] = llx.IntData(tlsPort)
	args["tlsEnabled"] = llx.BoolData(tlsPort != 0)
	args["tlsAuthClients"] = llx.StringData(cfg["tls-auth-clients"])
	args["aclFile"] = llx.StringData(cfg["aclfile"])

	res, err := CreateResource(runtime, "redisdb.instance", args)
	if err != nil {
		return nil, nil, err
	}
	res.(*mqlRedisdbInstance).configCache = cfg
	return nil, res, nil
}

// bindsAll reports whether a bind list exposes the server on all interfaces:
// an empty list (no bind configured) or a wildcard address.
func bindsAll(bind []string) bool {
	if len(bind) == 0 {
		return true
	}
	for _, b := range bind {
		if b == "0.0.0.0" || b == "*" || b == "::" || b == "::*" {
			return true
		}
	}
	return false
}

func (r *mqlRedisdbInstance) config() (*mqlRedisdbConfig, error) {
	cfg := r.configCache
	res, err := CreateResource(r.MqlRuntime, "redisdb.config", map[string]*llx.RawData{
		"__id":            llx.StringData(r.__id + "/config"),
		"save":            llx.StringData(cfg["save"]),
		"rdbEnabled":      llx.BoolData(strings.TrimSpace(cfg["save"]) != ""),
		"appendOnly":      llx.BoolData(cfg["appendonly"] == "yes"),
		"appendFsync":     llx.StringData(cfg["appendfsync"]),
		"maxmemory":       llx.IntData(atoiOr(cfg["maxmemory"], 0)),
		"maxmemoryPolicy": llx.StringData(cfg["maxmemory-policy"]),
		"logfile":         llx.StringData(cfg["logfile"]),
		"dir":             llx.StringData(cfg["dir"]),
		"dbFilename":      llx.StringData(cfg["dbfilename"]),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlRedisdbConfig), nil
}
