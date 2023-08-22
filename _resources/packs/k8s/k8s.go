// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package k8s

import (
	"go.mondoo.com/cnquery/resources/packs/core"
	"go.mondoo.com/cnquery/resources/packs/k8s/info"
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
