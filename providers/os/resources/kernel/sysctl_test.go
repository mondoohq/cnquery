package kernel

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/providers/os/connection/mock"
)

func TestSysctlDebian(t *testing.T) {
	mock, err := mock.New("./testdata/debian.toml", nil)
	require.NoError(t, err)

	c, err := mock.RunCommand("/sbin/sysctl -a")
	require.NoError(t, err)

	entries, err := ParseSysctl(c.Stdout, "=")
	require.NoError(t, err)

	assert.Equal(t, 32, len(entries))
	assert.Equal(t, "10000", entries["net.ipv4.conf.all.igmpv2_unsolicited_report_interval"])
}

func TestSysctlMacos(t *testing.T) {
	mock, err := mock.New("./testdata/osx.toml", nil)
	require.NoError(t, err)

	c, err := mock.RunCommand("sysctl -a")
	require.NoError(t, err)

	entries, err := ParseSysctl(c.Stdout, ":")
	require.NoError(t, err)

	assert.Equal(t, 17, len(entries))
	assert.Equal(t, "1024", entries["net.inet6.ip6.neighborgcthresh"])
}

func TestSysctlFreebsd(t *testing.T) {
	mock, err := mock.New("./testdata/freebsd12.toml", nil)
	require.NoError(t, err)

	c, err := mock.RunCommand("sysctl -a")
	require.NoError(t, err)

	entries, err := ParseSysctl(c.Stdout, ":")
	require.NoError(t, err)

	assert.Equal(t, 20, len(entries))
	assert.Equal(t, "1", entries["security.bsd.unprivileged_mlock"])
}
