package shared

import (
	"io/fs"
	"regexp"
	"strings"
)

func FindFiles(iofs fs.FS, from string, r *regexp.Regexp, typ string) ([]string, error) {
	matcher := createFindFilesMatcher(typ, r)
	matchedPaths := []string{}
	err := fs.WalkDir(iofs, from, func(p string, d fs.DirEntry, err error) error {
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
}

func (m findFilesMatcher) Match(path string, t fs.FileMode) bool {
	matchesType := m.matchesType(t)
	matchesRegex := m.matchesRegex(path)

	return matchesType && matchesRegex
}

func (m findFilesMatcher) matchesRegex(path string) bool {
	if m.r == nil {
		return true
	}
	// We don't use r.Match because we need the entire path to match
	// if we want to be compatible with find. It would probably be
	// more efficient add anchors to the regular expression
	return m.r.FindString(path) == path
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

func createFindFilesMatcher(typeStr string, r *regexp.Regexp) findFilesMatcher {
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
	}
}
