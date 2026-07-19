// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"sync"

	"github.com/Snowflake-Labs/terraform-provider-snowflake/pkg/sdk"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/snowflake/connection"
)

type mqlSnowflakeNotificationIntegrationInternal struct {
	descLock    sync.Mutex
	descLoaded  bool
	descProps   map[string]string
	descLoadErr error
}

func (r *mqlSnowflakeAccount) notificationIntegrations() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.SnowflakeConnection)
	client := conn.Client()
	ctx := context.Background()

	integrations, err := client.NotificationIntegrations.Show(ctx, sdk.NewShowNotificationIntegrationRequest())
	if err != nil {
		return nil, err
	}

	list := make([]any, 0, len(integrations))
	for i := range integrations {
		mqlIntegration, err := newMqlSnowflakeNotificationIntegration(r.MqlRuntime, integrations[i])
		if err != nil {
			return nil, err
		}
		list = append(list, mqlIntegration)
	}

	return list, nil
}

func newMqlSnowflakeNotificationIntegration(runtime *plugin.Runtime, integration sdk.NotificationIntegration) (*mqlSnowflakeNotificationIntegration, error) {
	r, err := CreateResource(runtime, "snowflake.notificationIntegration", map[string]*llx.RawData{
		"__id":      llx.StringData(integration.ID().FullyQualifiedName()),
		"name":      llx.StringData(integration.Name),
		"type":      llx.StringData(integration.NotificationType),
		"category":  llx.StringData(integration.Category),
		"enabled":   llx.BoolData(integration.Enabled),
		"comment":   llx.StringData(integration.Comment),
		"createdAt": llx.TimeData(integration.CreatedOn),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlSnowflakeNotificationIntegration), nil
}

func (r *mqlSnowflakeNotificationIntegration) describeProperties() (map[string]string, error) {
	if r.descLoaded {
		return r.descProps, r.descLoadErr
	}
	r.descLock.Lock()
	defer r.descLock.Unlock()
	if r.descLoaded {
		return r.descProps, r.descLoadErr
	}

	conn := r.MqlRuntime.Connection.(*connection.SnowflakeConnection)
	client := conn.Client()
	ctx := context.Background()

	props, err := client.NotificationIntegrations.Describe(ctx, sdk.NewAccountObjectIdentifier(r.Name.Data))
	if err != nil {
		r.descLoaded = true
		r.descLoadErr = err
		return nil, err
	}

	out := make(map[string]string, len(props))
	for _, p := range props {
		out[p.Name] = p.Value
	}
	r.descProps = out
	r.descLoaded = true
	return out, nil
}

func (r *mqlSnowflakeNotificationIntegration) properties() (map[string]any, error) {
	props, err := r.describeProperties()
	if err != nil {
		return nil, err
	}
	out := make(map[string]any, len(props))
	for k, v := range props {
		out[k] = v
	}
	return out, nil
}

func (r *mqlSnowflakeNotificationIntegration) prop(key string) (string, error) {
	props, err := r.describeProperties()
	if err != nil {
		return "", err
	}
	return props[key], nil
}

func (r *mqlSnowflakeNotificationIntegration) direction() (string, error) {
	return r.prop("DIRECTION")
}

func (r *mqlSnowflakeNotificationIntegration) notificationProvider() (string, error) {
	return r.prop("NOTIFICATION_PROVIDER")
}

func (r *mqlSnowflakeNotificationIntegration) awsSnsTopicArn() (string, error) {
	return r.prop("AWS_SNS_TOPIC_ARN")
}

func (r *mqlSnowflakeNotificationIntegration) awsSnsRoleArn() (string, error) {
	return r.prop("AWS_SNS_ROLE_ARN")
}

func (r *mqlSnowflakeNotificationIntegration) gcpPubsubSubscriptionName() (string, error) {
	return r.prop("GCP_PUBSUB_SUBSCRIPTION_NAME")
}

func (r *mqlSnowflakeNotificationIntegration) gcpPubsubTopicName() (string, error) {
	return r.prop("GCP_PUBSUB_TOPIC_NAME")
}

func (r *mqlSnowflakeNotificationIntegration) azureStorageQueuePrimaryUri() (string, error) {
	return r.prop("AZURE_STORAGE_QUEUE_PRIMARY_URI")
}

func (r *mqlSnowflakeNotificationIntegration) azureEventGridTopicEndpoint() (string, error) {
	return r.prop("AZURE_EVENT_GRID_TOPIC_ENDPOINT")
}

func (r *mqlSnowflakeNotificationIntegration) azureTenantId() (string, error) {
	return r.prop("AZURE_TENANT_ID")
}
