package plugin

// TODO: this may need a new home

func MondooAzureInstanceID(instanceID string) string {
	return "//platformid.api.mondoo.app/runtime/azure" + instanceID
}
