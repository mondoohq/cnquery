// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package registry

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"unicode/utf16"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
	"www.velocidex.com/golang/regparser"
)

// hiveFile reads a Windows registry hive file (SOFTWARE, SYSTEM, NTUSER.DAT, ...)
// without the Windows registry API, so a Linux or macOS scanner can read the
// registry of a mounted Windows disk. It is the cross-platform counterpart of
// RegLoadKey plus the native registry readers: values decode through
// decodeRawRegistryValue and a missing key is a NotFound error, as there.
//
// Key paths are relative to the hive root and use `\` as separator. Only `\`
// separates, because real key names contain `/` (for example
// `MIME\Database\Content Type\application/json`). Names match case-insensitively.
//
// A hive copied from a running or crashed system is often dirty: the sequence
// numbers in its base block differ, and the newest changes only exist in the
// transaction logs (<hive>.LOG1, <hive>.LOG2) next to it. When the hive is
// dirty and such logs exist, it is recovered into a temporary file that is
// deleted on Close. If recovery fails, the hive is read as it is.
type hiveFile struct {
	file *os.File
	// tmpPath is the recovered copy of the hive, deleted on Close. Empty when
	// the hive is read in place.
	tmpPath string
	reg     *regparser.Registry
	root    *regparser.CM_KEY_NODE

	mu       sync.Mutex
	children map[int64]*keyChildren
}

// keyChildren is the lazily built index of a key's subkeys.
type keyChildren struct {
	list   []*regparser.CM_KEY_NODE
	byName map[string]*regparser.CM_KEY_NODE
}

// openHiveFile opens and validates the hive at path.
func openHiveFile(path string) (*hiveFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	reg, err := regparser.NewRegistry(f)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("%s is not a valid registry hive: %w", path, err)
	}
	h := &hiveFile{file: f, reg: reg, children: map[int64]*keyChildren{}}

	if reg.BaseBlock.Sequence1() != reg.BaseBlock.Sequence2() {
		rec, err := recoverHive(f, path)
		if err != nil {
			log.Debug().Err(err).Str("path", path).Msg("registry hive is dirty and could not be recovered, reading it as is")
		} else if rreg, err := regparser.NewRegistry(rec); err != nil {
			rec.Close()
			os.Remove(rec.Name())
			log.Debug().Err(err).Str("path", path).Msg("recovered registry hive is not readable, reading the original as is")
		} else {
			f.Close()
			h.file, h.tmpPath, h.reg = rec, rec.Name(), rreg
		}
	}

	cell := h.reg.Profile.HCELL(h.reg.Reader, 0x1000+int64(h.reg.BaseBlock.RootCell()))
	h.root = cell.KeyNode()
	if h.root == nil {
		h.Close()
		return nil, fmt.Errorf("%s has no root key", path)
	}
	return h, nil
}

// recoverHive applies the transaction logs next to the hive into a temporary
// file. It fails when there is no log, so a dirty hive without logs is read as is.
func recoverHive(hive *os.File, path string) (*os.File, error) {
	var logs []*os.File
	defer func() {
		for _, l := range logs {
			l.Close()
		}
	}()
	for _, ext := range []string{".LOG1", ".LOG2", ".log1", ".log2"} {
		l, err := os.Open(path + ext)
		if err != nil {
			continue
		}
		// on case-insensitive file systems both spellings are the same file
		dup := false
		for _, other := range logs {
			dup = dup || sameFile(other, l)
		}
		if dup {
			l.Close()
			continue
		}
		logs = append(logs, l)
	}
	if len(logs) == 0 {
		return nil, errors.New("no transaction log files next to the hive")
	}
	if _, err := hive.Seek(0, 0); err != nil {
		return nil, err
	}
	return regparser.RecoverHive(hive, logs...)
}

func sameFile(a, b *os.File) bool {
	ai, err := a.Stat()
	if err != nil {
		return false
	}
	bi, err := b.Stat()
	if err != nil {
		return false
	}
	return os.SameFile(ai, bi)
}

// Close closes the hive and deletes the recovered copy, if any.
func (h *hiveFile) Close() error {
	err := h.file.Close()
	if h.tmpPath != "" {
		os.Remove(h.tmpPath)
	}
	return err
}

// keyName returns the key's name. Names are stored as extended ASCII when the
// compressed-name flag (0x20) is set, otherwise as UTF-16.
func keyName(nk *regparser.CM_KEY_NODE) string {
	return decodeHiveName(nk.Name(), nk.Flags()&0x20 != 0)
}

// valueName returns the value's name; flag 0x1 marks extended ASCII.
func valueName(vk *regparser.CM_KEY_VALUE) string {
	return decodeHiveName(vk.ValueName(), vk.Flags()&0x1 != 0)
}

func decodeHiveName(raw string, ascii bool) string {
	if !ascii {
		return string(utf16.Decode(utf16Units([]byte(raw))))
	}
	// extended ASCII is Latin-1
	r := make([]rune, len(raw))
	for i := 0; i < len(raw); i++ {
		r[i] = rune(raw[i])
	}
	return string(r)
}

// childrenOf returns the subkeys of nk, building the case-insensitive name
// index once per key. Callers list a key with thousands of subkeys and then
// read each child by path, so lookups must not rescan the list.
func (h *hiveFile) childrenOf(nk *regparser.CM_KEY_NODE) *keyChildren {
	h.mu.Lock()
	defer h.mu.Unlock()
	if c, ok := h.children[nk.Offset]; ok {
		return c
	}
	c := &keyChildren{byName: map[string]*regparser.CM_KEY_NODE{}}
	if counts := nk.SubKeyCounts(); len(counts) > 0 && counts[0] > 0 {
		c.list = nk.Subkeys()
		for _, sub := range c.list {
			name := strings.ToLower(keyName(sub))
			if _, dup := c.byName[name]; !dup {
				c.byName[name] = sub
			}
		}
	}
	h.children[nk.Offset] = c
	return c
}

// openKey walks the path from the root. It returns nil when a component is missing.
func (h *hiveFile) openKey(keyPath string) *regparser.CM_KEY_NODE {
	nk := h.root
	for _, part := range strings.Split(keyPath, "\\") {
		if part == "" {
			continue
		}
		sub, ok := h.childrenOf(nk).byName[strings.ToLower(part)]
		if !ok {
			return nil
		}
		nk = sub
	}
	return nk
}

func (h *hiveFile) mustOpenKey(keyPath string) (*regparser.CM_KEY_NODE, error) {
	nk := h.openKey(keyPath)
	if nk == nil {
		return nil, status.Error(codes.NotFound, "registry key not found: "+keyPath)
	}
	return nk, nil
}

// Exists reports whether the key exists, whatever it holds.
func (h *hiveFile) Exists(keyPath string) bool {
	return h.openKey(keyPath) != nil
}

// Children returns the names of the key's subkeys in hive order.
func (h *hiveFile) Children(keyPath string) ([]string, error) {
	nk, err := h.mustOpenKey(keyPath)
	if err != nil {
		return nil, err
	}
	subs := h.childrenOf(nk).list
	names := make([]string, len(subs))
	for i, sub := range subs {
		names[i] = keyName(sub)
	}
	return names, nil
}

// Items returns all values of the key in stored order. The default value has
// the empty name.
func (h *hiveFile) Items(keyPath string) ([]RegistryKeyItem, error) {
	nk, err := h.mustOpenKey(keyPath)
	if err != nil {
		return nil, err
	}
	res := []RegistryKeyItem{}
	if nk.ValueList().Count() == 0 {
		return res, nil
	}
	for _, vk := range nk.Values() {
		res = append(res, RegistryKeyItem{
			Key:   valueName(vk),
			Value: decodeRawRegistryValue(vk.Type(), valueBytes(vk)),
		})
	}
	return res, nil
}

// valueBytes returns the value's data exactly as stored.
func valueBytes(vk *regparser.CM_KEY_VALUE) []byte {
	size := vk.DataSize()
	if size == 0 {
		return nil
	}
	data := vk.ValueData().Data
	// inline data is always read as 4 bytes
	if int64(len(data)) > size {
		data = data[:size]
	}
	return data
}

// Item returns the named value, matching the name case-insensitively. Only the
// matching value is decoded.
func (h *hiveFile) Item(keyPath, name string) (RegistryKeyItem, error) {
	nk, err := h.mustOpenKey(keyPath)
	if err != nil {
		return RegistryKeyItem{}, err
	}
	if nk.ValueList().Count() > 0 {
		for _, vk := range nk.Values() {
			if vn := valueName(vk); strings.EqualFold(vn, name) {
				return RegistryKeyItem{Key: vn, Value: decodeRawRegistryValue(vk.Type(), valueBytes(vk))}, nil
			}
		}
	}
	return RegistryKeyItem{}, status.Error(codes.NotFound, fmt.Sprintf("registry value %s not found under %s", name, keyPath))
}
