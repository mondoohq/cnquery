package client

/*
This code is derived from https://github.com/vmware/goipmi

Copyright (c) 2014 VMware, Inc. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// OemVendorID aka IANA assigned Enterprise Number per:
// http://www.iana.org/assignments/enterprise-numbers/enterprise-numbers
// Note that constants defined here are the same subset that ipmitool recognizes.
type OemVendorID uint32

// IANA assigned manufacturer IDs
const (
	OemUnknown              = OemVendorID(0)
	OemHP                   = OemVendorID(11)
	OemSun                  = OemVendorID(42)
	OemNokia                = OemVendorID(94)
	OemBull                 = OemVendorID(107)
	OemHitachi116           = OemVendorID(116)
	OemNEC                  = OemVendorID(119)
	OemToshiba              = OemVendorID(186)
	OemIntel                = OemVendorID(343)
	OemTatung               = OemVendorID(373)
	OemHitachi399           = OemVendorID(399)
	OemDell                 = OemVendorID(674)
	OemLMC                  = OemVendorID(2168)
	OemRadiSys              = OemVendorID(4337)
	OemBroadcom             = OemVendorID(4413)
	OemMagnum               = OemVendorID(5593)
	OemTyan                 = OemVendorID(6653)
	OemNewisys              = OemVendorID(9237)
	OemFujitsuSiemens       = OemVendorID(10368)
	OemAvocent              = OemVendorID(10418)
	OemPeppercon            = OemVendorID(10437)
	OemSupermicro           = OemVendorID(10876)
	OemOSA                  = OemVendorID(11102)
	OemGoogle               = OemVendorID(11129)
	OemPICMG                = OemVendorID(12634)
	OemRaritan              = OemVendorID(13742)
	OemKontron              = OemVendorID(15000)
	OemPPS                  = OemVendorID(16394)
	OemAMI                  = OemVendorID(20974)
	OemNokiaSiemensNetworks = OemVendorID(28458)
	OemSupermicro47488      = OemVendorID(47488)
)

var oemStrings = map[OemVendorID]string{
	OemUnknown:              "Unknown",
	OemHP:                   "Hewlett-Packard",
	OemSun:                  "Sun Microsystems",
	OemNokia:                "Nokia",
	OemBull:                 "Bull Company",
	OemHitachi116:           "Hitachi",
	OemNEC:                  "NEEC",
	OemToshiba:              "Toshiba",
	OemIntel:                "Intel Corporation",
	OemTatung:               "Tatung",
	OemHitachi399:           "Hitachi",
	OemDell:                 "Dell Inc",
	OemLMC:                  "LMC",
	OemRadiSys:              "RadiSys Corporation",
	OemBroadcom:             "Broadcom Corporation",
	OemMagnum:               "Magnum Technologies",
	OemTyan:                 "Tyan Computer Corporation",
	OemNewisys:              "Newisys",
	OemFujitsuSiemens:       "Fujitsu Siemens",
	OemAvocent:              "Avocent",
	OemPeppercon:            "Peppercon AG",
	OemSupermicro:           "Supermicro",
	OemOSA:                  "OSA",
	OemGoogle:               "Google",
	OemPICMG:                "PICMG",
	OemRaritan:              "Raritan",
	OemKontron:              "Kontron",
	OemPPS:                  "Pigeon Point Systems",
	OemAMI:                  "AMI",
	OemNokiaSiemensNetworks: "Nokia Siemens Networks",
	OemSupermicro47488:      "Supermicro",
}

func (id OemVendorID) String() string {
	if s, ok := oemStrings[id]; ok {
		return s
	}
	return "Unknown"
}
