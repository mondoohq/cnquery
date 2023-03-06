package azure

func MondooAzureInstanceID(instanceID string) string {
	return "//platformid.api.mondoo.app/runtime/azure" + instanceID
}
