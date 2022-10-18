package platform

import (
	"strings"

	"go.mondoo.com/cnquery/motor/providers"
)

//go:generate protoc --proto_path=../../:. --go_out=. --go_opt=paths=source_relative --rangerrpc_out=. platform.proto

// often used family names
var (
	FAMILY_UNIX    = "unix"
	FAMILY_DARWIN  = "darwin"
	FAMILY_LINUX   = "linux"
	FAMILY_BSD     = "bsd"
	FAMILY_WINDOWS = "windows"
)

func (p *Platform) IsFamily(family string) bool {
	for i := range p.Family {
		if p.Family[i] == family {
			return true
		}
	}
	return false
}

func (p *Platform) PrettyTitle() string {
	prettyTitle := p.Title

	// extend the title only for OS and k8s objects
	if !(p.IsFamily("k8s-workload") || p.IsFamily("os")) {
		return prettyTitle
	}

	var runtimeNiceName string
	runtimeName := p.Runtime
	if runtimeName != "" {
		switch runtimeName {
		case providers.RUNTIME_AWS_EC2:
			runtimeNiceName = "AWS EC2 Instance"
		case providers.RUNTIME_AZ_COMPUTE:
			runtimeNiceName = "Azure Virtual Machine"
		case providers.RUNTIME_DOCKER_CONTAINER:
			runtimeNiceName = "Docker Container"
		case providers.RUNTIME_DOCKER_IMAGE:
			runtimeNiceName = "Docker Image"
		case providers.RUNTIME_GCP_COMPUTE:
			runtimeNiceName = "GCP Virtual Machine"
		case providers.RUNTIME_KUBERNETES_CLUSTER:
			runtimeNiceName = "Kubernetes Cluster"
		case providers.RUNTIME_KUBERNETES_MANIFEST:
			runtimeNiceName = "Kubernetes Manifest File"
		case providers.RUNTIME_VSPHERE_HOSTS:
			runtimeNiceName = "vSphere Host"
		case providers.RUNTIME_VSPHERE_VM:
			runtimeNiceName = "vSphere Virtual Machine"
		}
	} else {
		runtimeKind := p.Kind
		switch runtimeKind {
		case providers.Kind_KIND_BARE_METAL:
			runtimeNiceName = "bare metal"
		case providers.Kind_KIND_CONTAINER:
			runtimeNiceName = "Container"
		case providers.Kind_KIND_CONTAINER_IMAGE:
			runtimeNiceName = "Container Image"
		case providers.Kind_KIND_VIRTUAL_MACHINE:
			runtimeNiceName = "Virtual Machine"
		case providers.Kind_KIND_VIRTUAL_MACHINE_IMAGE:
			runtimeNiceName = "Virtual Machine Image"
		}
	}
	// e.g. ", Kubernetes Cluster" and also "Kubernetes, Kubernetes Cluster" do not look nice, so prevent them
	if prettyTitle == "" || strings.Contains(runtimeNiceName, prettyTitle) {
		return runtimeNiceName
	}

	// do not add runtime name when the title is already obvious, e.g. "Network API, Network"
	if !strings.Contains(prettyTitle, runtimeNiceName) {
		prettyTitle += ", " + runtimeNiceName
	}

	return prettyTitle
}
