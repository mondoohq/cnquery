package sdk

import "github.com/okta/okta-sdk-golang/v2/okta"

// ApiExtension handles cases where Okta's SDK doesn't support a particular API
type ApiExtension struct {
	RequestExecutor *okta.RequestExecutor
}
