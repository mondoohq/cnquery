// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package sshd

import (
	"strings"
)

type MatchBlock struct {
	Criteria string
	// Note: we set the value type to any, but it must be a string.
	// This is done due to type limitations in go and MQL's internal processing
	Params map[string]any
}

type MatchBlocks []*MatchBlock

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
			if x, ok := res[k]; ok {
				res[k] = x.(string) + "," + v.(string)
			} else {
				res[k] = v
			}
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

func ParseBlocks(content string) (MatchBlocks, error) {
	lines := strings.Split(content, "\n")

	curBlock := &MatchBlock{
		Criteria: "",
		Params:   map[string]any{},
	}
	res := []*MatchBlock{curBlock}
	matchConditions := map[string]*MatchBlock{
		"": curBlock,
	}

	for _, textLine := range lines {
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

		if key == "Match" {
			// This key is the only that we don't add to any params. It is stored
			// in the condition of each block and can be accessed there.
			condition := l.args
			if b, ok := matchConditions[condition]; ok {
				curBlock = b
			} else {
				curBlock = &MatchBlock{
					Criteria: condition,
					Params:   map[string]any{},
				}
				matchConditions[condition] = curBlock
				res = append(res, curBlock)
			}
			continue
		}

		if x, ok := curBlock.Params[key]; ok {
			curBlock.Params[key] = x.(string) + "," + l.args
		} else {
			curBlock.Params[key] = l.args
		}
	}

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
