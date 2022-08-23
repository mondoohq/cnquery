package k8s

import "go.mondoo.io/mondoo/resources/packs/core"

func (k *mqlK8s) GetApiResources() ([]interface{}, error) {
	kt, err := k8sProvider(k.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	resources, err := kt.SupportedResourceTypes()
	if err != nil {
		return nil, err
	}

	// convert to MQL resources
	list := resources.Resources()
	resp := []interface{}{}
	for i := range list {
		entry := list[i]

		mqlK8SResource, err := k.MotorRuntime.CreateResource("k8s.apiresource",
			"name", entry.Resource.Name,
			"singularName", entry.Resource.SingularName,
			"namespaced", entry.Resource.Namespaced,
			"group", entry.Resource.Group,
			"version", entry.Resource.Version,
			"kind", entry.Resource.Kind,
			"shortNames", core.StrSliceToInterface(entry.Resource.ShortNames),
			"categories", core.StrSliceToInterface(entry.Resource.Categories),
		)
		if err != nil {
			return nil, err
		}
		resp = append(resp, mqlK8SResource)
	}

	return resp, nil
}

func (k *mqlK8sApiresource) id() (string, error) {
	return k.Name()
}
