// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"gopkg.in/yaml.v3"
	kustomizeTypes "sigs.k8s.io/kustomize/api/types"
)

const (
	patchFormatStrategicMerge = "strategicMerge"
	patchFormatJSON6902       = "json6902"
)

// formatHint forces a patch's classification when the source field is
// unambiguous (the legacy patchesStrategicMerge / patchesJson6902 lists).
// An empty hint means "inspect the content shape".
type formatHint string

const (
	hintNone           formatHint = ""
	hintStrategicMerge formatHint = patchFormatStrategicMerge
	hintJSON6902       formatHint = patchFormatJSON6902
)

// jsonPatchOp is one decomposed RFC6902 operation.
type jsonPatchOp struct {
	op    string
	path  string
	value any
	// hasValue distinguishes an explicit null value (add/replace/test) from
	// an operation that carries no value at all (remove/move/copy).
	hasValue bool
}

type mqlKustomizePatchInternal struct {
	format string
	ops    []jsonPatchOp
}

func newMqlKustomizePatch(runtime *plugin.Runtime, kustPath string, index int, p *kustomizeTypes.Patch, hint formatHint) (*mqlKustomizePatch, error) {
	targetGroup := ""
	targetVersion := ""
	targetKind := ""
	targetName := ""
	targetNamespace := ""
	targetLabelSelector := ""
	targetAnnotationSelector := ""

	if p.Target != nil {
		targetGroup = p.Target.Group
		targetVersion = p.Target.Version
		targetKind = p.Target.Kind
		targetName = p.Target.Name
		targetNamespace = p.Target.Namespace
		targetLabelSelector = p.Target.LabelSelector
		targetAnnotationSelector = p.Target.AnnotationSelector
	}

	// Read the raw patch bytes: inline content wins, otherwise the file the
	// patch points at (relative to the kustomization directory). `content`
	// tracks the same bytes so a file-based patch surfaces its body through
	// the `content` field rather than an empty string.
	raw := []byte(p.Patch)
	content := p.Patch
	if len(raw) == 0 && p.Path != "" {
		// Best-effort read; a missing/unreadable file falls back to
		// strategic-merge with no operations rather than failing the audit.
		// Constrain the read to the kustomization directory so a malicious
		// patch path (e.g. "../../etc/passwd") — or a symlink inside the
		// directory whose target is outside it — can't escape the scan root.
		// Both the base and the candidate are symlink-resolved before the
		// containment check so a symlinked scan root (e.g. /tmp on macOS,
		// which resolves to /private/tmp) doesn't cause false rejections.
		full := filepath.Join(kustPath, p.Path)
		// Prefer symlink-resolved paths for the containment check (this catches
		// a symlink inside the directory whose target escapes it). When symlink
		// resolution fails for any reason other than the target being a
		// resolvable escape — e.g. a broken/unresolvable symlink component in
		// kustPath itself — fall back to a lexical containment check so a
		// perfectly readable patch file isn't silently dropped and misclassified
		// as an empty strategic-merge patch.
		base, target := filepath.Clean(kustPath), filepath.Clean(full)
		if rb, err1 := filepath.EvalSymlinks(kustPath); err1 == nil {
			if rt, err2 := filepath.EvalSymlinks(full); err2 == nil {
				base, target = rb, rt
			}
		}
		if rel, err := filepath.Rel(base, target); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			data, readErr := os.ReadFile(target)
			switch {
			case readErr == nil:
				raw = data
				content = string(data)
			case !os.IsNotExist(readErr):
				// A genuinely missing file is the expected best-effort case; any
				// other read failure would silently misclassify the patch, so
				// surface it.
				log.Warn().Err(readErr).Str("path", p.Path).Msg("kustomize: could not read patch file; treating it as an empty patch")
			}
		} else {
			log.Warn().Str("path", p.Path).Msg("kustomize: patch path escapes the kustomization directory; ignoring")
		}
	}

	format, ops := classifyPatch(raw, hint)

	id := "kustomize.patch:" + kustPath + ":" + strconv.Itoa(index)

	res, err := CreateResource(runtime, "kustomize.patch", map[string]*llx.RawData{
		"__id":                     llx.StringData(id),
		"content":                  llx.StringData(content),
		"path":                     llx.StringData(p.Path),
		"format":                   llx.StringData(format),
		"targetGroup":              llx.StringData(targetGroup),
		"targetVersion":            llx.StringData(targetVersion),
		"targetKind":               llx.StringData(targetKind),
		"targetName":               llx.StringData(targetName),
		"targetNamespace":          llx.StringData(targetNamespace),
		"targetLabelSelector":      llx.StringData(targetLabelSelector),
		"targetAnnotationSelector": llx.StringData(targetAnnotationSelector),
	})
	if err != nil {
		return nil, err
	}
	mqlP := res.(*mqlKustomizePatch)
	mqlP.format = format
	mqlP.ops = ops
	return mqlP, nil
}

// classifyPatch inspects raw patch bytes (YAML or JSON) and returns the
// patch format plus, for JSON6902 patches, the decomposed operations. It
// never panics on malformed input: anything it can't decode as a JSON6902
// operation sequence is treated as a strategic-merge patch with no
// operations. A non-empty hint forces the format.
func classifyPatch(raw []byte, hint formatHint) (string, []jsonPatchOp) {
	// A forced strategic-merge patch never carries operations.
	if hint == hintStrategicMerge {
		return patchFormatStrategicMerge, nil
	}

	ops, ok := decodeJSON6902(raw)
	switch {
	case hint == hintJSON6902:
		// Forced JSON6902: decode what we can, even an empty list.
		return patchFormatJSON6902, ops
	case ok:
		return patchFormatJSON6902, ops
	default:
		return patchFormatStrategicMerge, nil
	}
}

// decodeJSON6902 attempts to decode raw bytes as a sequence of RFC6902
// operations. It returns ok=true only when the content is a YAML/JSON
// sequence whose elements are all mappings carrying an `op` key — the
// shape that unambiguously identifies a JSON6902 patch.
func decodeJSON6902(raw []byte) ([]jsonPatchOp, bool) {
	if len(raw) == 0 {
		return nil, false
	}

	var seq []map[string]any
	if err := yaml.Unmarshal(raw, &seq); err != nil {
		return nil, false
	}
	if len(seq) == 0 {
		return nil, false
	}

	ops := make([]jsonPatchOp, 0, len(seq))
	for _, elem := range seq {
		if elem == nil {
			return nil, false
		}
		opVal, hasOp := elem["op"]
		if !hasOp {
			return nil, false
		}
		opStr, _ := opVal.(string)

		pathVal, _ := elem["path"].(string)
		value, hasValue := elem["value"]
		if hasValue {
			// yaml.v3 decodes integer scalars to Go `int`, which the llx dict
			// serializer rejects (it accepts only int64/float64 among numbers).
			// Normalize every value to JSON-native types so a common patch such
			// as `value: 3` doesn't error when the `value` field is queried.
			value = toJSONNative(value)
		}

		ops = append(ops, jsonPatchOp{
			op:       opStr,
			path:     pathVal,
			value:    value,
			hasValue: hasValue,
		})
	}
	return ops, true
}

// toJSONNative round-trips a yaml.v3-decoded value through encoding/json so
// every number, map, and slice is expressed with the JSON-native Go types the
// llx dict serializer accepts (float64/string/bool/[]any/map[string]any/nil).
// yaml.v3 hands back Go `int` for integer scalars, which dict serialization
// rejects; the round-trip converts those to float64. On the (practically
// impossible for yaml-derived data) marshal error, the original value is
// returned unchanged so behavior is never worse than before.
func toJSONNative(v any) any {
	if v == nil {
		return nil
	}
	data, err := json.Marshal(v)
	if err != nil {
		return v
	}
	var out any
	if err := json.Unmarshal(data, &out); err != nil {
		return v
	}
	return out
}

func (c *mqlKustomizePatch) operations() ([]any, error) {
	if c.format != patchFormatJSON6902 || len(c.ops) == 0 {
		return []any{}, nil
	}

	mqlOps := make([]any, 0, len(c.ops))
	for i, op := range c.ops {
		args := map[string]*llx.RawData{
			"__id": llx.StringData(c.__id + "/op[" + strconv.Itoa(i) + "]"),
			"op":   llx.StringData(op.op),
			"path": llx.StringData(op.path),
		}
		if op.hasValue {
			args["value"] = llx.DictData(op.value)
		} else {
			args["value"] = llx.NilData
		}

		res, err := CreateResource(c.MqlRuntime, "kustomize.patch.operation", args)
		if err != nil {
			return nil, err
		}
		mqlOps = append(mqlOps, res)
	}
	return mqlOps, nil
}

var (
	_ plugin.Resource = (*mqlKustomizePatch)(nil)
	_ plugin.Resource = (*mqlKustomizePatchOperation)(nil)
)
