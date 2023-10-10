package connection

import "go.mondoo.com/cnquery/v9/providers-sdk/v1/inventory"

func (a *AtlassianConnection) PlatformInfo() *inventory.Platform {
	return GetPlatformForObject(a.PlatformOverride)
}

func GetPlatformForObject(platformName string) *inventory.Platform {
	if platformName != "atlassian" && platformName != "" {
		return &inventory.Platform{
			Name:    platformName,
			Title:   "atlassian cloud",
			Kind:    "atlassian",
			Runtime: "atlassian",
		}
	}
	return &inventory.Platform{
		Name:    "atlassian",
		Title:   "atlassian cloud",
		Kind:    "api",
		Runtime: "atlassian",
	}
}
