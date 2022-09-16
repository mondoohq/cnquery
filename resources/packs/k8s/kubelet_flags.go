// copyright: 2019, Dominik Richter and Christoph Hartmann
// author: Dominik Richter
// author: Christoph Hartmann

package k8s

import (
	"strings"

	"go.mondoo.com/cnquery/resources/packs/core"
)

// parseFlagsIntoConfig adds flags to the kubelet config
// It does not take care of deprecated flags
// The flags are not just kubelet specific, but can also be global flags
// That also means, that some flags do not have a matching parameter in the kubelet config file and are added as is
// The list of flags is taken from
// /var/lib/minikube/binaries/v1.24.3/kubelet --help | grep -v DEPRECATED | grep -v -E "(vmodule|version|help|v level)"
func parseFlagsIntoConfig(kubeletConfig map[string]interface{}, flags map[string]interface{}) error {
	cliOnlyFlags := []string{
		"azure-container-registry-config",
		"bootstrap-kubeconfig",
		"cert-dir",
		"config",
		"container-runtime-endpoint",
		"exit-on-lock-contention",
		"hostname-override",
		"housekeeping-interval",
		"image-credential-provider-bin-dir",
		"image-credential-provider-config",
		"image-service-endpoint",
		"kubeconfig",
		"lock-file",
		"log-flush-frequency",
		"logging-format",
		"node-ip",
		"root-dir",
		"runtime-cgroups",
		"seccomp-default",
	}
	for _, key := range cliOnlyFlags {
		if _, ok := flags[key]; ok {
			kubeletConfig[key] = flags[key]
		}
	}

	if _, ok := flags["node-labels"]; ok {
		nodeLabels := map[string]string{}
		for _, label := range strings.Split(flags["node-labels"].(string), ",") {
			labelSplit := strings.Split(label, "=")
			nodeLabels[labelSplit[0]] = labelSplit[1]
		}
		data, err := core.JsonToDict(nodeLabels)
		if err != nil {
			return err
		}
		kubeletConfig["node-labels"] = data
	}

	return nil
}

// parseDeprecatedFlagsIntoConfig merges deprecated cli flags into the kubelet config
// It only takes care of deprecated flags.
// This is a seperate functzion in hope we can get rid of it in the future
// The list of flags is taken from
// https://github.com/kubernetes/kubernetes/blob/release-1.25/cmd/kubelet/app/options/options.go
func parseDeprecatedFlagsIntoConfig(kubeletConfig map[string]interface{}, flags map[string]interface{}) error {
	if _, ok := flags["enable-server"]; ok {
		kubeletConfig["enableServer"] = flags["enable-server"]
	}
	if _, ok := flags["fail-swap-on"]; ok {
		kubeletConfig["failSwapOn"] = flags["fail-swap-on"]
	}
	if _, ok := flags["pod-manifest-path"]; ok {
		kubeletConfig["staticPodPath"] = flags["pod-manifest-path"]
	}
	if _, ok := flags["sync-frequency"]; ok {
		kubeletConfig["syncFrequency"] = flags["sync-frequency"]
	}
	if _, ok := flags["file-check-frequency"]; ok {
		kubeletConfig["fileCheckFrequency"] = flags["file-check-frequency"]
	}
	if _, ok := flags["http-check-frequency"]; ok {
		kubeletConfig["httpCheckFrequency"] = flags["http-check-frequency"]
	}
	if _, ok := flags["manifest-url"]; ok {
		kubeletConfig["staticPodURL"] = flags["manifest-url"]
	}
	if _, ok := flags["manifest-url-header"]; ok {
		urlHeaders := map[string]string{}
		for _, urlHeader := range strings.Split(flags["manifest-url-header"].(string), ",") {
			urlHeaderSplit := strings.Split(urlHeader, "=")
			urlHeaders[urlHeaderSplit[0]] = urlHeaderSplit[1]
		}
		data, err := core.JsonToDict(urlHeaders)
		if err != nil {
			return err
		}
		kubeletConfig["staticPodURLHeader"] = data
	}
	if _, ok := flags["address"]; ok {
		kubeletConfig["address"] = flags["address"]
	}
	if _, ok := flags["port"]; ok {
		kubeletConfig["port"] = flags["port"]
	}
	if _, ok := flags["read-only-port"]; ok {
		kubeletConfig["readOnlyPort"] = flags["read-only-port"]
	}
	if _, ok := flags["anonymous-auth"]; ok {
		auth := kubeletConfig["authentication"].(map[string]interface{})
		anon := auth["anonymous"].(map[string]interface{})
		anon["enabled"] = flags["anonymous-auth"]
	}
	if _, ok := flags["authentication-token-webhook"]; ok {
		auth := kubeletConfig["authentication"].(map[string]interface{})
		webhook := auth["webhook"].(map[string]interface{})
		webhook["enabled"] = flags["authentication-token-webhook"]
	}
	if _, ok := flags["authentication-token-webhook-cache-ttl"]; ok {
		auth := kubeletConfig["authentication"].(map[string]interface{})
		webhook := auth["webhook"].(map[string]interface{})
		webhook["cacheTTL"] = flags["authentication-token-webhook-cache-ttl"].(string)
		kubeletConfig["authentication"] = auth
	}
	if _, ok := flags["client-ca-file"]; ok {
		authz := kubeletConfig["authorization"].(map[string]interface{})
		x509 := authz["x509"].(map[string]interface{})
		x509["clientCAFile"] = flags["client-ca-file"]
		kubeletConfig["authorization"] = authz
	}
	if _, ok := flags["authorization-mode"]; ok {
		authz := kubeletConfig["authorization"].(map[string]interface{})
		authz["mode"] = flags["authorization-mode"]
		kubeletConfig["authorization"] = authz
	}
	if _, ok := flags["authorization-webhook-cache-authorized-ttl"]; ok {
		authz := kubeletConfig["authorization"].(map[string]interface{})
		webhook := authz["webhook"].(map[string]interface{})
		webhook["cacheAuthorizedTTL"] = flags["authorization-webhook-cache-authorized-ttl"]
		kubeletConfig["authorization"] = authz
	}
	if _, ok := flags["authorization-webhook-cache-unauthorized-ttl"]; ok {
		authz := kubeletConfig["authorization"].(map[string]interface{})
		webhook := authz["webhook"].(map[string]interface{})
		webhook["cacheUnauthorizedTTL"] = flags["authorization-webhook-cache-unauthorized-ttl"]
		kubeletConfig["authorization"] = authz
	}
	if _, ok := flags["tls-cert-file"]; ok {
		kubeletConfig["tlsCertFile"] = flags["tls-cert-file"]
	}
	if _, ok := flags["tls-private-key-file"]; ok {
		kubeletConfig["tlsPrivateKeyFile"] = flags["tls-private-key-file"]
	}
	if _, ok := flags["rotate-server-certificates"]; ok {
		kubeletConfig["serverTLSBootstrap"] = flags["rotate-server-certificates"]
	}
	if _, ok := flags["tls-cipher-suites"]; ok {
		ciphers := strings.Split(flags["tls-cipher-suites"].(string), ",")
		data, err := core.JsonToDictSlice(ciphers)
		if err != nil {
			return err
		}
		kubeletConfig["tlsCipherSuites"] = data
	}
	if _, ok := flags["tls-min-version"]; ok {
		kubeletConfig["tlsMinVersion"] = flags["tls-min-version"]
	}
	if _, ok := flags["rotate-certificates"]; ok {
		kubeletConfig["rotateCertificates"] = flags["rotate-certificates"]
	}
	if _, ok := flags["registry-qps"]; ok {
		kubeletConfig["registryPullQPS"] = flags["registry-qps"]
	}
	if _, ok := flags["registry-burst"]; ok {
		kubeletConfig["registryBurst"] = flags["registry-burst"]
	}
	if _, ok := flags["event-qps"]; ok {
		kubeletConfig["eventRecordQPS"] = flags["event-qps"]
	}
	if _, ok := flags["event-burst"]; ok {
		kubeletConfig["eventBurst"] = flags["event-burst"]
	}
	if _, ok := flags["enable-debugging-handlers"]; ok {
		kubeletConfig["enableDebuggingHandlers"] = flags["enable-debugging-handlers"]
	}
	if _, ok := flags["contention-profiling"]; ok {
		kubeletConfig["enableContentionProfiling"] = flags["contention-profiling"]
	}
	if _, ok := flags["healthz-port"]; ok {
		kubeletConfig["healthzPort"] = flags["healthz-port"]
	}
	if _, ok := flags["healthz-bind-address"]; ok {
		kubeletConfig["healthzBindAddress"] = flags["healthz-bind-address"]
	}
	if _, ok := flags["oom-score-adj"]; ok {
		kubeletConfig["oomScoreAdj"] = flags["oom-score-adj"]
	}
	if _, ok := flags["cluster-domain"]; ok {
		kubeletConfig["clusterDomain"] = flags["cluster-domain"]
	}
	if _, ok := flags["volume-plugin-dir"]; ok {
		kubeletConfig["volumePluginDir"] = flags["volume-plugin-dir"]
	}
	if _, ok := flags["cluster-dns"]; ok {
		kubeletConfig["clusterDNS"] = flags["cluster-dns"]
	}
	if _, ok := flags["streaming-connection-idle-timeout"]; ok {
		kubeletConfig["streamingConnectionIdleTimeout"] = flags["streaming-connection-idle-timeout"]
	}
	if _, ok := flags["node-status-update-frequency"]; ok {
		kubeletConfig["nodeStatusUpdateFrequency"] = flags["node-status-update-frequency"]
	}
	if _, ok := flags["minimum-image-ttl-duration"]; ok {
		kubeletConfig["imageMinimumGCAge"] = flags["minimum-image-ttl-duration"]
	}
	if _, ok := flags["image-gc-high-threshold"]; ok {
		kubeletConfig["imageGCHighThresholdPercent"] = flags["image-gc-high-threshold"]
	}
	if _, ok := flags["image-gc-low-threshold"]; ok {
		kubeletConfig["imageGCLowThresholdPercent"] = flags["image-gc-low-threshold"]
	}
	if _, ok := flags["volume-stats-agg-period"]; ok {
		kubeletConfig["volumeStatsAggPeriod"] = flags["volume-stats-agg-period"]
	}
	if _, ok := flags["feature-gates"]; ok {
		featureFlags := map[string]string{}
		for _, feature := range strings.Split(flags["feature-gates"].(string), ",") {
			featureSplit := strings.Split(feature, "=")
			featureFlags[featureSplit[0]] = featureSplit[1]
		}
		data, err := core.JsonToDict(featureFlags)
		if err != nil {
			return err
		}
		kubeletConfig["featureGates"] = data
	}
	if _, ok := flags["kubelet-cgroups"]; ok {
		kubeletConfig["kubeletCgroups"] = flags["kubelet-cgroups"]
	}
	if _, ok := flags["system-cgroups"]; ok {
		kubeletConfig["systemCgroups"] = flags["system-cgroups"]
	}
	if _, ok := flags["provider-id"]; ok {
		kubeletConfig["providerID"] = flags["provider-id"]
	}
	if _, ok := flags["cgroups-per-qos"]; ok {
		kubeletConfig["cgroupsPerQOS"] = flags["cgroups-per-qos"]
	}
	if _, ok := flags["cgroup-driver"]; ok {
		kubeletConfig["cgroupDriver"] = flags["cgroup-driver"]
	}
	if _, ok := flags["cgroup-root"]; ok {
		kubeletConfig["cgroupRoot"] = flags["cgroup-root"]
	}
	if _, ok := flags["cpu-manager-policy"]; ok {
		kubeletConfig["cpuManagerPolicy"] = flags["cpu-manager-policy"]
	}
	if _, ok := flags["cpu-manager-policy-options"]; ok {
		cpuPolicies := map[string]string{}
		for _, cpuPolicy := range strings.Split(flags["cpu-manager-policy-options"].(string), ",") {
			cpuPolicySplit := strings.Split(cpuPolicy, "=")
			cpuPolicies[cpuPolicySplit[0]] = cpuPolicySplit[1]
		}
		data, err := core.JsonToDict(cpuPolicies)
		if err != nil {
			return err
		}
		kubeletConfig["cpuManagerPolicyOptions"] = data
	}
	if _, ok := flags["cpu-manager-reconcile-period"]; ok {
		kubeletConfig["cpuManagerReconcilePeriod"] = flags["cpu-manager-reconcile-period"]
	}
	if _, ok := flags["qos-reserved"]; ok {
		qosReserved := map[string]string{}
		for _, qosReserve := range strings.Split(flags["qos-reserved"].(string), ",") {
			qosReserveSplit := strings.Split(qosReserve, "=")
			qosReserved[qosReserveSplit[0]] = qosReserveSplit[1]
		}
		data, err := core.JsonToDict(qosReserved)
		if err != nil {
			return err
		}
		kubeletConfig["qosReserved"] = data
	}
	if _, ok := flags["topology-manager-policy"]; ok {
		kubeletConfig["topologyManagerPolicy"] = flags["topology-manager-policy"]
	}
	if _, ok := flags["runtime-request-timeout"]; ok {
		kubeletConfig["runtimeRequestTimeout"] = flags["runtime-request-timeout"]
	}
	if _, ok := flags["hairpin-mode"]; ok {
		kubeletConfig["hairpinMode"] = flags["hairpin-mode"]
	}
	if _, ok := flags["max-pods"]; ok {
		kubeletConfig["maxPods"] = flags["max-pods"]
	}
	if _, ok := flags["pod-cidr"]; ok {
		kubeletConfig["podCIDR"] = flags["pod-cidr"]
	}
	if _, ok := flags["pod-max-pids"]; ok {
		kubeletConfig["podPidsLimit"] = flags["pod-max-pids"]
	}
	if _, ok := flags["resolv-conf"]; ok {
		kubeletConfig["resolverConfig"] = flags["resolv-conf"]
	}
	if _, ok := flags["runonce"]; ok {
		kubeletConfig["runOnce"] = flags["runonce"]
	}
	if _, ok := flags["cpu-cfs-quota"]; ok {
		kubeletConfig["cpuCFSQuota"] = flags["cpu-cfs-quota"]
	}
	if _, ok := flags["cpu-cfs-quota-period"]; ok {
		kubeletConfig["cpuCFSQuotaPeriod"] = flags["cpu-cfs-quota-period"]
	}
	if _, ok := flags["enable-controller-attach-detach"]; ok {
		kubeletConfig["enableControllerAttachDetach"] = flags["enable-controller-attach-detach"]
	}
	if _, ok := flags["make-iptables-util-chains"]; ok {
		kubeletConfig["makeIPTablesUtilChains"] = flags["make-iptables-util-chains"]
	}
	if _, ok := flags["iptables-masquerade-bit"]; ok {
		kubeletConfig["iptablesMasqueradeBit"] = flags["iptables-masquerade-bit"]
	}
	if _, ok := flags["iptables-drop-bit"]; ok {
		kubeletConfig["iptablesDropBit"] = flags["iptables-drop-bit"]
	}
	if _, ok := flags["container-log-max-size"]; ok {
		kubeletConfig["containerLogMaxSize"] = flags["container-log-max-size"]
	}
	if _, ok := flags["container-log-max-files"]; ok {
		kubeletConfig["containerLogMaxFiles"] = flags["container-log-max-files"]
	}
	if _, ok := flags["allowed-unsafe-sysctls"]; ok {
		kubeletConfig["allowedUnsafeSysctls"] = flags["allowed-unsafe-sysctls"]
	}
	if _, ok := flags["node-status-max-images"]; ok {
		kubeletConfig["nodeStatusMaxImages"] = flags["node-status-max-images"]
	}
	if _, ok := flags["kernel-memcg-notification"]; ok {
		kubeletConfig["kernelMemcgNotification"] = flags["kernel-memcg-notification"]
	}
	if _, ok := flags["local-storage-capacity-isolation"]; ok {
		kubeletConfig["localStorageCapacityIsolation"] = flags["local-storage-capacity-isolation"]
	}
	if _, ok := flags["max-open-files"]; ok {
		kubeletConfig["maxOpenFiles"] = flags["max-open-files"]
	}
	if _, ok := flags["kube-api-content-type"]; ok {
		kubeletConfig["contentType"] = flags["kube-api-content-type"]
	}
	if _, ok := flags["kube-api-qps"]; ok {
		kubeletConfig["kubeAPIQPS"] = flags["kube-api-qps"]
	}
	if _, ok := flags["kube-api-burst"]; ok {
		kubeletConfig["kubeAPIBurst"] = flags["kube-api-burst"]
	}
	if _, ok := flags["serialize-image-pulls"]; ok {
		kubeletConfig["serializeImagePulls"] = flags["serialize-image-pulls"]
	}
	if _, ok := flags["eviction-hard"]; ok {
		evictions := map[string]string{}
		for _, eviction := range strings.Split(flags["eviction-hard"].(string), ",") {
			evictionSplit := strings.Split(eviction, "=")
			evictions[evictionSplit[0]] = evictionSplit[1]
		}
		data, err := core.JsonToDict(evictions)
		if err != nil {
			return err
		}
		kubeletConfig["evictionHard"] = data
	}
	if _, ok := flags["eviction-soft"]; ok {
		evictions := map[string]string{}
		for _, eviction := range strings.Split(flags["eviction-soft"].(string), ",") {
			evictionSplit := strings.Split(eviction, "=")
			evictions[evictionSplit[0]] = evictionSplit[1]
		}
		data, err := core.JsonToDict(evictions)
		if err != nil {
			return err
		}
		kubeletConfig["evictionSoft"] = data
	}
	if _, ok := flags["eviction-soft-grace-period"]; ok {
		softPeriods := map[string]string{}
		for _, softPeriod := range strings.Split(flags["eviction-soft-grace-period"].(string), ",") {
			softPeriodSplit := strings.Split(softPeriod, "=")
			softPeriods[softPeriodSplit[0]] = softPeriodSplit[1]
		}
		data, err := core.JsonToDict(softPeriods)
		if err != nil {
			return err
		}
		kubeletConfig["evictionSoftGracePeriod"] = data
	}
	if _, ok := flags["eviction-pressure-transition-period"]; ok {
		kubeletConfig["evictionPressureTransitionPeriod"] = flags["eviction-pressure-transition-period"]
	}
	if _, ok := flags["eviction-max-pod-grace-period"]; ok {
		kubeletConfig["evictionMaxPodGracePeriod"] = flags["eviction-max-pod-grace-period"]
	}
	if _, ok := flags["eviction-minimum-reclaim"]; ok {
		minReclaims := map[string]string{}
		for _, minReclaim := range strings.Split(flags["eviction-minimum-reclaim"].(string), ",") {
			minReclaimSplit := strings.Split(minReclaim, "=")
			minReclaims[minReclaimSplit[0]] = minReclaimSplit[1]
		}
		data, err := core.JsonToDict(minReclaims)
		if err != nil {
			return err
		}
		kubeletConfig["evictionMinimumReclaim"] = data
	}
	if _, ok := flags["pods-per-core"]; ok {
		kubeletConfig["podsPerCore"] = flags["pods-per-core"]
	}
	if _, ok := flags["protect-kernel-defaults"]; ok {
		kubeletConfig["protectKernelDefaults"] = flags["protect-kernel-defaults"]
	}
	if _, ok := flags["reserved-cpus"]; ok {
		kubeletConfig["reservedSystemCPUs"] = flags["reserved-cpus"]
	}
	if _, ok := flags["topology-manager-scope"]; ok {
		kubeletConfig["topologyManagerScope"] = flags["topology-manager-scope"]
	}
	if _, ok := flags["system-reserved"]; ok {
		systemReserved := map[string]string{}
		for _, systemReserve := range strings.Split(flags["system-reserved"].(string), ",") {
			systemReserveSplit := strings.Split(systemReserve, "=")
			systemReserved[systemReserveSplit[0]] = systemReserveSplit[1]
		}
		data, err := core.JsonToDict(systemReserved)
		if err != nil {
			return err
		}
		kubeletConfig["systemReserved"] = data
	}
	if _, ok := flags["kube-reserved"]; ok {
		kubeReserved := map[string]string{}
		for _, kubeReserve := range strings.Split(flags["kube-reserved"].(string), ",") {
			kubeReserveSplit := strings.Split(kubeReserve, "=")
			kubeReserved[kubeReserveSplit[0]] = kubeReserveSplit[1]
		}
		data, err := core.JsonToDict(kubeReserved)
		if err != nil {
			return err
		}
		kubeletConfig["kubeReserved"] = data
	}
	if _, ok := flags["enforce-node-allocatable"]; ok {
		kubeletConfig["enforceNodeAllocatable"] = flags["enforce-node-allocatable"]
	}
	if _, ok := flags["system-reserved-cgroup"]; ok {
		kubeletConfig["systemReservedCgroup"] = flags["system-reserved-cgroup"]
	}
	if _, ok := flags["kube-reserved-cgroup"]; ok {
		kubeletConfig["kubeReservedCgroup"] = flags["kube-reserved-cgroup"]
	}
	if _, ok := flags["memory-manager-policy"]; ok {
		kubeletConfig["memoryManagerPolicy"] = flags["memory-manager-policy"]
	}
	if _, ok := flags["reserved-memory"]; ok {
		reservations := strings.Split(flags["reserved-memory"].(string), ",")
		data, err := core.JsonToDictSlice(reservations)
		if err != nil {
			return err
		}
		kubeletConfig["reservedMemory"] = data
	}
	if _, ok := flags["register-node"]; ok {
		kubeletConfig["registerNode"] = flags["register-node"]
	}
	if _, ok := flags["register-with-taints"]; ok {
		taints := strings.Split(flags["register-with-taints"].(string), ",")
		data, err := core.JsonToDictSlice(taints)
		if err != nil {
			return err
		}
		kubeletConfig["registerWithTaints"] = data
	}

	/*
	  Looks like these flags do not have a corresponding config option in the file:
	*/
	deprecatedCliOnlyFlags := []string{
		"minimum-container-ttl-duration",
		"maximum-dead-containers-per-container",
		"maximum-dead-containers",
		"master-service-namespace",
		"register-schedulable",
		"keep-terminated-pod-volumes",
		"experimental-mounter-path",
		"cloud-provider",
		"cloud-config",
		"experimental-allocatable-ignore-eviction",
	}
	for _, key := range deprecatedCliOnlyFlags {
		if _, ok := flags[key]; ok {
			kubeletConfig[key] = flags[key]
		}
	}

	return nil
}
