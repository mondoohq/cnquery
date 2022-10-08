package resources

import (
	"fmt"
	"strings"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type ApiResource struct {
	Resource     metav1.APIResource
	GroupVersion schema.GroupVersion
}

func (a ApiResource) GroupVersionResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    a.GroupVersion.Group,
		Version:  a.GroupVersion.Version,
		Resource: a.Resource.Name,
	}
}

func (a ApiResource) FullApiName() string {
	sgv := a.GroupVersionResource()
	return strings.Join([]string{sgv.Resource, sgv.Version, sgv.Group}, ".")
}

// ApiNames return all potential names for api resource
func (r ApiResource) ApiNames() []string {
	apiRes := r.Resource
	gv := r.GroupVersion

	singularName := apiRes.SingularName
	if len(singularName) == 0 {
		singularName = strings.ToLower(apiRes.Kind)
	}
	names := append([]string{singularName, apiRes.Name}, apiRes.ShortNames...)

	// expand names with api values
	var res []string
	for i := range names {
		// e.g. deployment
		name := names[i]
		// e.g. deployment.apps
		nameWithGroup := strings.Join([]string{name, gv.Group}, ".")
		// e.g. deployment.v1.apps
		nameWithGroupVersion := strings.Join([]string{name, gv.Version, gv.Group}, ".")

		res = append(res, name, nameWithGroup, nameWithGroupVersion)
	}
	return res
}

func NewApiResourceIndex() *ApiResourceIndex {
	ri := &ApiResourceIndex{
		index:          make(ApiResourceKindIndex),
		shortnameindex: make(ApiResourceKindIndex),
	}
	return ri
}

type ApiResourceKindIndex map[string][]ApiResource

type ApiResourceIndex struct {
	list           []ApiResource        // all api resources
	index          ApiResourceKindIndex // api resources by kind
	shortnameindex ApiResourceKindIndex // api resources by shortname of kind
}

func (ri *ApiResourceIndex) find(s string) []ApiResource {
	// discover resource type by kind
	m, ok := ri.index[s]
	if ok {
		return m
	}

	// discover resource type by shortname
	m, ok = ri.shortnameindex[s]
	if ok {
		return m
	}

	// nothing found
	return []ApiResource{}
}

func (ri *ApiResourceIndex) Resources() []ApiResource {
	return ri.list
}

// Add stores api resource for quick access
func (ri *ApiResourceIndex) Add(r ApiResource) {
	names := r.ApiNames()
	// klog.V(6).Infof("names: %s", strings.Join(names, ", "))
	for _, name := range names {
		ri.index[name] = append(ri.index[name], r)
	}

	shortnames := r.Resource.ShortNames
	// klog.V(6).Infof("shortnames: %s", strings.Join(shortnames, ", "))
	for _, name := range shortnames {
		ri.shortnameindex[name] = append(ri.index[name], r)
	}

	ri.list = append(ri.list, r)
}

// find api for kind
func (ri *ApiResourceIndex) Lookup(kind string) (*ApiResource, error) {
	// lookup overrides for certain service types
	kind = strings.ToLower(kind)
	switch kind {
	case "svc", "service", "services":
		// prefer v1.Service to avoid conflicts with knative
		out := ri.find("service.v1.")
		if len(out) != 0 {
			return &out[0], nil
		}

	// prevent "Error: ambiguous kind "cronjobs". use one of these as the KIND disambiguate: [cronjobs.v1.batch, cronjobs.v1beta1.batch]"
	case "cronjob", "cronjobs":
		// cronjobs should be in batch/v1
		out := ri.find("cronjobs.v1.batch")
		if len(out) != 0 {
			return &out[0], nil
		}

		// v1beta1 is deprecated since 1.21
		// https://kubernetes.io/docs/reference/using-api/deprecation-guide/#cronjob-v125
		out = ri.find("cronjobs.v1beta1.batch")
		if len(out) != 0 {
			return &out[0], nil
		}

	case "deploy", "deployment", "deployments":
		// deployments should be in apps/v1
		out := ri.find("deployment.v1.apps")
		if len(out) != 0 {
			return &out[0], nil
		}

		// extensions/v1beta1, apps/v1beta1, and apps/v1beta2 are deprecated
		// see https://kubernetes.io/blog/2019/07/18/api-deprecations-in-1-16/
		out = ri.find("deployment.v1beta2.apps")
		if len(out) != 0 {
			return &out[0], nil
		}
		out = ri.find("deployment.v1beta1.apps")
		if len(out) != 0 {
			return &out[0], nil
		}
		out = ri.find("deployment.v1beta1.extensions")
		if len(out) != 0 {
			return &out[0], nil
		}
	case "admissionreview.v1.admission":
		// AdmissionReview resources are special since they don't exist in the public
		// k8s API. However, we do work with them since we scan them via our admission
		// controller. To work around this we manually create the AdmissionReview ApiResource
		// and give it "list" access so the rest of our code can handle this type.
		return &ApiResource{
			Resource: metav1.APIResource{
				Name:  "admissionreviews",
				Kind:  "AdmissionReview",
				Verbs: []string{"list"},
			},
			GroupVersion: admissionv1.SchemeGroupVersion,
		}, nil
	}

	apiResults := ri.find(kind)
	if len(apiResults) == 0 {
		return nil, fmt.Errorf("could not find api kind %q", kind)
	} else if len(apiResults) > 1 {
		// handle case where resource name is not specific enough
		names := make([]string, 0, len(apiResults))
		for _, a := range apiResults {
			names = append(names, a.FullApiName())
		}
		return nil, fmt.Errorf("ambiguous kind %q. use one of these as the KIND disambiguate: [%s]", kind, strings.Join(names, ", "))
	}
	return &apiResults[0], nil
}
