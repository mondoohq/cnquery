// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testApparmorJSON = `{"version": "2", "profiles": {"/usr/bin/man": "enforce", "/usr/lib/NetworkManager/nm-dhcp-client.action": "enforce", "/usr/lib/connman/scripts/dhclient-script": "enforce", "/usr/sbin/chronyd": "enforce", "/usr/{lib/NetworkManager,libexec}/nm-dhcp-helper": "enforce", "/{,usr/}sbin/dhclient": "enforce", "docker-default": "enforce", "libvirtd": "enforce", "libvirtd//qemu_bridge_helper": "enforce", "lsb_release": "enforce", "man_filter": "enforce", "man_groff": "enforce", "nvidia_modprobe": "enforce", "nvidia_modprobe//kmod": "enforce", "virt-aa-helper": "enforce", "Xorg": "complain", "plasmashell": "complain", "plasmashell//QtWebEngineProcess": "complain", "sbuild": "complain", "sbuild-abort": "complain", "sbuild-adduser": "complain", "sbuild-apt": "complain", "sbuild-checkpackages": "complain", "sbuild-clean": "complain", "sbuild-createchroot": "complain", "sbuild-destroychroot": "complain", "sbuild-distupgrade": "complain", "sbuild-hold": "complain", "sbuild-shell": "complain", "sbuild-unhold": "complain", "sbuild-update": "complain", "sbuild-upgrade": "complain", "transmission-cli": "complain", "transmission-daemon": "complain", "transmission-gtk": "complain", "transmission-qt": "complain", "unix-chkpwd": "complain", "unprivileged_userns": "complain", "1password": "unconfined", "Discord": "unconfined", "MongoDB Compass": "unconfined", "QtWebEngineProcess": "unconfined", "balena-etcher": "unconfined", "brave": "unconfined", "buildah": "unconfined", "busybox": "unconfined", "cam": "unconfined", "ch-checkns": "unconfined", "ch-run": "unconfined", "chrome": "unconfined", "chromium": "unconfined", "crun": "unconfined", "devhelp": "unconfined", "element-desktop": "unconfined", "epiphany": "unconfined", "evolution": "unconfined", "firefox": "unconfined", "flatpak": "unconfined", "foliate": "unconfined", "geary": "unconfined", "github-desktop": "unconfined", "goldendict": "unconfined", "ipa_verify": "unconfined", "kchmviewer": "unconfined", "keybase": "unconfined", "lc-compliance": "unconfined", "libcamerify": "unconfined", "linux-sandbox": "unconfined", "loupe": "unconfined", "lxc-attach": "unconfined", "lxc-create": "unconfined", "lxc-destroy": "unconfined", "lxc-execute": "unconfined", "lxc-stop": "unconfined", "lxc-unshare": "unconfined", "lxc-usernsexec": "unconfined", "mmdebstrap": "unconfined", "msedge": "unconfined", "nautilus": "unconfined", "notepadqq": "unconfined", "obsidian": "unconfined", "opam": "unconfined", "opera": "unconfined", "pageedit": "unconfined", "polypane": "unconfined", "privacybrowser": "unconfined", "qcam": "unconfined", "qmapshack": "unconfined", "qutebrowser": "unconfined", "rootlesskit": "unconfined", "rpm": "unconfined", "rssguard": "unconfined", "runc": "unconfined", "scide": "unconfined", "signal-desktop": "unconfined", "slack": "unconfined", "slirp4netns": "unconfined", "steam": "unconfined", "stress-ng": "unconfined", "surfshark": "unconfined", "systemd-coredump": "unconfined", "toybox": "unconfined", "trinity": "unconfined", "tup": "unconfined", "tuxedo-control-center": "unconfined", "userbindmount": "unconfined", "uwsgi-core": "unconfined", "vdens": "unconfined", "virtiofsd": "unconfined", "vivaldi-bin": "unconfined", "vpnns": "unconfined", "vscode": "unconfined", "wike": "unconfined", "wpcom": "unconfined"}, "processes": {"/portainer": [{"profile": "docker-default", "pid": "1607926", "status": "enforce"}], "/usr/sbin/chronyd": [{"profile": "/usr/sbin/chronyd", "pid": "1858", "status": "enforce"}, {"profile": "/usr/sbin/chronyd", "pid": "1861", "status": "enforce"}]}}`

func TestParseApparmorStatus(t *testing.T) {
	var status apparmorStatus
	err := json.Unmarshal([]byte(testApparmorJSON), &status)
	require.NoError(t, err)

	assert.Equal(t, "2", status.Version)
	assert.Len(t, status.Profiles, 114)
	assert.Equal(t, "enforce", status.Profiles["/usr/bin/man"])
	assert.Equal(t, "complain", status.Profiles["Xorg"])
	assert.Equal(t, "unconfined", status.Profiles["firefox"])

	assert.Len(t, status.Processes, 2)

	portainer := status.Processes["/portainer"]
	require.Len(t, portainer, 1)
	assert.Equal(t, "docker-default", portainer[0].Profile)
	assert.Equal(t, "1607926", portainer[0].PID)
	assert.Equal(t, "enforce", portainer[0].Status)

	chronyd := status.Processes["/usr/sbin/chronyd"]
	require.Len(t, chronyd, 2)
	assert.Equal(t, "1858", chronyd[0].PID)
	assert.Equal(t, "1861", chronyd[1].PID)
}

func TestParseApparmorStatusEmpty(t *testing.T) {
	var status apparmorStatus
	err := json.Unmarshal([]byte(`{"version": "2", "profiles": {}, "processes": {}}`), &status)
	require.NoError(t, err)

	assert.Equal(t, "2", status.Version)
	assert.Empty(t, status.Profiles)
	assert.Empty(t, status.Processes)
}
