package resources

import (
	"context"
	"io/ioutil"
	"os"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/disk"
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

	var cachedClient discovery.CachedDiscoveryInterface
	if os.Getenv("DEBUG") == "1" {
		cachedClient, err = disk.NewCachedDiscoveryClientForConfig(restConfig, ".cache/k8s", "", time.Hour)
		if err != nil {
			return nil, errors.Wrap(err, "failed to construct discovery client")
		}
	} else {
		dClient, err := discovery.NewDiscoveryClientForConfig(restConfig)
		if err != nil {
			return nil, errors.Wrap(err, "failed to construct discovery client")
		}
		cachedClient = memory.NewMemCacheClient(dClient)
	}

	// Always request fresh data from the server
	cachedClient.Invalidate()
	serverVersion, err := cachedClient.ServerVersion()
	if err != nil {
		return nil, err
	}

	log.Info().Msg("retrieving all k8s resources")

	var mu sync.Mutex
	cache := make(map[schema.GroupVersionResource][]runtime.Object)
	resTypes, err := supportedResourceTypes(cachedClient)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	var wg sync.WaitGroup
	for _, r := range resTypes.Resources() {
		wg.Add(1)
		go func(r ApiResource) {
			defer wg.Done()
			objs, _ := getKindResources(ctx, dynClient, r, "", true) // Load the resource for all namespaces
			mu.Lock()
			cache[r.GroupVersionResource()] = objs
			mu.Unlock()
		}(r)
	}

	wg.Wait()
	log.Debug().Msg("warmed up k8s resources cache")

	return &Discovery{
		resCache:        cache,
		discoveryClient: cachedClient,
		dynClient:       dynClient,
		ServerVersion:   serverVersion,
	}, nil
}

type Discovery struct {
	resCache        map[schema.GroupVersionResource][]runtime.Object
	dynClient       dynamic.Interface
	discoveryClient discovery.CachedDiscoveryInterface
	ServerVersion   *version.Info
}

func (d *Discovery) SupportedResourceTypes() (*ApiResourceIndex, error) {
	return supportedResourceTypes(d.discoveryClient)
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
	objs, ok := d.resCache[apiRes.GroupVersionResource()]
	if !ok {
		log.Debug().Msgf("couldn't load %s from cache. Attempting new retrieval...", apiRes.GroupVersionResource())
		return getKindResources(ctx, d.dynClient, apiRes, ns, allNs)
	}

	// If the resource is namespaced and there is ns filter provided, we filter the cached slice.
	if apiRes.Resource.Namespaced && !allNs && ns != "" {
		var filtered []runtime.Object
		for _, o := range objs {
			obj, err := meta.Accessor(o)

			// There should be no errors here as we already know the list contains API objects but just for the sake
			// of not crashing, make sure there is no error.
			if err == nil && obj.GetNamespace() == ns {
				filtered = append(filtered, o)
			}
		}
		objs = filtered // Replace the slice to be returned with the slice filtered on namespace
	}

	log.Debug().Msgf("loaded %s from cache", apiRes.GroupVersionResource())
	return objs, nil
}

func supportedResourceTypes(discoveryClient discovery.CachedDiscoveryInterface) (*ApiResourceIndex, error) {
	log.Debug().Msg("query api resource types")
	resList, err := discoveryClient.ServerPreferredResources()
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch api resource types from kubernetes")
	}
	log.Debug().Msgf("found %d api resource types", len(resList))

	return ResourceIndex(resList)
}

func getKindResources(ctx context.Context, client dynamic.Interface, apiRes ApiResource, ns string, allNs bool) ([]runtime.Object, error) {
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

		out = append(out, UnstructuredListToObjectList(resp.Items)...)

		next = resp.GetContinue()
		if next == "" {
			break
		}
	}
	return out, nil
}

func contains(v []string, s string) bool {
	for _, vv := range v {
		if vv == s {
			return true
		}
	}
	return false
}
