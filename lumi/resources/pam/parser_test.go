package pam

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseLine(t *testing.T) {
	t.Run("parsing conf lines", func(t *testing.T) {
		line := "account    required       pam_opendirectory.so"
		expected := &PamLine{
			PamType: "account",
			Control: "required",
			Module:  "pam_opendirectory.so",
			Options: []interface{}{},
		}
		result, err := ParseLine(line)
		require.NoError(t, err)
		require.Equal(t, expected, result)
	})

	t.Run("parsing conf lines with options", func(t *testing.T) {
		line := "account    required       pam_opendirectory.so no_warn group=admin,wheel"
		expected := &PamLine{
			PamType: "account",
			Control: "required",
			Module:  "pam_opendirectory.so",
			Options: []interface{}{"no_warn", "group=admin,wheel"},
		}
		result, err := ParseLine(line)
		require.NoError(t, err)
		require.Equal(t, expected, result)
	})

	t.Run("parsing conf lines with complicated control", func(t *testing.T) {
		line := "account     [default=bad success=ok user_unknown=ignore] pam_sss.so"
		expected := &PamLine{
			PamType: "account",
			Control: "[default=bad success=ok user_unknown=ignore]",
			Module:  "pam_sss.so",
			Options: []interface{}{},
		}
		result, err := ParseLine(line)
		require.NoError(t, err)
		require.Equal(t, expected, result)
	})

	t.Run("parsing conf lines with complicated control and options", func(t *testing.T) {
		line := "account    [default=bad success=ok user_unknown=ignore]       pam_opendirectory.so no_warn group=admin,wheel"
		expected := &PamLine{
			PamType: "account",
			Control: "[default=bad success=ok user_unknown=ignore]",
			Module:  "pam_opendirectory.so",
			Options: []interface{}{"no_warn", "group=admin,wheel"},
		}
		result, err := ParseLine(line)
		require.NoError(t, err)
		require.Equal(t, expected, result)
	})
}
