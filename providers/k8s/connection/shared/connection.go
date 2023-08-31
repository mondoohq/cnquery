// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package shared

import (
	"fmt"
	"strings"

	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers/k8s/connection/shared/resources"
	admissionv1 "k8s.io/api/admission/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/version"
)

const (
	OPTION_MANIFEST          = "path"
	OPTION_IMMEMORY_CONTENT  = "manifest-content"
	OPTION_NAMESPACE         = "namespace"
	OPTION_NAMESPACE_EXCLUDE = "namespace-exclude"
	OPTION_ADMISSION         = "k8s-admission-review"
	OPTION_OBJECT_KIND       = "object-kind"
	OPTION_CONTEXT           = "context"
)

type ConnectionType string

type Connection interface {
	ID() uint32
	Name() string

	// Resources returns the resources that match the provided kind and name. If not kind and name
	// are provided, then all cluster resources are returned.
	Resources(kind string, name string, namespace string) (*ResourceResult, error)
	ServerVersion() *version.Info
	SupportedResourceTypes() (*resources.ApiResourceIndex, error)
	Platform() *inventory.Platform
	Asset() *inventory.Asset
	AssetId() (string, error)

	AdmissionReviews() ([]admissionv1.AdmissionReview, error)
	Namespace(name string) (*v1.Namespace, error)
	Namespaces() ([]v1.Namespace, error)

	InventoryConfig() *inventory.Config
}

type ClusterInfo struct {
	Name string
}

type ResourceResult struct {
	Name         string
	Kind         string
	ResourceType *resources.ApiResource // resource type that matched kind

	// Resources the resources that match the name, kind and namespace
	Resources []runtime.Object
	Namespace string
	AllNs     bool
}

func getPlatformInfo(objectKind string, runtime string) *inventory.Platform {
	// We need this at two places (discovery and provider)
	// Here it is needed for the transport and this is what is shown on the cli
	platformData := &inventory.Platform{
		Family:  []string{"k8s", "k8s-workload"},
		Kind:    "k8s-object",
		Runtime: runtime,
	}
	switch objectKind {
	case "pod":
		platformData.Name = "k8s-pod"
		platformData.Title = "Kubernetes Pod"
		return platformData
	case "cronjob":
		platformData.Name = "k8s-cronjob"
		platformData.Title = "Kubernetes CronJob"
		return platformData
	case "statefulset":
		platformData.Name = "k8s-statefulset"
		platformData.Title = "Kubernetes StatefulSet"
		return platformData
	case "deployment":
		platformData.Name = "k8s-deployment"
		platformData.Title = "Kubernetes Deployment"
		return platformData
	case "job":
		platformData.Name = "k8s-job"
		platformData.Title = "Kubernetes Job"
		return platformData
	case "replicaset":
		platformData.Name = "k8s-replicaset"
		platformData.Title = "Kubernetes ReplicaSet"
		return platformData
	case "daemonset":
		platformData.Name = "k8s-daemonset"
		platformData.Title = "Kubernetes DaemonSet"
		return platformData
	case "ingress":
		platformData.Name = "k8s-ingress"
		platformData.Title = "Kubernetes Ingress"
		return platformData
	case "namespace":
		platformData.Name = "k8s-namespace"
		platformData.Title = "Kubernetes Namespace"
		return platformData
	}

	return nil
}

func sliceToPtrSlice[T any](items []T) []*T {
	ptrItems := make([]*T, 0, len(items))
	for i := range items {
		ptrItems = append(ptrItems, &items[i])
	}
	return ptrItems
}

func NewPlatformId(assetId string) string {
	return "//platformid.api.mondoo.app/runtime/k8s/uid/" + assetId
}

func NewWorkloadPlatformId(clusterIdentifier, workloadType, namespace, name, uid string) string {
	if workloadType == "namespace" {
		return NewNamespacePlatformId(clusterIdentifier, name, uid)
	}

	platformIdentifier := clusterIdentifier
	// when mondoo is called with "--namespace xyz" the cluster identifier already contains the namespace
	// when called without the namespace, it is missing, but we need it to identify workloads
	if !strings.Contains(clusterIdentifier, "namespace") && namespace != "" {
		platformIdentifier += "/namespace/" + namespace
	}
	// add plural "s"
	platformIdentifier += "/" + workloadType + "s" + "/name/" + name
	return platformIdentifier
}

func NewNamespacePlatformId(clusterIdentifier, name, uid string) string {
	if clusterIdentifier == "" {
		return fmt.Sprintf("//platformid.api.mondoo.app/runtime/k8s/namespace/%s", name)
	}

	return fmt.Sprintf("%s/namespace/%s/uid/%s", clusterIdentifier, name, uid)
}
