// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package manifestmf

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestManifestLicense is the regression test for Bundle-License being parsed
// into Headers and then never reported: a bundle naming its license reported
// none.
//
// Every input below is the shape of a real manifest — they were taken from a
// sample of MANIFEST.MF files on GitHub, which is also where the frequencies
// cited in the comments come from.
func TestManifestLicense(t *testing.T) {
	for _, c := range []struct {
		name   string
		header string
		want   string
	}{
		{
			// The canonical form: an identifier plus a link to its text.
			name:   "identifier with a link",
			header: `Bundle-License: Apache-2.0;link="https://www.apache.org/licenses/LICENSE-2.0"`,
			want:   "Apache-2.0",
		},
		{
			// Not SPDX, but it is what the bundle said, and normalizing it is
			// a consumer's decision.
			name:   "bare identifier",
			header: "Bundle-License: Apache 2.0",
			want:   "Apache 2.0",
		},
		{
			name:   "identifier with a link, no quotes",
			header: "Bundle-License: EPL-2.0;link=https://www.eclipse.org/legal/epl-2.0",
			want:   "EPL-2.0",
		},
		{
			// The grammar makes ',' a separator, but Eclipse/Tycho bundles
			// write an identifier that contains one. OR-joining this would
			// report the bundle as offered under "The Apache License" or
			// "Version 2.0" — a choice it never offered.
			name:   "identifier containing a comma",
			header: `Bundle-License: The Apache License, Version 2.0;link="http://www.apache.org/licenses/LICENSE-2.0.txt"`,
			want:   "The Apache License, Version 2.0",
		},
		{
			name:   "identifier with a version suffix",
			header: `Bundle-License: Eclipse Public License v2.0;link="http://www.eclipse.org/legal/epl-2.0"`,
			want:   "Eclipse Public License v2.0",
		},
		{
			// A quoted attribute value carries both separators, so splitting
			// must not see them.
			name:   "comma inside a quoted description",
			header: `Bundle-License: Apache-2.0;description="Apache License, Version 2.0";link="https://www.apache.org/licenses/LICENSE-2.0"`,
			want:   "Apache-2.0",
		},
		{
			// Two licenses, both stated as identifiers. Rejoined rather than
			// OR-joined, because a comma here is not reliably a separator.
			name:   "two identifiers",
			header: `Bundle-License: MIT;link="https://opensource.org/licenses/MIT",Apache-2.0;link="https://www.apache.org/licenses/LICENSE-2.0"`,
			want:   "MIT, Apache-2.0",
		},
		{
			// The most common real header by a wide margin, and the reason
			// this fix leaves most bundles reporting nothing. A link to the
			// terms is not the identity of the terms: a URL in a field
			// consumers compare against "Apache-2.0" matches nothing while
			// reading as a stated identifier.
			name:   "url only",
			header: "Bundle-License: https://www.apache.org/licenses/LICENSE-2.0.txt",
			want:   "",
		},
		{
			// A genuine dual license, stated as two URLs. Nothing nameable.
			name:   "two urls",
			header: "Bundle-License: https://www.eclipse.org/org/documents/epl-2.0/EPL-2.0.txt, https://www.gnu.org/software/classpath/license.html",
			want:   "",
		},
		{
			// The description names the license where the identifier is a URL,
			// but it is prose — bnd's own example is a whole sentence — so it
			// is not promoted to an identifier.
			name:   "url identifier with a describing attribute",
			header: "Bundle-License: https://opensource.org/licenses/BSD-2-Clause;description=BSD 2-Clause License",
			want:   "",
		},
		{
			// The magic value states that the license is *not* here.
			name:   "external",
			header: "Bundle-License: <<EXTERNAL>>",
			want:   "",
		},
		{
			name:   "no Bundle-License header",
			header: "Bundle-Name: Example",
			want:   "",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			mf := "Manifest-Version: 1.0\nBundle-SymbolicName: com.example\nBundle-Version: 1.0.0\n" + c.header + "\n"
			info, err := (&Extractor{}).Parse(strings.NewReader(mf), "META-INF/MANIFEST.MF")
			require.NoError(t, err)

			root := info.Root()
			require.NotNil(t, root)
			assert.Equal(t, c.want, root.License)
		})
	}
}

// TestManifestLicenseContinuationLine covers a Bundle-License wrapped across
// source lines, which is how a manifest carrying a link is normally written —
// the header limit is 72 bytes, so the real ones are nearly always wrapped.
func TestManifestLicenseContinuationLine(t *testing.T) {
	mf := "Manifest-Version: 1.0\n" +
		"Bundle-SymbolicName: com.example\n" +
		"Bundle-License: The Apache License, Version 2.0;link=\"http://www.apache.\n" +
		" org/licenses/LICENSE-2.0.txt\"\n"

	info, err := (&Extractor{}).Parse(strings.NewReader(mf), "META-INF/MANIFEST.MF")
	require.NoError(t, err)

	root := info.Root()
	require.NotNil(t, root)
	assert.Equal(t, "The Apache License, Version 2.0", root.License)
}

// TestManifestLicenseFixtures pins the checked-in fixtures. The OSGi one states
// its license as a URL, so it reports none — the same as before this fix, and
// deliberately: it must not start emitting the URL as an identifier.
func TestManifestLicenseFixtures(t *testing.T) {
	for _, c := range []struct{ fixture, want string }{
		{"./testdata/osgi.MANIFEST.MF", ""},
		{"./testdata/simple.MANIFEST.MF", ""},
	} {
		t.Run(c.fixture, func(t *testing.T) {
			f, err := os.Open(c.fixture)
			require.NoError(t, err)
			defer f.Close()

			info, err := (&Extractor{}).Parse(f, "META-INF/MANIFEST.MF")
			require.NoError(t, err)
			assert.Equal(t, c.want, info.Root().License)
		})
	}
}
