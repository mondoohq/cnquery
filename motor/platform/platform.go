package platform

import (
	"strings"

	"go.mondoo.io/mondoo/motor/providers"
)

//go:generate protoc --proto_path=../../:. --go_out=. --go_opt=paths=source_relative --rangerrpc_out=. --iam-actions_out=. platform.proto

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
		case providers.RUNTIME_AWS:
			runtimeNiceName = "Amazon Web Services"
		case providers.RUNTIME_AWS_EC2:
			runtimeNiceName = "AWS EC2 Instance"
		case providers.RUNTIME_AWS_EC2_EBS:
			runtimeNiceName = "AWS EC2 EBS Volume"
		case providers.RUNTIME_AWS_ECR:
			runtimeNiceName = "AWS Elastic Container Registry"
		case providers.RUNTIME_AZ:
			runtimeNiceName = "Microsoft Azure"
		case providers.RUNTIME_AZ_COMPUTE:
			runtimeNiceName = "Azure Virtual Machine"
		case providers.RUNTIME_DOCKER_CONTAINER:
			runtimeNiceName = "Docker Container"
		case providers.RUNTIME_DOCKER_IMAGE:
			runtimeNiceName = "Docker Image"
		case providers.RUNTIME_DOCKER_REGISTRY:
			runtimeNiceName = "Docker Container Registry"
		case providers.RUNTIME_EQUINIX_METAL:
			runtimeNiceName = "Equinix Metal"
		case providers.RUNTIME_MICROSOFT_GRAPH:
			runtimeNiceName = "Microsoft Graph"
		case providers.RUNTIME_GCP:
			runtimeNiceName = "Google Cloud Platform"
		case providers.RUNTIME_GCP_COMPUTE:
			runtimeNiceName = "GCP Virtual Machine"
		case providers.RUNTIME_GCP_GCR:
			runtimeNiceName = "Google Container Registry"
		case providers.RUNTIME_GITHUB:
			runtimeNiceName = "GitHub"
		case providers.RUNTIME_GITLAB:
			runtimeNiceName = "GitLab"
		case providers.RUNTIME_KUBERNETES:
			runtimeNiceName = "Kubernetes"
		case providers.RUNTIME_KUBERNETES_CLUSTER:
			runtimeNiceName = "Kubernetes Cluster"
		case providers.RUNTIME_KUBERNETES_MANIFEST:
			runtimeNiceName = "Kubernetes Manifest File"
		case providers.RUNTIME_VSPHERE:
			runtimeNiceName = "vSphere"
		case providers.RUNTIME_VSPHERE_HOSTS:
			runtimeNiceName = "vSphere Host"
		case providers.RUNTIME_VSPHERE_VM:
			runtimeNiceName = "vSphere Virtual Machine"
		default:
			runtimeNiceName = runtimeName
		}
	} else {
		runtimeKind := p.Kind
		switch runtimeKind {
		case providers.Kind_KIND_API:
			runtimeNiceName = "API"
		case providers.Kind_KIND_BARE_METAL:
			runtimeNiceName = "bare metal"
		case providers.Kind_KIND_CODE:
			runtimeNiceName = "code"
		case providers.Kind_KIND_CONTAINER:
			runtimeNiceName = "Container"
		case providers.Kind_KIND_CONTAINER_IMAGE:
			runtimeNiceName = "Container Image"
		case providers.Kind_KIND_K8S_OBJECT:
			runtimeNiceName = "Kubernetes Object"
		case providers.Kind_KIND_NETWORK:
			runtimeNiceName = "Network"
		case providers.Kind_KIND_PACKAGE:
			runtimeNiceName = "Software Package"
		case providers.Kind_KIND_PROCESS:
			runtimeNiceName = "Process"
		case providers.Kind_KIND_UNKNOWN:
			runtimeNiceName = "Unknown"
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
