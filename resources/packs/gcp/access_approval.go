package gcp

import (
	"context"
	"fmt"

	accessapproval "cloud.google.com/go/accessapproval/apiv1"
	accessapprovalpb "cloud.google.com/go/accessapproval/apiv1/accessapprovalpb"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	"google.golang.org/api/option"
)

func (g *mqlGcpOrganization) GetAccessApprovalSettings() (interface{}, error) {
	id, err := g.Id()
	if err != nil {
		return nil, err
	}

	return accessApprovalSettings(
		g.MotorRuntime.Motor.Provider, g.MotorRuntime, fmt.Sprintf("organizations/%s/accessApprovalSettings", id))
}

func (g *mqlGcpProject) GetAccessApprovalSettings() (interface{}, error) {
	id, err := g.Id()
	if err != nil {
		return nil, err
	}

	return accessApprovalSettings(
		g.MotorRuntime.Motor.Provider, g.MotorRuntime, fmt.Sprintf("projects/%s/accessApprovalSettings", id))
}

func (g *mqlGcpAccessApprovalSettings) id() (string, error) {
	return g.ResourcePath()
}

func accessApprovalSettings(motorProvider providers.Instance, runtime *resources.Runtime, settingsName string) (interface{}, error) {
	provider, err := gcpProvider(motorProvider)
	if err != nil {
		return nil, err
	}

	credentials, err := provider.Credentials(accessapproval.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	c, err := accessapproval.NewClient(ctx, option.WithCredentials(credentials))
	if err != nil {
		return nil, err
	}
	defer c.Close()

	settings, err := c.GetAccessApprovalSettings(ctx, &accessapprovalpb.GetAccessApprovalSettingsMessage{
		Name: settingsName,
	})
	if err != nil {
		return nil, err
	}

	mqlEnrolledServices := make([]interface{}, 0, len(settings.EnrolledServices))
	for _, s := range settings.EnrolledServices {
		mqlEnrolledServices = append(mqlEnrolledServices, map[string]interface{}{
			"cloudProduct":    s.CloudProduct,
			"enrollmentLevel": s.EnrollmentLevel.String(),
		})
	}

	return runtime.CreateResource("gcp.accessApprovalSettings",
		"resourcePath", settings.Name,
		"notificationEmails", core.SliceToInterfaceSlice(settings.NotificationEmails),
		"enrolledServices", mqlEnrolledServices,
		"enrolledAncestor", settings.EnrolledAncestor,
		"activeKeyVersion", settings.ActiveKeyVersion,
		"ancestorHasActiveKeyVersion", settings.AncestorHasActiveKeyVersion,
		"invalidKeyVersion", settings.InvalidKeyVersion,
	)
}
