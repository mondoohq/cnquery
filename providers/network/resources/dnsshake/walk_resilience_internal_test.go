// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package dnsshake

import (
	"net"
	"strconv"
	"sync"
	"testing"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// authoritativeNameservers walks up the labels until a name answers with NS
// records, and reads a query it could not get an answer to as that name not
// being a zone. It then climbs, and answers with the parent's nameservers.
//
// That is only a safe reading if a query that could have been answered was, and
// one UDP datagram to one resolver is not that. A lost reply at a delegated
// subzone silently returned the parent's nameservers, which serve referrals
// rather than the records the caller asked for. Since authoritativeRecords
// exists to read the TTLs a zone is configured with rather than a cache's
// countdown, that is a wrong answer, not a slow one.

// walkFixture serves a small zone tree, optionally dropping the first replies
// for one name so a caller sees a lost datagram before a real answer:
//
//   - example.test answers NS at its own name
//   - corp.example.test is delegated, so it answers NS too and is a zone of its
//     own, which is the name a climb would skip past
//   - every other name answers NOERROR with nothing
type walkFixture struct {
	mu        sync.Mutex
	dropName  string
	dropFirst int
	queries   int
	dropped   int
}

func (f *walkFixture) serve(t *testing.T) *dns.ClientConfig {
	t.Helper()

	delegations := map[string][]string{
		"example.test.":      {"ns1.example.test.", "ns2.example.test."},
		"corp.example.test.": {"ns1.corp.example.test."},
	}

	mux := dns.NewServeMux()
	mux.HandleFunc(".", func(w dns.ResponseWriter, r *dns.Msg) {
		q := r.Question[0]

		f.mu.Lock()
		f.queries++
		drop := q.Name == f.dropName && f.dropped < f.dropFirst
		if drop {
			f.dropped++
		}
		f.mu.Unlock()

		if drop {
			// No reply at all: the client waits and reports a timeout, which is
			// what a lost datagram looks like.
			return
		}

		m := new(dns.Msg)
		m.SetReply(r)
		m.Authoritative = true
		if q.Qtype == dns.TypeNS {
			for _, ns := range delegations[q.Name] {
				m.Answer = append(m.Answer, &dns.NS{
					Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeNS, Class: dns.ClassINET, Ttl: 300},
					Ns:  ns,
				})
			}
		}
		_ = w.WriteMsg(m)
	})

	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)

	srv := &dns.Server{PacketConn: pc, Handler: mux}
	go func() { _ = srv.ActivateAndServe() }()
	t.Cleanup(func() { _ = srv.Shutdown() })

	host, port, err := net.SplitHostPort(pc.LocalAddr().String())
	require.NoError(t, err)

	// Attempts is honoured by queryNS. Timeout is not, because queryDnsTypeAt
	// builds a dns.Client with the library default and never reads it.
	return &dns.ClientConfig{Servers: []string{host}, Port: port, Attempts: 2}
}

func (f *walkFixture) counts() (queries, dropped int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.queries, f.dropped
}

// resolveEachNameserver resolves any nameserver name to a distinct address and
// records what it was asked, which is how these tests see which zone the walk
// settled on.
func resolveEachNameserver(asked *[]string) func(string) ([]string, error) {
	addrs := map[string]string{
		"ns1.corp.example.test": "192.0.2.10",
		"ns1.example.test":      "192.0.2.1",
		"ns2.example.test":      "192.0.2.2",
	}
	return func(name string) ([]string, error) {
		*asked = append(*asked, name)
		if ip, ok := addrs[name]; ok {
			return []string{ip}, nil
		}
		return nil, &net.DNSError{Err: "no such host", Name: name, IsNotFound: true}
	}
}

// deadResolverConfig points at a port nothing listens on, so every query is
// lost rather than answered.
func deadResolverConfig(t *testing.T, servers, attempts int) *dns.ClientConfig {
	t.Helper()

	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	_, port, err := net.SplitHostPort(pc.LocalAddr().String())
	require.NoError(t, err)
	require.NoError(t, pc.Close())

	addrs := make([]string, servers)
	for i := range addrs {
		addrs[i] = "127.0.0.1"
	}
	return &dns.ClientConfig{Servers: addrs, Port: port, Attempts: attempts}
}

// TestAuthoritativeNameserversRetriesALostReply is the bug: one lost datagram at
// a delegated subzone used to hand back the parent's nameservers, and nothing
// said so.
func TestAuthoritativeNameserversRetriesALostReply(t *testing.T) {
	f := &walkFixture{dropName: "corp.example.test.", dropFirst: 1}

	var asked []string
	c := &DnsClient{
		config:     f.serve(t),
		fqdn:       "vpn.corp.example.test",
		lookupHost: resolveEachNameserver(&asked),
	}

	addrs, err := c.authoritativeNameservers()
	require.NoError(t, err)

	assert.Equal(t, []string{"192.0.2.10"}, addrs,
		"a retried query should settle on corp.example.test, not climb to its parent")
	assert.Equal(t, []string{"ns1.corp.example.test"}, asked,
		"the parent's nameservers should never have been reached")

	_, dropped := f.counts()
	assert.Equal(t, 1, dropped, "the fixture should have dropped exactly one reply")
}

// TestAuthoritativeNameserversFailsOverToTheNextResolver: resolv.conf commonly
// lists several nameservers, and asking only the first throws away the
// redundancy the host is configured for.
func TestAuthoritativeNameserversFailsOverToTheNextResolver(t *testing.T) {
	f := &walkFixture{}
	live := f.serve(t)

	var asked []string
	c := &DnsClient{
		config: &dns.ClientConfig{
			// Nothing listens on 127.0.0.2 at the fixture's port, so answering
			// at all requires moving on to the second resolver.
			Servers:  []string{"127.0.0.2", live.Servers[0]},
			Port:     live.Port,
			Attempts: 1,
		},
		fqdn:       "corp.example.test",
		lookupHost: resolveEachNameserver(&asked),
	}

	addrs, err := c.authoritativeNameservers()
	require.NoError(t, err, "an unreachable first resolver must fail over, not fall through the walk")
	assert.Equal(t, []string{"192.0.2.10"}, addrs)
}

// TestAuthoritativeNameserversStillClimbsPastAnUnresolvableDelegation is the
// leniency this backport keeps. A delegation whose nameservers do not resolve is
// not usable, so the walk continues to one that is. This is behaviour, not an
// accident, and the retry must not change it.
func TestAuthoritativeNameserversStillClimbsPastAnUnresolvableDelegation(t *testing.T) {
	f := &walkFixture{}

	var asked []string
	c := &DnsClient{
		config: f.serve(t),
		fqdn:   "vpn.corp.example.test",
		lookupHost: func(name string) ([]string, error) {
			asked = append(asked, name)
			if name == "ns1.corp.example.test" {
				return nil, &net.DNSError{Err: "no such host", Name: name, IsNotFound: true}
			}
			return resolveEachNameserver(new([]string))(name)
		},
	}

	addrs, err := c.authoritativeNameservers()
	require.NoError(t, err)
	assert.Equal(t, []string{"192.0.2.1", "192.0.2.2"}, addrs)
	assert.Equal(t, []string{"ns1.corp.example.test", "ns1.example.test", "ns2.example.test"}, asked,
		"the unresolvable delegation is tried first, then the walk continues to the parent")
}

// TestAuthoritativeNameserversEndsTheWalkOnAQueryFailure pins the behaviour
// change this backport makes to a shipped field. Before it, a lost reply at the
// queried name climbed to the parent and returned the parent's nameservers,
// which answer referrals rather than the records the caller wanted. It is an
// error now, and after queryNS's retry it takes a resolver that really cannot
// answer.
func TestAuthoritativeNameserversEndsTheWalkOnAQueryFailure(t *testing.T) {
	c := &DnsClient{config: deadResolverConfig(t, 2, 2), fqdn: "vpn.corp.example.test"}

	addrs, err := c.authoritativeNameservers()
	require.Error(t, err)
	assert.Empty(t, addrs)
	assert.Contains(t, err.Error(), "querying NS for",
		"the terminal error should carry the query that could not be answered")
}

func TestAuthoritativeNameserversRejectsRootStill(t *testing.T) {
	for _, fqdn := range []string{"", "."} {
		c := &DnsClient{config: deadResolverConfig(t, 1, 1), fqdn: fqdn}
		_, err := c.authoritativeNameservers()
		assert.Error(t, err, "fqdn %q must not walk to the root", fqdn)
	}
}

func TestQueryNSWithoutAResolverIsAnError(t *testing.T) {
	c := &DnsClient{config: &dns.ClientConfig{Port: "53"}, fqdn: "example.test"}

	_, err := c.queryNS("example.test")
	require.Error(t, err, "no configured resolver cannot silently mean no delegation")
}

func TestWalkFixturePortIsNumeric(t *testing.T) {
	// Guards the fixture itself: a non-numeric port would make every query fail
	// and every assertion above would be testing the failure path by accident.
	f := &walkFixture{}
	_, err := strconv.Atoi(f.serve(t).Port)
	require.NoError(t, err)
}
