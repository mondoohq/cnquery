// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package shared

import (
	"fmt"
	"strings"

	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/k8s/connection/shared/resources"
	admissionv1 "k8s.io/api/admission/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/version"
)

const (
	OPTION_MANIFEST          = "path"
	OPTION_IMMEMORY_CONTENT  = "manifest-content"
	OPTION_NAMESPACE         = "namespaces"
	OPTION_NAMESPACE_EXCLUDE = "namespaces-exclude"
	OPTION_ADMISSION         = "k8s-admission-review"
	OPTION_OBJECT_KIND       = "object-kind"
	OPTION_CONTEXT           = "context"
	idPrefix                 = "//platformid.api.mondoo.app/runtime/k8s/uid/"
)

type ConnectionType string

type Connection interface {
	plugin.Connection
	Name() string
	Runtime() string

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

func sliceToPtrSlice[T any](items []T) []*T {
	ptrItems := make([]*T, 0, len(items))
	for i := range items {
		ptrItems = append(ptrItems, &items[i])
	}
	return ptrItems
}

func NewPlatformId(assetId string) string {
	return idPrefix + assetId
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
		return fmt.Sprintf("%snamespace/%s", idPrefix, name)
	}

	return fmt.Sprintf("%s/namespace/%s/uid/%s", clusterIdentifier, name, uid)
}
