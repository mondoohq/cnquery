package resources

import (
	"context"
	"fmt"
	"io/ioutil"
	"sync"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // https://github.com/kubernetes/client-go/issues/242
	"k8s.io/client-go/rest"
)

func NewDiscovery(restConfig *rest.Config) (*Discovery, error) {
	// hide deprecation warnings for go api
	// see https://kubernetes.io/blog/2020/09/03/warnings/#customize-client-handling
	rest.SetDefaultWarningHandler(
		rest.NewWarningWriter(ioutil.Discard, rest.WarningWriterOptions{}),
	)

	dynClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to construct dynamic client")
	}
	dClient, err := discovery.NewDiscoveryClientForConfig(restConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to construct discovery client")
	}
	cachedClient := memory.NewMemCacheClient(dClient)

	// Always request fresh data from the server
	cachedClient.Invalidate()
	serverVersion, err := dClient.ServerVersion()
	if err != nil {
		return nil, err
	}

	return &Discovery{
		discoveryClient: cachedClient,
		dynClient:       dynClient,
		ServerVersion:   serverVersion,
	}, nil
}

type Discovery struct {
	dynClient       dynamic.Interface
	discoveryClient discovery.CachedDiscoveryInterface
	ServerVersion   *version.Info
}

func (d *Discovery) SupportedResourceTypes() (*ApiResourceIndex, error) {
	log.Debug().Msg("query api resource types")
	resList, err := d.discoveryClient.ServerPreferredResources()
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch api resource types from kubernetes")
	}
	log.Debug().Msgf("found %d api resource types", len(resList))

	ri := NewApiResourceIndex()
	for _, group := range resList {
		log.Debug().Msgf("iterating over group %s/%s (%d api resources types)", group.GroupVersion, group.APIVersion, len(group.APIResources))
		gv, err := schema.ParseGroupVersion(group.GroupVersion)
		if err != nil {
			return nil, errors.Wrapf(err, "%q cannot be parsed into groupversion", group.GroupVersion)
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

func (d *Discovery) GetAllResources(ctx context.Context, resTypes *ApiResourceIndex, ns string, allNs bool) ([]runtime.Object, error) {
	apis := resTypes.Resources()
	log.Debug().Msgf("query %d api resources concurrently", len(apis))

	var out []runtime.Object
	var mu sync.Mutex
	var wg sync.WaitGroup

	var collectErr error
	for _, api := range apis {
		wg.Add(1)
		go func(a ApiResource) {
			defer wg.Done()
			log.Debug().Msgf("query api resources: %s", a.GroupVersionResource())
			v, err := d.GetKindResources(ctx, a, ns, allNs)
			if err != nil {
				log.Debug().Msgf("query api resources error: %s, error=%v", a.GroupVersionResource(), err)
				collectErr = err
				return
			}
			mu.Lock()
			out = append(out, v...)
			mu.Unlock()
			log.Debug().Msgf("query api resources done: %s, found %d resources", a.GroupVersionResource(), len(v))
		}(api)
	}

	log.Debug().Msg("waiting for all queries to return")
	wg.Wait()
	log.Debug().Msgf("query api resources completed: objects=%d, error=%v", len(out), collectErr)
	return out, collectErr
}

func (d *Discovery) GetKindResources(ctx context.Context, apiRes ApiResource, ns string, allNs bool) ([]runtime.Object, error) {
	client := d.dynClient
	var out []runtime.Object

	var next string
	for {
		var intf dynamic.ResourceInterface
		nintf := client.Resource(apiRes.GroupVersionResource())
		log.Debug().Msgf("query resources for %s (namespaced: %t)", apiRes.Resource.Name, apiRes.Resource.Namespaced)
		if apiRes.Resource.Namespaced && !allNs {
			intf = nintf.Namespace(ns)
		} else {
			intf = nintf
		}
		resp, err := intf.List(ctx, metav1.ListOptions{
			Limit:    250,
			Continue: next,
		})
		// this error will happen when users have no permission
		if err != nil {
			log.Debug().Err(err).Msgf("could not fetch resources for: %v", apiRes.GroupVersionResource())
			break
		}

		objects, err := UnstructuredListToObjectList(resp.Items)
		if err != nil {
			return nil, fmt.Errorf("could not parse resources for %s: %w", apiRes.GroupVersionResource(), err)
		}

		out = append(out, objects...)

		next = resp.GetContinue()
		if next == "" {
			break
		}
	}
	return out, nil
}

func (d *Discovery) FilterResource(resourceTypes *ApiResourceIndex, resourceObjects []runtime.Object, kind string, name string) (*ApiResource, []runtime.Object, error) {
	// look up resource type for kind
	resType, err := resourceTypes.Lookup(kind)
	if err != nil {
		return nil, nil, err
	}

	// filter root resources
	roots := filterResource(resourceObjects, resType.Resource.Kind, name)
	return resType, roots, nil
}

func contains(v []string, s string) bool {
	for _, vv := range v {
		if vv == s {
			return true
		}
	}
	return false
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
