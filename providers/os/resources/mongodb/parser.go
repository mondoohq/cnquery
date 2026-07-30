// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package mongodb contains the parser for MongoDB's on-disk configuration
// file, mongod.conf. The parser operates on already-read file content so it
// doesn't depend on a particular filesystem implementation, which lets it be
// unit-tested against inlined fixtures and re-used over different transports
// (local, SSH, container snapshot, ...).
//
// Unlike the option files of MySQL and PostgreSQL, mongod.conf is a nested
// YAML document and has no include mechanism, so a parse reads exactly one
// file and the result is a tree rather than a flat key/value map. Lookups
// therefore address a value by its key path ("net", "tls", "mode").
package mongodb

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Conf is the result of parsing a mongod.conf.
type Conf struct {
	// Params is the configuration tree, normalized so every value is
	// JSON-native (bool, int64, float64, string, []any, map[string]any, or
	// nil). Normalization is mandatory rather than cosmetic: llx rejects a
	// dict holding any other Go type at query time, and a plain YAML decode
	// yields `int` and `time.Time` values that would trip that check.
	Params map[string]any
}

// ParseConf parses the content of a mongod.conf.
//
// An empty file is not an error: mongod accepts one and runs entirely on its
// built-in defaults, so it parses to an empty tree and every accessor reports
// the default it would have used.
func ParseConf(content string) (*Conf, error) {
	var root any
	if err := yaml.Unmarshal([]byte(content), &root); err != nil {
		return nil, err
	}

	if root == nil {
		return &Conf{Params: map[string]any{}}, nil
	}

	params, ok := normalize(root).(map[string]any)
	if !ok {
		// A YAML document whose root is a scalar or a sequence is not a
		// mongod.conf. Reporting it as an empty config would present a
		// malformed file as a server running on defaults.
		return nil, errors.New("mongod.conf must be a YAML mapping at the top level")
	}
	return &Conf{Params: params}, nil
}

// normalize rewrites a decoded YAML value into the JSON-native subset llx
// accepts for a dict. Numbers are the reason this exists: yaml.v3 decodes
// integers as `int`, which llx has no encoding for.
func normalize(v any) any {
	switch x := v.(type) {
	case nil, bool, string, int64, float64:
		return x
	case int:
		return int64(x)
	case int8:
		return int64(x)
	case int16:
		return int64(x)
	case int32:
		return int64(x)
	case uint:
		return int64(x)
	case uint8:
		return int64(x)
	case uint16:
		return int64(x)
	case uint32:
		return int64(x)
	case uint64:
		// Values above the int64 ceiling are not representable. No mongod
		// setting is anywhere near it, so a string keeps the value visible
		// rather than silently wrapping it negative.
		if x > 1<<63-1 {
			return strconv.FormatUint(x, 10)
		}
		return int64(x)
	case float32:
		return float64(x)
	case time.Time:
		// YAML auto-detects unquoted ISO-8601 scalars as timestamps. No
		// mongod setting is a timestamp, so this only fires on a value that
		// happens to look like one, and the original text is what matters.
		return x.Format(time.RFC3339)
	case []any:
		out := make([]any, len(x))
		for i := range x {
			out[i] = normalize(x[i])
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, val := range x {
			out[k] = normalize(val)
		}
		return out
	case map[any]any:
		// yaml.v3 produces map[string]any for string keys, but a document
		// using non-string keys still decodes to this shape.
		out := make(map[string]any, len(x))
		for k, val := range x {
			out[fmt.Sprintf("%v", k)] = normalize(val)
		}
		return out
	default:
		return fmt.Sprintf("%v", x)
	}
}

// ---------------------------------------------------------------------------
// lookups
// ---------------------------------------------------------------------------

// Lookup walks the key path and reports the value at it. The second return
// distinguishes an unset key from a key explicitly set to null, which matters
// because the accessors fall back to mongod's own default only for the former.
func Lookup(params map[string]any, path ...string) (any, bool) {
	if len(path) == 0 {
		return nil, false
	}
	cur := params
	for i, key := range path {
		v, ok := cur[key]
		if !ok {
			return nil, false
		}
		if i == len(path)-1 {
			return v, true
		}
		next, ok := v.(map[string]any)
		if !ok {
			return nil, false
		}
		cur = next
	}
	return nil, false
}

// String reports the value at path as a string, empty when unset. Scalars
// that YAML typed as something other than a string are formatted back, so a
// numeric-looking setting still reads as written.
func String(params map[string]any, path ...string) string {
	v, ok := Lookup(params, path...)
	if !ok || v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case bool:
		return strconv.FormatBool(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	default:
		return ""
	}
}

// Bool reports the value at path, falling back to def when the key is unset.
//
// The caller supplies the default rather than the function assuming false,
// because mongod defaults several security-relevant settings to true
// (security.javascriptEnabled, setParameter.enableLocalhostAuthBypass). A
// hardcoded false fallback would report an unhardened host as hardened.
func Bool(params map[string]any, def bool, path ...string) bool {
	v, ok := Lookup(params, path...)
	if !ok || v == nil {
		return def
	}
	switch x := v.(type) {
	case bool:
		return x
	case string:
		// setParameter values are frequently written as quoted strings.
		b, err := strconv.ParseBool(strings.TrimSpace(x))
		if err != nil {
			return def
		}
		return b
	case int64:
		return x != 0
	default:
		return def
	}
}

// Int reports the value at path, falling back to def when the key is unset or
// holds something that is not a whole number.
func Int(params map[string]any, def int64, path ...string) int64 {
	v, ok := Lookup(params, path...)
	if !ok || v == nil {
		return def
	}
	switch x := v.(type) {
	case int64:
		return x
	case float64:
		return int64(x)
	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(x), 10, 64)
		if err != nil {
			return def
		}
		return n
	default:
		return def
	}
}

// List reports the value at path as a list of strings.
//
// It accepts both spellings mongod does. Settings such as net.bindIp and
// net.tls.disabledProtocols are documented as a single comma-delimited
// string, but a YAML sequence is common in the wild and mongod reads it, so
// treating only the documented form would drop every entry on those hosts.
func List(params map[string]any, path ...string) []string {
	v, ok := Lookup(params, path...)
	if !ok || v == nil {
		return nil
	}
	switch x := v.(type) {
	case []any:
		out := make([]string, 0, len(x))
		for _, item := range x {
			s := strings.TrimSpace(scalarString(item))
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	case string:
		return splitList(x)
	default:
		s := strings.TrimSpace(scalarString(v))
		if s == "" {
			return nil
		}
		return []string{s}
	}
}

func splitList(raw string) []string {
	out := []string{}
	for part := range strings.SplitSeq(raw, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func scalarString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case bool:
		return strconv.FormatBool(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	default:
		return ""
	}
}

// ---------------------------------------------------------------------------
// TLS
// ---------------------------------------------------------------------------

// tlsLegacyKeys maps a net.tls key to its net.ssl spelling for the two
// settings MongoDB renamed rather than moved when it introduced net.tls in
// 4.2. Every other key kept its name under the new parent.
var tlsLegacyKeys = map[string]string{
	"certificateKeyFile":         "PEMKeyFile",
	"certificateKeyFilePassword": "PEMKeyPassword",
}

// tlsPaths returns the key paths to try for a net.tls setting, newest first.
//
// mongod still accepts the pre-4.2 net.ssl tree as an alias, and plenty of
// deployed configs use it, so reading only net.tls would report TLS as
// unconfigured on a host that has it configured the old way.
func tlsPaths(key string) [][]string {
	paths := [][]string{{"net", "tls", key}}
	legacy := key
	if alias, ok := tlsLegacyKeys[key]; ok {
		legacy = alias
	}
	paths = append(paths, []string{"net", "ssl", legacy})
	return paths
}

// TLSString reads a net.tls setting, falling back to its net.ssl spelling.
func TLSString(params map[string]any, key string) string {
	for _, path := range tlsPaths(key) {
		if v := String(params, path...); v != "" {
			return v
		}
	}
	return ""
}

// TLSBool reads a net.tls setting, falling back to its net.ssl spelling.
func TLSBool(params map[string]any, def bool, key string) bool {
	for _, path := range tlsPaths(key) {
		if _, ok := Lookup(params, path...); ok {
			return Bool(params, def, path...)
		}
	}
	return def
}

// TLSList reads a net.tls list setting, falling back to its net.ssl spelling.
func TLSList(params map[string]any, key string) []string {
	for _, path := range tlsPaths(key) {
		if v := List(params, path...); len(v) > 0 {
			return v
		}
	}
	return nil
}

// legacyTLSModes maps each pre-4.2 net.ssl.mode value to the net.tls.mode
// value it is an exact alias of.
var legacyTLSModes = map[string]string{
	"allowSSL":   "allowTLS",
	"preferSSL":  "preferTLS",
	"requireSSL": "requireTLS",
}

// TLSMode reports net.tls.mode, normalized to the modern spelling.
//
// The pre-4.2 net.ssl.mode values are exact aliases of the net.tls.mode ones,
// so they are reported under the modern name. Without that, an audit asserting
// requireTLS would fail on a correctly configured host that spells it
// requireSSL. The default is disabled, which is what mongod uses when neither
// tree sets a mode.
func TLSMode(params map[string]any) string {
	mode := TLSString(params, "mode")
	if mode == "" {
		return "disabled"
	}
	if modern, ok := legacyTLSModes[mode]; ok {
		return modern
	}
	return mode
}

// ---------------------------------------------------------------------------
// version
// ---------------------------------------------------------------------------

// reVersion matches the banner `mongod --version` opens with, for example
// "db version v7.0.14".
var reVersion = regexp.MustCompile(`(?i)db version\s+v?(\d+\.\d+(?:\.\d+)?)`)

// ParseVersion extracts the server version from `mongod --version` output,
// returning the empty string when the output carries no recognizable banner.
func ParseVersion(output string) string {
	m := reVersion.FindStringSubmatch(output)
	if m == nil {
		return ""
	}
	return m[1]
}
