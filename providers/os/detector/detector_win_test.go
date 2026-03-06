// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package detector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers/os/connection/mock"
)

func TestDetectIntuneDeviceID(t *testing.T) {
	// Hash of the encoded Intune PowerShell command
	const intuneCommandHash = "ee17502eeba7988928f8b668d05e2cb9e35bdc4cf9ebcde632470d87aa6a6905"

	intuneEnrolledMock := &mock.TomlData{
		Commands: map[string]*mock.Command{
			intuneCommandHash: {
				Stdout: `{"EnrollmentGUID":"12345678-1234-1234-1234-123456789012","EntDMID":"abcdef12-3456-7890-abcd-ef1234567890"}`,
			},
		},
	}

	intuneNotEnrolledMock := &mock.TomlData{
		Commands: map[string]*mock.Command{
			intuneCommandHash: {
				Stdout: "",
			},
		},
	}

	t.Run("workstation should detect Intune device ID", func(t *testing.T) {
		conn, err := mock.New(0, &inventory.Asset{}, mock.WithData(intuneEnrolledMock))
		require.NoError(t, err)

		pf := &inventory.Platform{
			Title: "Windows 10 Enterprise",
			Labels: map[string]string{
				"windows.mondoo.com/product-type": "1",
			},
		}

		detectIntuneDeviceID(pf, conn)
		assert.Equal(t, "abcdef12-3456-7890-abcd-ef1234567890", pf.Labels["windows.mondoo.com/intune-device-id"])
	})

	t.Run("workstation not enrolled should not set label", func(t *testing.T) {
		conn, err := mock.New(0, &inventory.Asset{}, mock.WithData(intuneNotEnrolledMock))
		require.NoError(t, err)

		pf := &inventory.Platform{
			Title: "Windows 10 Enterprise",
			Labels: map[string]string{
				"windows.mondoo.com/product-type": "1",
			},
		}

		detectIntuneDeviceID(pf, conn)
		_, exists := pf.Labels["windows.mondoo.com/intune-device-id"]
		assert.False(t, exists)
	})

	t.Run("Windows 11 multi-session should detect Intune device ID", func(t *testing.T) {
		conn, err := mock.New(0, &inventory.Asset{}, mock.WithData(intuneEnrolledMock))
		require.NoError(t, err)

		pf := &inventory.Platform{
			Title: "Windows 11 Enterprise Multi-Session",
			Labels: map[string]string{
				"windows.mondoo.com/product-type": "3",
			},
		}

		detectIntuneDeviceID(pf, conn)
		assert.Equal(t, "abcdef12-3456-7890-abcd-ef1234567890", pf.Labels["windows.mondoo.com/intune-device-id"])
	})

	t.Run("Windows Server should skip Intune detection", func(t *testing.T) {
		conn, err := mock.New(0, &inventory.Asset{}, mock.WithData(intuneEnrolledMock))
		require.NoError(t, err)

		pf := &inventory.Platform{
			Title: "Windows Server 2022 Datacenter",
			Labels: map[string]string{
				"windows.mondoo.com/product-type": "3",
			},
		}

		detectIntuneDeviceID(pf, conn)
		_, exists := pf.Labels["windows.mondoo.com/intune-device-id"]
		assert.False(t, exists)
	})

	t.Run("Windows Server 2025 should skip Intune detection", func(t *testing.T) {
		conn, err := mock.New(0, &inventory.Asset{}, mock.WithData(intuneEnrolledMock))
		require.NoError(t, err)

		pf := &inventory.Platform{
			Title: "Windows Server 2025 Datacenter",
			Labels: map[string]string{
				"windows.mondoo.com/product-type": "3",
			},
		}

		detectIntuneDeviceID(pf, conn)
		_, exists := pf.Labels["windows.mondoo.com/intune-device-id"]
		assert.False(t, exists)
	})

	t.Run("domain controller should skip Intune detection", func(t *testing.T) {
		conn, err := mock.New(0, &inventory.Asset{}, mock.WithData(intuneEnrolledMock))
		require.NoError(t, err)

		pf := &inventory.Platform{
			Title: "Windows Server 2022 Datacenter",
			Labels: map[string]string{
				"windows.mondoo.com/product-type": "2",
			},
		}

		detectIntuneDeviceID(pf, conn)
		_, exists := pf.Labels["windows.mondoo.com/intune-device-id"]
		assert.False(t, exists)
	})

	t.Run("empty product-type should skip Intune detection", func(t *testing.T) {
		conn, err := mock.New(0, &inventory.Asset{}, mock.WithData(intuneEnrolledMock))
		require.NoError(t, err)

		pf := &inventory.Platform{
			Title:  "Windows",
			Labels: map[string]string{},
		}

		detectIntuneDeviceID(pf, conn)
		_, exists := pf.Labels["windows.mondoo.com/intune-device-id"]
		assert.False(t, exists)
	})
}
