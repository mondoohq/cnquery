// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"go.mondoo.com/cnquery/v12/providers-sdk/v1/testutils"
)

func TestAwsIamSamlProviders(t *testing.T) {
	t.Run("list SAML providers", func(t *testing.T) {
		res := testutils.InitTester(t, testutils.LinuxMock())
		r := res.TestQuery(t, "aws.iam.samlProviders")
		assert := r.ErrorAsAssertion
		assert.NoError()
	})
}

func TestAwsIamOidcProviders(t *testing.T) {
	t.Run("list OIDC providers", func(t *testing.T) {
		res := testutils.InitTester(t, testutils.LinuxMock())
		r := res.TestQuery(t, "aws.iam.oidcProviders")
		assert := r.ErrorAsAssertion
		assert.NoError()
	})
}

func TestAwsIamSamlProviderFields(t *testing.T) {
	t.Run("access SAML provider fields", func(t *testing.T) {
		res := testutils.InitTester(t, testutils.LinuxMock())
		r := res.TestQuery(t, "aws.iam.samlProviders { arn name createdAt validUntil tags }")
		assert := r.ErrorAsAssertion
		assert.NoError()
	})
}

func TestAwsIamOidcProviderFields(t *testing.T) {
	t.Run("access OIDC provider fields", func(t *testing.T) {
		res := testutils.InitTester(t, testutils.LinuxMock())
		r := res.TestQuery(t, "aws.iam.oidcProviders { arn url clientIds thumbprints createdAt tags }")
		assert := r.ErrorAsAssertion
		assert.NoError()
	})
}
