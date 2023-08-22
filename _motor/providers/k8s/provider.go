// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package k8s

//go:generate  go run github.com/golang/mock/mockgen -source=./provider.go -destination=./mock_provider.go -package=k8s

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/rs/zerolog/log"
	platform "go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/k8s/resources"
	admissionv1 "k8s.io/api/admission/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/version"
)

const (
	OPTION_MANIFEST         = "path"
	OPTION_IMMEMORY_CONTENT = "manifest-content"
	OPTION_NAMESPACE        = "namespace"
	OPTION_ADMISSION        = "k8s-admission-review"
	OPTION_OBJECT_KIND      = "object-kind"
	OPTION_CONTEXT          = "context"
)

var (
	_ providers.Instance = (*apiProvider)(nil)
	_ providers.Instance = (*manifestProvider)(nil)
)

type KubernetesProvider interface {
	providers.Instance
	providers.PlatformIdentifier
	Name() (string, error)
	PlatformInfo() *platform.Platform

	// Resources returns the resources that match the provided kind and name. If not kind and name
	// are provided, then all cluster resources are returned.
	Resources(kind string, name string, namespace string) (*ResourceResult, error)
	ServerVersion() *version.Info
	SupportedResourceTypes() (*resources.ApiResourceIndex, error)

	Nodes() ([]v1.Node, error)
	Namespace(name string) (*v1.Namespace, error)
	Namespaces() ([]v1.Namespace, error)
	Pod(namespace, name string) (*v1.Pod, error)
	Pods(namespace v1.Namespace) ([]*v1.Pod, error)
	CronJob(namespace, name string) (*batchv1.CronJob, error)
	CronJobs(namespace v1.Namespace) ([]*batchv1.CronJob, error)
	StatefulSet(namespace, name string) (*appsv1.StatefulSet, error)
	StatefulSets(namespace v1.Namespace) ([]*appsv1.StatefulSet, error)
	Deployment(namespace, name string) (*appsv1.Deployment, error)
	Deployments(namespace v1.Namespace) ([]*appsv1.Deployment, error)
	Job(namespace, name string) (*batchv1.Job, error)
	Jobs(namespace v1.Namespace) ([]*batchv1.Job, error)
	ReplicaSet(namespace, name string) (*appsv1.ReplicaSet, error)
	ReplicaSets(namespace v1.Namespace) ([]*appsv1.ReplicaSet, error)
	DaemonSet(namespace, name string) (*appsv1.DaemonSet, error)
	DaemonSets(namespace v1.Namespace) ([]*appsv1.DaemonSet, error)
	Secret(namespace, name string) (*v1.Secret, error)
	AdmissionReviews() ([]admissionv1.AdmissionReview, error)
	Ingress(namespace, name string) (*networkingv1.Ingress, error)
	Ingresses(namespace v1.Namespace) ([]*networkingv1.Ingress, error)
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

// New initializes the k8s provider and loads a configuration.
// Supported options are:
// - namespace: limits the resources to a specific namespace
// - path: use a manifest file instead of live API
func New(ctx context.Context, pc *providers.Config) (KubernetesProvider, error) {
	if pc.Backend != providers.ProviderType_K8S {
		return nil, providers.ErrProviderTypeDoesNotMatch
	}

	if manifestContent, manifestDefined := pc.Options[OPTION_IMMEMORY_CONTENT]; manifestDefined {
		log.Debug().Msg("use in-memory manifest content")
		data, err := base64.StdEncoding.DecodeString(manifestContent)
		if err != nil {
			return nil, err
		}
		return newManifestProvider(pc.PlatformId, pc.Options[OPTION_OBJECT_KIND], WithManifestContent(data), WithNamespace(pc.Options[OPTION_NAMESPACE]))
	}

	if manifestFile, manifestDefined := pc.Options[OPTION_MANIFEST]; manifestDefined {
		log.Debug().Msg("use manifest file")
		return newManifestProvider(pc.PlatformId, pc.Options[OPTION_OBJECT_KIND], WithManifestFile(manifestFile), WithNamespace(pc.Options[OPTION_NAMESPACE]))
	}

	if data, admissionDefined := pc.Options[OPTION_ADMISSION]; admissionDefined {
		log.Debug().Msg("use admission review")
		return newAdmissionProvider(data, pc.PlatformId, pc.Options[OPTION_OBJECT_KIND])
	}

	// initialize resource cache, so that the same k8s resources can be re-used
	log.Debug().Msg("use Kubernetes API")
	dCache, ok := resources.GetDiscoveryCache(ctx)
	if !ok {
		return nil, fmt.Errorf("context does not have an initialized discovery cache")
	}

	return newApiProvider(pc.Options[OPTION_CONTEXT], pc.Options[OPTION_NAMESPACE], pc.Options[OPTION_OBJECT_KIND], pc.PlatformId, dCache)
}

func getPlatformInfo(objectKind string, runtime string) *platform.Platform {
	// We need this at two places (discovery and provider)
	// Here it is needed for the transport and this is what is shown on the cli
	platformData := &platform.Platform{
		Family:  []string{"k8s", "k8s-workload"},
		Kind:    providers.Kind_KIND_K8S_OBJECT,
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
