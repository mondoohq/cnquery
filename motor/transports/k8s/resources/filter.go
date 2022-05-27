package resources

import (
	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
)

func FilterResource(resourceTypes *ApiResourceIndex, resourceObjects []runtime.Object, kind string, name string) (*ApiResource, []runtime.Object, error) {
	// look up resource type for kind
	resType, err := resourceTypes.Lookup(kind)
	if err != nil {
		return nil, nil, err
	}

	// filter root resources
	roots := filterResource(resourceObjects, resType.Resource.Kind, name)
	return resType, roots, nil
}

func filterResource(resources []runtime.Object, kind string, name string) []runtime.Object {
	filtered := []runtime.Object{}

	for i := range resources {
		res := resources[i]

		o, err := meta.Accessor(res)
		if err != nil {
			log.Error().Err(err).Msgf("could not filter resource")
			continue
		}

		if res.GetObjectKind().GroupVersionKind().Kind == kind {
			if len(name) > 0 && o.GetName() == name {
				filtered = append(filtered, res)
			} else if len(name) == 0 {
				filtered = append(filtered, res)
			}
		}
	}
	return filtered
}
