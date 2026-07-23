// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"errors"
	"testing"

	"github.com/go-routeros/routeros/v3"
	"github.com/go-routeros/routeros/v3/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsUnknownCommandErr(t *testing.T) {
	// RouterOS returns this trap when a menu/command isn't available on the
	// device (e.g. /interface/wifi without the wifi package).
	assert.True(t, isUnknownCommandErr(errors.New("no such command prefix")))
	assert.True(t, isUnknownCommandErr(errors.New("from RouterOS: No Such Command")))

	assert.False(t, isUnknownCommandErr(nil))
	assert.False(t, isUnknownCommandErr(errors.New("connection refused")))
	assert.False(t, isUnknownCommandErr(errors.New("permission denied")))
}

// fakeClient is a rosClient that serves canned replies/errors per command and
// counts how many times Run is invoked, so tests can assert cache behavior.
type fakeClient struct {
	replies map[string]*routeros.Reply
	errs    map[string]error
	runs    map[string]int
}

func newFakeClient() *fakeClient {
	return &fakeClient{
		replies: map[string]*routeros.Reply{},
		errs:    map[string]error{},
		runs:    map[string]int{},
	}
}

func (f *fakeClient) Run(sentences ...string) (*routeros.Reply, error) {
	cmd := sentences[0]
	f.runs[cmd]++
	if err, ok := f.errs[cmd]; ok {
		return nil, err
	}
	return f.replies[cmd], nil
}

func (f *fakeClient) Close() error { return nil }

// reply builds a routeros.Reply whose sentences carry the given attribute maps.
func reply(rows ...map[string]string) *routeros.Reply {
	r := &routeros.Reply{}
	for _, row := range rows {
		r.Re = append(r.Re, &proto.Sentence{Map: row})
	}
	return r
}

func TestPrintCachesPerMenu(t *testing.T) {
	fake := newFakeClient()
	fake.replies["/interface/print"] = reply(
		map[string]string{"name": "ether1"},
		map[string]string{"name": "ether2"},
	)
	conn := &MikrotikConnection{client: fake}

	first, err := conn.Print("/interface")
	require.NoError(t, err)
	assert.Len(t, first, 2)
	assert.Equal(t, "ether1", first[0]["name"])

	// second call for the same menu must hit the cache, not the device
	second, err := conn.Print("/interface")
	require.NoError(t, err)
	assert.Equal(t, first, second)
	assert.Equal(t, 1, fake.runs["/interface/print"], "menu should only be fetched once")
}

func TestPrintCopiesRowsOutOfLibraryState(t *testing.T) {
	fake := newFakeClient()
	fake.replies["/ip/pool/print"] = reply(map[string]string{"name": "dhcp-pool"})
	conn := &MikrotikConnection{client: fake}

	rows, err := conn.Print("/ip/pool")
	require.NoError(t, err)
	// a caller mutating the returned map must not corrupt the cached copy
	rows[0]["name"] = "tampered"

	again, err := conn.Print("/ip/pool")
	require.NoError(t, err)
	assert.Equal(t, "dhcp-pool", again[0]["name"], "cache must be isolated from caller mutation")
}

func TestPrintPropagatesError(t *testing.T) {
	fake := newFakeClient()
	fake.errs["/user/print"] = errors.New("permission denied")
	conn := &MikrotikConnection{client: fake}

	_, err := conn.Print("/user")
	assert.Error(t, err)
	// a failed fetch must not be cached, so a retry re-issues the command
	_, _ = conn.Print("/user")
	assert.Equal(t, 2, fake.runs["/user/print"], "errors must not be cached")
}

func TestPrintOptionalDegradesOnUnknownCommand(t *testing.T) {
	fake := newFakeClient()
	fake.errs["/interface/wifi/print"] = errors.New("no such command prefix")
	conn := &MikrotikConnection{client: fake}

	rows, err := conn.PrintOptional("/interface/wifi")
	require.NoError(t, err, "an unsupported menu must degrade to empty, not error")
	assert.Empty(t, rows)
}

func TestPrintOptionalPropagatesRealError(t *testing.T) {
	fake := newFakeClient()
	fake.errs["/interface/wifi/print"] = errors.New("permission denied")
	conn := &MikrotikConnection{client: fake}

	_, err := conn.PrintOptional("/interface/wifi")
	assert.Error(t, err, "a genuine failure must still surface")
}

func TestPrintOne(t *testing.T) {
	fake := newFakeClient()
	fake.replies["/system/identity/print"] = reply(map[string]string{"name": "router-a"})
	fake.replies["/snmp/print"] = reply() // no records
	conn := &MikrotikConnection{client: fake}

	row, err := conn.PrintOne("/system/identity")
	require.NoError(t, err)
	assert.Equal(t, "router-a", row["name"])

	empty, err := conn.PrintOne("/snmp")
	require.NoError(t, err)
	assert.Empty(t, empty, "a menu with no records yields an empty map, not nil access")
}
