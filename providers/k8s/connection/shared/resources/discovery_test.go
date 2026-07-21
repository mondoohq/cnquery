// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"
)

// newFakeDynamicClient builds a fake dynamic client with the pods list kind
// registered so List() can resolve the list GVK before the reactors fire.
func newFakeDynamicClient() *dynamicfake.FakeDynamicClient {
	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	return dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		runtime.NewScheme(),
		map[schema.GroupVersionResource]string{gvr: "PodList"},
	)
}

func podsApiResource() ApiResource {
	return ApiResource{
		Resource:     metav1.APIResource{Name: "pods", Namespaced: false, Kind: "Pod"},
		GroupVersion: schema.GroupVersion{Group: "", Version: "v1"},
	}
}

func unstructuredPod(name string) unstructured.Unstructured {
	return unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1",
		"kind":       "Pod",
		"metadata":   map[string]any{"name": name, "namespace": "default"},
	}}
}

// TestGetKindResources_ForbiddenIsSwallowed verifies that a permission error
// (the expected case when a user's RBAC doesn't cover a discovered kind) is
// swallowed: the kind is skipped, no error is returned.
func TestGetKindResources_ForbiddenIsSwallowed(t *testing.T) {
	client := newFakeDynamicClient()
	client.PrependReactor("list", "pods", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, k8sErrors.NewForbidden(schema.GroupResource{Resource: "pods"}, "", errors.New("forbidden"))
	})

	d := &Discovery{dynClient: client}
	out, err := d.GetKindResources(context.Background(), podsApiResource(), "", true)
	require.NoError(t, err, "a forbidden error must be swallowed, not propagated")
	assert.Empty(t, out)
}

// TestGetKindResources_OtherErrorPropagates verifies that a non-permission
// error (throttling, transient API-server failure, expired continue token) is
// propagated instead of being silently turned into an empty result that looks
// like "this kind has no objects".
func TestGetKindResources_OtherErrorPropagates(t *testing.T) {
	client := newFakeDynamicClient()
	client.PrependReactor("list", "pods", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("the server was unable to handle the request")
	})

	d := &Discovery{dynClient: client}
	_, err := d.GetKindResources(context.Background(), podsApiResource(), "", true)
	require.Error(t, err, "a non-permission error must propagate")
}

// TestGetKindResources_Pagination verifies that all pages are concatenated by
// following the continue token rather than truncating at the first page.
func TestGetKindResources_Pagination(t *testing.T) {
	client := newFakeDynamicClient()

	call := 0
	client.PrependReactor("list", "pods", func(k8stesting.Action) (bool, runtime.Object, error) {
		call++
		list := &unstructured.UnstructuredList{}
		switch call {
		case 1:
			list.Items = []unstructured.Unstructured{unstructuredPod("pod-1")}
			list.SetContinue("page-2-token")
		default:
			list.Items = []unstructured.Unstructured{unstructuredPod("pod-2")}
			list.SetContinue("")
		}
		return true, list, nil
	})

	d := &Discovery{dynClient: client}
	out, err := d.GetKindResources(context.Background(), podsApiResource(), "", true)
	require.NoError(t, err)
	assert.Equal(t, 2, call, "both pages must be requested")
	require.Len(t, out, 2, "results from every page must be concatenated")
}
