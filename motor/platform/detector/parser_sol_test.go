package detector_test

import (
	"testing"

	"go.mondoo.com/cnquery/motor/platform/detector"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenSolaris2009Release(t *testing.T) {
	input := `
                     OpenSolaris 2009.06 snv_111b X86
       Copyright 2009 Sun Microsystems, Inc.  All Rights Reserved.
                    Use is subject to license terms.
                          Assembled 07 May 2009
`

	r, err := detector.ParseSolarisRelease(input)
	require.NoError(t, err)

	assert.Equal(t, "opensolaris", r.ID)
	assert.Equal(t, "OpenSolaris", r.Title)
	assert.Equal(t, "2009.06", r.Release)
}

func TestSolaris11Release(t *testing.T) {
	input := `
                  Oracle Solaris 11 Express snv_151a X86
 Copyright (c) 2010, Oracle and/or its affiliates.  All rights reserved.
					   Assembled 04 November 2010
`

	r, err := detector.ParseSolarisRelease(input)
	require.NoError(t, err)

	assert.Equal(t, "solaris", r.ID)
	assert.Equal(t, "Oracle Solaris", r.Title)
	assert.Equal(t, "11", r.Release)
}

func TestSolaris10Release(t *testing.T) {
	input := `
                        Solaris 10 5/08 s10x_u5wos_10 X86
           Copyright 2008 Sun Microsystems, Inc.  All Rights Reserved.
                        Use is subject to license terms.
                             Assembled 24 March 2008

                Solaris 10 10/09 (Update 8) Patch Bundle applied.
`

	r, err := detector.ParseSolarisRelease(input)
	require.NoError(t, err)

	assert.Equal(t, "solaris", r.ID)
	assert.Equal(t, "Solaris", r.Title)
	assert.Equal(t, "10", r.Release)
}
