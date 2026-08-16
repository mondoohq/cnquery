// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io"
	"path"
	"sort"
	"strings"
	"sync"

	"github.com/spf13/afero"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"go.mondoo.com/mql/v13/providers/os/resources/java"
)

type mqlJavaKeystoreInternal struct {
	lock     sync.Mutex
	parsed   *java.Keystore
	parseErr error
}

type mqlJavaKeystoreEntryInternal struct {
	// certs holds the entry's DER-encoded certificates, carried over from the
	// parse so the entry does not have to re-read the store.
	certs [][]byte
}

func initJavaKeystore(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if x, ok := args["path"]; ok {
		p, ok := x.Value.(string)
		if !ok {
			return nil, nil, errors.New("wrong type for 'path' in java.keystore initialization, it must be a string")
		}
		f, err := CreateResource(runtime, "file", map[string]*llx.RawData{
			"path": llx.StringData(p),
		})
		if err != nil {
			return nil, nil, err
		}
		args["file"] = llx.ResourceData(f, "file")
	}
	return args, nil, nil
}

func (s *mqlJavaKeystore) id() (string, error) {
	return "java.keystore/" + s.Path.Data, nil
}

func (s *mqlJavaKeystore) file() (*mqlFile, error) {
	f, err := CreateResource(s.MqlRuntime, "file", map[string]*llx.RawData{
		"path": llx.StringData(s.Path.Data),
	})
	if err != nil {
		return nil, err
	}
	return f.(*mqlFile), nil
}

// read parses the store once and caches it on the resource. Every field goes
// through here so a store is read a single time however many are asked for.
func (s *mqlJavaKeystore) read() (*java.Keystore, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.parsed != nil {
		return s.parsed, nil
	}
	if s.parseErr != nil {
		return nil, s.parseErr
	}

	conn := s.MqlRuntime.Connection.(shared.Connection)
	afs := &afero.Afero{Fs: conn.FileSystem()}

	f, err := afs.Open(s.Path.Data)
	if err != nil {
		s.parseErr = err
		return nil, err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		s.parseErr = err
		return nil, err
	}

	// The password is only consulted for PKCS#12; a JKS store does not need one.
	ks, err := java.Parse(data, "")
	if err != nil {
		s.parseErr = err
		return nil, err
	}

	s.parsed = ks
	return ks, nil
}

func (s *mqlJavaKeystore) format() (string, error) {
	ks, err := s.read()
	if err != nil {
		return "", err
	}
	return ks.Format, nil
}

func (s *mqlJavaKeystore) entries() ([]any, error) {
	ks, err := s.read()
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(ks.Entries))
	for i := range ks.Entries {
		entry := ks.Entries[i]

		createdAt := llx.NilData
		if !entry.CreatedAt.IsZero() {
			at := entry.CreatedAt
			createdAt = llx.TimeData(at)
		}

		raw, err := CreateResource(s.MqlRuntime, "java.keystore.entry", map[string]*llx.RawData{
			"__id":                 llx.StringData(s.Path.Data + "/" + entry.Alias),
			"alias":                llx.StringData(entry.Alias),
			"isTrustedCertificate": llx.BoolData(entry.Trusted),
			"createdAt":            createdAt,
		})
		if err != nil {
			return nil, err
		}
		mqlEntry := raw.(*mqlJavaKeystoreEntry)
		mqlEntry.certs = entry.Certs
		res = append(res, mqlEntry)
	}
	return res, nil
}

func (s *mqlJavaKeystore) certificates() ([]any, error) {
	ks, err := s.read()
	if err != nil {
		return nil, err
	}

	var der [][]byte
	for i := range ks.Entries {
		der = append(der, ks.Entries[i].Certs...)
	}
	return certificatesFromDER(s.MqlRuntime, der)
}

// unreadableCertificates counts what certificates() had to skip.
func (s *mqlJavaKeystore) unreadableCertificates() (int64, error) {
	ks, err := s.read()
	if err != nil {
		return 0, err
	}

	var der [][]byte
	for i := range ks.Entries {
		der = append(der, ks.Entries[i].Certs...)
	}
	_, unreadable := readableDER(der)
	return int64(unreadable), nil
}

func (s *mqlJavaKeystoreEntry) id() (string, error) {
	return s.__id, nil
}

func (s *mqlJavaKeystoreEntry) certificates() ([]any, error) {
	return certificatesFromDER(s.MqlRuntime, s.certs)
}

// readableDER splits certificates into those the X.509 parser accepts and a
// count of those it does not.
//
// A trust store assembled over years holds certificates that no longer parse —
// a negative serial number is the common one, legal when the certificate was
// issued and rejected by Go today. RHEL 7's store has exactly one such
// certificate out of 133. Handing the whole batch downstream fails on the first
// of them and the store reports nothing at all, so one twenty-year-old CA
// blinds every check. Skipping them keeps the other 132 assertable; the count is
// what stops that being a silent loss.
func readableDER(der [][]byte) ([][]byte, int) {
	out := make([][]byte, 0, len(der))
	unreadable := 0
	for i := range der {
		if _, err := x509.ParseCertificate(der[i]); err != nil {
			unreadable++
			continue
		}
		out = append(out, der[i])
	}
	return out, unreadable
}

// certificatesFromDER hands DER-encoded certificates to the shared
// `certificates` resource, which is what produces `network.certificate`. The
// bytes are PEM-wrapped first because that is the interface it takes.
func certificatesFromDER(runtime *plugin.Runtime, der [][]byte) ([]any, error) {
	der, _ = readableDER(der)
	if len(der) == 0 {
		return []any{}, nil
	}

	var buf strings.Builder
	for i := range der {
		if err := pem.Encode(&buf, &pem.Block{Type: "CERTIFICATE", Bytes: der[i]}); err != nil {
			return nil, err
		}
	}

	certs, err := runtime.CreateSharedResource("certificates", map[string]*llx.RawData{
		"pem": llx.StringData(buf.String()),
	})
	if err != nil {
		return nil, err
	}

	list, err := runtime.GetSharedData("certificates", certs.MqlID(), "list")
	if err != nil {
		return nil, err
	}
	return list.Value.([]any), nil
}

// javaTruststoreDirs are the directories a JDK or JRE keeps its trust store in,
// relative to the runtime's own root. JDK 9 moved it up out of `jre/`.
var javaTruststoreDirs = []string{
	"lib/security",
	"jre/lib/security",
}

// javaHomeRoots are the directories JVMs get installed under. Each is scanned
// one level deep, since a machine commonly has several runtimes side by side.
var javaHomeRoots = []string{
	"/usr/lib/jvm",
	"/usr/java",
	"/opt/java",
	"/opt/jdk",
	"/Library/Java/JavaVirtualMachines",
}

// javaTruststoreFiles are absolute paths that hold a trust store in their own
// right: a distribution-managed store, or a JVM installed at a fixed location.
var javaTruststoreFiles = []string{
	"/etc/ssl/certs/java/cacerts",
	"/etc/pki/java/cacerts",
	"/opt/java/openjdk/lib/security/cacerts",
}

func (s *mqlJavaTruststores) id() (string, error) {
	return "java.truststores", nil
}

func (s *mqlJavaTruststores) paths() ([]any, error) {
	conn := s.MqlRuntime.Connection.(shared.Connection)
	afs := &afero.Afero{Fs: conn.FileSystem()}

	found := map[string]struct{}{}

	add := func(p string) {
		if ok, _ := afs.Exists(p); ok {
			found[p] = struct{}{}
		}
	}

	for _, p := range javaTruststoreFiles {
		add(p)
	}

	// A machine usually has more than one runtime installed, and they do not
	// have to agree about what they trust, so every one of them is reported
	// rather than the first that turns up.
	for _, root := range javaHomeRoots {
		entries, err := afs.ReadDir(root)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			// macOS nests the runtime one level further, under Contents/Home.
			for _, home := range []string{
				path.Join(root, entry.Name()),
				path.Join(root, entry.Name(), "Contents", "Home"),
			} {
				for _, dir := range javaTruststoreDirs {
					add(path.Join(home, dir, "cacerts"))
				}
			}
		}
	}

	// Sorted so the list does not depend on directory iteration order, which
	// would make a check's output shuffle between scans of the same host.
	out := make([]string, 0, len(found))
	for p := range found {
		out = append(out, p)
	}
	sort.Strings(out)

	res := make([]any, 0, len(out))
	for _, p := range out {
		res = append(res, p)
	}
	return res, nil
}

func (s *mqlJavaTruststores) list(paths []any) ([]any, error) {
	res := make([]any, 0, len(paths))
	for i := range paths {
		p, ok := paths[i].(string)
		if !ok {
			continue
		}
		raw, err := CreateResource(s.MqlRuntime, "java.keystore", map[string]*llx.RawData{
			"path": llx.StringData(p),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, raw)
	}
	return res, nil
}
