package azure

import (
	"time"

	"github.com/Azure/go-autorest/autorest/date"
	"github.com/gofrs/uuid"
	"go.mondoo.io/mondoo/resources/packs/core"
)

func (a *mqlAzurerm) id() (string, error) {
	return "azurerm", nil
}

func azureTagsToInterface(data map[string]*string) map[string]interface{} {
	labels := make(map[string]interface{})
	for key := range data {
		labels[key] = core.ToString(data[key])
	}
	return labels
}

func azureRmTime(d *date.Time) *time.Time {
	if d == nil {
		return nil
	}
	return &d.Time
}

func azureRmUnixTime(d *date.UnixTime) *time.Time {
	if d == nil {
		return nil
	}

	// cast
	stamp := time.Time(*d)
	return &stamp
}

func uuidToString(u *uuid.UUID) string {
	if u == nil {
		return ""
	}
	return u.String()
}
