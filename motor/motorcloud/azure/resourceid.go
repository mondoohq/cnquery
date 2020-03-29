package azure

import (
	"fmt"
	"net/url"
	"strings"
)

// really azure? https://github.com/Azure/azure-sdk-for-go/issues/3080
// we need to parse the resource id to extract the individual parts to regenerate the url
type ResourceID struct {
	SubscriptionID string
	ResourceGroup  string
	Provider       string
	Path           map[string]string
}

// ParseResourceID parses a fully qualified ID into its components
// Resources have the following format:
// /subscriptions/{guid}/resourceGroups/{resource-group-name}/{resource-provider-namespace}/{resource-type}/{resource-name}
// @see https://docs.microsoft.com/bs-latn-ba/rest/api/resources/resources/getbyid
func ParseResourceID(id string) (*ResourceID, error) {
	// sanitize resource id
	idURL, err := url.ParseRequestURI(id)
	if err != nil {
		return nil, fmt.Errorf("cannot parse azure resource id: %s", err)
	}
	path := idURL.Path
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/")
	components := strings.Split(path, "/")

	// check if we have an even number of key-value pairs
	if len(components)%2 != 0 {
		return nil, fmt.Errorf("uneven number of components %q", path)
	}

	// parse key-value pairs
	subscriptionID := ""
	componentMap := make(map[string]string, len(components)/2)
	for current := 0; current < len(components); current += 2 {
		key := components[current]
		value := components[current+1]

		// validate that key and value are properly set
		if key == "" || value == "" {
			return nil, fmt.Errorf("invalid key value pair: '%s' -> '%s'", key, value)
		}

		// some urls use the same component id twice, ensure we store the subscription id
		// e.g. Azure Service Bus subscription resource
		if key == "subscriptions" && subscriptionID == "" {
			subscriptionID = value
		} else {
			componentMap[key] = value
		}
	}
	resID := &ResourceID{
		SubscriptionID: subscriptionID,
		Path:           componentMap,
	}

	if len(resID.SubscriptionID) == 0 {
		return nil, fmt.Errorf("no subscription ID found in resource id: %s", path)
	}

	// parse resource group
	if resourceGroup, ok := componentMap["resourceGroups"]; ok {
		resID.ResourceGroup = resourceGroup
		delete(componentMap, "resourceGroups")
	}

	// parse provider
	if provider, ok := componentMap["providers"]; ok {
		resID.Provider = provider
		delete(componentMap, "providers")
	}

	return resID, nil
}

// Component returns a component from parsed resource id
func (id *ResourceID) Component(name string) (string, error) {
	val, ok := id.Path[name]
	if !ok {
		return "", fmt.Errorf("ID was missing the `%s` element", name)
	}
	return val, nil
}
