package sample

import (
	"errors"
	"fmt"

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

func (g *mqlSampleProjectComputeInstance) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return "sample.project.computeService.instance/" + id, nil
}

func (g *mqlSampleProject) GetCompute() (interface{}, error) {
	return g.MotorRuntime.CreateResource("sample.project.computeService",
		"projectId", "sampleProjectId",
	)
}

func (g *mqlSampleProjectComputeService) GetInstances() ([]interface{}, error) {
	res := make([]interface{}, 0, 3)
	for i := 0; i < 3; i++ {
		r, err := g.MotorRuntime.CreateResource("sample.project.compute.instance",
			"id", fmt.Sprintf("sampleId-%d", i),
			"name", fmt.Sprintf("compute-instance-%d", i),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}
