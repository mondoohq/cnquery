package k8s

import (
	"bytes"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/k8s"
	"go.mondoo.com/cnquery/motor/providers/k8s/resources"
	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/meta"
)

// ListAdmissionReviews list all AdmissionReview resources in the manifest.
func ListAdmissionReviews(
	p k8s.KubernetesProvider,
	connection *providers.Config,
	clusterIdentifier string,
	od *k8s.PlatformIdOwnershipDirectory,
) ([]*asset.Asset, error) {
	admissionReviews, err := p.AdmissionReviews()
	if err != nil {
		return nil, errors.Wrap(err, "failed to list AdmissionReviews")
	}

	assets := []*asset.Asset{}
	for i := range admissionReviews {
		aReview := admissionReviews[i]
		od.Add(&aReview)

		asset, err := assetFromAdmissionReview(aReview, p.Runtime(), connection, clusterIdentifier)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create asset from admission review")
		}

		log.Debug().Str("connection", asset.Connections[0].Host).Msg("resolved AdmissionReview")

		assets = append(assets, asset)
	}

	return assets, nil
}

func assetFromAdmissionReview(a admissionv1.AdmissionReview, runtime string, connection *providers.Config, clusterIdentifier string) (*asset.Asset, error) {
	// Use the meta from the request object.
	obj, err := resources.ResourcesFromManifest(bytes.NewReader(a.Request.Object.Raw))
	if err != nil {
		log.Error().Err(err).Msg("failed to parse object from admission review")
		return nil, err
	}
	objMeta, err := meta.Accessor(obj[0])
	if err != nil {
		log.Error().Err(err).Msg("could not access object attributes")
		return nil, err
	}
	objType, err := meta.TypeAccessor(&a)
	if err != nil {
		log.Error().Err(err).Msg("could not access object attributes")
		return nil, err
	}

	objectKind := objType.GetKind()
	platformData, err := createPlatformData(objectKind, runtime)
	if err != nil {
		return nil, err
	}
	platformData.Version = objType.GetAPIVersion()
	platformData.Build = objMeta.GetResourceVersion()
	platformData.Labels = map[string]string{
		"uid": string(objMeta.GetUID()),
	}

	assetLabels := objMeta.GetLabels()
	if assetLabels == nil {
		assetLabels = map[string]string{}
	}
	ns := objMeta.GetNamespace()
	var name string
	if ns != "" {
		name = ns + "/" + objMeta.GetName()
		platformData.Labels["namespace"] = ns
		assetLabels["namespace"] = ns
	} else {
		name = objMeta.GetName()
	}

	asset := &asset.Asset{
		PlatformIds: []string{k8s.NewPlatformWorkloadId(clusterIdentifier, strings.ToLower(objectKind), objMeta.GetNamespace(), objMeta.GetName())},
		Name:        name,
		Platform:    platformData,
		Connections: []*providers.Config{connection},
		State:       asset.State_STATE_ONLINE,
		Labels:      assetLabels,
	}

	return asset, nil
}
