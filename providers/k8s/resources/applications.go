// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"slices"
	"sort"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/types"
)

const (
	// https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/
	K8sApplicationName      = "app.kubernetes.io/name"
	K8sApplicationInstance  = "app.kubernetes.io/instance"
	K8sApplicationVersion   = "app.kubernetes.io/version"
	K8sApplicationComponent = "app.kubernetes.io/component"
	K8sApplicationPartOf    = "app.kubernetes.io/part-of"
	K8sApplicationManagedBy = "app.kubernetes.io/managed-by"
)

func (k *mqlK8s) apps() ([]interface{}, error) {
	apps := map[string]k8sapp{}

	// fetch deployment resources
	deployments := k.GetDeployments()
	if deployments.Error != nil {
		return nil, deployments.Error
	}

	for i := range deployments.Data {
		d := deployments.Data[i].(*mqlK8sDeployment)
		labels := d.GetLabels().Data
		extractApp(apps, labels)
	}

	// fetch daemonset resources
	daemonsets := k.GetDaemonsets()
	if daemonsets.Error != nil {
		return nil, daemonsets.Error
	}

	for i := range daemonsets.Data {
		d := daemonsets.Data[i].(*mqlK8sDaemonset)
		labels := d.GetLabels().Data
		extractApp(apps, labels)
	}

	// return k8s app list
	appList := []interface{}{}
	for _, app := range apps {
		r, err := CreateResource(k.MqlRuntime, "k8s.app", map[string]*llx.RawData{
			"__id":       llx.StringData(app.name + "/" + app.instance),
			"name":       llx.StringData(app.name),
			"version":    llx.StringData(app.version),
			"instance":   llx.StringData(app.instance),
			"managedBy":  llx.StringData(app.managedBy),
			"partOf":     llx.StringData(app.partOf),
			"components": llx.ArrayData(convert.SliceAnyToInterface(app.components), types.String),
		})
		if err != nil {
			return nil, err
		}

		appList = append(appList, r)
	}

	return appList, nil
}

type k8sapp struct {
	name       string
	version    string
	instance   string
	components []string
	partOf     string
	managedBy  string
}

func extractApp(apps map[string]k8sapp, labels map[string]interface{}) {
	name, nameOk := labels[K8sApplicationName]
	instance, instanceOK := labels[K8sApplicationInstance]
	version, versionOK := labels[K8sApplicationVersion]
	component, componentOK := labels[K8sApplicationComponent]
	partOf, partOfOK := labels[K8sApplicationPartOf]
	managedBy, managedByOK := labels[K8sApplicationManagedBy]

	if !nameOk {
		// if the name is not set, we cannot create an app
		return
	}

	app := k8sapp{
		name: name.(string),
	}
	if instanceOK {
		app.instance = instance.(string)
	}
	if versionOK {
		app.version = version.(string)
	}
	if componentOK {
		app.components = []string{component.(string)}
	}
	if partOfOK {
		app.partOf = partOf.(string)
	}
	if managedByOK {
		app.managedBy = managedBy.(string)
	}

	key := app.name + app.instance
	if existing, ok := apps[key]; ok {
		// if the app already exists, we need to merge the components
		components := append(existing.components, app.components...)
		sort.Strings(components)
		components = slices.Compact(components)
		existing.components = components
		apps[app.name+app.instance] = existing
	} else {
		apps[app.name+app.instance] = app
	}
}
