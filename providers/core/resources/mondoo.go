package resources

import (
	"runtime"

	"go.mondoo.com/cnquery"
	"go.mondoo.com/cnquery/cli/execruntime"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers/os/connection/shared"
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

func (m *mqlMondoo) capabilities() ([]interface{}, error) {
	conn := m.MqlRuntime.Connection.(shared.Connection)
	caps := conn.Capabilities().String()
	return llx.TArr2Raw(caps), nil
}
