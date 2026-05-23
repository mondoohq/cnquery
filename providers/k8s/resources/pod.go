// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"sync"
	"time"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/k8s/connection/shared/resources"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type mqlK8sPodInternal struct {
	lock sync.Mutex
	obj  runtime.Object
}

func (k *mqlK8sPod) getPod() (*corev1.Pod, error) {
	p, ok := k.obj.(*corev1.Pod)
	if ok {
		return p, nil
	}
	return nil, errors.New("invalid k8s pod")
}

func (k *mqlK8s) pods() ([]any, error) {
	return k8sResourceToMql(k.MqlRuntime, gvkString(corev1.SchemeGroupVersion.WithKind("pods")), func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (any, error) {
		ts := obj.GetCreationTimestamp()

		r, err := CreateResource(k.MqlRuntime, "k8s.pod", map[string]*llx.RawData{
			"id":              llx.StringData(objIdFromK8sObj(obj, objT)),
			"uid":             llx.StringData(string(obj.GetUID())),
			"resourceVersion": llx.StringData(obj.GetResourceVersion()),
			"name":            llx.StringData(obj.GetName()),
			"namespace":       llx.StringData(obj.GetNamespace()),
			"apiVersion":      llx.StringData(objT.GetAPIVersion()),
			"kind":            llx.StringData(objT.GetKind()),
			"created":         llx.TimeData(ts.Time),
		})
		if err != nil {
			return nil, err
		}

		r.(*mqlK8sPod).obj = resource
		return r, nil
	})
}

func (k *mqlK8sPod) manifest() (map[string]any, error) {
	manifest, err := convert.JsonToDict(k.obj)
	if err != nil {
		return nil, err
	}
	return manifest, nil
}

func (k *mqlK8sPod) podSpec() (map[string]any, error) {
	pod, err := k.getPod()
	if err != nil {
		return nil, err
	}
	podSpec, err := resources.GetPodSpec(pod)
	if err != nil {
		return nil, err
	}
	dict, err := convert.JsonToDict(podSpec)
	if err != nil {
		return nil, err
	}
	return dict, nil
}

func (k *mqlK8sPod) id() (string, error) {
	return k.Id.Data, nil
}

func initK8sPod(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initNamespacedResource[*mqlK8sPod](runtime, args, func(k *mqlK8s) *plugin.TValue[[]any] { return k.GetPods() })
}

func (k *mqlK8sPod) initContainers() ([]any, error) {
	pod, err := k.getPod()
	if err != nil {
		return nil, err
	}
	return getContainers(pod, &pod.ObjectMeta, k.MqlRuntime, InitContainerType)
}

func (k *mqlK8sPod) ephemeralContainers() ([]any, error) {
	pod, err := k.getPod()
	if err != nil {
		return nil, err
	}
	return getContainers(pod, &pod.ObjectMeta, k.MqlRuntime, EphemeralContainerType)
}

func (k *mqlK8sPod) containers() ([]any, error) {
	pod, err := k.getPod()
	if err != nil {
		return nil, err
	}
	return getContainers(pod, &pod.ObjectMeta, k.MqlRuntime, ContainerContainerType)
}

func (k *mqlK8sPod) containerStatuses() ([]any, error) {
	pod, err := k.getPod()
	if err != nil {
		return nil, err
	}

	resp := []any{}
	for _, c := range pod.Status.ContainerStatuses {
		args := map[string]*llx.RawData{
			"__id":         llx.StringData(string(pod.GetUID()) + "-containerstatus-" + c.Name),
			"name":         llx.StringData(c.Name),
			"ready":        llx.BoolData(c.Ready),
			"restartCount": llx.IntData(int64(c.RestartCount)),
			"image":        llx.StringData(c.Image),
			"imageId":      llx.StringData(c.ImageID),
			"containerId":  llx.StringData(c.ContainerID),
		}
		mqlContainer, err := CreateResource(k.MqlRuntime, ResourceK8sContainerStatus, args)
		if err != nil {
			return nil, err
		}
		resp = append(resp, mqlContainer)
	}
	return resp, nil
}

func (k *mqlK8sPod) annotations() (map[string]any, error) {
	pod, err := k.getPod()
	if err != nil {
		return nil, err
	}
	return convert.MapToInterfaceMap(pod.GetAnnotations()), nil
}

func (k *mqlK8sPod) labels() (map[string]any, error) {
	pod, err := k.getPod()
	if err != nil {
		return nil, err
	}
	return convert.MapToInterfaceMap(pod.GetLabels()), nil
}

func (k *mqlK8sPod) node() (*mqlK8sNode, error) {
	pod, err := k.getPod()
	if err != nil {
		return nil, err
	}
	podSpec, err := resources.GetPodSpec(pod)
	if err != nil {
		return nil, err
	}

	node, err := NewResource(k.MqlRuntime, "k8s.node", map[string]*llx.RawData{
		"name": llx.StringData(podSpec.NodeName),
	})
	if err != nil {
		return nil, err
	}

	return node.(*mqlK8sNode), nil
}

func (k *mqlK8sPod) podSpecTyped() (*corev1.PodSpec, error) {
	pod, err := k.getPod()
	if err != nil {
		return nil, err
	}
	return &pod.Spec, nil
}

func (k *mqlK8sPod) nodeName() (string, error) {
	spec, err := k.podSpecTyped()
	if err != nil {
		return "", err
	}
	return spec.NodeName, nil
}

func (k *mqlK8sPod) nodeSelector() (map[string]any, error) {
	spec, err := k.podSpecTyped()
	if err != nil {
		return nil, err
	}
	return convert.MapToInterfaceMap(spec.NodeSelector), nil
}

func (k *mqlK8sPod) tolerations() ([]any, error) {
	spec, err := k.podSpecTyped()
	if err != nil {
		return nil, err
	}
	return convert.JsonToDictSlice(spec.Tolerations)
}

func (k *mqlK8sPod) topologySpreadConstraints() ([]any, error) {
	spec, err := k.podSpecTyped()
	if err != nil {
		return nil, err
	}
	return convert.JsonToDictSlice(spec.TopologySpreadConstraints)
}

func (k *mqlK8sPod) affinity() (map[string]any, error) {
	spec, err := k.podSpecTyped()
	if err != nil {
		return nil, err
	}
	return convert.JsonToDict(spec.Affinity)
}

func (k *mqlK8sPod) priorityClassName() (string, error) {
	spec, err := k.podSpecTyped()
	if err != nil {
		return "", err
	}
	return spec.PriorityClassName, nil
}

func (k *mqlK8sPod) priorityClass() (*mqlK8sPriorityclass, error) {
	spec, err := k.podSpecTyped()
	if err != nil {
		return nil, err
	}
	if spec.PriorityClassName == "" {
		k.PriorityClass.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	pc, err := NewResource(k.MqlRuntime, "k8s.priorityclass", map[string]*llx.RawData{
		"name": llx.StringData(spec.PriorityClassName),
	})
	if err != nil {
		return nil, err
	}
	return pc.(*mqlK8sPriorityclass), nil
}

func (k *mqlK8sPod) preemptionPolicy() (string, error) {
	spec, err := k.podSpecTyped()
	if err != nil {
		return "", err
	}
	if spec.PreemptionPolicy == nil {
		return "", nil
	}
	return string(*spec.PreemptionPolicy), nil
}

func (k *mqlK8sPod) schedulerName() (string, error) {
	spec, err := k.podSpecTyped()
	if err != nil {
		return "", err
	}
	return spec.SchedulerName, nil
}

func (k *mqlK8sPod) runtimeClassName() (string, error) {
	spec, err := k.podSpecTyped()
	if err != nil {
		return "", err
	}
	if spec.RuntimeClassName == nil {
		return "", nil
	}
	return *spec.RuntimeClassName, nil
}

func (k *mqlK8sPod) serviceAccountName() (string, error) {
	spec, err := k.podSpecTyped()
	if err != nil {
		return "", err
	}
	return spec.ServiceAccountName, nil
}

func (k *mqlK8sPod) serviceAccount() (*mqlK8sServiceaccount, error) {
	pod, err := k.getPod()
	if err != nil {
		return nil, err
	}
	if pod.Spec.ServiceAccountName == "" {
		k.ServiceAccount.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	sa, err := NewResource(k.MqlRuntime, "k8s.serviceaccount", map[string]*llx.RawData{
		"name":      llx.StringData(pod.Spec.ServiceAccountName),
		"namespace": llx.StringData(pod.Namespace),
	})
	if err != nil {
		return nil, err
	}
	return sa.(*mqlK8sServiceaccount), nil
}

func (k *mqlK8sPod) automountServiceAccountToken() (bool, error) {
	spec, err := k.podSpecTyped()
	if err != nil {
		return false, err
	}
	if spec.AutomountServiceAccountToken == nil {
		// Defaults to true when unset.
		return true, nil
	}
	return *spec.AutomountServiceAccountToken, nil
}

func (k *mqlK8sPod) hostNetwork() (bool, error) {
	spec, err := k.podSpecTyped()
	if err != nil {
		return false, err
	}
	return spec.HostNetwork, nil
}

func (k *mqlK8sPod) hostPID() (bool, error) {
	spec, err := k.podSpecTyped()
	if err != nil {
		return false, err
	}
	return spec.HostPID, nil
}

func (k *mqlK8sPod) hostIPC() (bool, error) {
	spec, err := k.podSpecTyped()
	if err != nil {
		return false, err
	}
	return spec.HostIPC, nil
}

func (k *mqlK8sPod) shareProcessNamespace() (bool, error) {
	spec, err := k.podSpecTyped()
	if err != nil {
		return false, err
	}
	if spec.ShareProcessNamespace == nil {
		return false, nil
	}
	return *spec.ShareProcessNamespace, nil
}

func (k *mqlK8sPod) securityContext() (map[string]any, error) {
	spec, err := k.podSpecTyped()
	if err != nil {
		return nil, err
	}
	return convert.JsonToDict(spec.SecurityContext)
}

func (k *mqlK8sPod) dnsPolicy() (string, error) {
	spec, err := k.podSpecTyped()
	if err != nil {
		return "", err
	}
	return string(spec.DNSPolicy), nil
}

func (k *mqlK8sPod) dnsConfig() (map[string]any, error) {
	spec, err := k.podSpecTyped()
	if err != nil {
		return nil, err
	}
	return convert.JsonToDict(spec.DNSConfig)
}

func (k *mqlK8sPod) hostname() (string, error) {
	spec, err := k.podSpecTyped()
	if err != nil {
		return "", err
	}
	return spec.Hostname, nil
}

func (k *mqlK8sPod) subdomain() (string, error) {
	spec, err := k.podSpecTyped()
	if err != nil {
		return "", err
	}
	return spec.Subdomain, nil
}

func (k *mqlK8sPod) hostAliases() ([]any, error) {
	spec, err := k.podSpecTyped()
	if err != nil {
		return nil, err
	}
	return convert.JsonToDictSlice(spec.HostAliases)
}

func (k *mqlK8sPod) restartPolicy() (string, error) {
	spec, err := k.podSpecTyped()
	if err != nil {
		return "", err
	}
	return string(spec.RestartPolicy), nil
}

func (k *mqlK8sPod) terminationGracePeriodSeconds() (int64, error) {
	spec, err := k.podSpecTyped()
	if err != nil {
		return 0, err
	}
	if spec.TerminationGracePeriodSeconds == nil {
		return 0, nil
	}
	return *spec.TerminationGracePeriodSeconds, nil
}

func (k *mqlK8sPod) activeDeadlineSeconds() (int64, error) {
	spec, err := k.podSpecTyped()
	if err != nil {
		return 0, err
	}
	if spec.ActiveDeadlineSeconds == nil {
		return 0, nil
	}
	return *spec.ActiveDeadlineSeconds, nil
}

func (k *mqlK8sPod) readinessGates() ([]any, error) {
	spec, err := k.podSpecTyped()
	if err != nil {
		return nil, err
	}
	return convert.JsonToDictSlice(spec.ReadinessGates)
}

func (k *mqlK8sPod) enableServiceLinks() (bool, error) {
	spec, err := k.podSpecTyped()
	if err != nil {
		return false, err
	}
	if spec.EnableServiceLinks == nil {
		// Defaults to true when unset.
		return true, nil
	}
	return *spec.EnableServiceLinks, nil
}

func (k *mqlK8sPod) imagePullSecrets() ([]any, error) {
	spec, err := k.podSpecTyped()
	if err != nil {
		return nil, err
	}
	return convert.JsonToDictSlice(spec.ImagePullSecrets)
}

func (k *mqlK8sPod) overhead() (map[string]any, error) {
	spec, err := k.podSpecTyped()
	if err != nil {
		return nil, err
	}
	if spec.Overhead == nil {
		return map[string]any{}, nil
	}
	out := make(map[string]any, len(spec.Overhead))
	for name, qty := range spec.Overhead {
		out[string(name)] = qty.String()
	}
	return out, nil
}

func (k *mqlK8sPod) os() (map[string]any, error) {
	spec, err := k.podSpecTyped()
	if err != nil {
		return nil, err
	}
	return convert.JsonToDict(spec.OS)
}

func (k *mqlK8sPod) phase() (string, error) {
	pod, err := k.getPod()
	if err != nil {
		return "", err
	}
	return string(pod.Status.Phase), nil
}

func (k *mqlK8sPod) qosClass() (string, error) {
	pod, err := k.getPod()
	if err != nil {
		return "", err
	}
	return string(pod.Status.QOSClass), nil
}

func (k *mqlK8sPod) podIP() (string, error) {
	pod, err := k.getPod()
	if err != nil {
		return "", err
	}
	return pod.Status.PodIP, nil
}

func (k *mqlK8sPod) podIPs() ([]any, error) {
	pod, err := k.getPod()
	if err != nil {
		return nil, err
	}
	out := make([]any, len(pod.Status.PodIPs))
	for i, ip := range pod.Status.PodIPs {
		out[i] = ip.IP
	}
	return out, nil
}

func (k *mqlK8sPod) hostIP() (string, error) {
	pod, err := k.getPod()
	if err != nil {
		return "", err
	}
	return pod.Status.HostIP, nil
}

func (k *mqlK8sPod) nominatedNodeName() (string, error) {
	pod, err := k.getPod()
	if err != nil {
		return "", err
	}
	return pod.Status.NominatedNodeName, nil
}

func (k *mqlK8sPod) startTime() (*time.Time, error) {
	pod, err := k.getPod()
	if err != nil {
		return nil, err
	}
	if pod.Status.StartTime == nil {
		k.StartTime.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	t := pod.Status.StartTime.Time
	return &t, nil
}

func (k *mqlK8sPod) conditions() ([]any, error) {
	pod, err := k.getPod()
	if err != nil {
		return nil, err
	}
	return convert.JsonToDictSlice(pod.Status.Conditions)
}

func (k *mqlK8sPod) reason() (string, error) {
	pod, err := k.getPod()
	if err != nil {
		return "", err
	}
	return pod.Status.Reason, nil
}

func (k *mqlK8sPod) message() (string, error) {
	pod, err := k.getPod()
	if err != nil {
		return "", err
	}
	return pod.Status.Message, nil
}
