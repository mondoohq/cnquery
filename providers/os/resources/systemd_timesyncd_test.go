// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Timesyncd state assembly mirrors the production code paths so changes
// to the property names or NTPSynchronized semantics get caught.

func TestTimesyncd_PropertyParsing(t *testing.T) {
	// `timedatectl show --no-pager` properties.
	show := `Timezone=UTC
LocalRTC=no
CanNTP=yes
NTP=yes
NTPSynchronized=yes
TimeUSec=Mon 2026-05-25 01:30:00 UTC
RTCTimeUSec=Mon 2026-05-25 01:30:00 UTC
`
	props := parseSystemdShowOutput(show)
	assert.Equal(t, "yes", props["NTPSynchronized"])
	assert.Equal(t, "UTC", props["Timezone"])

	// `timedatectl show-timesync --no-pager` properties.
	timesync := `SystemNTPServers=time.cloudflare.com
FallbackNTPServers=0.fedora.pool.ntp.org 1.fedora.pool.ntp.org
ServerName=time.cloudflare.com
ServerAddress=1.1.1.1
RootDistanceMaxUSec=5s
PollIntervalMinUSec=32s
PollIntervalMaxUSec=34min 8s
PollIntervalUSec=256000000
LeapStatus=normal
`
	tprops := parseSystemdShowOutput(timesync)
	assert.Equal(t, "time.cloudflare.com", tprops["SystemNTPServers"])
	assert.Equal(t,
		"0.fedora.pool.ntp.org 1.fedora.pool.ntp.org",
		tprops["FallbackNTPServers"])
	assert.Equal(t, "time.cloudflare.com", tprops["ServerName"])
	assert.Equal(t, "1.1.1.1", tprops["ServerAddress"])
	assert.Equal(t, "256000000", tprops["PollIntervalUSec"])
	assert.Equal(t, "normal", tprops["LeapStatus"])
}

func TestTimesyncd_NTPSynchronizedNo(t *testing.T) {
	// Most common failure mode: timesyncd is up but hasn't reached a
	// server yet — NTPSynchronized=no rather than the property being
	// absent.
	show := `NTPSynchronized=no
`
	props := parseSystemdShowOutput(show)
	assert.Equal(t, "no", props["NTPSynchronized"])
	assert.False(t, props["NTPSynchronized"] == "yes")
}

func TestTimesyncd_NoServersConfigured(t *testing.T) {
	// When no NTP servers are configured, SystemNTPServers is the empty
	// string, and LinkNTPServers may also be absent.
	timesync := `SystemNTPServers=
FallbackNTPServers=
ServerName=
ServerAddress=
LeapStatus=
`
	props := parseSystemdShowOutput(timesync)
	assert.Equal(t, "", props["SystemNTPServers"])
	assert.Equal(t, "", props["FallbackNTPServers"])
}

func TestTimesyncd_StatusFallback(t *testing.T) {
	// systemd < 239 (e.g. v237 on Ubuntu 18.04) has no `timedatectl show`
	// verb, so `synchronized` falls back to parsing `timedatectl status`.
	status := `                      Local time: Wed 2026-06-17 06:34:18 CEST
                  Universal time: Wed 2026-06-17 04:34:18 UTC
                        RTC time: Wed 2026-06-17 04:34:18
                       Time zone: Europe/Berlin (CEST, +0200)
       System clock synchronized: yes
systemd-timesyncd.service active: yes
                 RTC in local TZ: no
`
	assert.True(t, parseTimedatectlStatusSynchronized(status))

	notSynced := `       System clock synchronized: no
systemd-timesyncd.service active: yes
`
	assert.False(t, parseTimedatectlStatusSynchronized(notSynced))
}

func TestTimesyncd_StatusFallbackEdgeCases(t *testing.T) {
	// Missing line -> not synchronized rather than a false positive.
	assert.False(t, parseTimedatectlStatusSynchronized(""))
	assert.False(t, parseTimedatectlStatusSynchronized("Time zone: UTC (UTC, +0000)\n"))

	// Case-insensitive value, and the "active" line must not be mistaken
	// for the synchronized line.
	assert.True(t, parseTimedatectlStatusSynchronized("System clock synchronized: Yes\n"))
	assert.False(t, parseTimedatectlStatusSynchronized("systemd-timesyncd.service active: yes\n"))
}
