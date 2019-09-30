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
	assert.Equal(t, &uptime.UnixUptimeResult{
		Duration:           106140000000000,
		Users:              0,
		LoadOneMinute:      float64(0.0),
		LoadFiveMinutes:    float64(0.13),
		LoadFifteenMinutes: float64(0.22),
	}, duration)
	assert.Equal(t, "29h29m0s", time.Duration(duration.Duration).String())

	data = "23:41:57 up 16 min,  0 users,  load average: 0.06, 0.02, 0.00"
	duration, err = uptime.ParseUnixUptime(data)
	assert.Nil(t, err)
	assert.Equal(t, &uptime.UnixUptimeResult{
		Duration:           960000000000,
		Users:              0,
		LoadOneMinute:      float64(0.06),
		LoadFiveMinutes:    float64(0.02),
		LoadFifteenMinutes: float64(0.00),
	}, duration)
	assert.Equal(t, "16m0s", time.Duration(duration.Duration).String())
}

func TestAlpineUptime(t *testing.T) {
	// alpine
	data := " 08:45:41 up 22 min,  load average: 0.19, 0.15, 0.09"
	duration, err := uptime.ParseUnixUptime(data)
	assert.Nil(t, err)
	assert.Equal(t, &uptime.UnixUptimeResult{
		Duration:           1320000000000,
		Users:              0,
		LoadOneMinute:      float64(0.19),
		LoadFiveMinutes:    float64(0.15),
		LoadFifteenMinutes: float64(0.09),
	}, duration)
	assert.Equal(t, "22m0s", time.Duration(duration.Duration).String())
}

func TestDebianUptime(t *testing.T) {
	// debian
	data := " 08:45:19 up 21 min,  0 users,  load average: 0.10, 0.13, 0.09"
	duration, err := uptime.ParseUnixUptime(data)
	assert.Nil(t, err)
	assert.Equal(t, &uptime.UnixUptimeResult{
		Duration:           1260000000000,
		Users:              0,
		LoadOneMinute:      float64(0.10),
		LoadFiveMinutes:    float64(0.13),
		LoadFifteenMinutes: float64(0.09),
	}, duration)
	assert.Equal(t, "21m0s", time.Duration(duration.Duration).String())
}

func TestRhelUptime(t *testing.T) {
	// rehl
	data := " 12:27:22 up 8 min,  1 user,  load average: 0.01, 0.02, 0.00"
	duration, err := uptime.ParseUnixUptime(data)
	assert.Nil(t, err)
	assert.Equal(t, &uptime.UnixUptimeResult{
		Duration:           480000000000,
		Users:              1,
		LoadOneMinute:      float64(0.01),
		LoadFiveMinutes:    float64(0.02),
		LoadFifteenMinutes: float64(0.00),
	}, duration)
	assert.Equal(t, "8m0s", time.Duration(duration.Duration).String())

	data = "13:24:35 up  1:05,  2 users,  load average: 0.00, 0.00, 0.00"
	duration, err = uptime.ParseUnixUptime(data)
	assert.Nil(t, err)
	assert.Equal(t, &uptime.UnixUptimeResult{
		Duration:           3900000000000,
		Users:              2,
		LoadOneMinute:      float64(0.00),
		LoadFiveMinutes:    float64(0.00),
		LoadFifteenMinutes: float64(0.00),
	}, duration)
	assert.Equal(t, "1h5m0s", time.Duration(duration.Duration).String())
}

func TestBusyboxUptime(t *testing.T) {
	data := " 08:56:57 up 33 min,  0 users,  load average: 0.09, 0.09, 0.08"
	duration, err := uptime.ParseUnixUptime(data)
	assert.Nil(t, err)
	assert.Equal(t, &uptime.UnixUptimeResult{
		Duration:           1980000000000,
		Users:              0,
		LoadOneMinute:      float64(0.09),
		LoadFiveMinutes:    float64(0.09),
		LoadFifteenMinutes: float64(0.08),
	}, duration)
	assert.Equal(t, "33m0s", time.Duration(duration.Duration).String())
}

func TestMacOSUptime(t *testing.T) {
	// uptime
	data := "23:04  up 24 days, 13:07, 9 users, load averages: 4.81 5.21 5.15"
	duration, err := uptime.ParseUnixUptime(data)
	assert.Nil(t, err)
	assert.Equal(t, &uptime.UnixUptimeResult{
		Duration:           2120820000000000,
		Users:              9,
		LoadOneMinute:      float64(4.81),
		LoadFiveMinutes:    float64(5.21),
		LoadFifteenMinutes: float64(5.15),
	}, duration)
	assert.Equal(t, "589h7m0s", time.Duration(duration.Duration).String())

	data = "10:52  up 38 mins, 9 users, load averages: 2.27 2.54 3.72"
	duration, err = uptime.ParseUnixUptime(data)
	assert.Nil(t, err)
	assert.Equal(t, &uptime.UnixUptimeResult{
		Duration:           2280000000000,
		Users:              9,
		LoadOneMinute:      float64(2.27),
		LoadFiveMinutes:    float64(2.54),
		LoadFifteenMinutes: float64(3.72),
	}, duration)
	assert.Equal(t, "38m0s", time.Duration(duration.Duration).String())
}
