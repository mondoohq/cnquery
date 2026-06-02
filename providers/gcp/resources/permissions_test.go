// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"os"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validatedGCPPermissions is the authoritative set of GCP IAM permissions that
// our manifest (gcp.permissions.json) must contain — no more, no less.
//
// IMPORTANT: Do NOT edit this list just to make the test pass after a provider
// code change. Every entry here has been validated as a REAL GCP IAM permission
// against the live IAM API (`gcloud iam list-testable-permissions` on a project
// and an organization, cross-checked against predefined roles). The permission
// scanner derives permission strings heuristically from SDK method names and
// regularly emits names that are NOT real permissions (wrong casing, singular
// vs. plural, abbreviated resource segments such as certificatemanager.certs,
// or an entirely different resource). When the scanner produces a new
// permission, first confirm the real permission name, add a
// gcpPermissionOverrides entry in
// providers-sdk/v1/util/permissions/permissions.go if the derived name is
// wrong, regenerate the manifest, and only then add the validated name here.
//
// These entries are real permissions that list-testable-permissions does not
// return for a project/organization resource (Cloud Identity is directory-
// scoped; DLP discovery configs follow Google's documented naming), so they
// were validated from documentation rather than the testable-permissions API:
//   - cloudidentity.groups.list
//   - cloudidentity.memberships.list
//   - dlp.discoveryConfigs.list
var validatedGCPPermissions = []string{
	"accessapproval.settings.get",
	"aiplatform.customJobs.get",
	"aiplatform.customJobs.list",
	"aiplatform.datasets.list",
	"aiplatform.endpoints.list",
	"aiplatform.featureOnlineStores.list",
	"aiplatform.indexEndpoints.list",
	"aiplatform.indexes.list",
	"aiplatform.metadataStores.list",
	"aiplatform.models.list",
	"aiplatform.pipelineJobs.list",
	"aiplatform.tensorboards.list",
	"alloydb.backups.list",
	"alloydb.clusters.list",
	"alloydb.instances.list",
	"alloydb.users.list",
	"apikeys.keys.list",
	"artifactregistry.repositories.getIamPolicy",
	"artifactregistry.repositories.list",
	"backupdr.backupPlans.list",
	"backupdr.backupVaults.list",
	"backupdr.bvdataSources.list",
	"backupdr.managementServers.list",
	"batch.jobs.list",
	"bigquery.connections.list",
	"bigquery.reservations.list",
	"bigtable.appProfiles.list",
	"binaryauthorization.attestors.list",
	"binaryauthorization.policy.get",
	"certificatemanager.certissuanceconfigs.get",
	"certificatemanager.certissuanceconfigs.list",
	"certificatemanager.certmapentries.list",
	"certificatemanager.certmaps.list",
	"certificatemanager.certs.get",
	"certificatemanager.certs.list",
	"certificatemanager.dnsauthorizations.get",
	"certificatemanager.dnsauthorizations.list",
	"certificatemanager.trustconfigs.list",
	"cloudbuild.builds.list",
	"cloudbuild.workerpools.list",
	"clouddeploy.deliveryPipelines.list",
	"clouddeploy.releases.list",
	"clouddeploy.targets.list",
	"cloudfunctions.functions.list",
	"cloudfunctions.iamPolicy.get",
	"cloudidentity.groups.list",
	"cloudidentity.memberships.list",
	"cloudkms.cryptoKeyVersions.list",
	"cloudkms.cryptoKeys.get",
	"cloudkms.cryptoKeys.getIamPolicy",
	"cloudkms.cryptoKeys.list",
	"cloudkms.ekmConnections.list",
	"cloudkms.importJobs.list",
	"cloudkms.keyRings.list",
	"cloudkms.locations.list",
	"cloudkms.retiredResources.list",
	"cloudscheduler.jobs.list",
	"cloudsql.backupRuns.list",
	"cloudsql.databases.list",
	"cloudsql.instances.list",
	"cloudsql.sslCerts.list",
	"cloudsql.users.list",
	"cloudtasks.iamPolicy.get",
	"cloudtasks.queues.list",
	"composer.environments.list",
	"compute.addresses.list",
	"compute.backendBuckets.list",
	"compute.backendServices.list",
	"compute.disks.list",
	"compute.externalVpnGateways.list",
	"compute.firewallPolicies.get",
	"compute.firewallPolicies.list",
	"compute.firewalls.list",
	"compute.healthChecks.list",
	"compute.images.getIamPolicy",
	"compute.images.list",
	"compute.instanceGroupManagers.list",
	"compute.instanceGroups.list",
	"compute.instanceTemplates.list",
	"compute.instances.list",
	"compute.interconnectAttachments.list",
	"compute.interconnects.list",
	"compute.machineTypes.get",
	"compute.machineTypes.list",
	"compute.networkEndpointGroups.list",
	"compute.networks.get",
	"compute.networks.list",
	"compute.packetMirrorings.list",
	"compute.projects.get",
	"compute.publicAdvertisedPrefixes.list",
	"compute.regions.get",
	"compute.regions.list",
	"compute.routers.list",
	"compute.routes.list",
	"compute.securityPolicies.get",
	"compute.securityPolicies.list",
	"compute.serviceAttachments.list",
	"compute.snapshots.getIamPolicy",
	"compute.snapshots.list",
	"compute.sslCertificates.list",
	"compute.sslPolicies.list",
	"compute.storagePools.list",
	"compute.targetHttpProxies.list",
	"compute.targetHttpsProxies.list",
	"compute.targetPools.list",
	"compute.targetSslProxies.list",
	"compute.targetTcpProxies.list",
	"compute.urlMaps.list",
	"compute.vpnGateways.list",
	"compute.vpnTunnels.list",
	"compute.zones.get",
	"compute.zones.list",
	"container.clusters.list",
	"containeranalysis.occurrences.list",
	"dataflow.jobs.list",
	"dataplex.assets.list",
	"dataplex.lakes.list",
	"dataplex.zones.list",
	"dataproc.autoscalingPolicies.list",
	"dataproc.clusters.getIamPolicy",
	"dataproc.clusters.list",
	"dataproc.jobs.list",
	"datastore.backupSchedules.list",
	"datastore.databases.list",
	"datastore.indexes.list",
	"datastream.connectionProfiles.get",
	"datastream.connectionProfiles.list",
	"datastream.privateConnections.get",
	"datastream.privateConnections.list",
	"datastream.routes.list",
	"datastream.streams.list",
	"dlp.columnDataProfiles.list",
	"dlp.connections.list",
	"dlp.deidentifyTemplates.list",
	"dlp.discoveryConfigs.list",
	"dlp.fileStoreProfiles.list",
	"dlp.inspectTemplates.list",
	"dlp.jobTriggers.list",
	"dlp.jobs.list",
	"dlp.projectDataProfiles.list",
	"dlp.storedInfoTypes.list",
	"dlp.tableDataProfiles.list",
	"dns.managedZones.getIamPolicy",
	"dns.managedZones.list",
	"dns.policies.list",
	"dns.resourceRecordSets.list",
	"dns.responsePolicies.list",
	"domains.registrations.list",
	"essentialcontacts.contacts.list",
	"eventarc.channels.list",
	"eventarc.triggers.list",
	"file.instances.list",
	"gkebackup.backupPlans.list",
	"gkebackup.restorePlans.list",
	"iam.denypolicies.list",
	"iam.iamPolicy.get",
	"iam.roles.list",
	"iam.serviceAccountKeys.list",
	"iam.serviceAccounts.list",
	"iam.workloadIdentityPoolProviders.list",
	"iam.workloadIdentityPools.list",
	"iap.projects.getSettings",
	"iap.tunnelDestGroups.list",
	"ids.endpoints.list",
	"logging.buckets.list",
	"logging.exclusions.list",
	"logging.settings.get",
	"logging.sinks.list",
	"logging.views.list",
	"memcache.instances.get",
	"memcache.instances.list",
	"memorystore.backupCollections.get",
	"memorystore.backupCollections.list",
	"memorystore.backups.list",
	"memorystore.instances.get",
	"memorystore.instances.list",
	"modelarmor.floorSettings.get",
	"modelarmor.templates.list",
	"monitoring.alertPolicies.list",
	"monitoring.dashboards.list",
	"monitoring.groups.list",
	"monitoring.notificationChannels.list",
	"monitoring.services.list",
	"monitoring.slos.list",
	"monitoring.uptimeCheckConfigs.list",
	"networksecurity.addressGroups.list",
	"networksecurity.authorizationPolicies.list",
	"networksecurity.clientTlsPolicies.list",
	"networksecurity.securityProfileGroups.list",
	"networksecurity.securityProfiles.list",
	"networksecurity.serverTlsPolicies.list",
	"networksecurity.tlsInspectionPolicies.list",
	"networksecurity.urlLists.list",
	"orgpolicy.constraints.list",
	"orgpolicy.customConstraints.list",
	"orgpolicy.policies.list",
	"osconfig.inventories.get",
	"osconfig.osPolicyAssignments.list",
	"osconfig.patchDeployments.list",
	"osconfig.vulnerabilityReports.get",
	"policyanalyzer.serviceAccountLastAuthenticationActivities.query",
	"privateca.caPools.list",
	"privateca.certificateAuthorities.list",
	"privateca.certificates.list",
	"pubsub.schemas.get",
	"pubsub.schemas.list",
	"redis.backups.list",
	"redis.clusters.list",
	"redis.instances.list",
	"resourcemanager.folders.getIamPolicy",
	"resourcemanager.projects.get",
	"resourcemanager.projects.getIamPolicy",
	"resourcemanager.resourceTagBindings.list",
	"run.jobs.getIamPolicy",
	"run.jobs.list",
	"run.operations.list",
	"run.services.getIamPolicy",
	"run.services.list",
	"secretmanager.secrets.getIamPolicy",
	"secretmanager.secrets.list",
	"secretmanager.versions.list",
	"serviceusage.services.get",
	"serviceusage.services.list",
	"source.repos.list",
	"spanner.backupSchedules.list",
	"spanner.backups.list",
	"spanner.databaseRoles.list",
	"spanner.databases.getDdl",
	"spanner.databases.getIamPolicy",
	"spanner.databases.list",
	"spanner.instanceConfigs.list",
	"spanner.instancePartitions.list",
	"spanner.instances.getIamPolicy",
	"spanner.instances.list",
	"storage.buckets.get",
	"storage.buckets.getIamPolicy",
	"storage.buckets.list",
	"workflows.workflows.list",
}

type permissionManifest struct {
	Permissions []string `json:"permissions"`
}

func TestGCPPermissionsMatchValidatedList(t *testing.T) {
	data, err := os.ReadFile("gcp.permissions.json")
	require.NoError(t, err, "failed to read gcp.permissions.json")

	var manifest permissionManifest
	require.NoError(t, json.Unmarshal(data, &manifest), "failed to parse gcp.permissions.json")

	actual := make([]string, len(manifest.Permissions))
	copy(actual, manifest.Permissions)
	sort.Strings(actual)

	expected := make([]string, len(validatedGCPPermissions))
	copy(expected, validatedGCPPermissions)
	sort.Strings(expected)

	// Build sets for detailed diff
	actualSet := map[string]bool{}
	for _, p := range actual {
		actualSet[p] = true
	}
	expectedSet := map[string]bool{}
	for _, p := range expected {
		expectedSet[p] = true
	}

	var unexpected, missing []string
	for _, p := range actual {
		if !expectedSet[p] {
			unexpected = append(unexpected, p)
		}
	}
	for _, p := range expected {
		if !actualSet[p] {
			missing = append(missing, p)
		}
	}

	if len(unexpected) > 0 {
		t.Errorf("unexpected permissions in manifest (not in validated list):\n")
		for _, p := range unexpected {
			t.Errorf("  - %s", p)
		}
	}
	if len(missing) > 0 {
		t.Errorf("missing permissions from manifest (expected but not found):\n")
		for _, p := range missing {
			t.Errorf("  - %s", p)
		}
	}

	assert.Equal(t, expected, actual, "permissions in gcp.permissions.json must exactly match the validated list")
}
