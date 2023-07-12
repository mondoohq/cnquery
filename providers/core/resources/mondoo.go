package resources

import (
	"runtime"

	"go.mondoo.com/cnquery"
	"go.mondoo.com/cnquery/cli/execruntime"
)

func (m *mqlMondoo) version() (string, error) {
	return cnquery.GetVersion(), nil
}

func (m *mqlMondoo) build() (string, error) {
	return cnquery.GetBuild(), nil
}

func (m *mqlMondoo) arch() (string, error) {
	return runtime.GOOS + "-" + runtime.GOARCH, nil
}

func (m *mqlMondoo) jobEnvironment() (map[string]interface{}, error) {
	// get the local agent runtime information
	ciEnv := execruntime.Detect()

	return map[string]interface{}{
		"id":   ciEnv.Namespace,
		"name": ciEnv.Name,
	}, nil
}
