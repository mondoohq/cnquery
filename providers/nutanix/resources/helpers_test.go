// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"
	"time"

	clustercommon "github.com/nutanix/ntnx-api-golang-clients/clustermgmt-go-client/v4/models/common/v1/config"
	netcommon "github.com/nutanix/ntnx-api-golang-clients/networking-go-client/v4/models/common/v1/config"
	netconfig "github.com/nutanix/ntnx-api-golang-clients/networking-go-client/v4/models/networking/v4/config"
	vmcommon "github.com/nutanix/ntnx-api-golang-clients/vmm-go-client/v4/models/common/v1/config"
)

func sp(s string) *string { return &s }
func ip(i int) *int       { return &i }
func i64(i int64) *int64  { return &i }
func bp(b bool) *bool     { return &b }

func TestDerefInt64(t *testing.T) {
	if got := derefInt64(nil); got != 0 {
		t.Errorf("derefInt64(nil) = %d, want 0", got)
	}
	if got := derefInt64(i64(42)); got != 42 {
		t.Errorf("derefInt64(42) = %d, want 42", got)
	}
}

func TestDerefInt(t *testing.T) {
	if got := derefInt(nil); got != 0 {
		t.Errorf("derefInt(nil) = %d, want 0", got)
	}
	if got := derefInt(ip(7)); got != 7 {
		t.Errorf("derefInt(7) = %d, want 7", got)
	}
}

func TestDerefBool(t *testing.T) {
	if derefBool(nil) {
		t.Error("derefBool(nil) = true, want false")
	}
	if !derefBool(bp(true)) {
		t.Error("derefBool(true) = false, want true")
	}
}

func TestUsecsToTime(t *testing.T) {
	if got := usecsToTime(nil); got != nil {
		t.Errorf("usecsToTime(nil) = %v, want nil", got)
	}
	if got := usecsToTime(i64(0)); got != nil {
		t.Errorf("usecsToTime(0) = %v, want nil", got)
	}
	want := time.UnixMicro(1_700_000_000_000_000).UTC()
	got := usecsToTime(i64(1_700_000_000_000_000))
	if got == nil || !got.Equal(want) {
		t.Errorf("usecsToTime = %v, want %v", got, want)
	}
}

func TestClusterIPToString(t *testing.T) {
	if got := clusterIPToString(nil); got != "" {
		t.Errorf("clusterIPToString(nil) = %q, want empty", got)
	}
	ipv4 := &clustercommon.IPAddress{Ipv4: &clustercommon.IPv4Address{Value: sp("10.0.0.5")}}
	if got := clusterIPToString(ipv4); got != "10.0.0.5" {
		t.Errorf("clusterIPToString(ipv4) = %q, want 10.0.0.5", got)
	}
	ipv6 := &clustercommon.IPAddress{Ipv6: &clustercommon.IPv6Address{Value: sp("2001:db8::1")}}
	if got := clusterIPToString(ipv6); got != "2001:db8::1" {
		t.Errorf("clusterIPToString(ipv6) = %q, want 2001:db8::1", got)
	}
}

func TestClusterIPOrFqdnToString(t *testing.T) {
	ipv4 := &clustercommon.IPAddressOrFQDN{Ipv4: &clustercommon.IPv4Address{Value: sp("8.8.8.8")}}
	if got := clusterIPOrFqdnToString(ipv4); got != "8.8.8.8" {
		t.Errorf("got %q, want 8.8.8.8", got)
	}
	fqdn := &clustercommon.IPAddressOrFQDN{Fqdn: &clustercommon.FQDN{Value: sp("ns.example.com")}}
	if got := clusterIPOrFqdnToString(fqdn); got != "ns.example.com" {
		t.Errorf("got %q, want ns.example.com", got)
	}
	if got := clusterIPOrFqdnToString(nil); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestVmIPv4ToString(t *testing.T) {
	if got := vmIPv4ToString(nil); got != "" {
		t.Errorf("got %q, want empty", got)
	}
	if got := vmIPv4ToString(&vmcommon.IPv4Address{Value: sp("192.168.1.10")}); got != "192.168.1.10" {
		t.Errorf("got %q, want 192.168.1.10", got)
	}
}

func TestNetIPToString(t *testing.T) {
	if got := netIPToString(nil); got != "" {
		t.Errorf("got %q, want empty", got)
	}
	ipv4 := &netcommon.IPAddress{Ipv4: &netcommon.IPv4Address{Value: sp("172.16.0.1")}}
	if got := netIPToString(ipv4); got != "172.16.0.1" {
		t.Errorf("got %q, want 172.16.0.1", got)
	}
}

func TestIPSubnetToString(t *testing.T) {
	if got := ipSubnetToString(nil); got != "" {
		t.Errorf("got %q, want empty", got)
	}
	s := &netconfig.IPSubnet{Ipv4: &netconfig.IPv4Subnet{
		Ip:           &netcommon.IPv4Address{Value: sp("10.0.0.0")},
		PrefixLength: ip(24),
	}}
	if got := ipSubnetToString(s); got != "10.0.0.0/24" {
		t.Errorf("got %q, want 10.0.0.0/24", got)
	}
}
