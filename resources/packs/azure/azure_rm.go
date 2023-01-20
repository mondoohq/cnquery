package azure

func (a *mqlAzure) id() (string, error) {
	return "azure", nil
}

func (a *mqlAzure) GetSubscription() (interface{}, error) {
	// the resource fetches the data itself
	return a.MotorRuntime.CreateResource("azure.subscription")
}
