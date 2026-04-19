// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package pubspeclock

// pubspecLock represents a parsed pubspec.lock file.
type pubspecLock struct {
	Packages map[string]pubspecPackage `yaml:"packages"`

	// evidence is a list of file paths where the pubspec.lock was found.
	evidence []string `yaml:"-"`
}

// pubspecPackage represents a single package entry in pubspec.lock.
type pubspecPackage struct {
	Dependency  string             `yaml:"dependency"`
	Description pubspecDescription `yaml:"description"`
	Source      string             `yaml:"source"`
	Version     string             `yaml:"version"`
}

// pubspecDescription can be either a map (hosted packages) or a string (sdk packages).
type pubspecDescription struct {
	Name   string `yaml:"name"`
	Sha256 string `yaml:"sha256"`
	URL    string `yaml:"url"`
}

// UnmarshalYAML handles the description field which can be a string or a map.
func (d *pubspecDescription) UnmarshalYAML(unmarshal func(any) error) error {
	// Try as string first (sdk packages use a plain string like "flutter")
	var str string
	if err := unmarshal(&str); err == nil {
		d.Name = str
		return nil
	}

	// Try as map (hosted packages)
	type plain pubspecDescription
	var p plain
	if err := unmarshal(&p); err != nil {
		return err
	}
	*d = pubspecDescription(p)
	return nil
}

// isDirectMain returns true if this is a direct production dependency.
func (p *pubspecPackage) isDirectMain() bool {
	return p.Dependency == "direct main"
}
