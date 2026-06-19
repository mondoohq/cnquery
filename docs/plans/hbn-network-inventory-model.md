# HBN Network Inventory Model Plan

## Scope

Add the MQL provider model required to represent HBN network intent, secondary-interface network policies, egress routing, NAT, and internet exposure in Kubernetes clusters. Operator RBAC/deployment and cnspec policy content are covered in separate PRs.

## Implementation status

This PR implements the first normalized posture slice:

- `k8s.networkExposure`, `k8s.egressRoute`, `k8s.egressNat`, and `k8s.networkPolicyCoverage` summaries.
- Kubernetes Service, Ingress, Gateway API gateway/route exposure, AdminNetworkPolicy/BaselineAdminNetworkPolicy, MultiNetworkPolicy, Calico, Cilium, Coil `Egress`, Calico IPPool, and supported HBN current/legacy signals.
- Current HBN coverage is limited to available `network-connector.sylvaproject.org` network connector/inbound-style signals. Legacy HBN coverage covers the listed `network.t-caas.telekom.com` route/configuration resources where they can be normalized.

The raw first-class resources listed below remain the target model for follow-up PRs. They are not all shipped by this normalized posture slice.

## Inputs researched

- `telekom/multi-networkpolicy-nftables`
  - Default branch: `nftables`.
  - Runs as a DaemonSet.
  - Watches `MultiNetworkPolicy`, Pods, Namespaces, and `NetworkAttachmentDefinition`.
  - Applies nftables rules inside pod network namespaces.
  - Uses `k8s.cni.cncf.io` MultiNetworkPolicy APIs from the upstream multi-networkpolicy project.
- `telekom/das-schiff-network-operator` PR 248
  - Open PR: `feat: intent-based network configuration - CRDs, controllers, agents, and E2E tests`.
  - Adds intent CRDs under `network-connector.sylvaproject.org`.
  - Keeps existing network operator CRDs under `network.t-caas.telekom.com`.
  - Important intent resources include `VRF`, `Network`, `Destination`, `Layer2Attachment`, `Inbound`, `Outbound`, `BGPPeering`, `Collector`, `TrafficMirror`, `NodeNetworkConfig`, `NodeNetworkStatus`, `NetworkConnector`, and `NetworkConnectorConfig`.

## Goals

- Inventory HBN intent resources and their compiled node status.
- Inventory MultiNetworkPolicy resources and associate them with secondary pod interfaces.
- Normalize internet-exposed services across Kubernetes Services, Ingress, Gateway API, and HBN Inbound intent.
- Normalize egress routing and NAT visibility from HBN Outbound, Destination, Network, VRF, and node status resources.
- Make native Kubernetes NetworkPolicy, MultiNetworkPolicy, Calico, and Cilium policy coverage comparable in Mondoo.
- Support optional observed-flow sources such as Calico Whisker and Cilium Hubble without making them required.

## Non-goals

- Do not enforce network policy or mutate cluster networking.
- Do not enter the admission path.
- Do not require HBN-specific CRDs in all clusters.
- Do not make packet-level guarantees from static API state. Observed flow integrations are separate and optional.

## Current implementation anchors

- `providers/k8s/resources/networkpolicy.go` exposes native Kubernetes NetworkPolicy.
- `providers/k8s/resources/service.go` exposes Services and LoadBalancer fields.
- `providers/k8s/resources/ingress.go` exposes Ingress.
- `providers/k8s/resources/gateway.go`, `httproute.go`, and related files expose Gateway API resources.
- `providers/k8s/resources/customresource.go` and the dynamic Kubernetes helpers are the right pattern for optional CRDs.

## Proposed MQL resources

### HBN intent resources

Add typed resources for the stable HBN API groups:

- `k8s.hbn.vrf`
- `k8s.hbn.network`
- `k8s.hbn.destination`
- `k8s.hbn.layer2Attachment`
- `k8s.hbn.inbound`
- `k8s.hbn.outbound`
- `k8s.hbn.bgpPeering`
- `k8s.hbn.collector`
- `k8s.hbn.trafficMirror`
- `k8s.hbn.nodeNetworkConfig`
- `k8s.hbn.nodeNetworkStatus`
- `k8s.hbn.networkConnector`
- `k8s.hbn.networkConnectorConfig`

Each resource should expose:

- `id`, `uid`, `name`, `namespace`, `kind`, `apiVersion`, `created`, `resourceVersion`.
- `labels`, `annotations`, `ownerReferences`, `managedFields`.
- `spec`, `status`, and `conditions`.
- Cross-links by reference fields where possible, for example `network.vrf`, `inbound.network`, `outbound.destinations`, and `nodeNetworkStatus.node`.

The implementation should use unstructured decoding first. Add typed Go structs only when the API is stable enough and the module dependency is acceptable.

### Legacy HBN resources

Add inventory for existing `network.t-caas.telekom.com` resources:

- `k8s.hbn.legacy.nodeNetworkConfig`
- `k8s.hbn.legacy.nodeNetplanConfig`
- `k8s.hbn.legacy.bgpPeering`
- `k8s.hbn.legacy.layer2NetworkConfiguration`
- `k8s.hbn.legacy.networkConfigRevision`
- `k8s.hbn.legacy.vrfRouteConfiguration`

These should be marked as legacy in metadata but normalized into exposure and egress summaries when possible.

### Secondary-interface policy resources

Add:

- `k8s.networkAttachmentDefinition`
- `k8s.multiNetworkPolicy`
- `k8s.pod.secondaryInterfaces`
- `k8s.pod.secondaryInterfacePolicies`

`k8s.multiNetworkPolicy` should support `k8s.cni.cncf.io/v1beta1` and `v1beta2` where the CRDs exist.

`k8s.pod.secondaryInterfaces` should derive interface attachment from:

- `k8s.v1.cni.cncf.io/networks` annotations.
- Multus network-status annotations when present.
- `NetworkAttachmentDefinition` references.
- HBN `PodNetwork` or `Layer2Attachment` resources when present.

### Normalized posture resources

Add normalized, query-friendly resources generated from the raw objects:

#### `k8s.networkExposure`

Fields:

- `id`
- `sourceKind`: `Service`, `Ingress`, `Gateway`, `HTTPRoute`, `HBNInbound`, or `LegacyHBN`
- `sourceRef`
- `namespace`
- `name`
- `addresses`
- `ports`
- `protocols`
- `internetExposed`: boolean
- `exposureReason`: `publicLoadBalancerIP`, `nodePort`, `publicIngressAddress`, `gatewayPublicAddress`, `hbnInboundPublicDestination`, `bgpPublicAnnouncement`, or `unknown`
- `networkClassifications`
- `vrf`
- `network`
- `routes`
- `policyCoverage`
- `confidence`: `high`, `medium`, or `low`

#### `k8s.egressRoute`

Fields:

- `id`
- `sourceRef`
- `vrf`
- `network`
- `destinations`
- `cidrs`
- `publicCidrs`
- `nat`
- `nodeStatuses`
- `bgpPeerings`
- `classification`
- `confidence`

#### `k8s.egressNat`

Fields:

- `id`
- `sourceRef`
- `vrf`
- `network`
- `addresses`
- `cidrs`
- `publicCidrs`
- `nodeStatuses`
- `classification`
- `owner`

#### `k8s.networkPolicyCoverage`

Fields:

- `id`
- `workloadRef`
- `podSelector`
- `interfaces`
- `nativeNetworkPolicies`
- `multiNetworkPolicies`
- `calicoPolicies`
- `ciliumPolicies`
- `defaultDenyIngress`
- `defaultDenyEgress`
- `secondaryInterfaceIngressCovered`
- `secondaryInterfaceEgressCovered`
- `coverageGaps`

## Classification rules

Internet exposure should be true when any high-confidence signal is present:

- Service type `LoadBalancer` has a public IP or hostname.
- Service type `NodePort` is reachable on nodes with public addresses.
- Ingress has public addresses or an externally routed ingress class.
- Gateway has public addresses or an externally routed gateway class.
- HBN `Inbound` maps a public destination or a public advertised address to a workload.
- BGP peering or node status indicates a public CIDR is announced for an inbound service.

Egress NAT should be true when:

- HBN `Outbound` references a NAT address, NAT pool, masquerade setting, or public egress destination.
- Compiled `NodeNetworkConfig` or `NodeNetworkStatus` contains NAT translation for the route.
- Legacy route configuration exposes NAT or masquerade fields.

Policy coverage should be interface-aware:

- Native NetworkPolicy applies to the primary pod interface only unless the CNI explicitly extends it.
- MultiNetworkPolicy applies to the referenced secondary network.
- Calico and Cilium policy resources should be normalized separately and linked when present.
- Unknown policy controller semantics should produce `coverageGaps`, not false confidence.

## Implementation phases

### Phase 1: CRD detection and raw inventory

- Add discovery helpers for optional API resources.
- Detect `network-connector.sylvaproject.org`, `network.t-caas.telekom.com`, and `k8s.cni.cncf.io`.
- Add raw unstructured resources with manifest, spec, status, conditions, labels, and annotations.
- Ensure missing CRDs return empty lists without errors.

### Phase 2: Cross-resource linking

- Link HBN references across VRF, Network, Destination, Inbound, Outbound, BGP, and node status.
- Link `NetworkAttachmentDefinition` to pods and MultiNetworkPolicies.
- Link Services, Ingresses, Gateways, and Routes back to HBN Inbound intent where labels, owner references, or explicit refs allow it.

### Phase 3: Normalized exposure and egress model

- Build `k8s.networkExposure`, `k8s.egressRoute`, and `k8s.egressNat`.
- Classify public CIDRs with `ipaddr` logic and Kubernetes node address data.
- Include confidence and reason fields for every classification.

### Phase 4: Policy coverage model

- Extend existing NetworkPolicy inventory into `k8s.networkPolicyCoverage` by aggregating native policies with the same namespace and pod selector, so separate ingress and egress policies are evaluated as one selected-workload coverage record.
- Add MultiNetworkPolicy selectors and secondary-interface coverage.
- Add optional Calico and Cilium CRD support behind API discovery. Missing or forbidden optional CRDs are treated as absent optional inventory so restricted service accounts can still scan core Kubernetes network posture.

### Phase 5: Observed-flow integrations

- Add optional resources for Calico Whisker and Cilium Hubble summaries only when endpoints are configured.
- Keep flow collection bounded and sampled.
- Merge observed flows into exposure and egress resources as evidence, not as the only source of truth.

## Test plan

Focused tests:

- `cd providers/k8s && go test ./...`
- Fixture tests for every HBN CRD listed above.
- Fixture tests for clusters without any HBN or MultiNetworkPolicy CRDs.
- MultiNetworkPolicy v1beta1 and v1beta2 fixtures.
- NetworkAttachmentDefinition and Multus pod annotation fixtures.
- Exposure classifier tests for public/private LoadBalancer, NodePort, Ingress, Gateway, and HBN Inbound.
- Egress NAT classifier tests for HBN Outbound and compiled node status.
- Policy coverage tests for native NetworkPolicy only, MultiNetworkPolicy on secondary interfaces, and mixed primary/secondary coverage.

Full repo verification:

- `make providers/test`
- `make test`

## Acceptance criteria

- Supported HBN intent is visible through normalized MQL resources when CRDs exist. Raw first-class HBN resources remain follow-up work.
- Clusters without HBN or MultiNetworkPolicy CRDs scan successfully with empty resources.
- Internet-exposed services can be discovered from Kubernetes and HBN inputs.
- Egress NAT and public destinations are visible in normalized MQL resources.
- Secondary-interface MultiNetworkPolicy coverage can be queried alongside native NetworkPolicy coverage.
- Optional Calico and Cilium resources do not affect default scan behavior when their CRDs are absent.
