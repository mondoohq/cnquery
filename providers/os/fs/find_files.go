// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package fs

import (
	"io/fs"
	"os"
	"regexp"
	"strings"
)

func FindFiles(iofs fs.FS, from string, r *regexp.Regexp, typ string, perm *uint32, depth *int) ([]string, error) {
	matcher := createFindFilesMatcher(iofs, typ, from, r, perm, depth)
	matchedPaths := []string{}
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
	matchesType := m.matchesType(t)
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

func (m findFilesMatcher) matchesType(entryType fs.FileMode) bool {
	if len(m.types) == 0 {
		return true
	}
	for _, at := range m.types {
		var matches bool
		switch at {
		case 'b':
			matches = (entryType&fs.ModeDevice) != 0 && (entryType&fs.ModeCharDevice) == 0
		case 'c':
			matches = (entryType&fs.ModeDevice) != 0 && (entryType&fs.ModeCharDevice) != 0
		case 'd':
			matches = entryType.IsDir()
		case 'p':
			matches = (entryType & fs.ModeNamedPipe) != 0
		case 'f':
			matches = entryType.IsRegular()
		case 'l':
			matches = (entryType & fs.ModeSymlink) != 0
		}
		if matches {
			return true
		}
	}
	return false
}

func (m findFilesMatcher) matchesPerm(path string) bool {
	if m.perm == nil {
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
