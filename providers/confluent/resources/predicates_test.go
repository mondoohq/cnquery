// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/llx"
)

// --- cluster endpoints ----------------------------------------------------

func TestEndpointViewsAreStableAndCarryTheAccessPoint(t *testing.T) {
	views := endpointViews(map[string]clusterEndpointRecord{
		"zzz":    {ConnectionType: "PRIVATE_LINK", KafkaBootstrapEndpoint: "z:9092", HTTPEndpoint: "https://z"},
		"PUBLIC": {ConnectionType: "PUBLIC", KafkaBootstrapEndpoint: "p:9092", HTTPEndpoint: "https://p"},
		"aaa":    {ConnectionType: "PRIVATE_NETWORK_INTERFACE", KafkaBootstrapEndpoint: "a:9092", HTTPEndpoint: "https://a"},
	})

	// Map iteration order is random, so the flattening has to impose one or the
	// endpoints list would reorder between two runs of the same scan.
	require.Len(t, views, 3)
	assert.Equal(t, []string{"PUBLIC", "aaa", "zzz"}, []string{views[0].AccessPointID, views[1].AccessPointID, views[2].AccessPointID})
	assert.Equal(t, "https://p", views[0].HTTPEndpoint)

	assert.Nil(t, endpointViews(nil))
	assert.Nil(t, endpointViews(map[string]clusterEndpointRecord{}))
}

func TestPreferredEndpoint(t *testing.T) {
	public := endpointView{AccessPointID: "PUBLIC", ConnectionType: connectionTypePublic, HTTPEndpoint: "https://public"}
	private := endpointView{AccessPointID: "ap1", ConnectionType: connectionTypePrivateLink, HTTPEndpoint: "https://private"}

	// The public endpoint wins even when it is not first, because it is the one
	// a client outside the network reaches the cluster on.
	assert.Equal(t, public, preferredEndpoint([]endpointView{private, public}))
	assert.Equal(t, private, preferredEndpoint([]endpointView{private}))
	assert.Equal(t, endpointView{}, preferredEndpoint(nil))
}

func TestEndpointConnectionTypes(t *testing.T) {
	got := endpointConnectionTypes([]endpointView{
		{ConnectionType: "PUBLIC"},
		{ConnectionType: "PRIVATE_LINK"},
		{ConnectionType: "PUBLIC"},
		{ConnectionType: ""},
	})
	assert.Equal(t, []string{"PRIVATE_LINK", "PUBLIC"}, got)
	assert.Equal(t, []string{}, endpointConnectionTypes(nil))
}

func TestEndpointExposurePredicates(t *testing.T) {
	tests := []struct {
		name        string
		types       []string
		wantPublic  bool
		wantPrivate bool
	}{
		{"public only", []string{connectionTypePublic}, true, false},
		{"private link only", []string{connectionTypePrivateLink}, false, true},
		{"private network interface only", []string{connectionTypePrivateInterface}, false, true},
		// a cluster can carry both, so the two predicates are not each other's
		// negation
		{"both", []string{connectionTypePublic, connectionTypePrivateLink}, true, true},
		{"none", nil, false, false},
		{"an unknown type is neither", []string{"SOMETHING_NEW"}, false, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			views := make([]endpointView, 0, len(tc.types))
			for _, connType := range tc.types {
				views = append(views, endpointView{ConnectionType: connType})
			}
			assert.Equal(t, tc.wantPublic, hasPublicEndpoint(views))
			assert.Equal(t, tc.wantPrivate, hasPrivateEndpoint(views))
		})
	}
}

func TestClusterCku(t *testing.T) {
	two := int32(2)
	four := int32(4)

	t.Run("the provisioned count wins over the requested one", func(t *testing.T) {
		record := &kafkaClusterRecord{
			Spec:   &clusterSpecRecord{Config: &clusterConfigRecord{Cku: &four}},
			Status: &clusterStatusRecord{Cku: &two},
		}
		require.NotNil(t, clusterCku(record))
		assert.EqualValues(t, 2, *clusterCku(record))
	})

	t.Run("the spec answers when the status carries no count", func(t *testing.T) {
		record := &kafkaClusterRecord{
			Spec:   &clusterSpecRecord{Config: &clusterConfigRecord{Cku: &four}},
			Status: &clusterStatusRecord{},
		}
		require.NotNil(t, clusterCku(record))
		assert.EqualValues(t, 4, *clusterCku(record))
	})

	t.Run("a cluster type without units reports none", func(t *testing.T) {
		record := &kafkaClusterRecord{Spec: &clusterSpecRecord{Config: &clusterConfigRecord{Kind: "Basic"}}}
		assert.Nil(t, clusterCku(record))
		assert.Nil(t, clusterCku(nil))
		assert.Nil(t, clusterCku(&kafkaClusterRecord{}))
	})
}

// --- ACL predicates -------------------------------------------------------

func TestAclGrantsWildcardResource(t *testing.T) {
	tests := []struct {
		name        string
		patternType string
		resource    string
		want        bool
	}{
		{"literal star covers every resource", "LITERAL", "*", true},
		{"literal name is narrow", "LITERAL", "payments", false},
		// an empty prefix matches every name, which is the same reach written
		// differently
		{"empty prefix covers every resource", "PREFIXED", "", true},
		{"a real prefix is narrow", "PREFIXED", "payments-", false},
		{"prefix of a single star is narrow", "PREFIXED", "*", false},
		{"any star", "ANY", "*", true},
		{"match star", "MATCH", "*", true},
		{"any name", "ANY", "payments", false},
		{"lower case pattern type still matches", "literal", "*", true},
		{"unknown pattern type is not a wildcard", "UNKNOWN", "*", false},
		{"empty pattern type is not a wildcard", "", "*", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, aclGrantsWildcardResource(tc.patternType, tc.resource))
		})
	}
}

func TestAclGrantsAnyPrincipal(t *testing.T) {
	tests := map[string]bool{
		"User:*":         true,
		"UserV2:*":       true,
		"Group:*":        true,
		"*":              true,
		"User:sa-abc123": false,
		"User:u-abc123":  false,
		"UserV2:sa-1":    false,
		"User:":          false,
		"":               false,
		// a name that merely contains a star is not a wildcard principal
		"User:sa-*-x": false,
	}
	for principal, want := range tests {
		t.Run(principal, func(t *testing.T) {
			assert.Equal(t, want, aclGrantsAnyPrincipal(principal))
		})
	}
}

func TestAclGrantsAllOperations(t *testing.T) {
	assert.True(t, aclGrantsAllOperations("ALL"))
	assert.True(t, aclGrantsAllOperations("all"))
	assert.False(t, aclGrantsAllOperations("READ"))
	assert.False(t, aclGrantsAllOperations("ALTER"))
	assert.False(t, aclGrantsAllOperations(""))
}

func TestAclMatchesTopic(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		patternType  string
		resourceName string
		topic        string
		want         bool
	}{
		{"literal exact", "TOPIC", "LITERAL", "payments", "payments", true},
		{"literal other", "TOPIC", "LITERAL", "orders", "payments", false},
		{"literal star", "TOPIC", "LITERAL", "*", "payments", true},
		{"prefixed hit", "TOPIC", "PREFIXED", "pay", "payments", true},
		{"prefixed miss", "TOPIC", "PREFIXED", "ord", "payments", false},
		// an empty prefix reaches every topic
		{"empty prefix", "TOPIC", "PREFIXED", "", "payments", true},
		{"any star", "TOPIC", "ANY", "*", "payments", true},
		// a group grant must not be attributed to a topic of the same name
		{"group resource type", "GROUP", "LITERAL", "payments", "payments", false},
		{"cluster resource type", "CLUSTER", "LITERAL", "kafka-cluster", "payments", false},
		{"any resource type", "ANY", "LITERAL", "*", "payments", true},
		{"lower case types still match", "topic", "literal", "payments", "payments", true},
		{"unknown pattern type does not match", "TOPIC", "UNKNOWN", "payments", "payments", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, aclMatchesTopic(tc.resourceType, tc.patternType, tc.resourceName, tc.topic))
		})
	}
}

// Kafka gives an access control entry no identifier, so the whole tuple is its
// identity. Two entries that collapse to one cache key would report a single
// grant where the cluster holds two, and the second entry's values would be
// dropped in favour of the first.
func TestAclIDCarriesEveryIdentityDimension(t *testing.T) {
	base := aclRecord{
		ClusterID:    "lkc-1",
		ResourceType: "TOPIC",
		ResourceName: "payments",
		PatternType:  "LITERAL",
		Principal:    "User:sa-1",
		Host:         "*",
		Operation:    "READ",
		Permission:   "ALLOW",
	}

	variants := map[string]func(r *aclRecord){
		"cluster":      func(r *aclRecord) { r.ClusterID = "lkc-2" },
		"resourceType": func(r *aclRecord) { r.ResourceType = "GROUP" },
		"resourceName": func(r *aclRecord) { r.ResourceName = "orders" },
		"patternType":  func(r *aclRecord) { r.PatternType = "PREFIXED" },
		"principal":    func(r *aclRecord) { r.Principal = "User:sa-2" },
		"host":         func(r *aclRecord) { r.Host = "10.0.0.1" },
		"operation":    func(r *aclRecord) { r.Operation = "WRITE" },
		"permission":   func(r *aclRecord) { r.Permission = "DENY" },
	}

	baseID := aclID(&base)
	seen := map[string]string{baseID: "base"}
	for name, mutate := range variants {
		t.Run(name, func(t *testing.T) {
			variant := base
			mutate(&variant)
			id := aclID(&variant)
			assert.NotEqual(t, baseID, id, "an entry differing in %s must get its own identifier", name)
			if other, ok := seen[id]; ok {
				t.Fatalf("identifier collides with the %s variant", other)
			}
			seen[id] = name
		})
	}

	// The identical entry must keep the identical key, or the resource would
	// change identity between two runs of the same scan.
	assert.Equal(t, baseID, aclID(&aclRecord{
		ClusterID: "lkc-1", ResourceType: "TOPIC", ResourceName: "payments",
		PatternType: "LITERAL", Principal: "User:sa-1", Host: "*",
		Operation: "READ", Permission: "ALLOW",
	}))
}

// A consumer group identifier may contain the separator, which would otherwise
// shift the remaining components and let two different entries agree.
func TestAclIDEscapesSeparators(t *testing.T) {
	first := aclID(&aclRecord{
		ClusterID: "lkc-1", ResourceType: "GROUP", ResourceName: "a/LITERAL/b",
		PatternType: "LITERAL", Principal: "User:sa-1", Host: "*",
		Operation: "READ", Permission: "ALLOW",
	})
	second := aclID(&aclRecord{
		ClusterID: "lkc-1", ResourceType: "GROUP", ResourceName: "a",
		PatternType: "LITERAL", Principal: "User:sa-1", Host: "*",
		Operation: "READ", Permission: "ALLOW",
	})
	assert.NotEqual(t, first, second)
	assert.NotContains(t, first, "a/LITERAL/b")
}

func TestTopicIDIsQualifiedByCluster(t *testing.T) {
	// The same topic name on two clusters is two topics.
	assert.NotEqual(t, topicID("lkc-1", "payments"), topicID("lkc-2", "payments"))
	assert.Equal(t, topicID("lkc-1", "payments"), topicID("lkc-1", "payments"))
	// A name carrying the separator must not read as a deeper path.
	assert.NotEqual(t, topicID("lkc-1", "a/topic/b"), topicID("lkc-1/topic/a", "b"))
}

func TestPrincipalAccountID(t *testing.T) {
	tests := map[string]string{
		"User:sa-abc123":   "sa-abc123",
		"UserV2:sa-abc123": "sa-abc123",
		"user:u-abc123":    "u-abc123",
		"User:*":           "",
		"Group:admins":     "",
		"sa-abc123":        "",
		"":                 "",
		"User:":            "",
		// the legacy numeric form names no object this provider models
		"User:12345": "12345",
	}
	for principal, want := range tests {
		t.Run(principal, func(t *testing.T) {
			assert.Equal(t, want, principalAccountID(principal))
		})
	}

	assert.True(t, isServiceAccountID("sa-abc123"))
	assert.False(t, isServiceAccountID("u-abc123"))
	assert.False(t, isServiceAccountID("12345"))
	assert.False(t, isServiceAccountID(""))
}

// --- Confluent Resource Names ---------------------------------------------

func TestParseCRN(t *testing.T) {
	segments := parseCRN("crn://confluent.cloud/organization=o-1/environment=env-1/cloud-cluster=lkc-1")
	assert.Equal(t, []crnSegment{
		{Key: "organization", Value: "o-1"},
		{Key: "environment", Value: "env-1"},
		{Key: "cloud-cluster", Value: "lkc-1"},
	}, segments)

	assert.Equal(t, "o-1", crnValue(segments, "organization"))
	assert.Equal(t, "env-1", crnValue(segments, "environment"))
	assert.Empty(t, crnValue(segments, "kafka"))
	assert.Equal(t, "cloud-cluster", crnScopeKind(segments))
	assert.Equal(t, "lkc-1", crnClusterID(segments))

	assert.Nil(t, parseCRN(""))
	assert.Nil(t, parseCRN("   "))
	assert.Empty(t, crnScopeKind(nil))
	assert.Empty(t, crnClusterID(nil))
}

func TestCRNScopeShapes(t *testing.T) {
	tests := []struct {
		name        string
		crn         string
		wantScope   string
		wantEnv     string
		wantCluster string
	}{
		{
			name:      "organization scope",
			crn:       "crn://confluent.cloud/organization=o-1",
			wantScope: "organization",
		},
		{
			name:      "environment scope",
			crn:       "crn://confluent.cloud/organization=o-1/environment=env-1",
			wantScope: "environment",
			wantEnv:   "env-1",
		},
		{
			name:        "cluster scope",
			crn:         "crn://confluent.cloud/organization=o-1/environment=env-1/cloud-cluster=lkc-1",
			wantScope:   "cloud-cluster",
			wantEnv:     "env-1",
			wantCluster: "lkc-1",
		},
		{
			// a resource-level binding names the cluster with the `kafka`
			// segment rather than `cloud-cluster`
			name:        "topic scope",
			crn:         "crn://confluent.cloud/organization=o-1/environment=env-1/cloud-cluster=lkc-1/kafka=lkc-1/topic=payments",
			wantScope:   "topic",
			wantEnv:     "env-1",
			wantCluster: "lkc-1",
		},
		{
			name:        "kafka segment only",
			crn:         "crn://confluent.cloud/organization=o-1/environment=env-1/kafka=lkc-9/topic=*",
			wantScope:   "topic",
			wantEnv:     "env-1",
			wantCluster: "lkc-9",
		},
		{
			// a wildcard names no cluster, so nothing must resolve to one
			name:      "wildcard cluster",
			crn:       "crn://confluent.cloud/organization=o-1/environment=env-1/cloud-cluster=*",
			wantScope: "cloud-cluster",
			wantEnv:   "env-1",
		},
		{
			name:      "schema registry scope",
			crn:       "crn://confluent.cloud/organization=o-1/environment=env-1/schema-registry=lsrc-1",
			wantScope: "schema-registry",
			wantEnv:   "env-1",
		},
		{
			name:      "garbage carries no scope",
			crn:       "not-a-crn",
			wantScope: "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			segments := parseCRN(tc.crn)
			assert.Equal(t, tc.wantScope, crnScopeKind(segments))
			assert.Equal(t, tc.wantEnv, crnValue(segments, "environment"))
			assert.Equal(t, tc.wantCluster, crnClusterID(segments))
		})
	}
}

// --- object references ----------------------------------------------------

func TestReferenceKind(t *testing.T) {
	assert.Equal(t, "cmk.v2.Cluster", referenceKind(&objectReference{APIVersion: "cmk/v2", Kind: "Cluster"}))
	assert.Equal(t, "srcm.v3.Cluster", referenceKind(&objectReference{APIVersion: "srcm/v3", Kind: "Cluster"}))
	// a half-formed reference must not produce a half-formed kind
	assert.Empty(t, referenceKind(&objectReference{APIVersion: "cmk/v2"}))
	assert.Empty(t, referenceKind(&objectReference{Kind: "Cluster"}))
	assert.Empty(t, referenceKind(nil))
}

func TestOwnerKindOf(t *testing.T) {
	assert.Equal(t, "ServiceAccount", ownerKindOf(&objectReference{ID: "sa-1", Kind: "ServiceAccount"}))
	assert.Equal(t, "User", ownerKindOf(&objectReference{ID: "u-1", Kind: "User"}))
	// the identifier prefix answers when the reference carries no kind
	assert.Equal(t, "ServiceAccount", ownerKindOf(&objectReference{ID: "sa-1"}))
	assert.Equal(t, "User", ownerKindOf(&objectReference{ID: "u-1"}))
	assert.Empty(t, ownerKindOf(&objectReference{ID: "x-1"}))
	assert.Empty(t, ownerKindOf(nil))
}

func TestRefID(t *testing.T) {
	assert.Equal(t, "env-1", refID(&objectReference{ID: "env-1"}))
	assert.Empty(t, refID(&objectReference{}))
	assert.Empty(t, refID(nil))
}

func TestKeyReferenceOf(t *testing.T) {
	assert.Equal(t, "arn:aws:kms:...", keyReferenceOf(&encryptionKeyDetailRecord{Kind: "AwsKey", KeyArn: "arn:aws:kms:..."}))
	assert.Equal(t, "https://vault/keys/k", keyReferenceOf(&encryptionKeyDetailRecord{Kind: "AzureKey", KeyID: "https://vault/keys/k"}))
	assert.Equal(t, "projects/p/...", keyReferenceOf(&encryptionKeyDetailRecord{Kind: "GcpKey", KeyID: "projects/p/..."}))
	// a kind Confluent adds later still reports whichever reference it filled
	assert.Equal(t, "some-id", keyReferenceOf(&encryptionKeyDetailRecord{Kind: "SomethingNew", KeyID: "some-id"}))
	assert.Equal(t, "some-arn", keyReferenceOf(&encryptionKeyDetailRecord{Kind: "SomethingNew", KeyArn: "some-arn"}))
	assert.Empty(t, keyReferenceOf(&encryptionKeyDetailRecord{}))
	assert.Empty(t, keyReferenceOf(nil))
}

// --- derived scalars ------------------------------------------------------

func TestAgeInDays(t *testing.T) {
	now := time.Date(2026, 1, 31, 12, 0, 0, 0, time.UTC)

	t.Run("whole days since creation", func(t *testing.T) {
		created := now.Add(-400 * 24 * time.Hour)
		assert.Equal(t, int64(400), ageInDays(&created, now).Value)
	})

	t.Run("a partial day rounds down", func(t *testing.T) {
		created := now.Add(-36 * time.Hour)
		assert.Equal(t, int64(1), ageInDays(&created, now).Value)
	})

	t.Run("a key created moments ago is zero days old", func(t *testing.T) {
		created := now.Add(-time.Minute)
		assert.Equal(t, int64(0), ageInDays(&created, now).Value)
	})

	// A key whose creation time the API did not report is of unknown age. Zero
	// would read as a key created today, which is the safest possible answer
	// and the wrong one.
	t.Run("an absent creation time is null, not zero", func(t *testing.T) {
		assert.Equal(t, llx.NilData, ageInDays(nil, now))
	})

	// Clock skew must not produce a negative age, which no rotation policy is
	// written against.
	t.Run("a creation time in the future is zero", func(t *testing.T) {
		created := now.Add(48 * time.Hour)
		assert.Equal(t, int64(0), ageInDays(&created, now).Value)
	})
}

func TestOptionalInt(t *testing.T) {
	value := int32(2)
	assert.Equal(t, int64(2), optionalInt(&value).Value)
	// a cluster type that carries no unit count must report null rather than a
	// zero that reads as a real size
	assert.Equal(t, llx.NilData, optionalInt(nil))
}

func TestStrSliceAndMapWidening(t *testing.T) {
	assert.Equal(t, []any{"a", "b"}, strSliceToAny([]string{"a", "b"}))
	assert.Equal(t, []any{}, strSliceToAny(nil))
	assert.Equal(t, map[string]any{"a": "1"}, strMapToAny(map[string]string{"a": "1"}))
	assert.Equal(t, map[string]any{}, strMapToAny(nil))
}
