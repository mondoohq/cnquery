package k8s

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/transports/k8s/resources"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/version"
)

func (t *Transport) ServerVersion() *version.Info {
	return t.d.ServerVersion
}

// discover api and resources that have a list method
func (t *Transport) SupportedResources() (*resources.ApiResourceIndex, error) {
	// TODO: this should likely be cached
	return t.d.SupportedResourceTypes()
}

type ResourceResult struct {
	Name          string
	Kind          string
	ResourceType  *resources.ApiResource // resource type that matched kind
	AllResources  []runtime.Object
	RootResources []runtime.Object
	Namespace     string
	AllNs         bool
}

func (t *Transport) Resources(kind string, name string) (*ResourceResult, error) {
	ctx := context.Background()
	ns := t.opts[OPTION_NAMESPACE]
	allNs := false
	if len(ns) == 0 {
		allNs = true
	}

	var err error
	var resourceObjects []runtime.Object

	// TODO: this should only apply for api calls
	resTypes, err := t.SupportedResources()
	if err != nil {
		return nil, err
	}
	log.Debug().Msg("completed querying resource types")

	if len(t.manifestFile) > 0 {
		log.Debug().Str("file", t.manifestFile).Msg("load resources from manifest files")
		var input io.Reader

		// if content is piped
		if t.manifestFile == "-" {
			input = os.Stdin
		} else {
			// return all resources from manifest
			filenames := []string{}

			fi, err := os.Stat(t.manifestFile)
			if err != nil {
				return nil, err
			}

			if fi.IsDir() {
				// NOTE: we are not using filepath.WalkDir since we do not net recursive walking
				files, err := ioutil.ReadDir(t.manifestFile)
				if err != nil {
					return nil, err
				}
				for i := range files {
					f := files[i]
					if f.IsDir() {
						continue
					}
					filename := path.Join(t.manifestFile, f.Name())

					// only load yaml files for now
					ext := filepath.Ext(filename)
					if ext == ".yaml" || ext == ".yml" {
						log.Debug().Str("file", filename).Msg("add file to manifest loading")
						filenames = append(filenames, filename)
					} else {
						log.Debug().Str("file", filename).Msg("ignore file")
					}

				}

			} else {
				filenames = append(filenames, t.manifestFile)
			}

			input, err = resources.MergeManifestFiles(filenames)
			if err != nil {
				return nil, err
			}
		}

		resourceObjects, err = resources.ResourcesFromManifest(input)
		if err != nil {
			return nil, errors.Wrap(err, "could not query resource objects")
		}
		log.Debug().Msgf("found %d resource objects", len(resourceObjects))
	} else {
		// return all resources for specified resource tpyes and namespace
		log.Debug().Msg("fetch all resource objects")
		resourceObjects, err = t.d.GetAllResources(ctx, resTypes, ns, allNs)
		if err != nil {
			return nil, errors.Wrap(err, "could not query resource objects")
		}
		log.Debug().Msgf("found %d resource objects", len(resourceObjects))
	}

	// find root nodes
	resType, rootResources, err := t.d.FilterResource(resTypes, resourceObjects, kind, name)

	return &ResourceResult{
		Name:          name,
		Kind:          kind,
		ResourceType:  resType,
		AllResources:  resourceObjects,
		RootResources: rootResources,
		Namespace:     ns,
		AllNs:         allNs,
	}, err
}
