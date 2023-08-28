// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package shared

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ManifestParserSuite struct {
	suite.Suite
	manifestParser ManifestParser
}

func (s *ManifestParserSuite) SetupSuite() {
	manifest, err := LoadManifestFile("./resources/testdata/mixed.yaml")
	s.Require().NoError(err)
	manP, err := NewManifestParser(manifest, "", "")
	s.Require().NoError(err)

	s.manifestParser = manP
}

func (s *ManifestParserSuite) TestNamespace() {
	ns, err := s.manifestParser.Namespace("default")
	s.Require().NoError(err)
	s.Equal("default", ns.Name)
	s.Equal("Namespace", ns.Kind)
}

func (s *ManifestParserSuite) TestNamespaces() {
	nss, err := s.manifestParser.Namespaces()
	s.Require().NoError(err)
	s.Len(nss, 2)

	nsNames := make([]string, 0, len(nss))
	for _, ns := range nss {
		nsNames = append(nsNames, ns.Name)
		s.Equal("Namespace", ns.Kind)
	}
	s.ElementsMatch([]string{"default", "custom"}, nsNames)
}

func TestManifestParserSuite(t *testing.T) {
	suite.Run(t, new(ManifestParserSuite))
}
