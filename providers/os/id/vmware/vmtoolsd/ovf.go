// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package vmtoolsd

import (
	"encoding/xml"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
)

// OVFEnv tries to fetch the Open Virtualization Format settings using `vmtoolsd`
//
// https://www.dmtf.org/sites/default/files/standards/documents/DSP0243_2.1.0.pdf
func (m *CommandInstanceMetadata) OVFEnv() (*Env, error) {
	ovfEnvXML, err := m.vmtoolsdGuestInfo("ovfEnv")
	if err == nil && ovfEnvXML != "" {
		var env Env
		if err := xml.Unmarshal([]byte(ovfEnvXML), &env); err == nil {
			return &env, nil
		}
		log.Debug().Err(err).Msg("unable to unmarshal XML OVF env")
	}

	log.Debug().Err(err).Msg("unable to get vmtoolsd ovfEnv data")
	return nil, errors.New("unable to detect OVF environment")
}

// This is a copy and past from from https://github.com/vmware/govmomi/blob/main/ovf/env.go
// so we don't import `govmomi` into our `os` provider, why, to avoid increasing the binary
// size, though if we don't care about it, we can just import it.

type Env struct {
	XMLName   xml.Name `xml:"http://schemas.dmtf.org/ovf/environment/1 Environment" json:"xmlName"`
	ID        string   `xml:"id,attr" json:"id"`
	EsxID     string   `xml:"http://www.vmware.com/schema/ovfenv esxId,attr" json:"esxID"`
	VCenterID string   `xml:"http://www.vmware.com/schema/ovfenv vCenterId,attr" json:"vCenterID"`

	Platform *PlatformSection `xml:"PlatformSection" json:"platformSection,omitempty"`
	Property *PropertySection `xml:"PropertySection" json:"propertySection,omitempty"`
}

type PlatformSection struct {
	Kind    string `xml:"Kind" json:"kind,omitempty"`
	Version string `xml:"Version" json:"version,omitempty"`
	Vendor  string `xml:"Vendor" json:"vendor,omitempty"`
	Locale  string `xml:"Locale" json:"locale,omitempty"`
}

type PropertySection struct {
	Properties []EnvProperty `xml:"Property" json:"property,omitempty"`
}

type EnvProperty struct {
	Key   string `xml:"key,attr" json:"key"`
	Value string `xml:"value,attr" json:"value,omitempty"`
}
