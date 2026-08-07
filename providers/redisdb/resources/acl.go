// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/types"
)

// aclUser is the parsed form of one ACL LIST line.
type aclUser struct {
	name            string
	enabled         bool
	nopass          bool
	passwordCount   int64
	keyPatterns     []any
	channelPatterns []any
	commandRules    []any
}

// parseACLLine parses a single ACL LIST line in ACL-file format:
//
//	user <name> on|off [nopass|#<hash>...] [~<key>|allkeys|%...] [&<chan>|allchannels] [+cmd|-cmd|+@cat|-@cat]...
func parseACLLine(line string) aclUser {
	u := aclUser{keyPatterns: []any{}, channelPatterns: []any{}, commandRules: []any{}}
	fields := strings.Fields(line)
	if len(fields) < 2 || fields[0] != "user" {
		return u
	}
	u.name = fields[1]
	for _, tok := range fields[2:] {
		switch {
		case tok == "on":
			u.enabled = true
		case tok == "off":
			u.enabled = false
		case tok == "nopass":
			u.nopass = true
		case strings.HasPrefix(tok, "#"), strings.HasPrefix(tok, ">"):
			u.passwordCount++
		case tok == "allkeys":
			u.keyPatterns = append(u.keyPatterns, "*")
		case strings.HasPrefix(tok, "~"):
			u.keyPatterns = append(u.keyPatterns, strings.TrimPrefix(tok, "~"))
		case strings.HasPrefix(tok, "%"):
			// Read/write-qualified key pattern (Redis 7.0+), e.g. "%RW~foo:*".
			if i := strings.IndexByte(tok, '~'); i >= 0 {
				u.keyPatterns = append(u.keyPatterns, tok[i+1:])
			}
		case tok == "allchannels":
			u.channelPatterns = append(u.channelPatterns, "*")
		case strings.HasPrefix(tok, "&"):
			u.channelPatterns = append(u.channelPatterns, strings.TrimPrefix(tok, "&"))
		case strings.HasPrefix(tok, "+"), strings.HasPrefix(tok, "-"):
			u.commandRules = append(u.commandRules, tok)
		}
		// Other flags (sanitize-payload, resetchannels, selectors) are ignored.
	}
	return u
}

func (r *mqlRedisdbInstance) users() ([]any, error) {
	conn := redisdbConnection(r.MqlRuntime)
	client, err := conn.Client()
	if err != nil {
		return nil, err
	}
	lines, err := client.ACLList(conn.Context()).Result()
	if err != nil {
		// Reading the ACL roster needs the +acl privilege; treat a denial as no
		// visible users rather than failing the whole asset.
		if isNoPerm(err) {
			return []any{}, nil
		}
		return nil, err
	}

	serverID := conn.ServerID()
	list := []any{}
	for _, line := range lines {
		u := parseACLLine(line)
		if u.name == "" {
			continue
		}
		res, err := CreateResource(r.MqlRuntime, "redisdb.acl.user", map[string]*llx.RawData{
			"__id":            llx.StringData(serverID + "/user/" + u.name),
			"name":            llx.StringData(u.name),
			"isDefault":       llx.BoolData(u.name == "default"),
			"enabled":         llx.BoolData(u.enabled),
			"nopass":          llx.BoolData(u.nopass),
			"passwordCount":   llx.IntData(u.passwordCount),
			"keyPatterns":     llx.ArrayData(u.keyPatterns, types.String),
			"channelPatterns": llx.ArrayData(u.channelPatterns, types.String),
			"commandRules":    llx.ArrayData(u.commandRules, types.String),
		})
		if err != nil {
			return nil, err
		}
		list = append(list, res)
	}
	return list, nil
}
