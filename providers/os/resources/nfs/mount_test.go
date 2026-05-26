// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package nfs

import (
	"reflect"
	"testing"
)

func TestIsNFSFsType(t *testing.T) {
	cases := []struct {
		fstype string
		want   bool
	}{
		{"nfs", true},
		{"NFS", true},
		{"nfs3", true},
		{"nfs4", true},
		{"nfsv4", true},
		{"nfs4.1", true},
		{"nfs4.2", true},
		{"ext4", false},
		{"", false},
		{"nfsd", false}, // nfsd is a daemon, not a mountable fs
		{"smbfs", false},
		{"nfsv", false}, // "nfsv" with no digits is rejected
	}
	for _, c := range cases {
		if got := IsNFSFsType(c.fstype); got != c.want {
			t.Errorf("IsNFSFsType(%q) = %v, want %v", c.fstype, got, c.want)
		}
	}
}

func TestSplitNFSDevice(t *testing.T) {
	cases := []struct {
		device               string
		wantServer, wantPath string
	}{
		{"server:/export/data", "server", "/export/data"},
		{"192.168.1.10:/srv", "192.168.1.10", "/srv"},
		{"[fe80::1]:/v6/path", "[fe80::1]", "/v6/path"},
		{"[2001:db8::1]:/path/with:colon", "[2001:db8::1]", "/path/with:colon"},
		{":/relative", "", "/relative"},
		{"nocolon", "nocolon", ""},
		{"", "", ""},
	}
	for _, c := range cases {
		gotServer, gotPath := splitNFSDevice(c.device)
		if gotServer != c.wantServer || gotPath != c.wantPath {
			t.Errorf("splitNFSDevice(%q) = (%q, %q), want (%q, %q)",
				c.device, gotServer, gotPath, c.wantServer, c.wantPath)
		}
	}
}

func TestNFSVersion(t *testing.T) {
	cases := []struct {
		name    string
		options map[string]string
		fstype  string
		want    string
	}{
		{"vers from option", map[string]string{"vers": "4.1"}, "nfs", "4.1"},
		{"nfsvers wins ahead of fallback", map[string]string{"nfsvers": "3"}, "nfs4", "3"},
		{"version key honoured", map[string]string{"version": "4"}, "nfs", "4"},
		{"fallback from fstype nfs4", map[string]string{}, "nfs4", "4"},
		{"fallback from fstype nfsv4", map[string]string{}, "nfsv4", "4"},
		{"plain nfs no version", map[string]string{}, "nfs", ""},
		{"non-nfs fstype no version", map[string]string{}, "ext4", ""},
		{"empty option falls through", map[string]string{"vers": ""}, "nfs4", "4"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := nfsVersion(c.options, c.fstype); got != c.want {
				t.Errorf("nfsVersion = %q, want %q", got, c.want)
			}
		})
	}
}

func TestBuildMountInfo(t *testing.T) {
	cases := []struct {
		name    string
		device  string
		mp      string
		fstype  string
		options map[string]string
		want    MountInfo
	}{
		{
			name:    "linux nfs4 with krb5",
			device:  "filer.example.com:/exports/home",
			mp:      "/mnt/home",
			fstype:  "nfs4",
			options: map[string]string{"rw": "", "vers": "4.1", "sec": "krb5p", "hard": "", "rsize": "1048576"},
			want: MountInfo{
				Device:     "filer.example.com:/exports/home",
				MountPoint: "/mnt/home",
				Server:     "filer.example.com",
				RemotePath: "/exports/home",
				Version:    "4.1",
				Security:   "krb5p",
				HardMount:  true,
				ReadOnly:   false,
				Options:    []string{"hard", "rsize=1048576", "rw", "sec=krb5p", "vers=4.1"},
			},
		},
		{
			name:    "soft mount marks hardMount false",
			device:  "10.0.0.5:/data",
			mp:      "/mnt/data",
			fstype:  "nfs",
			options: map[string]string{"soft": "", "ro": "", "vers": "3"},
			want: MountInfo{
				Device:     "10.0.0.5:/data",
				MountPoint: "/mnt/data",
				Server:     "10.0.0.5",
				RemotePath: "/data",
				Version:    "3",
				Security:   "",
				HardMount:  false,
				ReadOnly:   true,
				Options:    []string{"ro", "soft", "vers=3"},
			},
		},
		{
			name:    "fstype-based version fallback",
			device:  "nas:/share",
			mp:      "/mnt/nas",
			fstype:  "nfs4",
			options: map[string]string{"rw": ""},
			want: MountInfo{
				Device:     "nas:/share",
				MountPoint: "/mnt/nas",
				Server:     "nas",
				RemotePath: "/share",
				Version:    "4",
				Security:   "",
				HardMount:  true,
				ReadOnly:   false,
				Options:    []string{"rw"},
			},
		},
		{
			name:    "ipv6 server with brackets",
			device:  "[fe80::1]:/v6",
			mp:      "/mnt/v6",
			fstype:  "nfs",
			options: nil,
			want: MountInfo{
				Device:     "[fe80::1]:/v6",
				MountPoint: "/mnt/v6",
				Server:     "[fe80::1]",
				RemotePath: "/v6",
				Version:    "",
				Security:   "",
				HardMount:  true,
				ReadOnly:   false,
				Options:    nil,
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := BuildMountInfo(c.device, c.mp, c.fstype, c.options)
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("BuildMountInfo mismatch:\n got:  %#v\n want: %#v", got, c.want)
			}
		})
	}
}

func TestFlattenOptions(t *testing.T) {
	got := flattenOptions(map[string]string{
		"rw":   "",
		"vers": "4.1",
		"sec":  "krb5",
		"hard": "",
	})
	want := []string{"hard", "rw", "sec=krb5", "vers=4.1"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("flattenOptions = %#v, want %#v", got, want)
	}

	if got := flattenOptions(nil); got != nil {
		t.Errorf("flattenOptions(nil) = %#v, want nil", got)
	}
}
