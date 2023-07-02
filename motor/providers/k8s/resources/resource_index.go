package resources

import (
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func ResourceIndex(resList []*metav1.APIResourceList) (*ApiResourceIndex, error) {
	ri := NewApiResourceIndex()
	for _, group := range resList {
		log.Debug().Msgf("iterating over group %s/%s (%d api resources types)", group.GroupVersion, group.APIVersion, len(group.APIResources))
		gv, err := schema.ParseGroupVersion(group.GroupVersion)
		if err != nil {
			return nil, errors.Join(err, errors.New(fmt.Sprintf("%q cannot be parsed into groupversion", group.GroupVersion)))
		}

		for _, apiRes := range group.APIResources {
			log.Debug().Msgf("api=%s namespaced=%v", apiRes.Name, apiRes.Namespaced)
			if !contains(apiRes.Verbs, "list") {
				log.Debug().Msgf("api resource type (%s) is missing required verb 'list', skipping: %v", apiRes.Name, apiRes.Verbs)
				continue
			}
			v := ApiResource{
				GroupVersion: gv,
				Resource:     apiRes,
			}
			ri.Add(v)
		}
	}
	log.Debug().Msgf("found %d api resource types", len(ri.Resources()))
	return ri, nil
}
