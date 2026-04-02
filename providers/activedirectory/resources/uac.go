// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

const (
	UACAccountDisable             int64 = 0x0002
	UACLockout                    int64 = 0x0010
	UACPasswordNotRequired        int64 = 0x0020
	UACPasswordCantChange         int64 = 0x0040
	UACEncryptedTextPwdAllowed    int64 = 0x0080
	UACNormalAccount              int64 = 0x0200
	UACServerTrustAccount         int64 = 0x2000
	UACDontExpirePassword         int64 = 0x10000
	UACSmartcardRequired          int64 = 0x40000
	UACTrustedForDelegation       int64 = 0x80000
	UACNotDelegated               int64 = 0x100000
	UACUseDesKeyOnly              int64 = 0x200000
	UACDontRequirePreauth         int64 = 0x400000
	UACPasswordExpired            int64 = 0x800000
	UACTrustedToAuthForDelegation int64 = 0x1000000
	UACPartialSecretsAccount      int64 = 0x04000000
)

func uacHasFlag(uac int64, flag int64) bool {
	return (uac & flag) != 0
}
