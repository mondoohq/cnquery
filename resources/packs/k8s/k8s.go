package k8s

import (
	"go.mondoo.io/mondoo/resources/packs/core"
	"go.mondoo.io/mondoo/resources/packs/k8s/info"
)

var Registry = info.Registry

func init() {
	Init(Registry)
}

func (k *mqlK8s) id() (string, error) {
	return "k8s", nil
}

func (k *mqlK8s) GetServerVersion() (interface{}, error) {
	kt, err := k8sProvider(k.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	return core.JsonToDict(kt.ServerVersion())
}
