package platform

import (
	"strings"

	"go.mondoo.io/mondoo/motor/transports"
)

//go:generate protoc --proto_path=../../:. --go_out=. --go_opt=paths=source_relative --falcon_out=. --iam-actions_out=. platform.proto

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

	var runtimeNiceName string
	runtimeName := p.Runtime
	if runtimeName != "" {
		switch runtimeName {
		case transports.RUNTIME_AWS:
			runtimeNiceName = "Amazon Web Services"
		case transports.RUNTIME_AWS_EC2:
			runtimeNiceName = "AWS EC2 Instance"
		case transports.RUNTIME_AWS_EC2_EBS:
			runtimeNiceName = "AWS EC2 EBS Volume"
		case transports.RUNTIME_AWS_ECR:
			runtimeNiceName = "AWS Elastic Container Registry"
		case transports.RUNTIME_AZ:
			runtimeNiceName = "Microsoft Azure"
		case transports.RUNTIME_AZ_COMPUTE:
			runtimeNiceName = "Azure Virtual Machine"
		case transports.RUNTIME_DOCKER_CONTAINER:
			runtimeNiceName = "Docker Container"
		case transports.RUNTIME_DOCKER_IMAGE:
			runtimeNiceName = "Docker Image"
		case transports.RUNTIME_DOCKER_REGISTRY:
			runtimeNiceName = "Docker Container Registry"
		case transports.RUNTIME_EQUINIX_METAL:
			runtimeNiceName = "Equinix Metal"
		case transports.RUNTIME_MICROSOFT_GRAPH:
			runtimeNiceName = "Microsoft Graph"
		case transports.RUNTIME_GCP:
			runtimeNiceName = "Google Cloud Platform"
		case transports.RUNTIME_GCP_COMPUTE:
			runtimeNiceName = "GCP Virtual Machine"
		case transports.RUNTIME_GCP_GCR:
			runtimeNiceName = "Google Container Registry"
		case transports.RUNTIME_GITHUB:
			runtimeNiceName = "GitHub"
		case transports.RUNTIME_GITLAB:
			runtimeNiceName = "GitLab"
		case transports.RUNTIME_KUBERNETES:
			runtimeNiceName = "Kubernetes"
		case transports.RUNTIME_KUBERNETES_CLUSTER:
			runtimeNiceName = "Kubernetes Cluster"
		case transports.RUNTIME_KUBERNETES_MANIFEST:
			runtimeNiceName = "Kubernetes Manifest File"
		case transports.RUNTIME_VSPHERE:
			runtimeNiceName = "vSphere"
		case transports.RUNTIME_VSPHERE_HOSTS:
			runtimeNiceName = "vSphere Host"
		case transports.RUNTIME_VSPHERE_VM:
			runtimeNiceName = "vSphere Virtual Machine"
		default:
			runtimeNiceName = runtimeName
		}
	} else {
		runtimeKind := p.Kind
		switch runtimeKind {
		case transports.Kind_KIND_API:
			runtimeNiceName = "API"
		case transports.Kind_KIND_BARE_METAL:
			runtimeNiceName = "bare metal"
		case transports.Kind_KIND_CODE:
			runtimeNiceName = "code"
		case transports.Kind_KIND_CONTAINER:
			runtimeNiceName = "Container"
		case transports.Kind_KIND_CONTAINER_IMAGE:
			runtimeNiceName = "Container Image"
		case transports.Kind_KIND_K8S_OBJECT:
			runtimeNiceName = "Kubernetes Object"
		case transports.Kind_KIND_NETWORK:
			runtimeNiceName = "Network"
		case transports.Kind_KIND_PACKAGE:
			runtimeNiceName = "Software Package"
		case transports.Kind_KIND_PROCESS:
			runtimeNiceName = "Process"
		case transports.Kind_KIND_UNKNOWN:
			runtimeNiceName = "Unknown"
		case transports.Kind_KIND_VIRTUAL_MACHINE:
			runtimeNiceName = "Virtual Machine"
		case transports.Kind_KIND_VIRTUAL_MACHINE_IMAGE:
			runtimeNiceName = "Virtual Machine Image"
		}
	}
	// do not add runtime name when the title is already obvious, e.g. "Network API, Network"
	if !strings.Contains(prettyTitle, runtimeNiceName) {
		prettyTitle += ", " + runtimeNiceName
	}

	return prettyTitle
}

// map that is organized by platform name, to quickly determine its families
var osTree = platfromPartens(OperatingSystems)

func platfromPartens(r *PlatformResolver) map[string][]string {
	return traverseFamily(r, []string{})
}

func traverseFamily(r *PlatformResolver, parents []string) map[string][]string {
	if r.IsFamiliy {
		// make sure we completely copy the values, otherwise they are going to overwrite themselves
		p := make([]string, len(parents))
		copy(p, parents)
		// add the current family
		p = append(p, r.Name)
		res := map[string][]string{}

		// iterate over children
		for i := range r.Children {
			child := r.Children[i]
			// recursively walk through the tree
			collect := traverseFamily(child, p)
			for k := range collect {
				res[k] = collect[k]
			}
		}
		return res
	}

	// return child (no family)
	return map[string][]string{
		r.Name: parents,
	}
}

func Family(platform string) []string {
	parents, ok := osTree[platform]
	if !ok {
		return []string{}
	}
	return parents
}

// gathers the family for the provided platform
// NOTE: at this point only operating systems have families
func IsFamily(platform string, family string) bool {
	// 1. determine the families of the platform
	parents, ok := osTree[platform]
	if !ok {
		return false
	}

	// 2. check that the platform is part of the family
	for i := range parents {
		if parents[i] == family {
			return true
		}
	}
	return false
}
