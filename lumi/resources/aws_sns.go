package resources

func (t *lumiAwsSnsTopic) id() (string, error) {
	return t.Arn()
}

func (t *lumiAwsSnsSubscription) id() (string, error) {
	return t.Arn()
}
