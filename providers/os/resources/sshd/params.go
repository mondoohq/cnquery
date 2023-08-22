// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package sshd

import (
	"strings"
)

func Params(content string) (map[string]string, error) {
	lines := strings.Split(content, "\n")

	res := make(map[string]string)
	for _, textLine := range lines {
		l, err := ParseLine([]rune(textLine))
		if err != nil {
			return nil, err
		}

		k := l.key
		if k == "" {
			continue
		}

		// handle lower case entries and use proper ssh camel case
		if sshKey, ok := SSH_Keywords[strings.ToLower(k)]; ok {
			k = sshKey
		}

		// check if we have an entry already
		if val, ok := res[k]; ok {
			res[k] = val + "," + l.args
		} else {
			res[k] = l.args
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
