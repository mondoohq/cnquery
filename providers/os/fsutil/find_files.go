// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package fsutil

import (
	"io/fs"
	"os"
	"regexp"
	"strings"
)

func FindFiles(iofs fs.FS, from string, r *regexp.Regexp, typ string, perm *uint32, depth *int) ([]string, error) {
	matcher := createFindFilesMatcher(iofs, typ, from, r, perm, depth)
	matchedPaths := []string{}
	// Note: matcher.Match resolves a symlink to its target type so a symlink is
	// reported under the type it points at (mirroring `find -L`), but WalkDir
	// itself does not follow symlinks during traversal. So a symlinked directory
	// is reported for type:"directory" yet its contents are not walked. Full
	// `find -L` recursion would need explicit symlink following with loop
	// detection; it isn't required for the config-file discovery this powers.
	err := fs.WalkDir(iofs, from, func(p string, d fs.DirEntry, err error) error {
		if d != nil && d.IsDir() && matcher.DepthReached(p) {
			return fs.SkipDir
		}

		skipFile, err := handleFsError(err)
		if err != nil {
			return err
		}

		if skipFile {
			return nil
		}
		if matcher.Match(p, d.Type()) {
			matchedPaths = append(matchedPaths, p)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return matchedPaths, nil
}

type findFilesMatcher struct {
	types []byte
	r     *regexp.Regexp
	perm  *uint32
	depth *int
	from  string
	iofs  fs.FS
}

// Depth 0 means we only walk the current directory
// Depth 1 means we walk the current directory and its children
// Depth 2 means we walk the current directory, its children and their children
func (m findFilesMatcher) DepthReached(p string) bool {
	if m.depth == nil {
		return false
	}

	trimmed := strings.TrimPrefix(p, m.from)
	// WalkDir always uses slash for separating, ignoring the OS separator. This is why we need to replace it.
	normalized := strings.ReplaceAll(trimmed, string(os.PathSeparator), "/")
	depth := strings.Count(normalized, "/")
	return depth > *m.depth
}

func (m findFilesMatcher) Match(path string, t fs.FileMode) bool {
	matchesType := m.matchesType(path, t)
	matchesRegex := m.matchesRegex(path)
	matchesPerm := m.matchesPerm(path)

	return matchesType && matchesRegex && matchesPerm
}

func (m findFilesMatcher) matchesRegex(path string) bool {
	if m.r == nil {
		return true
	}
	// We don't use r.Match because we need the entire path to match
	// if we want to be compatible with find. It would probably be
	// more efficient add anchors to the regular expression
	match := m.r.FindString(path)
	return match == path
}

func (m findFilesMatcher) matchesType(path string, entryType fs.FileMode) bool {
	if len(m.types) == 0 {
		return true
	}

	// WalkDir reports lstat info, so a symlink keeps fs.ModeSymlink and never
	// matches file/dir/etc. on its own. The command-based backend uses
	// `find -L`, which follows symlinks and tests the target's type; mirror
	// that here by resolving the symlink's target. Without this, a symlinked
	// regular file (e.g. authselect manages /etc/pam.d service files as
	// symlinks into /etc/authselect) is skipped by type:"file".
	// See https://github.com/mondoohq/mql/issues/8467
	// This is purely additive: a symlink still matches type:"link" (handled
	// against the original lstat mode below), it now ALSO matches the type of
	// whatever it points at. Broken symlinks fail to resolve and keep their
	// symlink type.
	resolvedType := entryType
	if entryType&fs.ModeSymlink != 0 && m.wantsNonSymlinkType() {
		if info, err := fs.Stat(m.iofs, path); err == nil {
			resolvedType = info.Mode()
		}
	}

	for _, at := range m.types {
		var matches bool
		switch at {
		case 'b':
			matches = (resolvedType&fs.ModeDevice) != 0 && (resolvedType&fs.ModeCharDevice) == 0
		case 'c':
			matches = (resolvedType&fs.ModeDevice) != 0 && (resolvedType&fs.ModeCharDevice) != 0
		case 'd':
			matches = resolvedType.IsDir()
		case 'p':
			matches = (resolvedType & fs.ModeNamedPipe) != 0
		case 'f':
			matches = resolvedType.IsRegular()
		case 'l':
			// match against the original mode so any symlink matches, including
			// one that points at a regular file or directory.
			matches = (entryType & fs.ModeSymlink) != 0
		}
		if matches {
			return true
		}
	}
	return false
}

// wantsNonSymlinkType reports whether any requested type other than "link" is
// set, i.e. whether resolving a symlink's target type could change the result.
// It lets us skip the extra stat for plain type:"link" lookups.
func (m findFilesMatcher) wantsNonSymlinkType() bool {
	for _, at := range m.types {
		if at != 'l' {
			return true
		}
	}
	return false
}

func (m findFilesMatcher) matchesPerm(path string) bool {
	if m.perm == nil {
		return true
	}
	// A zero permission mask is "no permission filter", matching `find -perm -0`
	// (every file has all of zero bits set) and the command-based find backend.
	// files.find created via CreateResource without a `permissions` argument
	// arrives here with a 0 mask (initFilesFind's 0o777 default only applies to
	// MQL-instantiated resources), and the pam/sudoers/modprobe/rsyslog/... `*.d`
	// discoveries all rely on that. Without this guard a 0 mask matched nothing
	// (`mode & 0 == 0`), so those lookups returned no files on filesystem and
	// tar connections. See https://github.com/mondoohq/mql/issues/8467
	if *m.perm == 0 {
		return true
	}
	info, err := fs.Stat(m.iofs, path)
	if err != nil {
		return false
	}

	// If the permissions don't match continue
	if uint32(info.Mode().Perm())&*m.perm == 0 {
		return false
	}
	return true
}

func createFindFilesMatcher(iofs fs.FS, typeStr string, from string, r *regexp.Regexp, perm *uint32, depth *int) findFilesMatcher {
	allowed := []byte{}
	types := strings.Split(typeStr, ",")
	for _, t := range types {
		if len(t) == 0 {
			continue
		}
		firstChar := t[0]
		switch firstChar {
		case 'b', 'c', 'd', 'p', 'f', 'l':
			allowed = append(allowed, firstChar)
		default:
		}
	}
	return findFilesMatcher{
		types: allowed,
		r:     r,
		perm:  perm,
		iofs:  iofs,
		depth: depth,
		from:  from,
	}
}
