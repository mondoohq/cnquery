package sample

import (
	"errors"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/sample/info"
)

var Registry = info.Registry

func init() {
	Init(Registry)
}

func (g *mqlSampleProject) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return "sample.project/" + id, nil
}

func (g *mqlSampleProject) init(args *resources.Args) (*resources.Args, SampleProject, error) {
	if len(*args) > 1 {
		return args, nil, nil
	}

	(*args)["id"] = "sampleProjectId"

	return args, nil, nil
}

func (g *mqlSampleProject) GetId() (string, error) {
	// placeholder to convince MQL that this is an optional field
	// should never be called since the data is initialized in init
	return "", errors.New("not implemented")
}

func (g *mqlSampleProjectComputeService) id() (string, error) {
	id, err := g.ProjectId()
	if err != nil {
		return "", err
	}
	return "sample.project.computeService/" + id, nil
}

func (g *mqlSampleProjectComputeServiceInstance) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return "sample.project.computeService.instance/" + id, nil
}

func (g *mqlSampleProjectComputeServiceDisk) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return "sample.project.computeService.disk/" + id, nil
}

func (g *mqlSampleProject) GetCompute() (interface{}, error) {
	return g.MotorRuntime.CreateResource("sample.project.computeService",
		"projectId", "sampleProjectId",
	)
}

func (g *mqlSampleProjectComputeService) GetInstances() ([]interface{}, error) {
	res := make([]interface{}, 0, 3)
	for i := 0; i < 3; i++ {
		r, err := g.MotorRuntime.CreateResource("sample.project.computeService.instance",
			"id", fmt.Sprintf("sampleId-%d", i),
			"projectId", "sampleProjectId",
			"name", fmt.Sprintf("compute-instance-%d", i),
			"disk", nil,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}

func (g *mqlSampleProjectComputeService) GetDisks() ([]interface{}, error) {
	res := make([]interface{}, 0, 3)
	for i := 0; i < 3; i++ {
		r, err := g.MotorRuntime.CreateResource("sample.project.computeService.disk",
			"id", fmt.Sprintf("sampleId-%d", i),
			"projectId", "sampleProjectId",
			"name", fmt.Sprintf("compute-disk-%d", i),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}

func (g *mqlSampleProject) GetGke() (interface{}, error) {
	return g.MotorRuntime.CreateResource("sample.project.gkeService",
		"projectId", "sampleProjectId",
	)
}

func (g *mqlSampleProjectGkeService) id() (string, error) {
	id, err := g.ProjectId()
	if err != nil {
		return "", err
	}
	return "sample.project.gkeService/" + id, nil
}

func (g *mqlSampleProjectGkeServiceCluster) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return "sample.project.gkeService.cluster/" + id, nil
}

func (g *mqlSampleProjectGkeService) GetClusters() ([]interface{}, error) {
	res := make([]interface{}, 0, 3)
	for i := 0; i < 3; i++ {
		r, err := g.MotorRuntime.CreateResource("sample.project.gkeService.cluster",
			"id", fmt.Sprintf("sampleId-%d", i),
			"projectId", "sampleProjectId",
			"name", fmt.Sprintf("gke-cluster-%d", i),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}

func (g *mqlSampleProjectComputeServiceInstance) init(args *resources.Args) (*resources.Args, SampleProjectComputeServiceInstance, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	if ids := getAssetIdentifier(g.MotorRuntime); ids != nil {
		(*args)["name"] = ids.name
		(*args)["projectId"] = ids.project
	}

	obj, err := g.MotorRuntime.CreateResource("sample.project.computeService", "projectId", (*args)["projectId"])
	if err != nil {
		return nil, nil, err
	}
	computeSvc := obj.(SampleProjectComputeService)
	instances, err := computeSvc.Instances()
	if err != nil {
		return nil, nil, err
	}

	for _, i := range instances {
		instance := i.(SampleProjectComputeServiceInstance)
		name, err := instance.Name()
		if err != nil {
			return nil, nil, err
		}
		projectId, err := instance.ProjectId()
		if err != nil {
			return nil, nil, err
		}

		if name == (*args)["name"] && projectId == (*args)["projectId"] {
			return args, instance, nil
		}
	}
	log.Error().Msgf("failed to find res %s %s", (*args)["name"], (*args)["projectId"])
	return nil, nil, &resources.ResourceNotFound{}
}

func (g *mqlSampleProjectGkeServiceCluster) init(args *resources.Args) (*resources.Args, SampleProjectGkeServiceCluster, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	if ids := getAssetIdentifier(g.MotorRuntime); ids != nil {
		(*args)["name"] = ids.name
		(*args)["projectId"] = ids.project
	}

	obj, err := g.MotorRuntime.CreateResource("sample.project.gkeService", "projectId", (*args)["projectId"])
	if err != nil {
		return nil, nil, err
	}
	gkeSvc := obj.(SampleProjectGkeService)
	clusters, err := gkeSvc.Clusters()
	if err != nil {
		return nil, nil, err
	}

	for _, c := range clusters {
		cluster := c.(SampleProjectGkeServiceCluster)
		name, err := cluster.Name()
		if err != nil {
			return nil, nil, err
		}
		projectId, err := cluster.ProjectId()
		if err != nil {
			return nil, nil, err
		}

		if name == (*args)["name"] && projectId == (*args)["projectId"] {
			return args, cluster, nil
		}
	}
	return nil, nil, &resources.ResourceNotFound{}
}

type assetIdentifier struct {
	name    string
	project string
}

func getAssetIdentifier(runtime *resources.Runtime) *assetIdentifier {
	a := runtime.Motor.GetAsset()
	if a == nil {
		return nil
	}
	var name, project string
	for _, id := range a.PlatformIds {
		if strings.HasPrefix(id, "//platformid.api.mondoo.app/runtime/sample/") {
			// "//platformid.api.mondoo.app/runtime/gcp/{o.service}/v1/projects/{project}/{objectType}/{name}"
			segments := strings.Split(id, "/")
			name = segments[len(segments)-1]
			project = segments[8]
			break
		}
	}
	return &assetIdentifier{name: name, project: project}
}
