// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/gcp/connection"
	"go.mondoo.com/mql/v13/types"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	iampb "google.golang.org/genproto/googleapis/iam/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (g *mqlGcpProjectSecretmanagerService) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	projectId := g.ProjectId.Data
	return fmt.Sprintf("%s/gcp.project.secretmanagerService", projectId), nil
}

func initGcpProjectSecretmanagerService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}

	conn, ok := runtime.Connection.(*connection.GcpConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not a GCP connection")
	}

	projectId := conn.ResourceID()
	args["projectId"] = llx.StringData(projectId)

	return args, nil, nil
}

type mqlGcpProjectSecretmanagerServiceInternal struct {
	serviceEnabled bool
}

func (g *mqlGcpProject) secretmanager() (*mqlGcpProjectSecretmanagerService, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	projectId := g.Id.Data

	res, err := CreateResource(g.MqlRuntime, "gcp.project.secretmanagerService", map[string]*llx.RawData{
		"projectId": llx.StringData(projectId),
	})
	if err != nil {
		return nil, err
	}

	serviceEnabled, err := g.isServiceEnabled(service_secretmanager)
	if err != nil {
		return nil, err
	}

	svc := res.(*mqlGcpProjectSecretmanagerService)
	svc.serviceEnabled = serviceEnabled
	if !serviceEnabled {
		log.Debug().Str("service", service_secretmanager).Msg("gcp service is not enabled, skipping")
	}

	return svc, nil
}

func (g *mqlGcpProjectSecretmanagerService) secrets() ([]any, error) {
	// when the service is not enabled, we return nil
	if !g.serviceEnabled {
		return nil, nil
	}

	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	creds, err := conn.Credentials(secretmanager.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	client, err := secretmanager.NewClient(ctx, option.WithCredentials(creds), connection.GRPCClientTraceOption())
	if err != nil {
		return nil, err
	}
	defer client.Close()

	it := client.ListSecrets(ctx, &secretmanagerpb.ListSecretsRequest{
		Parent: fmt.Sprintf("projects/%s", projectId),
	})

	var secrets []any
	for {
		s, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var replicationDict map[string]any
		var replicationType string
		if s.Replication != nil {
			replicationDict, err = secretReplicationToDict(s.Replication)
			if err != nil {
				log.Error().Err(err).Str("secret", s.Name).Msg("failed to convert replication")
				continue
			}
			if s.Replication.GetAutomatic() != nil {
				replicationType = "AUTOMATIC"
			} else if s.Replication.GetUserManaged() != nil {
				replicationType = "USER_MANAGED"
			}
		}

		topicNames := make([]any, 0, len(s.Topics))
		for _, t := range s.Topics {
			topicNames = append(topicNames, t.Name)
		}

		var rotationDict map[string]any
		var rotationPeriod string
		var nextRotationTime *time.Time
		if s.Rotation != nil {
			rotationPeriod = durationToString(s.Rotation.RotationPeriod)
			nextRotationTime = timestampAsTimePtr(s.Rotation.NextRotationTime)
			rotationDict, err = convert.JsonToDict(mqlSecretRotation{
				NextRotationTime: timestampToString(s.Rotation.NextRotationTime),
				RotationPeriod:   rotationPeriod,
			})
			if err != nil {
				log.Error().Err(err).Str("secret", s.Name).Msg("failed to convert rotation")
				continue
			}
		}

		ttl := durationToString(s.GetTtl())

		versionAliasesMap := make(map[string]any)
		for k, v := range s.VersionAliases {
			versionAliasesMap[k] = int64(v)
		}

		var mqlVersionDestroyTtl *time.Time
		if s.VersionDestroyTtl != nil {
			v := llx.DurationToTime(s.VersionDestroyTtl.Seconds)
			mqlVersionDestroyTtl = &v
		}

		cmeKeys := extractCustomerManagedEncryptionKeys(s)

		mqlSecret, err := CreateResource(g.MqlRuntime, "gcp.project.secretmanagerService.secret", map[string]*llx.RawData{
			"projectId":                 llx.StringData(projectId),
			"resourcePath":              llx.StringData(s.Name),
			"name":                      llx.StringData(parseResourceName(s.Name)),
			"createTime":                llx.TimeDataPtr(timestampAsTimePtr(s.CreateTime)),
			"labels":                    llx.MapData(convert.MapToInterfaceMap(s.Labels), types.String),
			"replication":               llx.DictData(replicationDict),
			"replicationType":           llx.StringData(replicationType),
			"topics":                    llx.ArrayData(topicNames, types.String),
			"expireTime":                llx.TimeDataPtr(timestampAsTimePtr(s.GetExpireTime())),
			"ttl":                       llx.StringData(ttl),
			"etag":                      llx.StringData(s.Etag),
			"rotation":                  llx.DictData(rotationDict),
			"rotationPeriod":            llx.StringData(rotationPeriod),
			"nextRotationTime":          llx.TimeDataPtr(nextRotationTime),
			"versionAliases":            llx.MapData(versionAliasesMap, types.Int),
			"annotations":               llx.MapData(convert.MapToInterfaceMap(s.Annotations), types.String),
			"versionDestroyTtl":         llx.TimeDataPtr(mqlVersionDestroyTtl),
			"customerManagedEncryption": llx.ArrayData(cmeKeys, types.String),
			"tags":                      llx.MapData(convert.MapToInterfaceMap(s.Tags), types.String),
		})
		if err != nil {
			return nil, err
		}
		secrets = append(secrets, mqlSecret)
	}
	return secrets, nil
}

func (g *mqlGcpProjectSecretmanagerServiceSecret) id() (string, error) {
	return g.ResourcePath.Data, g.ResourcePath.Error
}

func initGcpProjectSecretmanagerServiceSecret(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	// If we already have all the fields populated (e.g., from CreateResource in secrets()), just return.
	if len(args) > 3 {
		return args, nil, nil
	}

	// Resolve from asset identifier when accessed as a discovered asset
	if len(args) == 0 {
		if args == nil {
			args = make(map[string]*llx.RawData)
		}
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["name"] = llx.StringData(ids.name)
			args["projectId"] = llx.StringData(ids.project)
		} else {
			return nil, nil, errors.New("no asset identifier found")
		}
	}

	obj, err := CreateResource(runtime, "gcp.project.secretmanagerService", map[string]*llx.RawData{
		"projectId": args["projectId"],
	})
	if err != nil {
		return nil, nil, err
	}
	svc := obj.(*mqlGcpProjectSecretmanagerService)
	secrets := svc.GetSecrets()
	if secrets.Error != nil {
		return nil, nil, secrets.Error
	}

	nameRaw := args["name"]
	if nameRaw == nil {
		return nil, nil, errors.New("gcp.project.secretmanagerService.secret requires a \"name\" argument")
	}
	nameVal, _ := nameRaw.Value.(string)
	for _, s := range secrets.Data {
		secret := s.(*mqlGcpProjectSecretmanagerServiceSecret)
		if secret.Name.Data == nameVal {
			return args, secret, nil
		}
	}

	return nil, nil, fmt.Errorf("secret %q not found", nameVal)
}

func (g *mqlGcpProjectSecretmanagerServiceSecret) versions() ([]any, error) {
	if g.ResourcePath.Error != nil {
		return nil, g.ResourcePath.Error
	}
	secretPath := g.ResourcePath.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	creds, err := conn.Credentials(secretmanager.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	client, err := secretmanager.NewClient(ctx, option.WithCredentials(creds), connection.GRPCClientTraceOption())
	if err != nil {
		return nil, err
	}
	defer client.Close()

	it := client.ListSecretVersions(ctx, &secretmanagerpb.ListSecretVersionsRequest{
		Parent: secretPath,
	})

	var versions []any
	for {
		v, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var cmeStatusDict map[string]any
		if v.CustomerManagedEncryption != nil {
			cmeStatusDict, err = convert.JsonToDict(mqlCustomerManagedEncryptionStatus{
				KmsKeyVersionName: v.CustomerManagedEncryption.KmsKeyVersionName,
			})
			if err != nil {
				log.Error().Err(err).Str("version", v.Name).Msg("failed to convert customer managed encryption status")
				continue
			}
		}

		mqlVersion, err := CreateResource(g.MqlRuntime, "gcp.project.secretmanagerService.secret.version", map[string]*llx.RawData{
			"resourcePath":                   llx.StringData(v.Name),
			"name":                           llx.StringData(parseResourceName(v.Name)),
			"state":                          llx.StringData(v.State.String()),
			"created":                        llx.TimeDataPtr(timestampAsTimePtr(v.CreateTime)),
			"destroyed":                      llx.TimeDataPtr(timestampAsTimePtr(v.DestroyTime)),
			"etag":                           llx.StringData(v.Etag),
			"clientSpecifiedPayloadChecksum": llx.BoolData(v.ClientSpecifiedPayloadChecksum),
			"scheduledDestroyTime":           llx.TimeDataPtr(timestampAsTimePtr(v.ScheduledDestroyTime)),
			"customerManagedEncryption":      llx.DictData(cmeStatusDict),
		})
		if err != nil {
			return nil, err
		}
		versions = append(versions, mqlVersion)
	}
	return versions, nil
}

func (g *mqlGcpProjectSecretmanagerServiceSecretVersion) id() (string, error) {
	return g.ResourcePath.Data, g.ResourcePath.Error
}

func (g *mqlGcpProjectSecretmanagerServiceSecret) rotationEnabled() (bool, error) {
	v := g.GetRotation()
	if v.Error != nil {
		return false, v.Error
	}
	if v.Data == nil {
		return false, nil
	}
	m, ok := v.Data.(map[string]any)
	if !ok {
		return false, nil
	}
	if period, _ := m["rotationPeriod"].(string); period != "" {
		return true, nil
	}
	if next, _ := m["nextRotationTime"].(string); next != "" {
		return true, nil
	}
	return false, nil
}

func (g *mqlGcpProjectSecretmanagerServiceSecret) public() (bool, error) {
	bindings := g.GetIamPolicy()
	if bindings.Error != nil {
		return false, bindings.Error
	}
	return iamPolicyHasPublicMember(bindings.Data)
}

func (g *mqlGcpProjectSecretmanagerServiceSecret) kmsKeys() ([]any, error) {
	if g.CustomerManagedEncryption.Error != nil {
		return nil, g.CustomerManagedEncryption.Error
	}
	keys := g.CustomerManagedEncryption.Data
	if len(keys) == 0 {
		return []any{}, nil
	}
	res := make([]any, 0, len(keys))
	for _, raw := range keys {
		name, ok := raw.(string)
		if !ok || name == "" {
			continue
		}
		k, err := NewResource(g.MqlRuntime, "gcp.project.kmsService.keyring.cryptokey",
			map[string]*llx.RawData{"resourcePath": llx.StringData(name)})
		if err != nil {
			return nil, err
		}
		res = append(res, k)
	}
	return res, nil
}

func (g *mqlGcpProjectSecretmanagerServiceSecret) topicRefs() ([]any, error) {
	if g.Topics.Error != nil {
		return nil, g.Topics.Error
	}
	topics := g.Topics.Data
	if len(topics) == 0 {
		return []any{}, nil
	}
	res := make([]any, 0, len(topics))
	for _, raw := range topics {
		name, ok := raw.(string)
		if !ok || name == "" {
			continue
		}
		projectId := projectFromResourceName(name)
		if projectId == "" {
			continue
		}
		t, err := NewResource(g.MqlRuntime, "gcp.project.pubsubService.topic", map[string]*llx.RawData{
			"projectId": llx.StringData(projectId),
			"name":      llx.StringData(lastPathSegment(name)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, t)
	}
	return res, nil
}

func (g *mqlGcpProjectSecretmanagerServiceSecret) managedBy() (string, error) {
	return managedByFromLabels(g.GetLabels(), g.GetAnnotations())
}

func (g *mqlGcpProjectSecretmanagerServiceSecret) customerManagedEncryptionEnabled() (bool, error) {
	if g.CustomerManagedEncryption.Error != nil {
		return false, g.CustomerManagedEncryption.Error
	}
	return len(g.CustomerManagedEncryption.Data) > 0, nil
}

func (g *mqlGcpProjectSecretmanagerServiceSecret) iamPolicy() ([]any, error) {
	if g.ResourcePath.Error != nil {
		return nil, g.ResourcePath.Error
	}
	secretPath := g.ResourcePath.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	creds, err := conn.Credentials(secretmanager.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	client, err := secretmanager.NewClient(ctx, option.WithCredentials(creds), connection.GRPCClientTraceOption())
	if err != nil {
		return nil, err
	}
	defer client.Close()

	policy, err := client.GetIamPolicy(ctx, &iampb.GetIamPolicyRequest{
		Resource: secretPath,
		// Request policy schema v3 so conditional bindings (which can include
		// conditional allUsers/allAuthenticatedUsers grants) are returned;
		// the default v1 silently strips any binding carrying a condition.
		Options: &iampb.GetPolicyOptions{RequestedPolicyVersion: 3},
	})
	if err != nil {
		// Tolerate access-denied on a single secret so one inaccessible secret
		// doesn't fail the whole collection (mirrors the KMS accessors).
		if s, ok := status.FromError(err); ok && s.Code() == codes.PermissionDenied {
			return nil, nil
		}
		return nil, err
	}
	res := make([]any, 0, len(policy.Bindings))
	for i, b := range policy.Bindings {
		mqlBinding, err := CreateResource(g.MqlRuntime, "gcp.resourcemanager.binding", map[string]*llx.RawData{
			"id":                   llx.StringData(secretPath + "-" + strconv.Itoa(i)),
			"role":                 llx.StringData(b.Role),
			"members":              llx.ArrayData(convert.SliceAnyToInterface(b.Members), types.String),
			"conditionTitle":       llx.StringData(b.GetCondition().GetTitle()),
			"conditionExpression":  llx.StringData(b.GetCondition().GetExpression()),
			"conditionDescription": llx.StringData(b.GetCondition().GetDescription()),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlBinding)
	}
	return res, nil
}

// Helper types for dict conversion

type mqlSecretReplication struct {
	Type                      string                 `json:"type"`
	CustomerManagedEncryption string                 `json:"customerManagedEncryption,omitempty"`
	Replicas                  []mqlSecretReplicaInfo `json:"replicas,omitempty"`
}

type mqlSecretReplicaInfo struct {
	Location                  string `json:"location"`
	CustomerManagedEncryption string `json:"customerManagedEncryption,omitempty"`
}

type mqlSecretRotation struct {
	NextRotationTime string `json:"nextRotationTime,omitempty"`
	RotationPeriod   string `json:"rotationPeriod,omitempty"`
}

type mqlCustomerManagedEncryptionStatus struct {
	KmsKeyVersionName string `json:"kmsKeyVersionName"`
}

// extractCustomerManagedEncryptionKeys returns all CMEK key names from a secret,
// checking all possible locations depending on replication type:
// - Top-level Secret.CustomerManagedEncryption (regionalized secrets)
// - Replication.Automatic.CustomerManagedEncryption (automatic replication)
// - Replication.UserManaged.Replicas[].CustomerManagedEncryption (user-managed replication)
func extractCustomerManagedEncryptionKeys(s *secretmanagerpb.Secret) []any {
	var keys []any
	if s.CustomerManagedEncryption != nil {
		keys = append(keys, s.CustomerManagedEncryption.KmsKeyName)
	}
	if s.Replication != nil {
		if auto := s.Replication.GetAutomatic(); auto != nil && auto.CustomerManagedEncryption != nil {
			keys = append(keys, auto.CustomerManagedEncryption.KmsKeyName)
		}
		if um := s.Replication.GetUserManaged(); um != nil {
			for _, replica := range um.Replicas {
				if replica.CustomerManagedEncryption != nil {
					keys = append(keys, replica.CustomerManagedEncryption.KmsKeyName)
				}
			}
		}
	}
	return keys
}

func secretReplicationToDict(r *secretmanagerpb.Replication) (map[string]any, error) {
	if auto := r.GetAutomatic(); auto != nil {
		rep := mqlSecretReplication{Type: "AUTOMATIC"}
		if auto.CustomerManagedEncryption != nil {
			rep.CustomerManagedEncryption = auto.CustomerManagedEncryption.KmsKeyName
		}
		return convert.JsonToDict(rep)
	}
	if um := r.GetUserManaged(); um != nil {
		replicas := make([]mqlSecretReplicaInfo, 0, len(um.Replicas))
		for _, replica := range um.Replicas {
			info := mqlSecretReplicaInfo{Location: replica.Location}
			if replica.CustomerManagedEncryption != nil {
				info.CustomerManagedEncryption = replica.CustomerManagedEncryption.KmsKeyName
			}
			replicas = append(replicas, info)
		}
		return convert.JsonToDict(mqlSecretReplication{
			Type:     "USER_MANAGED",
			Replicas: replicas,
		})
	}
	return nil, nil
}

func timestampToString(ts *timestamppb.Timestamp) string {
	if ts == nil {
		return ""
	}
	return ts.AsTime().Format(time.RFC3339)
}

func durationToString(d *durationpb.Duration) string {
	if d == nil {
		return ""
	}
	return fmt.Sprintf("%ds", d.Seconds)
}
