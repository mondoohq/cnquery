// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package services

import (
	"errors"
	"regexp"

	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v11/providers/os/detector"
)

type Service struct {
	Name        string
	Description string
	State       State
	Type        string
	Installed   bool
	Running     bool
	Enabled     bool
	Masked      bool
}

type State string

const (
	ServiceContinuePending State = "ServiceContinuePending"
	ServicePausePending    State = "ServicePausePending"
	ServicePaused          State = "ServicePaused"
	ServiceRunning         State = "ServiceRunning"
	ServiceStartPending    State = "ServiceStartPending"
	ServiceStopPending     State = "ServiceStopPending"
	ServiceStopped         State = "ServiceStopped"
	ServiceUnknown         State = "ServiceUnknown"
)

type OSServiceManager interface {
	Name() string
	List() ([]*Service, error)
}

type noopOsServiceManager struct{}

func (n *noopOsServiceManager) Name() string {
	return "none"
}

func (n *noopOsServiceManager) List() ([]*Service, error) {
	return nil, nil
}

var amazonlinux1version = regexp.MustCompile(`^201\d`)

func ResolveManager(conn shared.Connection) (OSServiceManager, error) {
	var osm OSServiceManager

	asset := conn.Asset()
	if asset == nil || asset.Platform == nil {
		return nil, errors.New("cannot find OS information for package detection")
	}

	useNoopInit := false
	if asset.Platform.IsFamily("linux") {
		// If we're on linux, check if there is no init system. If there is no init system,
		// we don't have managed services. This happens in containers.
		if _, err := conn.FileInfo("/sbin/init"); err != nil {
			useNoopInit = true
		}
	}

	switch {
	case useNoopInit:
		osm = &noopOsServiceManager{}
	case asset.Platform.IsFamily("arch"): // arch family
		osm = ResolveSystemdServiceManager(conn)
	case asset.Platform.Name == "amazonlinux":
		if amazonlinux1version.MatchString(asset.Platform.Version) {
			osm = &UpstartServiceManager{SysVServiceManager{conn: conn}}
		} else {
			osm = ResolveSystemdServiceManager(conn)
		}
	case asset.Platform.Name == "photon":
		osm = ResolveSystemdServiceManager(conn)
	// NOTE: we need to check fedora before rhel family, since its also rhel family
	case asset.Platform.Name == "fedora":
		rv := detector.ParseOsVersion(asset.Platform.Version)
		v, err := rv.MajorAtoi()
		if err != nil {
			return nil, errors.New("unknown fedora version: " + asset.Platform.Version)
		}

		if v < 15 {
			// upstart is only used since fedora 11 but we do not support those older versions
			osm = &UpstartServiceManager{SysVServiceManager{conn: conn}}
		} else {
			osm = ResolveSystemdServiceManager(conn)
		}
	case asset.Platform.IsFamily("redhat"):
		rv := detector.ParseOsVersion(asset.Platform.Version)
		v, err := rv.MajorAtoi()
		if err != nil {
			return nil, errors.New("unknown redhat version: " + asset.Platform.Version)
		}
		if v < 7 {
			osm = &UpstartServiceManager{SysVServiceManager{conn: conn}}
		} else {
			osm = ResolveSystemdServiceManager(conn)
		}
	case asset.Platform.Name == "ubuntu" || asset.Platform.Name == "linuxmint" || asset.Platform.Name == "pop":
		rv := detector.ParseOsVersion(asset.Platform.Version)
		v, err := rv.MajorAtoi()
		if err != nil {
			return nil, errors.New("unknown ubuntu version: " + asset.Platform.Version)
		}

		if v < 15 {
			osm = &UpstartServiceManager{SysVServiceManager{conn: conn}}
		} else {
			osm = ResolveSystemdServiceManager(conn)
		}
	case asset.Platform.Name == "debian":
		rv := detector.ParseOsVersion(asset.Platform.Version)
		v, err := rv.MajorAtoi()
		if err != nil {
			return nil, errors.New("unknown debian version: " + asset.Platform.Version)
		}

		if v < 7 {
			osm = &SysVServiceManager{conn: conn}
		} else {
			osm = ResolveSystemdServiceManager(conn)
		}
	case asset.Platform.Name == "raspbian":
		osm = ResolveSystemdServiceManager(conn)
	case asset.Platform.Name == "suse-microos": // it is suse family but uses a different version scheme
		osm = ResolveSystemdServiceManager(conn)
	case asset.Platform.IsFamily("suse"):
		rv := detector.ParseOsVersion(asset.Platform.Version)
		v, err := rv.MajorAtoi()
		if err != nil {
			return nil, errors.New("unknown suse version: " + asset.Platform.Version)
		}

		// NOTE: opensuse-tumbleweed uses version numbers like 20200622
		if v < 12 {
			osm = &SysVServiceManager{conn: conn}
		} else {
			osm = ResolveSystemdServiceManager(conn)
		}
	case asset.Platform.IsFamily("darwin"): // "macos", "darwin"
		osm = &LaunchDServiceManager{conn: conn}
	case asset.Platform.Name == "freebsd" || asset.Platform.Name == "dragonflybsd" || asset.Platform.Name == "netbsd":
		osm = &BsdInitServiceManager{conn: conn}
	case asset.Platform.Name == "openbsd":
		osm = &OpenBsdRcctlServiceManager{conn: conn}
	case asset.Platform.Name == "windows":
		osm = &WindowsServiceManager{conn: conn}
	case asset.Platform.Name == "alpine":
		osm = &AlpineOpenrcServiceManager{conn: conn}
	case asset.Platform.Name == "cos":
		osm = ResolveSystemdServiceManager(conn)
	case asset.Platform.Name == "aix":
		osm = &AixServiceManager{conn: conn}
	case asset.Platform.Name == "kali": // debian based with versions from 2015 onwards being systemd based
		osm = ResolveSystemdServiceManager(conn)
	case asset.Platform.Name == "cloudlinux": // rhel based
		osm = ResolveSystemdServiceManager(conn)
	}

	if osm == nil {
		return nil, errors.New("could not detect suitable service manager for platform: " + asset.Platform.Name)
	}

	return osm, nil
}

func FindService(services []*Service, name string) (*Service, error) {
	// search for name
	for i := range services {
		service := services[i]
		if service.Name == name {
			return service, nil
		}
	}

	return nil, errors.New("service> " + name + " does not exist")
}
