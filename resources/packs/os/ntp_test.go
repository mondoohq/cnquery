package os_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResource_NtpConf(t *testing.T) {
	t.Run("ntp.conf settings", func(t *testing.T) {
		res := x.TestQuery(t, "ntp.conf.settings")
		assert.NotEmpty(t, res)
	})

	t.Run("ntp.conf servers", func(t *testing.T) {
		res := x.TestQuery(t, "ntp.conf.servers")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, []interface{}{
			"127.127.1.0", "66.187.224.4", "18.26.4.105", "128.249.1.10",
		}, res[0].Data.Value)
	})

	t.Run("ntp.conf restrict", func(t *testing.T) {
		res := x.TestQuery(t, "ntp.conf.restrict")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, []interface{}{
			"default ignore",
			"66.187.224.4 mask 255.255.255.255 nomodify notrap noquery",
			"18.26.4.105 mask 255.255.255.255 nomodify notrap noquery",
			"128.249.1.10 mask 255.255.255.255 nomodify notrap noquery",
		}, res[0].Data.Value)
	})

	t.Run("ntp.conf fudge", func(t *testing.T) {
		res := x.TestQuery(t, "ntp.conf.fudge")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, []interface{}{
			"127.127.1.0 stratum 10",
		}, res[0].Data.Value)
	})
}
