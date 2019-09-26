package uptime_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"go.mondoo.io/mondoo/lumi/resources/uptime"
)

func TestLinuxUptime(t *testing.T) {
	// uptime
	data := " 21:00:04 up 1 day,  5:29,  0 users,  load average: 0.00, 0.13, 0.22"
	duration, err := uptime.ParseUnixUptime(data)
	assert.Nil(t, err)
	assert.Equal(t, &uptime.UnixUptime{
		Time:               106140000000000,
		Users:              0,
		LoadOneMinute:      float64(0.0),
		LoadFiveMinutes:    float64(0.13),
		LoadFifteenMinutes: float64(0.22),
	}, duration)
	assert.Equal(t, "29h29m0s", time.Duration(duration.Time).String())

	data = "23:41:57 up 16 min,  0 users,  load average: 0.06, 0.02, 0.00"
	duration, err = uptime.ParseUnixUptime(data)
	assert.Nil(t, err)
	assert.Equal(t, &uptime.UnixUptime{
		Time:               960000000000,
		Users:              0,
		LoadOneMinute:      float64(0.06),
		LoadFiveMinutes:    float64(0.02),
		LoadFifteenMinutes: float64(0.00),
	}, duration)
	assert.Equal(t, "16m0s", time.Duration(duration.Time).String())
}

func TestMacOSUptime(t *testing.T) {
	// uptime
	data := "23:04  up 24 days, 13:07, 9 users, load averages: 4.81 5.21 5.15"
	duration, err := uptime.ParseUnixUptime(data)
	assert.Nil(t, err)
	assert.Equal(t, &uptime.UnixUptime{
		Time:               2120820000000000,
		Users:              9,
		LoadOneMinute:      float64(4.81),
		LoadFiveMinutes:    float64(5.21),
		LoadFifteenMinutes: float64(5.15),
	}, duration)
	assert.Equal(t, "589h7m0s", time.Duration(duration.Time).String())
}

// for windows wmic path Win32_OperatingSystem get LastBootUpTime
// https://www.windowscentral.com/how-check-your-computer-uptime-windows-10
