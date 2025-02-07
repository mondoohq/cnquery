// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package sshd

import (
	"strings"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/utils/sortx"
)

type MatchBlock struct {
	Criteria string
	// Note: we set the value type to any, but it must be a string.
	// This is done due to type limitations in go and MQL's internal processing
	Params  map[string]any
	Context Context
}

type Context struct {
	Path    string
	Range   llx.Range
	curLine int
}

func setParam(m map[string]any, key string, value string) {
	v, ok := m[key]
	if !ok {
		m[key] = value
	} else if isMultiParam[key] {
		m[key] = v.(string) + "," + value
	}
}

type MatchBlocks []*MatchBlock

var isMultiParam = map[string]bool{
	"AllowGroups":   true,
	"AllowUsers":    true,
	"DenyGroups":    true,
	"DenyUsers":     true,
	"ListenAddress": true,
	"Port":          true,
	"AcceptEnv":     true,
	"HostKey":       true,
}

func (m MatchBlocks) Flatten() map[string]any {
	if len(m) == 0 {
		return nil
	}
	if len(m) == 1 {
		return m[0].Params
	}

	// We are using the first block as a starting point for the size.
	// We can't just add the sizes of params across all blocks, because keys
	// may be used across multiple blocks. It is likely that the size will
	// have to grow, but it's the floor and a good starting point.
	res := make(map[string]any, len(m[0].Params))
	matchConditions := []string{}
	for i := range m {
		cur := m[i]

		if cur.Criteria != "" {
			matchConditions = append(matchConditions, cur.Criteria)
		}

		for k, v := range cur.Params {
			setParam(res, k, v.(string))
		}
	}

	// We are adding one flattened key for all match groups. This is
	// more to be informative and consistent, rather than useful.
	// The most useful way to access conditions is to cycle over all match blocks.
	if len(matchConditions) != 0 {
		res["Match"] = strings.Join(matchConditions, ",")
	}

	return res
}

func mergeIncludedBlocks(matchConditions map[string]*MatchBlock, blocks MatchBlocks, curBlock string) {
	for _, block := range blocks {
		// meaning:
		// 1. curBlock == "", we can always add all subblocks
		// 2. if block == "", we can add it to whatever current block is
		// 3. in all other cases the block criteria must match, or we move on
		if block.Criteria != curBlock && curBlock != "" && block.Criteria != "" {
			continue
		}

		var existing *MatchBlock
		if block.Criteria == "" {
			existing = matchConditions[curBlock]
		} else {
			existing := matchConditions[block.Criteria]
			if existing == nil {
				matchConditions[block.Criteria] = block
				continue
			}
		}

		for k, v := range block.Params {
			if _, ok := existing.Params[k]; !ok {
				existing.Params[k] = v
			}
		}
	}
}

func ParseBlocks(rootPath string, globPathContent func(string) (string, error)) (MatchBlocks, error) {
	content, err := globPathContent(rootPath)
	if err != nil {
		return nil, err
	}

	curBlock := &MatchBlock{
		Criteria: "",
		Params:   map[string]any{},
		Context: Context{
			Path:    rootPath,
			Range:   llx.NewRange(),
			curLine: 1,
		},
	}
	matchConditions := map[string]*MatchBlock{
		"": curBlock,
	}

	lines := strings.Split(content, "\n")
	for curLineIdx, textLine := range lines {
		l, err := ParseLine([]rune(textLine))
		if err != nil {
			return nil, err
		}

		key := l.key
		if key == "" {
			continue
		}

		// handle lower case entries and use proper ssh camel case
		if sshKey, ok := SSH_Keywords[strings.ToLower(key)]; ok {
			key = sshKey
		}

		if key == "Include" {
			// FIXME: parse multi-keys properly
			paths := strings.Split(l.args, " ")

			for _, path := range paths {
				subBlocks, err := ParseBlocks(path, globPathContent)
				if err != nil {
					return nil, err
				}
				mergeIncludedBlocks(matchConditions, subBlocks, curBlock.Criteria)
			}
			continue
		}

		if key == "Match" {
			// wrap up context on the previous block
			curBlock.Context.Range = curBlock.Context.Range.AddLineRange(uint32(curBlock.Context.curLine), uint32(curLineIdx-1))
			curBlock.Context.curLine = curLineIdx

			// This key is the only that we don't add to any params. It is stored
			// in the condition of each block and can be accessed there.
			condition := l.args
			if b, ok := matchConditions[condition]; ok {
				curBlock = b
			} else {
				curBlock = &MatchBlock{
					Criteria: condition,
					Params:   map[string]any{},
					Context: Context{
						curLine: curLineIdx + 1,
						Path:    rootPath,
						Range:   llx.NewRange(),
					},
				}
				matchConditions[condition] = curBlock
			}
			continue
		}

		setParam(curBlock.Params, key, l.args)
	}

	keys := sortx.Keys(matchConditions)
	res := make([]*MatchBlock, len(keys))
	i := 0
	for _, key := range keys {
		res[i] = matchConditions[key]
		i++
	}

	curBlock.Context.Range = curBlock.Context.Range.AddLineRange(uint32(curBlock.Context.curLine), uint32(len(lines)))

	return res, nil
}

var SSH_Keywords = map[string]string{
	"acceptenv":                       "AcceptEnv",
	"addressfamily":                   "AddressFamily",
	"allowagentforwarding":            "AllowAgentForwarding",
	"allowgroups":                     "AllowGroups",
	"allowstreamlocalforwarding":      "AllowStreamLocalForwarding",
	"allowtcpforwarding":              "AllowTcpForwarding",
	"allowusers":                      "AllowUsers",
	"authenticationmethods":           "AuthenticationMethods",
	"authorizedkeyscommand":           "AuthorizedKeysCommand",
	"authorizedkeyscommanduser":       "AuthorizedKeysCommandUser",
	"authorizedkeysfile":              "AuthorizedKeysFile",
	"authorizedprincipalscommand":     "AuthorizedPrincipalsCommand",
	"authorizedprincipalscommanduser": "AuthorizedPrincipalsCommandUser",
	"authorizedprincipalsfile":        "AuthorizedPrincipalsFile",
	"banner":                          "Banner",
	"casignaturealgorithms":           "CASignatureAlgorithms",
	"challengeresponseauthentication": "ChallengeResponseAuthentication",
	"chrootdirectory":                 "ChrootDirectory",
	"ciphers":                         "Ciphers",
	"clientalivecountmax":             "ClientAliveCountMax",
	"clientaliveinterval":             "ClientAliveInterval",
	"compression":                     "Compression",
	"denygroups":                      "DenyGroups",
	"denyusers":                       "DenyUsers",
	"disableforwarding":               "DisableForwarding",
	"exposeauthinfo":                  "ExposeAuthInfo",
	"fingerprinthash":                 "FingerprintHash",
	"forcecommand":                    "ForceCommand",
	"gssapiauthentication":            "GSSAPIAuthentication",
	"gssapicleanupcredentials":        "GSSAPICleanupCredentials",
	"gssapistrictacceptorcheck":       "GSSAPIStrictAcceptorCheck",
	"gatewayports":                    "GatewayPorts",
	"hostcertificate":                 "HostCertificate",
	"hostkey":                         "HostKey",
	"hostkeyagent":                    "HostKeyAgent",
	"hostkeyalgorithms":               "HostKeyAlgorithms",
	"hostbasedacceptedkeytypes":       "HostbasedAcceptedKeyTypes",
	"hostbasedauthentication":         "HostbasedAuthentication",
	"hostbasedusesnamefrompacketonly": "HostbasedUsesNameFromPacketOnly",
	"ipqos":                           "IPQoS",
	"ignorerhosts":                    "IgnoreRhosts",
	"ignoreuserknownhosts":            "IgnoreUserKnownHosts",
	"include":                         "Include",
	"kbdinteractiveauthentication":    "KbdInteractiveAuthentication",
	"kerberosauthentication":          "KerberosAuthentication",
	"kerberosgetafstoken":             "KerberosGetAFSToken",
	"kerberosorlocalpasswd":           "KerberosOrLocalPasswd",
	"kerberosticketcleanup":           "KerberosTicketCleanup",
	"kexalgorithms":                   "KexAlgorithms",
	"listenaddress":                   "ListenAddress",
	"loglevel":                        "LogLevel",
	"logingracetime":                  "LoginGraceTime",
	"macs":                            "MACs",
	"match":                           "Match",
	"maxauthtries":                    "MaxAuthTries",
	"maxsessions":                     "MaxSessions",
	"maxstartups":                     "MaxStartups",
	"passwordauthentication":          "PasswordAuthentication",
	"permitemptypasswords":            "PermitEmptyPasswords",
	"permitlisten":                    "PermitListen",
	"permitopen":                      "PermitOpen",
	"permitrootlogin":                 "PermitRootLogin",
	"permittty":                       "PermitTTY",
	"permittunnel":                    "PermitTunnel",
	"permituserenvironment":           "PermitUserEnvironment",
	"permituserrc":                    "PermitUserRC",
	"pidfile":                         "PidFile",
	"port":                            "Port",
	"printlastlog":                    "PrintLastLog",
	"printmotd":                       "PrintMotd",
	"pubkeyacceptedkeytypes":          "PubkeyAcceptedKeyTypes",
	"pubkeyauthoptions":               "PubkeyAuthOptions",
	"pubkeyauthentication":            "PubkeyAuthentication",
	"rdomain":                         "RDomain",
	"rekeylimit":                      "RekeyLimit",
	"revokedkeys":                     "RevokedKeys",
	"securitykeyprovider":             "SecurityKeyProvider",
	"setenv":                          "SetEnv",
	"streamlocalbindmask":             "StreamLocalBindMask",
	"streamlocalbindunlink":           "StreamLocalBindUnlink",
	"strictmodes":                     "StrictModes",
	"subsystem":                       "Subsystem",
	"syslogfacility":                  "SyslogFacility",
	"tcpkeepalive":                    "TCPKeepAlive",
	"trustedusercakeys":               "TrustedUserCAKeys",
	"usedns":                          "UseDNS",
	"usepam":                          "UsePAM",
	"versionaddendum":                 "VersionAddendum",
	"x11displayoffset":                "X11DisplayOffset",
	"x11forwarding":                   "X11Forwarding",
	"x11uselocalhost":                 "X11UseLocalhost",
	"xauthlocation":                   "XAuthLocation",
}
