// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build windows
// +build windows

package windows

import (
	"fmt"
	"os"
	"unsafe"

	"github.com/rs/zerolog/log"
	"golang.org/x/sys/windows"
)

var (
	modAdvapi32 = windows.NewLazySystemDLL("advapi32.dll")

	procAuditEnumerateCategories    = modAdvapi32.NewProc("AuditEnumerateCategories")
	procAuditEnumerateSubCategories = modAdvapi32.NewProc("AuditEnumerateSubCategories")
	procAuditQuerySystemPolicy      = modAdvapi32.NewProc("AuditQuerySystemPolicy")
	procAuditLookupSubCategoryName  = modAdvapi32.NewProc("AuditLookupSubCategoryNameW")
	procAuditFree                   = modAdvapi32.NewProc("AuditFree")
)

// Audit policy constants from ntsecapi.h
const (
	POLICY_AUDIT_EVENT_UNCHANGED = 0x00000000
	POLICY_AUDIT_EVENT_SUCCESS   = 0x00000001
	POLICY_AUDIT_EVENT_FAILURE   = 0x00000002
	POLICY_AUDIT_EVENT_NONE      = 0x00000004
)

// AUDIT_POLICY_INFORMATION structure
type auditPolicyInformation struct {
	AuditSubCategoryGuid windows.GUID
	AuditingInformation  uint32
	AuditCategoryGuid    windows.GUID
}

// GetNativeAuditpol retrieves audit policy using the native Windows API
func GetNativeAuditpol() ([]AuditpolEntry, error) {
	log.Debug().Msg("retrieving audit policy using native Windows API")

	hostname, err := os.Hostname()
	if err != nil {
		hostname = ""
	}

	var result []AuditpolEntry

	// Enumerate all audit categories
	var categoryCount uint32
	var categoryGuids *windows.GUID

	ret, _, err := procAuditEnumerateCategories.Call(
		uintptr(unsafe.Pointer(&categoryGuids)),
		uintptr(unsafe.Pointer(&categoryCount)),
	)
	if ret == 0 {
		return nil, fmt.Errorf("AuditEnumerateCategories failed: %w", err)
	}
	defer procAuditFree.Call(uintptr(unsafe.Pointer(categoryGuids)))

	// Convert to slice
	categories := unsafe.Slice(categoryGuids, categoryCount)

	for i := uint32(0); i < categoryCount; i++ {
		categoryGuid := categories[i]

		// Enumerate subcategories for this category
		var subCategoryCount uint32
		var subCategoryGuids *windows.GUID

		ret, _, err = procAuditEnumerateSubCategories.Call(
			uintptr(unsafe.Pointer(&categoryGuid)),
			0, // bRetrieveAllSubCategories = FALSE
			uintptr(unsafe.Pointer(&subCategoryGuids)),
			uintptr(unsafe.Pointer(&subCategoryCount)),
		)
		if ret == 0 {
			log.Debug().Err(err).Msg("AuditEnumerateSubCategories failed")
			continue
		}

		if subCategoryCount == 0 {
			continue
		}

		// Convert to slice
		subCategories := unsafe.Slice(subCategoryGuids, subCategoryCount)

		// Query the policy for all subcategories at once
		var policyInfo *auditPolicyInformation
		ret, _, err = procAuditQuerySystemPolicy.Call(
			uintptr(unsafe.Pointer(subCategoryGuids)),
			uintptr(subCategoryCount),
			uintptr(unsafe.Pointer(&policyInfo)),
		)
		if ret == 0 {
			procAuditFree.Call(uintptr(unsafe.Pointer(subCategoryGuids)))
			log.Debug().Err(err).Msg("AuditQuerySystemPolicy failed")
			continue
		}

		// Convert policy info to slice
		policies := unsafe.Slice(policyInfo, subCategoryCount)

		for j := uint32(0); j < subCategoryCount; j++ {
			subCategoryGuid := subCategories[j]
			policy := policies[j]

			// Get subcategory name
			subCategoryName, err := lookupSubCategoryName(&subCategoryGuid)
			if err != nil {
				log.Debug().Err(err).Str("guid", guidToString(subCategoryGuid)).Msg("failed to lookup subcategory name")
				subCategoryName = guidToString(subCategoryGuid)
			}

			// Convert auditing information to inclusion setting string
			inclusionSetting := auditingInfoToString(policy.AuditingInformation)

			entry := AuditpolEntry{
				MachineName:      hostname,
				PolicyTarget:     "System",
				Subcategory:      subCategoryName,
				SubcategoryGUID:  guidToString(subCategoryGuid),
				InclusionSetting: inclusionSetting,
				ExclusionSetting: "",
			}
			result = append(result, entry)
		}

		procAuditFree.Call(uintptr(unsafe.Pointer(policyInfo)))
		procAuditFree.Call(uintptr(unsafe.Pointer(subCategoryGuids)))
	}

	return result, nil
}

func lookupSubCategoryName(guid *windows.GUID) (string, error) {
	var namePtr *uint16
	ret, _, err := procAuditLookupSubCategoryName.Call(
		uintptr(unsafe.Pointer(guid)),
		uintptr(unsafe.Pointer(&namePtr)),
	)
	if ret == 0 {
		return "", fmt.Errorf("AuditLookupSubCategoryName failed: %w", err)
	}
	defer procAuditFree.Call(uintptr(unsafe.Pointer(namePtr)))

	return windows.UTF16PtrToString(namePtr), nil
}

func guidToString(guid windows.GUID) string {
	return fmt.Sprintf("%08X-%04X-%04X-%02X%02X-%02X%02X%02X%02X%02X%02X",
		guid.Data1, guid.Data2, guid.Data3,
		guid.Data4[0], guid.Data4[1],
		guid.Data4[2], guid.Data4[3], guid.Data4[4], guid.Data4[5], guid.Data4[6], guid.Data4[7])
}

func auditingInfoToString(info uint32) string {
	hasSuccess := (info & POLICY_AUDIT_EVENT_SUCCESS) != 0
	hasFailure := (info & POLICY_AUDIT_EVENT_FAILURE) != 0

	if hasSuccess && hasFailure {
		return "Success and Failure"
	} else if hasSuccess {
		return "Success"
	} else if hasFailure {
		return "Failure"
	}
	return "No Auditing"
}

// NativeAuditpolSupported returns true on Windows
func NativeAuditpolSupported() bool {
	// Check if the required DLL functions are available
	if err := procAuditEnumerateCategories.Find(); err != nil {
		return false
	}
	return true
}
