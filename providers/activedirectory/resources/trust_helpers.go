// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import "strings"

const (
	trustTypeDownlevel = 1
	trustTypeUplevel   = 2
	trustTypeMIT       = 3
	trustTypeDCE       = 4
	trustTypeAAD       = 5
)

const (
	trustAttrNonTransitive                        = 0x00000001
	trustAttrQuarantinedDomain                    = 0x00000004
	trustAttrForestTransitive                     = 0x00000008
	trustAttrCrossOrganization                    = 0x00000010
	trustAttrWithinForest                         = 0x00000020
	trustAttrTreatAsExternal                      = 0x00000040
	trustAttrUsesRC4Encryption                    = 0x00000080
	trustAttrUsesAESKeys                          = 0x00000100
	trustAttrCrossOrganizationNoTGTDelegation     = 0x00000200
	trustAttrPIMTrust                             = 0x00000400
	trustAttrCrossOrganizationEnableTGTDelegation = 0x00000800
)

func parseTrustType(sourceDomain, targetDomain string, trustType, trustAttrs int64) string {
	switch trustType {
	case trustTypeDownlevel:
		return "Downlevel"
	case trustTypeMIT:
		return "MIT"
	case trustTypeDCE:
		return "DCE"
	case trustTypeAAD:
		return "AzureAD"
	}

	if trustAttrs&trustAttrForestTransitive != 0 {
		return "Forest"
	}
	if trustAttrs&trustAttrWithinForest != 0 {
		if isParentChildTrust(sourceDomain, targetDomain) {
			return "ParentChild"
		}
		return "CrossLink"
	}
	if trustType == trustTypeUplevel {
		return "External"
	}
	return "Unknown"
}

func isParentChildTrust(sourceDomain, targetDomain string) bool {
	source := strings.ToLower(strings.TrimSpace(sourceDomain))
	target := strings.ToLower(strings.TrimSpace(targetDomain))
	if source == "" || target == "" || source == target {
		return false
	}
	return strings.HasSuffix(source, "."+target) || strings.HasSuffix(target, "."+source)
}

func trustUsesSelectiveAuthentication(trustAttrs int64) bool {
	return trustAttrs&trustAttrCrossOrganization != 0
}

func trustUsesRC4(trustType, trustAttrs int64) bool {
	if trustType == trustTypeMIT {
		return trustAttrs&trustAttrUsesRC4Encryption != 0
	}
	// Windows trusts default to RC4 unless AES keys are negotiated.
	if trustType == trustTypeUplevel {
		return trustAttrs&trustAttrUsesAESKeys == 0
	}
	return false
}

func trustUsesAES(trustAttrs int64) bool {
	return trustAttrs&trustAttrUsesAESKeys != 0
}

func trustAllowsTGTDelegation(trustAttrs int64) bool {
	return trustAttrs&trustAttrCrossOrganizationEnableTGTDelegation != 0
}

func trustHasSIDHistoryEnabled(trustAttrs int64) bool {
	return trustAttrs&trustAttrTreatAsExternal != 0
}

func trustHasSIDFilteringEnabled(trustAttrs int64) bool {
	return trustAttrs&trustAttrQuarantinedDomain != 0
}
