// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package pomproperties

// pomProperties represents the contents of a pom.properties file
// found at META-INF/maven/<groupId>/<artifactId>/pom.properties inside JARs.
type pomProperties struct {
	GroupId    string
	ArtifactId string
	Version    string

	// evidence is a list of file paths where the pom.properties was found.
	evidence []string `json:"-"`
}
