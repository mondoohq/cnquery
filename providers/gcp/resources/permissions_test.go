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
// our manifest must contain — no more, no less. This list was validated against
// the live GCP IAM API using queryTestablePermissions.
var validatedGCPPermissions = []string{
	"accessapproval.settings.get",
	"aiplatform.datasets.list",
	"aiplatform.endpoints.list",
	"aiplatform.featureOnlineStores.list",
	"aiplatform.models.list",
	"aiplatform.pipelineJobs.list",
	"alloydb.backups.list",
	"alloydb.clusters.list",
	"alloydb.instances.list",
	"apikeys.keys.list",
	"artifactregistry.repositories.getIamPolicy",
	"artifactregistry.repositories.list",
	"backupdr.backupPlans.list",
	"backupdr.backupVaults.list",
	"backupdr.bvdataSources.list",
	"backupdr.managementServers.list",
	"bigtable.appProfiles.list",
	"binaryauthorization.policy.get",
	"clouddeploy.deliveryPipelines.list",
	"clouddeploy.targets.list",
	"cloudfunctions.functions.list",
	"cloudkms.cryptoKeys.get",
	"cloudkms.cryptoKeys.getIamPolicy",
	"cloudkms.cryptoKeys.list",
	"cloudkms.cryptoKeyVersions.list",
	"cloudkms.keyRings.list",
	"cloudkms.locations.list",
	"cloudsql.databases.list",
	"cloudsql.instances.list",
	"cloudscheduler.jobs.list",
	"cloudtasks.queues.list",
	"compute.addresses.list",
	"compute.backendServices.list",
	"compute.disks.list",
	"compute.firewallPolicies.get",
	"compute.firewallPolicies.list",
	"compute.firewalls.list",
	"compute.healthChecks.list",
	"compute.images.list",
	"compute.instanceGroupManagers.list",
	"compute.instanceGroups.list",
	"compute.instances.list",
	"compute.machineTypes.get",
	"compute.machineTypes.list",
	"compute.networks.get",
	"compute.networks.list",
	"compute.projects.get",
	"compute.regions.list",
	"compute.routers.list",
	"compute.securityPolicies.get",
	"compute.securityPolicies.list",
	"compute.snapshots.list",
	"compute.sslCertificates.list",
	"compute.sslPolicies.list",
	"compute.storagePools.list",
	"compute.targetHttpProxies.list",
	"compute.targetHttpsProxies.list",
	"compute.urlMaps.list",
	"compute.vpnGateways.list",
	"compute.vpnTunnels.list",
	"compute.zones.get",
	"compute.zones.list",
	"container.clusters.list",
	"dataflow.jobs.list",
	"dataproc.clusters.list",
	"datastore.databases.list",
	"dns.managedZones.list",
	"dns.policies.list",
	"dns.resourceRecordSets.list",
	"essentialcontacts.contacts.list",
	"file.instances.list",
	"iam.roles.list",
	"iam.serviceAccountKeys.list",
	"iam.serviceAccounts.list",
	"logging.buckets.list",
	"monitoring.alertPolicies.list",
	"orgpolicy.policies.list",
	"privateca.caPools.list",
	"privateca.certificateAuthorities.list",
	"privateca.certificates.list",
	"redis.backups.list",
	"redis.clusters.list",
	"redis.instances.list",
	"resourcemanager.projects.get",
	"resourcemanager.projects.getIamPolicy",
	"run.jobs.list",
	"run.operations.list",
	"run.services.list",
	"secretmanager.secrets.getIamPolicy",
	"secretmanager.secrets.list",
	"secretmanager.versions.list",
	"serviceusage.services.get",
	"serviceusage.services.list",
	"spanner.backups.list",
	"spanner.databases.list",
	"spanner.instances.list",
	"storage.buckets.getIamPolicy",
	"storage.buckets.list",
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
