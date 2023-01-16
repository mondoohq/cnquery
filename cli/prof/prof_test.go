package prof

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	t.Run("no options provided", func(t *testing.T) {
		{
			opts, err := parseProf("")
			require.NoError(t, err)
			require.Equal(t, defaultOpts, opts)
		}

		{
			opts, err := parseProf("     ")
			require.NoError(t, err)
			require.Equal(t, defaultOpts, opts)
		}

		{
			opts, err := parseProf(" , ,,,   ")
			require.NoError(t, err)
			require.Equal(t, defaultOpts, opts)
		}
	})

	t.Run("enable", func(t *testing.T) {
		{
			expected := defaultOpts
			expected.Enabled = true

			opts, err := parseProf("enable")
			require.NoError(t, err)
			require.Equal(t, expected, opts)
		}

		{
			expected := defaultOpts
			expected.Enabled = true

			opts, err := parseProf("enable=true")
			require.NoError(t, err)
			require.Equal(t, expected, opts)
		}

		{
			expected := defaultOpts

			opts, err := parseProf("enable=truce")
			require.NoError(t, err)
			require.Equal(t, expected, opts)
		}
	})

	t.Run("enabled", func(t *testing.T) {
		{
			expected := defaultOpts
			expected.Enabled = true

			opts, err := parseProf("enabled")
			require.NoError(t, err)
			require.Equal(t, expected, opts)
		}

		{
			expected := defaultOpts
			expected.Enabled = true

			opts, err := parseProf("enabled=true")
			require.NoError(t, err)
			require.Equal(t, expected, opts)
		}

		{
			expected := defaultOpts

			opts, err := parseProf("enabled=truce")
			require.NoError(t, err)
			require.Equal(t, expected, opts)
		}
	})

	t.Run("listen", func(t *testing.T) {
		{
			expected := defaultOpts

			opts, err := parseProf("listen")
			require.NoError(t, err)
			require.Equal(t, expected, opts)
		}

		{
			expected := defaultOpts

			opts, err := parseProf("listen=")
			require.NoError(t, err)
			require.Equal(t, expected, opts)
		}

		{
			expected := defaultOpts
			expected.Listen = "localhost:7474"
			opts, err := parseProf("listen=localhost:7474")
			require.NoError(t, err)
			require.Equal(t, expected, opts)
		}
	})

	t.Run("memprofilerate", func(t *testing.T) {
		{
			expected := defaultOpts

			opts, err := parseProf("memprofilerate")
			require.NoError(t, err)
			require.Equal(t, expected, opts)
		}

		{
			expected := defaultOpts

			opts, err := parseProf("memprofilerate=")
			require.NoError(t, err)
			require.Equal(t, expected, opts)
		}

		{
			_, err := parseProf("memprofilerate=notanumber")
			require.Error(t, err)
		}

		{
			expected := defaultOpts
			expectedMemProfileRate := 43
			expected.MemProfileRate = &expectedMemProfileRate

			opts, err := parseProf("memprofilerate=43")
			require.NoError(t, err)
			require.Equal(t, expected, opts)
		}
	})

	t.Run("all together", func(t *testing.T) {
		{
			expected := defaultOpts
			expected.Enabled = true
			expectedMemProfileRate := 43
			expected.MemProfileRate = &expectedMemProfileRate
			expected.Listen = "localhost:7474"

			opts, err := parseProf("enabled,memprofilerate = 43, listen= localhost:7474")
			require.NoError(t, err)
			require.Equal(t, expected, opts)
		}
	})
}
