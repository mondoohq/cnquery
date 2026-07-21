// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"io"

	"github.com/spf13/afero"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/providers/os/connection/shared"
)

// computeHashes reads the file exactly once and computes its MD5, SHA1, and
// SHA256 digests together, caching all three on the resource. It is safe to call
// from any of the three getters; the first call does the work and the rest reuse
// the cached values. Directories, missing files, and unreadable paths resolve all
// three digests to null (rather than an empty-string hash) so policies can tell
// "no hash" apart from a real digest.
func (s *mqlFile) computeHashes(path string) error {
	if s.Sha256.IsSet() {
		// a previous getter already computed (or null-ed) the digests
		return nil
	}

	nullAll := func() {
		null := plugin.TValue[string]{State: plugin.StateIsSet | plugin.StateIsNull}
		s.Md5 = null
		s.Sha1 = null
		s.Sha256 = null
	}

	conn := s.MqlRuntime.Connection.(shared.Connection)
	afs := &afero.Afero{Fs: conn.FileSystem()}

	fi, err := afs.Stat(path)
	if err != nil || fi.IsDir() {
		nullAll()
		return nil
	}

	f, err := afs.Open(path)
	if err != nil {
		nullAll()
		return nil
	}
	defer f.Close()

	hMd5 := md5.New()
	hSha1 := sha1.New()
	hSha256 := sha256.New()
	if _, err := io.Copy(io.MultiWriter(hMd5, hSha1, hSha256), f); err != nil {
		nullAll()
		return nil
	}

	digest := func(h hash.Hash) plugin.TValue[string] {
		return plugin.TValue[string]{
			Data:  hex.EncodeToString(h.Sum(nil)),
			State: plugin.StateIsSet,
		}
	}
	s.Md5 = digest(hMd5)
	s.Sha1 = digest(hSha1)
	s.Sha256 = digest(hSha256)
	return nil
}

func (s *mqlFile) md5(path string) (string, error) {
	if err := s.computeHashes(path); err != nil {
		return "", err
	}
	return s.Md5.Data, nil
}

func (s *mqlFile) sha1(path string) (string, error) {
	if err := s.computeHashes(path); err != nil {
		return "", err
	}
	return s.Sha1.Data, nil
}

func (s *mqlFile) sha256(path string) (string, error) {
	if err := s.computeHashes(path); err != nil {
		return "", err
	}
	return s.Sha256.Data, nil
}
