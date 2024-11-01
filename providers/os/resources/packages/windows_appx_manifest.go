// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package packages

import "encoding/xml"

// AppxManifest represents the structure of an AppxManifest.xml file
type AppxManifest struct {
	XMLName             xml.Name `xml:"Package"`
	Text                string   `xml:",chardata"`
	Xmlns               string   `xml:"xmlns,attr"`
	Uap                 string   `xml:"uap,attr"`
	Build               string   `xml:"build,attr"`
	Mp                  string   `xml:"mp,attr"`
	IgnorableNamespaces string   `xml:"IgnorableNamespaces,attr"`
	Identity            struct {
		Text                  string `xml:",chardata"`
		Name                  string `xml:"Name,attr"`
		ProcessorArchitecture string `xml:"ProcessorArchitecture,attr"`
		Publisher             string `xml:"Publisher,attr"`
		Version               string `xml:"Version,attr"`
	} `xml:"Identity"`
	Properties struct {
		Text                 string `xml:",chardata"`
		Framework            string `xml:"Framework"`
		DisplayName          string `xml:"DisplayName"`
		PublisherDisplayName string `xml:"PublisherDisplayName"`
		Description          string `xml:"Description"`
		Logo                 string `xml:"Logo"`
	} `xml:"Properties"`
	Resources struct {
		Text     string `xml:",chardata"`
		Resource struct {
			Text     string `xml:",chardata"`
			Language string `xml:"Language,attr"`
		} `xml:"Resource"`
	} `xml:"Resources"`
	Dependencies struct {
		Text               string `xml:",chardata"`
		TargetDeviceFamily struct {
			Text             string `xml:",chardata"`
			Name             string `xml:"Name,attr"`
			MinVersion       string `xml:"MinVersion,attr"`
			MaxVersionTested string `xml:"MaxVersionTested,attr"`
		} `xml:"TargetDeviceFamily"`
	} `xml:"Dependencies"`
	PhoneIdentity struct {
		Text             string `xml:",chardata"`
		PhoneProductId   string `xml:"PhoneProductId,attr"`
		PhonePublisherId string `xml:"PhonePublisherId,attr"`
	} `xml:"PhoneIdentity"`
	Extensions struct {
		Text      string `xml:",chardata"`
		Extension []struct {
			Text            string `xml:",chardata"`
			Category        string `xml:"Category,attr"`
			InProcessServer struct {
				Text             string `xml:",chardata"`
				Path             string `xml:"Path"`
				ActivatableClass []struct {
					Text               string `xml:",chardata"`
					ActivatableClassId string `xml:"ActivatableClassId,attr"`
					ThreadingModel     string `xml:"ThreadingModel,attr"`
				} `xml:"ActivatableClass"`
			} `xml:"InProcessServer"`
		} `xml:"Extension"`
	} `xml:"Extensions"`
}
