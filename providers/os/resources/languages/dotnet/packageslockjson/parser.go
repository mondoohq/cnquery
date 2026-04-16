// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package packageslockjson

// packagesLock represents a parsed packages.lock.json file.
type packagesLock struct {
	Version      int                                       `json:"version"`
	Dependencies map[string]map[string]packagesLockPackage `json:"dependencies"`

	// evidence is a list of file paths where the lock file was found.
	evidence []string `json:"-"`
}

// packagesLockPackage represents a single dependency entry.
type packagesLockPackage struct {
	Type        string `json:"type"`
	Requested   string `json:"requested"`
	Resolved    string `json:"resolved"`
	ContentHash string `json:"contentHash"`
}

// isDirect returns true if this is a directly referenced package.
func (p *packagesLockPackage) isDirect() bool {
	return p.Type == "Direct"
}
