// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package project

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// vaultHeader is the magic prefix every Ansible-vault-encrypted file begins
// with, e.g. `$ANSIBLE_VAULT;1.1;AES256`.
const vaultHeader = "$ANSIBLE_VAULT"

// VaultFile is a detected vault-encrypted file. We never decrypt — without the
// vault password the contents are opaque — so we surface only the file and the
// metadata carried in its header.
type VaultFile struct {
	Path    string
	Format  string // vault format version, e.g. 1.1
	Cipher  string // cipher named in the header, e.g. AES256
	VaultID string // vault-id label from the header, when present (format 1.2+)
}

// VaultVariable is a single variable encrypted inline with a `!vault` tag inside
// an otherwise-plaintext file. Like VaultFile, the value is never decrypted.
type VaultVariable struct {
	Path string // file containing the encrypted variable
	Key  string // the variable's key path within the file
}

// detectVaultFiles walks the project tree for fully vault-encrypted files,
// skipping VCS and hidden directories.
func detectVaultFiles(root string) ([]*VaultFile, error) {
	var files []*VaultFile

	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if p != root && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		header := readHeader(p)
		if !strings.HasPrefix(header, vaultHeader) {
			return nil
		}
		format, cipher, vaultID := parseVaultHeader(header)
		files = append(files, &VaultFile{Path: p, Format: format, Cipher: cipher, VaultID: vaultID})
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

// isVaultEncrypted reports whether a file's bytes are a vault payload. Used to
// skip encrypted files during YAML parsing, which would otherwise fail.
func isVaultEncrypted(data []byte) bool {
	return bytes.HasPrefix(bytes.TrimSpace(data), []byte(vaultHeader))
}

// readHeader reads just the first line of a file, cheaply enough to scan a tree.
func readHeader(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	buf := make([]byte, 128)
	n, _ := f.Read(buf)
	line := string(buf[:n])
	if idx := strings.IndexByte(line, '\n'); idx >= 0 {
		line = line[:idx]
	}
	return strings.TrimSpace(line)
}

// parseVaultHeader extracts the format version, cipher, and optional vault-id
// from a header line of the form `$ANSIBLE_VAULT;<version>;<cipher>[;<vault-id>]`.
func parseVaultHeader(header string) (format, cipher, vaultID string) {
	parts := strings.Split(header, ";")
	if len(parts) > 1 {
		format = strings.TrimSpace(parts[1])
	}
	if len(parts) > 2 {
		cipher = strings.TrimSpace(parts[2])
	}
	if len(parts) > 3 {
		vaultID = strings.TrimSpace(parts[3])
	}
	return format, cipher, vaultID
}

// detectInlineVaultVars scans the project's YAML files for variables encrypted
// inline with a `!vault` tag, recording the file and key path of each. Fully
// encrypted files are skipped — they are reported by detectVaultFiles instead.
func detectInlineVaultVars(root string) ([]*VaultVariable, error) {
	var vars []*VaultVariable

	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if p != root && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !isYAMLPath(p) {
			return nil
		}
		data, readErr := os.ReadFile(p)
		if readErr != nil || isVaultEncrypted(data) {
			return nil
		}
		var node yaml.Node
		if yaml.Unmarshal(data, &node) != nil {
			return nil
		}
		for _, key := range findVaultKeys(&node, "") {
			vars = append(vars, &VaultVariable{Path: p, Key: key})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(vars, func(i, j int) bool {
		if vars[i].Path != vars[j].Path {
			return vars[i].Path < vars[j].Path
		}
		return vars[i].Key < vars[j].Key
	})
	return vars, nil
}

// findVaultKeys walks a YAML node tree and returns the key paths of every scalar
// carrying a `!vault` tag.
func findVaultKeys(node *yaml.Node, prefix string) []string {
	var keys []string
	switch node.Kind {
	case yaml.DocumentNode:
		for _, child := range node.Content {
			keys = append(keys, findVaultKeys(child, prefix)...)
		}
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			keyNode, valNode := node.Content[i], node.Content[i+1]
			path := keyNode.Value
			if prefix != "" {
				path = prefix + "." + keyNode.Value
			}
			if valNode.Tag == "!vault" {
				keys = append(keys, path)
			}
			keys = append(keys, findVaultKeys(valNode, path)...)
		}
	case yaml.SequenceNode:
		for idx, child := range node.Content {
			keys = append(keys, findVaultKeys(child, prefix+"["+strconv.Itoa(idx)+"]")...)
		}
	}
	return keys
}

func isYAMLPath(p string) bool {
	ext := filepath.Ext(p)
	return ext == ".yml" || ext == ".yaml"
}
