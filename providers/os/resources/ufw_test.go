// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseUfwKeyValue(t *testing.T) {
	input := `# /etc/ufw/ufw.conf
#

# Set to yes to start on boot.
ENABLED=yes

# Please use the 'ufw' command to set the loglevel.
LOGLEVEL=low
`
	result := parseUfwKeyValue(input)
	assert.Equal(t, "yes", result["ENABLED"])
	assert.Equal(t, "low", result["LOGLEVEL"])
}

func TestParseUfwKeyValueQuoted(t *testing.T) {
	input := `IPV6=yes
DEFAULT_INPUT_POLICY="DROP"
DEFAULT_OUTPUT_POLICY="ACCEPT"
DEFAULT_FORWARD_POLICY="DROP"
DEFAULT_APPLICATION_POLICY="SKIP"
`
	result := parseUfwKeyValue(input)
	assert.Equal(t, "yes", result["IPV6"])
	assert.Equal(t, "DROP", result["DEFAULT_INPUT_POLICY"])
	assert.Equal(t, "ACCEPT", result["DEFAULT_OUTPUT_POLICY"])
	assert.Equal(t, "DROP", result["DEFAULT_FORWARD_POLICY"])
	assert.Equal(t, "SKIP", result["DEFAULT_APPLICATION_POLICY"])
}

func TestParseUfwKeyValueDisabled(t *testing.T) {
	input := `ENABLED=no
LOGLEVEL=low
`
	result := parseUfwKeyValue(input)
	assert.Equal(t, "no", result["ENABLED"])
}

func TestParseUfwTuples(t *testing.T) {
	input := `*filter
:ufw-user-input - [0:0]
:ufw-user-output - [0:0]
:ufw-user-forward - [0:0]
:ufw-user-limit - [0:0]
:ufw-user-limit-accept - [0:0]
### RULES ###

### tuple ### allow tcp 22 0.0.0.0/0 any 0.0.0.0/0 in
-A ufw-user-input -p tcp --dport 22 -j ACCEPT

### tuple ### deny tcp 3306 0.0.0.0/0 any 0.0.0.0/0 in
-A ufw-user-input -p tcp --dport 3306 -j DROP

### tuple ### allow any 80 0.0.0.0/0 any 0.0.0.0/0 in
-A ufw-user-input -p tcp --dport 80 -j ACCEPT
-A ufw-user-input -p udp --dport 80 -j ACCEPT

### tuple ### limit tcp 22 0.0.0.0/0 any 0.0.0.0/0 in
-A ufw-user-input -p tcp --dport 22 -j ufw-user-limit
-A ufw-user-input -p tcp --dport 22 -j ufw-user-limit-accept

### tuple ### allow tcp 443 0.0.0.0/0 any 10.0.0.0/8 in_eth0
-A ufw-user-input -i eth0 -p tcp --dport 443 -s 10.0.0.0/8 -j ACCEPT

### tuple ### reject tcp 25 0.0.0.0/0 any 0.0.0.0/0 in
-A ufw-user-input -p tcp --dport 25 -j REJECT

### tuple ### allow udp 53 0.0.0.0/0 any 0.0.0.0/0 out
-A ufw-user-output -p udp --dport 53 -j ACCEPT

### tuple ### allow_log tcp 8080 0.0.0.0/0 any 0.0.0.0/0 in
-A ufw-user-logging-input -p tcp --dport 8080 -j LOG
-A ufw-user-input -p tcp --dport 8080 -j ACCEPT

-A ufw-user-limit -m limit --limit 3/minute -j LOG --log-prefix "[UFW LIMIT BLOCK] "
-A ufw-user-limit -j REJECT
-A ufw-user-limit-accept -j ACCEPT
COMMIT
`
	rules := parseUfwTuples(input)
	assert.Len(t, rules, 8)

	// allow tcp 22 in
	assert.Equal(t, "ALLOW", rules[0].action)
	assert.Equal(t, "tcp", rules[0].protocol)
	assert.Equal(t, "22", rules[0].port)
	assert.Equal(t, "0.0.0.0/0", rules[0].to)
	assert.Equal(t, "0.0.0.0/0", rules[0].from)
	assert.Equal(t, "IN", rules[0].direction)
	assert.Equal(t, "", rules[0].iface)

	// deny tcp 3306 in
	assert.Equal(t, "DENY", rules[1].action)
	assert.Equal(t, "tcp", rules[1].protocol)
	assert.Equal(t, "3306", rules[1].port)
	assert.Equal(t, "IN", rules[1].direction)

	// allow any 80 in
	assert.Equal(t, "ALLOW", rules[2].action)
	assert.Equal(t, "any", rules[2].protocol)
	assert.Equal(t, "80", rules[2].port)

	// limit tcp 22 in
	assert.Equal(t, "LIMIT", rules[3].action)
	assert.Equal(t, "tcp", rules[3].protocol)
	assert.Equal(t, "22", rules[3].port)

	// allow tcp 443 in_eth0
	assert.Equal(t, "ALLOW", rules[4].action)
	assert.Equal(t, "tcp", rules[4].protocol)
	assert.Equal(t, "443", rules[4].port)
	assert.Equal(t, "10.0.0.0/8", rules[4].from)
	assert.Equal(t, "IN", rules[4].direction)
	assert.Equal(t, "eth0", rules[4].iface)

	// reject tcp 25 in
	assert.Equal(t, "REJECT", rules[5].action)

	// allow udp 53 out
	assert.Equal(t, "ALLOW", rules[6].action)
	assert.Equal(t, "udp", rules[6].protocol)
	assert.Equal(t, "53", rules[6].port)
	assert.Equal(t, "OUT", rules[6].direction)

	// allow_log tcp 8080 in (log suffix stripped)
	assert.Equal(t, "ALLOW", rules[7].action)
	assert.Equal(t, "tcp", rules[7].protocol)
	assert.Equal(t, "8080", rules[7].port)
}

func TestParseUfwTuplesEmpty(t *testing.T) {
	// Real-world empty rules file from user's system
	input := `*filter
:ufw-user-input - [0:0]
:ufw-user-output - [0:0]
:ufw-user-forward - [0:0]
:ufw-user-limit - [0:0]
:ufw-user-limit-accept - [0:0]
### RULES ###
-A ufw-user-limit -m limit --limit 3/minute -j LOG --log-prefix "[UFW LIMIT BLOCK] "
-A ufw-user-limit -j REJECT
-A ufw-user-limit-accept -j ACCEPT
COMMIT
`
	rules := parseUfwTuples(input)
	assert.Empty(t, rules)
}

func TestParseUfwTuplesWithApps(t *testing.T) {
	// Tuple with application names (dapp/sapp fields before direction)
	input := `### tuple ### allow tcp 80 0.0.0.0/0 any 0.0.0.0/0 Apache%20Full - in
-A ufw-user-input -p tcp --dport 80 -j ACCEPT
`
	rules := parseUfwTuples(input)
	assert.Len(t, rules, 1)
	assert.Equal(t, "ALLOW", rules[0].action)
	assert.Equal(t, "tcp", rules[0].protocol)
	assert.Equal(t, "80", rules[0].port)
	assert.Equal(t, "IN", rules[0].direction)
}

func TestParseUfwTuplesLogAll(t *testing.T) {
	input := `### tuple ### deny_log-all tcp 9999 0.0.0.0/0 any 0.0.0.0/0 in
-A ufw-user-input -p tcp --dport 9999 -j DROP
`
	rules := parseUfwTuples(input)
	assert.Len(t, rules, 1)
	assert.Equal(t, "DENY", rules[0].action)
}
