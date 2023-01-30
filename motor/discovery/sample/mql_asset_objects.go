package sample

import (
	"github.com/cockroachdb/errors"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/motor"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers"
	sampleprovider "go.mondoo.com/cnquery/motor/providers/sample"
	"go.mondoo.com/cnquery/mql"
	"go.mondoo.com/cnquery/resources"
	resource_pack "go.mondoo.com/cnquery/resources/packs/sample"
)

type MqlDiscovery struct {
	rt *resources.Runtime
}

func NewMQLAssetsDiscovery(provider *sampleprovider.Provider) (*MqlDiscovery, error) {
	m, err := motor.New(provider)
	if err != nil {
		return nil, err
	}
	rt := resources.NewRuntime(resource_pack.Registry, m)
	return &MqlDiscovery{rt: rt}, nil
}

func (md *MqlDiscovery) Close() {
	if md.rt != nil && md.rt.Motor != nil {
		md.rt.Motor.Close()
	}
}

func (md *MqlDiscovery) GetList(query string) []interface{} {
	mqlExecutor := mql.New(md.rt, cnquery.DefaultFeatures)
	value, err := mqlExecutor.Exec(query, map[string]*llx.Primitive{})
	if err != nil {
		return nil
	}

	a := []interface{}{}
	d, _ := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result: &a,
	})
	d.Decode(value.Value)
	return a
}

func GatherAssets(tc *providers.Config, project string) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	pCfg := tc.Clone()
	at, err := sampleprovider.New(pCfg)
	if err != nil {
		return nil, err
	}
	m, err := NewMQLAssetsDiscovery(at)
	if err != nil {
		return nil, err
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, DiscoveryComputeInstances) {
		assets = append(assets, computeInstances(m, project, tc)...)
	}
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, DiscoveryGkeClusters) {
		assets = append(assets, gkeClusters(m, project, tc)...)
	}

	return assets, nil
}

func getTitleFamily(o sampleObject) (sampleObjectPlatformInfo, error) {
	switch o.service {
	case "compute":
		switch o.objectType {
		case "instance":
			return sampleObjectPlatformInfo{title: "Sample Compute Image", platform: "sample-compute-image"}, nil
		}
	case "gke":
		switch o.objectType {
		case "cluster":
			return sampleObjectPlatformInfo{title: "Sample GKE Image", platform: "sample-gke-cluster"}, nil
		}
	}

	return sampleObjectPlatformInfo{}, errors.Newf("missing runtime info for sample object service %s type %s", o.service, o.objectType)
}

func computeInstances(m *MqlDiscovery, project string, tc *providers.Config) []*asset.Asset {
	assets := []*asset.Asset{}
	images := m.GetList("return sample.project.compute.instances { id name }")
	for i := range images {
		b := images[i].(map[string]interface{})
		id := b["id"].(string)
		name := b["name"].(string)

		assets = append(assets, MqlObjectToAsset(project,
			mqlObject{
				name: name,
				gcpObject: sampleObject{
					project:    project,
					name:       name,
					id:         id,
					service:    "compute",
					objectType: "instance",
				},
			}, tc))
	}
	return assets
}

func gkeClusters(m *MqlDiscovery, project string, tc *providers.Config) []*asset.Asset {
	assets := []*asset.Asset{}
	images := m.GetList("return sample.project.gke.clusters { id name }")
	for i := range images {
		b := images[i].(map[string]interface{})
		id := b["id"].(string)
		name := b["name"].(string)

		assets = append(assets, MqlObjectToAsset(project,
			mqlObject{
				name: name,
				gcpObject: sampleObject{
					project:    project,
					name:       name,
					id:         id,
					service:    "gke",
					objectType: "cluster",
				},
			}, tc))
	}
	return assets
}

func validate(m mqlObject) error {
	if m.name == "" {
		return errors.New("name required for mql sample object to asset translation")
	}
	if m.gcpObject.id == "" {
		return errors.New("id required for mql sample object to asset translation")
	}
	if m.gcpObject.project == "" {
		return errors.New("project required for mql sample object to asset translation")
	}
	if m.gcpObject.name == "" {
		return errors.New("name required for mql sample object to asset translation")
	}
	return nil
}

func PlatformID(o sampleObject) string {
	return "//platformid.api.mondoo.app/runtime/sample/" + o.service + "/v1/projects/" + o.project + "/" + o.objectType + "/" + o.name
}

func MqlObjectToAsset(account string, mqlObject mqlObject, tc *providers.Config) *asset.Asset {
	if mqlObject.name == "" {
		mqlObject.name = mqlObject.gcpObject.id
	}
	if err := validate(mqlObject); err != nil {
		log.Error().Err(err).Msg("missing values in mql object to asset translation")
		return nil
	}
	info, err := getTitleFamily(mqlObject.gcpObject)
	if err != nil {
		log.Error().Err(err).Msg("missing runtime info")
		return nil
	}
	platformid := PlatformID(mqlObject.gcpObject)
	t := tc.Clone()
	t.PlatformId = platformid
	return &asset.Asset{
		PlatformIds: []string{platformid},
		Name:        mqlObject.name,
		Platform: &platform.Platform{
			Name:    info.platform,
			Title:   info.title,
			Kind:    providers.Kind_KIND_SAMPLE_OBJECT,
			Runtime: providers.RUNTIME_SAMPLE,
		},
		State:       asset.State_STATE_ONLINE,
		Labels:      mqlObject.labels,
		Connections: []*providers.Config{t},
	}
}

type mqlObject struct {
	name      string
	labels    map[string]string
	gcpObject sampleObject
}

type sampleObject struct {
	project    string
	id         string
	service    string
	objectType string
	name       string
}

type sampleObjectPlatformInfo struct {
	title    string
	platform string
}
