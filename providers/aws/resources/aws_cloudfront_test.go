// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// originMtlsClientCertificate returns null when no client cert ARN was extracted
// from the origin's CustomOriginConfig (i.e. origin mTLS isn't configured).
func TestCloudfrontOriginMtlsClientCertificateNullWhenCacheEmpty(t *testing.T) {
	o := &mqlAwsCloudfrontDistributionOrigin{}
	got, err := o.originMtlsClientCertificate()
	require.NoError(t, err)
	assert.Nil(t, got)
	assert.True(t, o.OriginMtlsClientCertificate.IsNull())
	assert.True(t, o.OriginMtlsClientCertificate.IsSet())
}
