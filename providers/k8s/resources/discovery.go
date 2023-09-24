// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/gobwas/glob"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers/k8s/connection/shared"
	"go.mondoo.com/cnquery/providers/k8s/connection/shared/resources"
	"go.mondoo.com/cnquery/types"
	"golang.org/x/exp/slices"
	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	DiscoveryAuto             = "auto"
	DiscoveryClusters         = "clusters"
	DiscoveryPods             = "pods"
	DiscoveryJobs             = "jobs"
	DiscoveryCronJobs         = "cronjobs"
	DiscoveryStatefulSets     = "statefulsets"
	DiscoveryDeployments      = "deployments"
	DiscoveryReplicaSets      = "replicasets"
	DiscoveryDaemonSets       = "daemonsets"
	DiscoveryContainerImages  = "container-images"
	DiscoveryAdmissionReviews = "admissionreviews"
	DiscoveryIngresses        = "ingresses"
	DiscoveryNamespaces       = "namespaces"
)

type NamespaceFilterOpts struct {
	include []string
	exclude []string
}

func (f *NamespaceFilterOpts) skipNamespace(namespace string) bool {
	// anything explicitly specified in the list of includes means accept only from that list
	if len(f.include) > 0 {
		for _, ns := range f.include {
			g, err := glob.Compile(ns)
			if err != nil {
				log.Error().Err(err).Msg("failed to compile glob")
				return false
			}
			if g.Match(namespace) {
				// stop looking, we found our match
				return false
			}
		}

		// didn't find it, so it must be skipped
		return true
	}

	// if nothing explicitly meant to be included, then check whether
	// it should be excluded
	for _, ns := range f.exclude {
		g, err := glob.Compile(ns)
		if err != nil {
			log.Error().Err(err).Msg("failed to compile glob")
			return false
		}
		if g.Match(namespace) {
			return true
		}
	}

	return false
}

func Discover(runtime *plugin.Runtime) (*inventory.Inventory, error) {
	conn := runtime.Connection.(shared.Connection)

	in := &inventory.Inventory{Spec: &inventory.InventorySpec{
		Assets: []*inventory.Asset{},
	}}

	if (conn.InventoryConfig().Discover == nil || len(conn.InventoryConfig().Discover.Targets) == 0) && conn.Asset() != nil {
		in.Spec.Assets = append(in.Spec.Assets, conn.Asset())
		return in, nil
	}

	invConfig := conn.InventoryConfig()

	res, err := runtime.CreateResource(runtime, "k8s", nil)
	if err != nil {
		return nil, err
	}
	k8s := res.(*mqlK8s)

	nsFilter := NamespaceFilterOpts{}
	if include, ok := invConfig.Options[shared.OPTION_NAMESPACE]; ok && len(include) > 0 {
		nsFilter.include = strings.Split(include, ",")
	}

	if exclude, ok := invConfig.Options[shared.OPTION_NAMESPACE_EXCLUDE]; ok && len(exclude) > 0 {
		nsFilter.exclude = strings.Split(exclude, ",")
	}

	// If we can discover the cluster asset, then we use that as root and build all
	// platform IDs for the assets based on it. If we cannot discover the cluster, we
	// discover the individual namespaces according to the ns filter and then build
	// the platform IDs for the assets based on the namespace.
	if len(nsFilter.include) == 0 && len(nsFilter.exclude) == 0 {
		assetId, err := conn.AssetId()
		if err != nil {
			return nil, err
		}

		root := &inventory.Asset{
			PlatformIds: []string{assetId},
			Name:        conn.Name(),
			Platform:    conn.Platform(),
			Connections: []*inventory.Config{invConfig.Clone(inventory.WithoutDiscovery())}, // pass-in the parent connection config
		}
		if slices.Contains(invConfig.Discover.Targets, DiscoveryClusters) {
			in.Spec.Assets = append(in.Spec.Assets, root)
		}

		od := NewPlatformIdOwnershipIndex(assetId)

		assets, err := discoverAssets(runtime, conn, invConfig, assetId, k8s, nsFilter, od, false)
		if err != nil {
			return nil, err
		}
		setRelatedAssets(conn, root, assets, od)
		in.Spec.Assets = append(in.Spec.Assets, assets...)
	} else {
		nss, err := discoverNamespaces(conn, invConfig, "", nil, nsFilter)
		if err != nil {
			return nil, err
		}

		in.Spec.Assets = append(in.Spec.Assets, nss...)

		// Discover the assets for each namespace and use the namespace platform ID as root
		for _, ns := range nss {
			nsFilter = NamespaceFilterOpts{include: []string{ns.Name}}

			od := NewPlatformIdOwnershipIndex(ns.PlatformIds[0])

			// We don't want to discover the namespaces again since we have already done this above
			assets, err := discoverAssets(runtime, conn, invConfig, ns.PlatformIds[0], k8s, nsFilter, od, true)
			if err != nil {
				return nil, err
			}
			setRelatedAssets(conn, ns, assets, od)
			in.Spec.Assets = append(in.Spec.Assets, assets...)
		}
	}

	return in, nil
}

func discoverAssets(
	runtime *plugin.Runtime,
	conn shared.Connection,
	invConfig *inventory.Config,
	clusterId string,
	k8s *mqlK8s,
	nsFilter NamespaceFilterOpts,
	od *PlatformIdOwnershipIndex,
	skipNsDiscovery bool,
) ([]*inventory.Asset, error) {
	var assets []*inventory.Asset
	var err error
	for _, target := range invConfig.Discover.Targets {
		var list []*inventory.Asset
		if target == DiscoveryPods || target == DiscoveryAuto {
			list, err = discoverPods(conn, invConfig, clusterId, k8s, od, nsFilter)
			if err != nil {
				return nil, err
			}
			assets = append(assets, list...)
		}
		if target == DiscoveryJobs || target == DiscoveryAuto {
			list, err = discoverJobs(conn, invConfig, clusterId, k8s, od, nsFilter)
			if err != nil {
				return nil, err
			}
			assets = append(assets, list...)
		}
		if target == DiscoveryCronJobs || target == DiscoveryAuto {
			list, err = discoverCronJobs(conn, invConfig, clusterId, k8s, od, nsFilter)
			if err != nil {
				return nil, err
			}
			assets = append(assets, list...)
		}
		if target == DiscoveryStatefulSets || target == DiscoveryAuto {
			list, err = discoverStatefulSets(conn, invConfig, clusterId, k8s, od, nsFilter)
			if err != nil {
				return nil, err
			}
			assets = append(assets, list...)
		}
		if target == DiscoveryDeployments || target == DiscoveryAuto {
			list, err = discoverDeployments(conn, invConfig, clusterId, k8s, od, nsFilter)
			if err != nil {
				return nil, err
			}
			assets = append(assets, list...)
		}
		if target == DiscoveryReplicaSets || target == DiscoveryAuto {
			list, err = discoverReplicaSets(conn, invConfig, clusterId, k8s, od, nsFilter)
			if err != nil {
				return nil, err
			}
			assets = append(assets, list...)
		}
		if target == DiscoveryDaemonSets || target == DiscoveryAuto {
			list, err = discoverDaemonSets(conn, invConfig, clusterId, k8s, od, nsFilter)
			if err != nil {
				return nil, err
			}
			assets = append(assets, list...)
		}
		if target == DiscoveryIngresses || target == DiscoveryAuto {
			list, err = discoverIngresses(conn, invConfig, clusterId, k8s, od, nsFilter)
			if err != nil {
				return nil, err
			}
			assets = append(assets, list...)
		}
		if target == DiscoveryAdmissionReviews {
			list, err = discoverAdmissionReviews(conn, invConfig, clusterId, k8s, od, nsFilter)
			if err != nil {
				return nil, err
			}
			assets = append(assets, list...)
		}
		if target == DiscoveryNamespaces && !skipNsDiscovery {
			list, err = discoverNamespaces(conn, invConfig, clusterId, od, nsFilter)
			if err != nil {
				return nil, err
			}
			assets = append(assets, list...)
		}
		if target == DiscoveryContainerImages || target == DiscoveryAuto {
			list, err = discoverContainerImages(runtime, invConfig, clusterId, k8s, nsFilter)
			if err != nil {
				return nil, err
			}
			assets = append(assets, list...)
		}
	}
	return assets, nil
}

func discoverPods(
	conn shared.Connection,
	invConfig *inventory.Config,
	clusterId string,
	k8s *mqlK8s,
	od *PlatformIdOwnershipIndex,
	nsFilter NamespaceFilterOpts,
) ([]*inventory.Asset, error) {
	pods := k8s.GetPods()
	if pods.Error != nil {
		return nil, pods.Error
	}

	assetList := make([]*inventory.Asset, 0, len(pods.Data))
	for _, p := range pods.Data {
		pod := p.(*mqlK8sPod)

		if skip := nsFilter.skipNamespace(pod.Namespace.Data); skip {
			continue
		}

		labels := map[string]string{}
		for k, v := range pod.GetLabels().Data {
			labels[k] = v.(string)
		}
		addMondooAssetLabels(labels, &pod.obj.ObjectMeta, clusterId)
		platform, err := createPlatformData(pod.Kind.Data, conn.Runtime())
		if err != nil {
			return nil, err
		}
		assetList = append(assetList, &inventory.Asset{
			PlatformIds: []string{
				shared.NewWorkloadPlatformId(clusterId, "pod", pod.Namespace.Data, pod.Name.Data, pod.Uid.Data),
			},
			Name:        pod.Namespace.Data + "/" + pod.Name.Data,
			Platform:    platform,
			Labels:      labels,
			Connections: []*inventory.Config{invConfig.Clone(inventory.WithoutDiscovery())}, // pass-in the parent connection config
		})
		od.Add(pod.obj)
	}
	return assetList, nil
}

func discoverJobs(
	conn shared.Connection,
	invConfig *inventory.Config,
	clusterId string,
	k8s *mqlK8s,
	od *PlatformIdOwnershipIndex,
	nsFilter NamespaceFilterOpts,
) ([]*inventory.Asset, error) {
	jobs := k8s.GetJobs()
	if jobs.Error != nil {
		return nil, jobs.Error
	}

	assetList := make([]*inventory.Asset, 0, len(jobs.Data))
	for _, j := range jobs.Data {
		job := j.(*mqlK8sJob)

		if skip := nsFilter.skipNamespace(job.Namespace.Data); skip {
			continue
		}

		labels := map[string]string{}
		for k, v := range job.GetLabels().Data {
			labels[k] = v.(string)
		}
		addMondooAssetLabels(labels, &job.obj.ObjectMeta, clusterId)
		platform, err := createPlatformData(job.Kind.Data, conn.Runtime())
		if err != nil {
			return nil, err
		}
		assetList = append(assetList, &inventory.Asset{
			PlatformIds: []string{
				shared.NewWorkloadPlatformId(clusterId, "job", job.Namespace.Data, job.Name.Data, job.Uid.Data),
			},
			Name:        job.Namespace.Data + "/" + job.Name.Data,
			Platform:    platform,
			Labels:      labels,
			Connections: []*inventory.Config{invConfig.Clone(inventory.WithoutDiscovery())}, // pass-in the parent connection config
		})
		od.Add(job.obj)
	}
	return assetList, nil
}

func discoverCronJobs(
	conn shared.Connection,
	invConfig *inventory.Config,
	clusterId string,
	k8s *mqlK8s,
	od *PlatformIdOwnershipIndex,
	nsFilter NamespaceFilterOpts,
) ([]*inventory.Asset, error) {
	cjs := k8s.GetCronjobs()
	if cjs.Error != nil {
		return nil, cjs.Error
	}

	assetList := make([]*inventory.Asset, 0, len(cjs.Data))
	for _, cj := range cjs.Data {
		cjob := cj.(*mqlK8sCronjob)

		if skip := nsFilter.skipNamespace(cjob.Namespace.Data); skip {
			continue
		}

		labels := map[string]string{}
		for k, v := range cjob.GetLabels().Data {
			labels[k] = v.(string)
		}
		addMondooAssetLabels(labels, &cjob.obj.ObjectMeta, clusterId)
		platform, err := createPlatformData(cjob.Kind.Data, conn.Runtime())
		if err != nil {
			return nil, err
		}
		assetList = append(assetList, &inventory.Asset{
			PlatformIds: []string{
				shared.NewWorkloadPlatformId(clusterId, "cronjob", cjob.Namespace.Data, cjob.Name.Data, cjob.Uid.Data),
			},
			Name:        cjob.Namespace.Data + "/" + cjob.Name.Data,
			Platform:    platform,
			Labels:      labels,
			Connections: []*inventory.Config{invConfig.Clone(inventory.WithoutDiscovery())}, // pass-in the parent connection config
		})
		od.Add(cjob.obj)
	}
	return assetList, nil
}

func discoverStatefulSets(
	conn shared.Connection,
	invConfig *inventory.Config,
	clusterId string,
	k8s *mqlK8s,
	od *PlatformIdOwnershipIndex,
	nsFilter NamespaceFilterOpts,
) ([]*inventory.Asset, error) {
	ss := k8s.GetStatefulsets()
	if ss.Error != nil {
		return nil, ss.Error
	}

	assetList := make([]*inventory.Asset, 0, len(ss.Data))
	for _, j := range ss.Data {
		statefulset := j.(*mqlK8sStatefulset)

		if skip := nsFilter.skipNamespace(statefulset.Namespace.Data); skip {
			continue
		}

		labels := map[string]string{}
		for k, v := range statefulset.GetLabels().Data {
			labels[k] = v.(string)
		}
		addMondooAssetLabels(labels, &statefulset.obj.ObjectMeta, clusterId)
		platform, err := createPlatformData(statefulset.Kind.Data, conn.Runtime())
		if err != nil {
			return nil, err
		}
		assetList = append(assetList, &inventory.Asset{
			PlatformIds: []string{
				shared.NewWorkloadPlatformId(clusterId, "statefulset", statefulset.Namespace.Data, statefulset.Name.Data, statefulset.Uid.Data),
			},
			Name:        statefulset.Namespace.Data + "/" + statefulset.Name.Data,
			Platform:    platform,
			Labels:      labels,
			Connections: []*inventory.Config{invConfig.Clone(inventory.WithoutDiscovery())}, // pass-in the parent connection config
		})
		od.Add(statefulset.obj)
	}
	return assetList, nil
}

func discoverDeployments(
	conn shared.Connection,
	invConfig *inventory.Config,
	clusterId string,
	k8s *mqlK8s,
	od *PlatformIdOwnershipIndex,
	nsFilter NamespaceFilterOpts,
) ([]*inventory.Asset, error) {
	ds := k8s.GetDeployments()
	if ds.Error != nil {
		return nil, ds.Error
	}

	assetList := make([]*inventory.Asset, 0, len(ds.Data))
	for _, d := range ds.Data {
		deployment := d.(*mqlK8sDeployment)

		if skip := nsFilter.skipNamespace(deployment.Namespace.Data); skip {
			continue
		}

		labels := map[string]string{}
		for k, v := range deployment.GetLabels().Data {
			labels[k] = v.(string)
		}
		addMondooAssetLabels(labels, &deployment.obj.ObjectMeta, clusterId)
		platform, err := createPlatformData(deployment.Kind.Data, conn.Runtime())
		if err != nil {
			return nil, err
		}
		assetList = append(assetList, &inventory.Asset{
			PlatformIds: []string{
				shared.NewWorkloadPlatformId(clusterId, "deployment", deployment.Namespace.Data, deployment.Name.Data, deployment.Uid.Data),
			},
			Name:        deployment.Namespace.Data + "/" + deployment.Name.Data,
			Platform:    platform,
			Labels:      labels,
			Connections: []*inventory.Config{invConfig.Clone(inventory.WithoutDiscovery())}, // pass-in the parent connection config
		})
		od.Add(deployment.obj)
	}
	return assetList, nil
}

func discoverReplicaSets(
	conn shared.Connection,
	invConfig *inventory.Config,
	clusterId string,
	k8s *mqlK8s,
	od *PlatformIdOwnershipIndex,
	nsFilter NamespaceFilterOpts,
) ([]*inventory.Asset, error) {
	rs := k8s.GetReplicasets()
	if rs.Error != nil {
		return nil, rs.Error
	}

	assetList := make([]*inventory.Asset, 0, len(rs.Data))
	for _, r := range rs.Data {
		replicaset := r.(*mqlK8sReplicaset)

		if skip := nsFilter.skipNamespace(replicaset.Namespace.Data); skip {
			continue
		}

		labels := map[string]string{}
		for k, v := range replicaset.GetLabels().Data {
			labels[k] = v.(string)
		}
		addMondooAssetLabels(labels, &replicaset.obj.ObjectMeta, clusterId)
		platform, err := createPlatformData(replicaset.Kind.Data, conn.Runtime())
		if err != nil {
			return nil, err
		}
		assetList = append(assetList, &inventory.Asset{
			PlatformIds: []string{
				shared.NewWorkloadPlatformId(clusterId, "replicaset", replicaset.Namespace.Data, replicaset.Name.Data, replicaset.Uid.Data),
			},
			Name:        replicaset.Namespace.Data + "/" + replicaset.Name.Data,
			Platform:    platform,
			Labels:      labels,
			Connections: []*inventory.Config{invConfig.Clone(inventory.WithoutDiscovery())}, // pass-in the parent connection config
		})
		od.Add(replicaset.obj)
	}
	return assetList, nil
}

func discoverDaemonSets(
	conn shared.Connection,
	invConfig *inventory.Config,
	clusterId string,
	k8s *mqlK8s,
	od *PlatformIdOwnershipIndex,
	nsFilter NamespaceFilterOpts,
) ([]*inventory.Asset, error) {
	ds := k8s.GetDaemonsets()
	if ds.Error != nil {
		return nil, ds.Error
	}

	assetList := make([]*inventory.Asset, 0, len(ds.Data))
	for _, d := range ds.Data {
		daemonset := d.(*mqlK8sDaemonset)

		if skip := nsFilter.skipNamespace(daemonset.Namespace.Data); skip {
			continue
		}

		labels := map[string]string{}
		for k, v := range daemonset.GetLabels().Data {
			labels[k] = v.(string)
		}
		addMondooAssetLabels(labels, &daemonset.obj.ObjectMeta, clusterId)
		platform, err := createPlatformData(daemonset.Kind.Data, conn.Runtime())
		if err != nil {
			return nil, err
		}
		assetList = append(assetList, &inventory.Asset{
			PlatformIds: []string{
				shared.NewWorkloadPlatformId(clusterId, "daemonset", daemonset.Namespace.Data, daemonset.Name.Data, daemonset.Uid.Data),
			},
			Name:        daemonset.Namespace.Data + "/" + daemonset.Name.Data,
			Platform:    platform,
			Labels:      labels,
			Connections: []*inventory.Config{invConfig.Clone(inventory.WithoutDiscovery())}, // pass-in the parent connection config
		})
		od.Add(daemonset.obj)
	}
	return assetList, nil
}

func discoverAdmissionReviews(
	conn shared.Connection,
	invConfig *inventory.Config,
	clusterId string,
	k8s *mqlK8s,
	od *PlatformIdOwnershipIndex,
	nsFilter NamespaceFilterOpts,
) ([]*inventory.Asset, error) {
	admissionReviews, err := conn.AdmissionReviews()
	if err != nil {
		return nil, err
	}

	var assetList []*inventory.Asset
	for i := range admissionReviews {
		aReview := admissionReviews[i]

		asset, err := assetFromAdmissionReview(aReview, conn.Runtime(), invConfig, clusterId)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create asset from admission review")
		}

		log.Debug().Str("connection", asset.Connections[0].Host).Msg("resolved AdmissionReview")

		assetList = append(assetList, asset)
	}

	return assetList, nil
}

func discoverIngresses(
	conn shared.Connection,
	invConfig *inventory.Config,
	clusterId string,
	k8s *mqlK8s,
	od *PlatformIdOwnershipIndex,
	nsFilter NamespaceFilterOpts,
) ([]*inventory.Asset, error) {
	is := k8s.GetIngresses()
	if is.Error != nil {
		return nil, is.Error
	}

	assetList := make([]*inventory.Asset, 0, len(is.Data))
	for _, d := range is.Data {
		ingress := d.(*mqlK8sIngress)

		if skip := nsFilter.skipNamespace(ingress.Namespace.Data); skip {
			continue
		}

		labels := map[string]string{}
		for k, v := range ingress.GetLabels().Data {
			labels[k] = v.(string)
		}
		addMondooAssetLabels(labels, &ingress.obj.ObjectMeta, clusterId)
		platform, err := createPlatformData(ingress.Kind.Data, conn.Runtime())
		if err != nil {
			return nil, err
		}
		assetList = append(assetList, &inventory.Asset{
			PlatformIds: []string{
				shared.NewWorkloadPlatformId(clusterId, "ingress", ingress.Namespace.Data, ingress.Name.Data, ingress.Uid.Data),
			},
			Name:        ingress.Namespace.Data + "/" + ingress.Name.Data,
			Platform:    platform,
			Labels:      labels,
			Connections: []*inventory.Config{invConfig.Clone(inventory.WithoutDiscovery())}, // pass-in the parent connection config
		})
		od.Add(ingress.obj)
	}
	return assetList, nil
}

func discoverNamespaces(
	conn shared.Connection,
	invConfig *inventory.Config,
	clusterId string,
	od *PlatformIdOwnershipIndex,
	nsFilter NamespaceFilterOpts,
) ([]*inventory.Asset, error) {
	// We don't use MQL here since we need to handle k8s permission errors
	nss, err := conn.Namespaces()
	if err != nil {
		if k8sErrors.IsForbidden(err) && len(nsFilter.include) > 0 {
			for _, ns := range nsFilter.include {
				n, err := conn.Namespace(ns)
				if err != nil {
					return nil, err
				}
				nss = append(nss, *n)
			}
		} else {
			return nil, errors.Wrap(err, "failed to list namespaces")
		}
	}

	assetList := make([]*inventory.Asset, 0, len(nss))
	for _, ns := range nss {
		if skip := nsFilter.skipNamespace(ns.Name); skip {
			continue
		}

		labels := map[string]string{}
		for k, v := range ns.Labels {
			labels[k] = v
		}
		addMondooAssetLabels(labels, &ns.ObjectMeta, clusterId)
		platform, err := createPlatformData(ns.Kind, conn.Runtime())
		if err != nil {
			return nil, err
		}
		assetList = append(assetList, &inventory.Asset{
			PlatformIds: []string{
				shared.NewNamespacePlatformId(clusterId, ns.Name, string(ns.UID)),
			},
			Name:        ns.Name,
			Platform:    platform,
			Labels:      labels,
			Connections: []*inventory.Config{invConfig.Clone(inventory.WithoutDiscovery())}, // pass-in the parent connection config
		})
		if od != nil {
			od.Add(&ns)
		}
	}
	return assetList, nil
}

func discoverContainerImages(runtime *plugin.Runtime, invConfig *inventory.Config, clusterId string, k8s *mqlK8s, nsFilter NamespaceFilterOpts) ([]*inventory.Asset, error) {
	pods := k8s.GetPods()
	if pods.Error != nil {
		return nil, pods.Error
	}

	runningImages := make(map[string]ContainerImage)
	for _, p := range pods.Data {
		pod := p.(*mqlK8sPod)

		if skip := nsFilter.skipNamespace(pod.Namespace.Data); skip {
			continue
		}

		podImages := UniqueImagesForPod(*pod.obj, runtime)
		runningImages = types.MergeMaps(runningImages, podImages)
	}

	assetList := make([]*inventory.Asset, 0, len(runningImages))
	for _, i := range runningImages {
		assetList = append(assetList, &inventory.Asset{
			Connections: []*inventory.Config{
				{
					Type: "registry-image",
					Host: i.resolvedImage,
				},
			},
		})
	}

	return assetList, nil
}

func addMondooAssetLabels(assetLabels map[string]string, objMeta metav1.Object, clusterIdentifier string) {
	ns := objMeta.GetNamespace()
	if ns != "" {
		assetLabels["k8s.mondoo.com/namespace"] = ns
	}
	assetLabels["k8s.mondoo.com/name"] = objMeta.GetName()
	if string(objMeta.GetUID()) != "" {
		// objects discovered from manifest do not necessarily have a UID
		assetLabels["k8s.mondoo.com/uid"] = string(objMeta.GetUID())
	}
	objType, err := meta.TypeAccessor(objMeta)
	if err == nil {
		assetLabels["k8s.mondoo.com/kind"] = objType.GetKind()
		assetLabels["k8s.mondoo.com/apiVersion"] = objType.GetAPIVersion()
	}
	if objMeta.GetResourceVersion() != "" {
		// objects discovered from manifest do not necessarily have a resource version
		assetLabels["k8s.mondoo.com/resource-version"] = objMeta.GetResourceVersion()
	}
	assetLabels["k8s.mondoo.com/cluster-id"] = clusterIdentifier

	owners := objMeta.GetOwnerReferences()
	if len(owners) > 0 {
		owner := owners[0]
		assetLabels["k8s.mondoo.com/owner-kind"] = owner.Kind
		assetLabels["k8s.mondoo.com/owner-name"] = owner.Name
		assetLabels["k8s.mondoo.com/owner-uid"] = string(owner.UID)
	}
}

func assetFromAdmissionReview(a admissionv1.AdmissionReview, runtime string, connection *inventory.Config, clusterIdentifier string) (*inventory.Asset, error) {
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
	platformData, err := createPlatformData(a.Kind, runtime)
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
	} else {
		name = objMeta.GetName()
	}

	addMondooAssetLabels(assetLabels, objMeta, clusterIdentifier)

	asset := &inventory.Asset{
		PlatformIds: []string{shared.NewWorkloadPlatformId(clusterIdentifier, strings.ToLower(objectKind), objMeta.GetNamespace(), objMeta.GetName(), string(objMeta.GetUID()))},
		Name:        name,
		Platform:    platformData,
		Connections: []*inventory.Config{connection},
		State:       inventory.State_STATE_ONLINE,
		Labels:      assetLabels,
	}

	return asset, nil
}

func createPlatformData(objectKind, runtime string) (*inventory.Platform, error) {
	platformData := &inventory.Platform{
		Family:  []string{"k8s"},
		Kind:    "k8s-object",
		Runtime: runtime,
	}

	switch objectKind {
	case "Node":
		platformData.Name = "k8s-node"
		platformData.Title = "Kubernetes Node"
	case "Pod":
		platformData.Family = append(platformData.Family, "k8s-workload")
		platformData.Name = "k8s-pod"
		platformData.Title = "Kubernetes Pod"
	case "CronJob":
		platformData.Family = append(platformData.Family, "k8s-workload")
		platformData.Name = "k8s-cronjob"
		platformData.Title = "Kubernetes CronJob"
	case "StatefulSet":
		platformData.Family = append(platformData.Family, "k8s-workload")
		platformData.Name = "k8s-statefulset"
		platformData.Title = "Kubernetes StatefulSet"
	case "Deployment":
		platformData.Family = append(platformData.Family, "k8s-workload")
		platformData.Name = "k8s-deployment"
		platformData.Title = "Kubernetes Deployment"
	case "Job":
		platformData.Family = append(platformData.Family, "k8s-workload")
		platformData.Name = "k8s-job"
		platformData.Title = "Kubernetes Job"
	case "ReplicaSet":
		platformData.Family = append(platformData.Family, "k8s-workload")
		platformData.Name = "k8s-replicaset"
		platformData.Title = "Kubernetes ReplicaSet"
	case "DaemonSet":
		platformData.Family = append(platformData.Family, "k8s-workload")
		platformData.Name = "k8s-daemonset"
		platformData.Title = "Kubernetes DaemonSet"
	case "AdmissionReview":
		platformData.Family = append(platformData.Family, "k8s-admission")
		platformData.Name = "k8s-admission"
		platformData.Title = "Kubernetes Admission Review"
	case "Ingress":
		platformData.Family = append(platformData.Family, "k8s-ingress")
		platformData.Name = "k8s-ingress"
		platformData.Title = "Kubernetes Ingress"
	case "Namespace":
		platformData.Family = append(platformData.Family, "k8s-namespace")
		platformData.Name = "k8s-namespace"
		platformData.Title = "Kubernetes Namespace"
	default:
		return nil, fmt.Errorf("could not determine object kind %s", objectKind)
	}
	return platformData, nil
}

func setRelatedAssets(conn shared.Connection, root *inventory.Asset, assets []*inventory.Asset, od *PlatformIdOwnershipIndex) {
	// everything is connected to the root asset
	root.RelatedAssets = append(root.RelatedAssets, assets...)

	// build a lookup on the k8s uid to look up individual assets to link
	platformIdToAssetMap := map[string]*inventory.Asset{}
	for _, a := range assets {
		for _, platformId := range a.PlatformIds {
			platformIdToAssetMap[platformId] = a
		}
	}

	for id, a := range platformIdToAssetMap {
		ownedBy := od.OwnedBy(id)
		for _, ownerPlatformId := range ownedBy {
			if aa, ok := platformIdToAssetMap[ownerPlatformId]; ok {
				a.RelatedAssets = append(a.RelatedAssets, aa)
			} else {
				// If the owner object is not scanned we can still add an asset as we know most of the information
				// from the ownerReference field
				if platformEntry, ok := od.GetKubernetesObjectData(ownerPlatformId); ok {
					platformData, err := createPlatformData(platformEntry.Kind, conn.Runtime())
					if err != nil {
						continue
					}
					a.RelatedAssets = append(a.RelatedAssets, &inventory.Asset{
						PlatformIds: []string{ownerPlatformId},
						Platform:    platformData,
						Name:        platformEntry.Namespace + "/" + platformEntry.Name,
					})
				}
			}
		}
	}
}
