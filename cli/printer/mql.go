// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package printer

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/mqlc"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/types"
	"go.mondoo.com/cnquery/v12/utils/multierr"
	"go.mondoo.com/cnquery/v12/utils/sortx"
	"golang.org/x/exp/slices"
)

const MsgUnsupported = "unsupported platform"

type printCache struct {
	bundle         *llx.CodeBundle
	checksumLookup map[string]uint64
	isInline       bool
}

func newPrintCache(bundle *llx.CodeBundle) *printCache {
	res := &printCache{
		bundle: bundle,
	}

	code := bundle.CodeV2
	lookup := make(map[string]uint64, len(code.Checksums))
	for k, v := range code.Checksums {
		lookup[v] = k
	}
	res.checksumLookup = lookup

	return res
}

func (print *Printer) Datas(bundle *llx.CodeBundle, results map[string]*llx.RawResult) string {
	var res strings.Builder
	i := 0
	for _, v := range results {
		res.WriteString(print.Result(v, bundle))
		if len(results) > 1 && len(results) != i+1 {
			res.WriteString("\n")
		}
		i++
	}
	return res.String()
}

// Results prints a full query with all data points
// NOTE: ensure that results only contains results that match the bundle!
func (print *Printer) Results(bundle *llx.CodeBundle, results map[string]*llx.RawResult) string {
	assessment := llx.Results2Assessment(bundle, results)

	if assessment != nil {
		return print.Assessment(bundle, assessment)
	}

	return print.Datas(bundle, results)
}

// Assessment prints a complete comparable assessment
func (print *Printer) Assessment(bundle *llx.CodeBundle, assessment *llx.Assessment) string {
	var res strings.Builder

	var indent string
	if len(assessment.Results) > 1 {
		if assessment.Success {
			res.WriteString(print.Secondary("[ok] "))
		} else {
			res.WriteString(print.Failed("[failed] "))
		}
		res.WriteString(bundle.Source)
		res.WriteString("\n")
		// once we displayed the overall status, lets indent the subs
		indent = "  "
	}

	cache := newPrintCache(bundle)

	// indent all sub-results
	for i := range assessment.Results {
		cur := print.assessment(assessment.Results[i], indent, cache)
		res.WriteString(indent)
		res.WriteString(cur)
		res.WriteString("\n")
	}

	return res.String()
}

func isBooleanOp(op string) bool {
	return op == "&&" || op == "||"
}

func (print *Printer) assessment(assessment *llx.AssessmentItem, indent string, cache *printCache) string {
	bundle := cache.bundle
	nextIndent := indent + "  "
	checksum := assessment.Checksum

	if assessment.Error != "" {
		var res strings.Builder
		res.WriteString(print.Failed("[failed] "))
		if bundle != nil && bundle.Labels != nil {
			label := bundle.Labels.Labels[checksum]
			res.WriteString(label)
		}
		res.WriteString("\n")
		res.WriteString(nextIndent)
		res.WriteString("error: ")
		res.WriteString(assessment.Error)
		return res.String()
	}

	if assessment.Template != "" {
		if assessment.Success {
			return print.Secondary("[ok] true")
		}

		data := make([]string, len(assessment.Data))
		for i := range assessment.Data {
			data[i] = print.primitive(assessment.Data[i], "", indent, cache)
		}
		res := print.Failed("[failed] ") +
			print.assessmentTemplate(assessment.Template, data)
		return indentBlock(res, indent)
	}

	if isBooleanOp(assessment.Operation) {
		if assessment.Success {
			return print.Secondary("[ok] true")
		}

		var res strings.Builder
		res.WriteString(print.Secondary("[failed] "))
		res.WriteString(print.primitive(assessment.Actual, "", nextIndent, cache))
		res.WriteString(" " + assessment.Operation + " ")
		res.WriteString(print.primitive(assessment.Expected, "", nextIndent, cache))
		return res.String()
	}

	if assessment.Success {
		if assessment.Actual == nil {
			return print.Secondary("[ok] ") + " (missing result)"
		}

		return print.Secondary("[ok]") +
			" value: " +
			print.primitive(assessment.Actual, "", indent, cache)
	}

	var res strings.Builder
	res.WriteString(print.Failed("[failed] "))
	if bundle != nil && bundle.Labels != nil {
		label, ok := bundle.Labels.Labels[checksum]
		if ok {
			res.WriteString(label)
		}
	}
	res.WriteString("\n")

	if assessment.Expected != nil {
		res.WriteString(nextIndent)
		res.WriteString("expected: " + assessment.Operation + " ")
		res.WriteString(print.primitive(assessment.Expected, "", nextIndent, cache))
		res.WriteString("\n")
	}

	if assessment.Actual != nil {
		res.WriteString(nextIndent)
		res.WriteString("actual:   ")
		res.WriteString(print.primitive(assessment.Actual, checksum, nextIndent, cache))
	}

	return res.String()
}

// Result prints the llx raw result into a string
func (print *Printer) Result(result *llx.RawResult, bundle *llx.CodeBundle) string {
	if result == nil {
		return "< no result? >"
	}

	return print.DataWithLabel(result.Data, result.CodeID, bundle, "")
}

func (print *Printer) label(ref string, bundle *llx.CodeBundle) string {
	if bundle == nil {
		return ""
	}

	labels := bundle.Labels
	if labels == nil {
		return fmt.Sprintf("["+print.Primary("%s")+"] ", ref)
	}

	label := labels.Labels[ref]
	if label == "" {
		return ""
	}

	return print.Primary(label) + ": "
}

func (print *Printer) defaultLabel(ref string, bundle *llx.CodeBundle) string {
	if bundle == nil {
		return ""
	}

	labels := bundle.Labels
	if labels == nil {
		return fmt.Sprintf("["+print.Primary("%s")+"] ", ref)
	}

	label := labels.Labels[ref]
	if label == "" {
		return ""
	}

	return print.Primary(label) + "="
}

func isDefaultField(ref string, bundle *llx.CodeBundle, defaultFields []string) bool {
	if bundle == nil {
		return false
	}

	labels := bundle.Labels
	if labels == nil {
		return false
	}

	label := labels.Labels[ref]
	if label == "" {
		return false
	}

	return slices.Contains(defaultFields, label)
}

func (print *Printer) array(typ types.Type, data []any, checksum string, indent string, cache *printCache) string {
	if len(data) == 0 {
		return "[]"
	}

	var res strings.Builder
	if cache.isInline {
		res.WriteString("[")
	} else {
		res.WriteString("[\n")
	}

	ep := cache.checksumLookup[checksum]
	listType := ""
	if ep != 0 {
		code := cache.bundle.CodeV2
		chunk := code.Chunk(ep)
		switch chunk.Id {
		case "$one", "$all", "$none", "$any":
			ref := chunk.Function.Binding
			listChunk := code.Chunk(ref)
			listType = types.Type(listChunk.Type()).Child().Label()
			listType += " "
		}
	}

	if cache.isInline {
		for i := range data {
			res.WriteString(print.data(typ.Child(), data[i], checksum, indent, cache))
			if len(data) != i+1 {
				res.WriteString(", ")
			}
		}
	} else {
		for i := range data {
			res.WriteString(fmt.Sprintf(
				indent+"  %d: %s%s\n",
				i,
				listType,
				print.data(typ.Child(), data[i], checksum, indent+"  ", cache),
			))
		}
	}

	res.WriteString(indent + "]")
	return res.String()
}

func (print *Printer) assessmentTemplate(template string, data []string) string {
	res := template
	for i := range data {
		r := "$" + strconv.Itoa(i)
		res = strings.ReplaceAll(res, r, data[i])
	}
	return res
}

func (print *Printer) dataAssessment(codeID string, data map[string]any, bundle *llx.CodeBundle) string {
	var assertion *llx.AssertionMessage
	var ok bool

	assertion, ok = bundle.Assertions[codeID]
	if !ok {
		return ""
	}

	checksums := assertion.Checksums
	if assertion.DecodeBlock {
		if len(assertion.Checksums) < 2 {
			return ""
		}

		v, ok := data[assertion.Checksums[0]]
		if !ok {
			return ""
		}
		raw, ok := v.(*llx.RawData)
		if !ok {
			return ""
		}
		data, ok = raw.Value.(map[string]any)
		if !ok {
			return ""
		}
		checksums = checksums[1:]
	}

	fields := make([]string, len(checksums))
	for i := range checksums {
		v, ok := data[checksums[i]]
		if !ok {
			return ""
		}

		val := v.(*llx.RawData)
		fields[i] = print.Data(val.Type, val.Value, "", bundle, "")
	}

	return print.assessmentTemplate(assertion.Template, fields)
}

func (print *Printer) refMap(data map[string]any, checksum string, indent string, cache *printCache) string {
	if len(data) == 0 {
		return "{}"
	}

	var res strings.Builder

	// we need to separate entries that are unlabelled (eg part of an assertion)
	labeledKeys := []string{}
	keys := sortx.Keys(data)
	for i := range keys {
		if _, ok := cache.bundle.Labels.Labels[keys[i]]; ok {
			labeledKeys = append(labeledKeys, keys[i])
		}
	}

	ep := cache.checksumLookup[checksum]
	listType := ""
	if ep != 0 {
		code := cache.bundle.CodeV2
		chunk := code.Chunk(ep)
		switch chunk.Id {
		case "$one", "$all", "$none", "$any":
			ref := chunk.Function.Binding
			listChunk := code.Chunk(ref)
			listType = types.Type(listChunk.Type()).Child().Label()
		}
	}

	nonDefaultFields := []string{}
	defaultFields := []string{}
	if listType != "" && print.schema != nil {
		resourceInfo := (print.schema).Lookup(listType)
		if resourceInfo != nil {
			defaultFields = strings.Split(resourceInfo.Defaults, " ")
		}
	}

	inlineCache := *cache
	inlineCache.isInline = true
	for _, k := range labeledKeys {
		if k == "_" {
			continue
		}

		if !isDefaultField(k, cache.bundle, defaultFields) {
			// save for later output after we wrote all the default fields
			nonDefaultFields = append(nonDefaultFields, k)
			continue
		}

		v := data[k]
		label := print.defaultLabel(k, cache.bundle)
		val := v.(*llx.RawData)

		if plugin.IsUnsupportedProviderError(val.Error) {
			res.WriteString("  " + label + print.Disabled(MsgUnsupported) + "\n")
			continue
		}
		if val.Error != nil {
			res.WriteString("  " + label + print.Error(val.Error.Error()) + " ")
			continue
		}

		data := print.data(val.Type, val.Value, k, "", &inlineCache)
		res.WriteString(label + data + " ")
	}

	if len(nonDefaultFields) > 0 {
		res.WriteString("{\n")
		for _, k := range nonDefaultFields {
			if k == "_" {
				continue
			}

			v := data[k]
			label := print.label(k, cache.bundle)
			val := v.(*llx.RawData)

			if plugin.IsUnsupportedProviderError(val.Error) {
				res.WriteString(indent + "  " + label + print.Disabled(MsgUnsupported) + "\n")
				continue
			}
			if val.Error != nil {
				res.WriteString(indent + "  " + label + print.Error(val.Error.Error()) + "\n")
				continue
			}

			if truthy, _ := val.IsTruthy(); !truthy {
				assertion := print.dataAssessment(k, data, cache.bundle)
				if assertion != "" {
					assertion = print.Failed("[failed]") + " " + strings.Trim(assertion, "\n\t ")
					assertion = indentBlock(assertion, indent+"  ")
					res.WriteString(indent + "  " + assertion + "\n")
					continue
				}
			}

			data := print.data(val.Type, val.Value, k, indent+"  ", cache)
			res.WriteString(indent + "  " + label + data + "\n")
		}

		res.WriteString(indent + "}")
	}
	return res.String()
}

func (print *Printer) stringMap(typ types.Type, data map[string]any, indent string, cache *printCache) string {
	if len(data) == 0 {
		return "{}"
	}

	var res strings.Builder
	res.WriteString("{\n")

	keys := sortx.Keys(data)
	for _, k := range keys {
		v := data[k]
		val := print.data(typ.Child(), v, k, indent+"  ", cache)
		res.WriteString(fmt.Sprintf(indent+"  %s: %s\n", k, val))
	}

	res.WriteString(indent + "}")
	return res.String()
}

func (print *Printer) dict(typ types.Type, raw any, indent string, cache *printCache) string {
	if raw == nil {
		return print.Secondary("null")
	}

	switch data := raw.(type) {
	case bool:
		if data {
			return print.Secondary("true")
		}
		return print.Secondary("false")

	case int64:
		return print.Secondary(fmt.Sprintf("%d", data))

	case float64:
		return print.Secondary(fmt.Sprintf("%f", data))

	case string:
		return print.Secondary(llx.PrettyPrintString(data))

	case time.Time:
		return print.Secondary(data.String())

	case []any:
		if len(data) == 0 {
			return "[]"
		}

		var res strings.Builder
		res.WriteString("[\n")

		for i := range data {
			res.WriteString(fmt.Sprintf(
				indent+"  %d: %s\n",
				i,
				print.dict(typ, data[i], indent+"  ", cache),
			))
		}

		res.WriteString(indent + "]")
		return res.String()

	case map[string]any:
		if len(data) == 0 {
			return "{}"
		}

		var res strings.Builder
		res.WriteString("{\n")

		keys := sortx.Keys(data)
		for _, k := range keys {
			s := print.dict(typ, data[k], indent+"  ", cache)
			res.WriteString(fmt.Sprintf(indent+"  %s: %s\n", k, s))
		}

		res.WriteString(indent + "}")
		return res.String()

	default:
		return print.Secondary(fmt.Sprintf("%+v", raw))
	}
}

func (print *Printer) intMap(typ types.Type, data map[int]any, indent string, cache *printCache) string {
	var res strings.Builder
	res.WriteString("{\n")

	for i := range data {
		value := data[i]
		res.WriteString(fmt.Sprintf(indent+"  %d: "+print.Secondary("%#v")+"\n", i, value))
	}

	res.WriteString(indent + "}")
	return res.String()
}

func (print *Printer) resourceContext(data any, checksum string, indent string, cache *printCache) (string, bool) {
	m, ok := data.(map[string]any)
	if !ok {
		return "", false
	}

	var path string
	var rnge llx.Range
	var content string

	for k, v := range m {
		label, ok := cache.bundle.Labels.Labels[k]
		if !ok {
			continue
		}
		vv, ok := v.(*llx.RawData)
		if !ok {
			continue
		}

		switch label {
		case "content":
			if vv.Type == types.String {
				content, _ = vv.Value.(string)
			}
		case "range":
			if vv.Type == types.Range {
				rnge, _ = vv.Value.(llx.Range)
			}
		case "path", "file.path":
			if vv.Type == types.String {
				path, _ = vv.Value.(string)
			}
		}
	}

	var res strings.Builder
	if path == "" {
		if !rnge.IsEmpty() {
			res.WriteString("<unknown>:")
			res.WriteString(rnge.String())
		}
	} else {
		res.WriteString(path)
		if !rnge.IsEmpty() {
			res.WriteByte(':')
			res.WriteString(rnge.String())
		}
	}
	if content != "" {
		res.WriteByte('\n')
		res.WriteString(indent)
		res.WriteString(indentBlock(content, indent))
	}

	r := res.String()
	if r == "" {
		return "", false
	}
	return r, true
}

func (print *Printer) autoExpand(blockRef uint64, data any, indent string, cache *printCache) string {
	var res strings.Builder

	if arr, ok := data.([]any); ok {
		if len(arr) == 0 {
			return "[]"
		}

		prefix := indent + "  "
		res.WriteString("[\n")
		for i := range arr {
			num := strconv.Itoa(i)
			autoIndent := prefix + strings.Repeat(" ", len(num)+2)
			c := print.autoExpand(blockRef, arr[i], autoIndent, cache)
			res.WriteString(prefix)
			res.WriteString(num)
			res.WriteString(": ")
			res.WriteString(c)
			res.WriteByte('\n')
		}
		res.WriteString(indent + "]")
		return res.String()
	}

	if data == nil {
		return print.Secondary("null")
	}

	m, ok := data.(map[string]any)
	if !ok {
		return "data is not a map to auto-expand"
	}

	block := cache.bundle.CodeV2.Block(blockRef)
	var name string

	self, ok := m["_"].(*llx.RawData)
	if ok {
		name = self.Type.ResourceName()
	} else if block != nil {
		// We end up here when we deal with array resources. In that case,
		// the block's first element is typed as an individual resource, so we can
		// use that.
		typ := block.Chunks[0].Type()
		name = typ.ResourceName()
	}
	if name == "" {
		name = "<unknown>"
	}

	res.WriteString(name)

	// hasContext := false
	// resourceInfo := print.schema.Lookup(name)
	// if resourceInfo != nil && resourceInfo.Context != "" {
	// 	hasContext = true
	// }

	if block != nil {
		// important to process them in this order
		for _, ref := range block.Entrypoints {
			checksum := cache.bundle.CodeV2.Checksums[ref]
			v, ok := m[checksum]
			if !ok {
				continue
			}
			vv, ok := v.(*llx.RawData)
			if !ok {
				continue
			}

			label := cache.bundle.Labels.Labels[checksum]

			// Note (Dom): We don't have precise matching on the resource context
			// just yet. Here we handle it via best-effort, i.e. if it's called
			// context, a block, and it's part of the resource's auto-expand, then we
			// treat it as a kind of resource context.
			if label == "context" && types.Type(vv.Type) == types.Block {
				if data, ok := print.resourceContext(vv.Value, checksum, indent, cache); ok {
					res.WriteByte('\n')
					res.WriteString(indent)
					res.WriteString("in ")
					res.WriteString(data)
					continue
				}
			}

			val := print.data(vv.Type, vv.Value, checksum, indent, cache)
			res.WriteByte(' ')
			res.WriteString(label)
			res.WriteByte('=')
			res.WriteString(val)
		}
	}

	return res.String()
}

func (print *Printer) Data(typ types.Type, data any, checksum string, bundle *llx.CodeBundle, indent string) string {
	return print.data(typ, data, checksum, indent, newPrintCache(bundle))
}

func (print *Printer) data(typ types.Type, data any, checksum string, indent string, cache *printCache) string {
	if typ.NotSet() {
		return "no data available"
	}

	if blockRef, ok := cache.bundle.AutoExpand[checksum]; ok {
		return print.autoExpand(blockRef, data, indent, cache)
	}

	switch typ.Underlying() {
	case types.Any:
		return indent + fmt.Sprintf("any: %#v", data)
	case types.Ref:
		if d, ok := data.(uint64); ok {
			return "ref" + print.Primary(fmt.Sprintf("<%d,%d>", d>>32, d&0xFFFFFFFF))
		} else {
			return "ref" + print.Primary(fmt.Sprintf("%d", data.(int32)))
		}
	case types.Nil:
		return print.Secondary("null")
	case types.Bool:
		if data == nil {
			return print.Secondary("null")
		}
		if data.(bool) {
			return print.Secondary("true")
		}
		return print.Secondary("false")
	case types.Int:
		if data == nil {
			return print.Secondary("null")
		}
		if data.(int64) == math.MaxInt64 {
			return print.Secondary("Infinity")
		}
		if data.(int64) == math.MinInt64 {
			return print.Secondary("-Infinity")
		}
		return print.Secondary(strconv.FormatInt(data.(int64), 10))
	case types.Float:
		if data == nil {
			return print.Secondary("null")
		}
		if math.IsInf(data.(float64), 1) {
			return print.Secondary("Infinity")
		}
		if math.IsInf(data.(float64), -1) {
			return print.Secondary("-Infinity")
		}
		return print.Secondary(strconv.FormatFloat(data.(float64), 'g', -1, 64))
	case types.String:
		if data == nil {
			return print.Secondary("null")
		}
		return print.Secondary(llx.PrettyPrintString(data.(string)))
	case types.Regex:
		if data == nil {
			return print.Secondary("null")
		}
		return print.Secondary(fmt.Sprintf("/%s/", data))
	case types.Time:
		if data == nil {
			return print.Secondary("null")
		}
		time := data.(*time.Time)
		if time == nil {
			return print.Secondary("null")
		}

		if *time == llx.NeverPastTime || *time == llx.NeverFutureTime {
			return print.Secondary("Never")
		}

		if time.Unix() > 0 {
			return print.Secondary(time.String())
		}

		durationStr := llx.TimeToDurationString(*time)

		return print.Secondary(durationStr)
	case types.Dict:
		return print.dict(typ, data, indent, cache)

	case types.Score:
		return llx.ScoreString(data.([]byte))

	case types.Block:
		return print.refMap(data.(map[string]any), checksum, indent, cache)

	case types.Version:
		return print.Secondary(data.(string))

	case types.IP:
		if data == nil {
			return print.Secondary("null")
		}
		return print.Secondary(data.(llx.RawIP).String())

	case types.ArrayLike:
		if data == nil {
			return print.Secondary("null")
		}
		return print.array(typ, data.([]any), checksum, indent, cache)

	case types.MapLike:
		if data == nil {
			return print.Secondary("null")
		}
		if typ.Key() == types.String {
			return print.stringMap(typ, data.(map[string]any), indent, cache)
		}
		if typ.Key() == types.Int {
			return print.intMap(typ, data.(map[int]any), indent, cache)
		}
		return "unable to render map, its type is not supported: " + typ.Label() + ", raw: " + fmt.Sprintf("%#v", data)

	case types.ResourceLike:
		if data == nil {
			return print.Secondary("null")
		}

		r := data.(llx.Resource)
		idline := r.MqlName()
		if id := r.MqlID(); id != "" {
			idline += " id = " + id
		}

		return idline

	case types.FunctionLike:
		if d, ok := data.(uint64); ok {
			return indent + fmt.Sprintf("=> <%d,0>", d>>32)
		} else {
			return indent + fmt.Sprintf("=> %#v", data)
		}

	case types.Range:
		if d, ok := data.(llx.Range); ok {
			return indent + d.String()
		} else {
			return indent + "<bad range>"
		}

	default:
		return indent + fmt.Sprintf("🤷 %#v", data)
	}
}

// For the printer we want to filter out UnsupportedProvider errors,
// since they are just reporting ignored bits.
func filterErrors(e error) error {
	if e == nil {
		return nil
	}
	if plugin.IsUnsupportedProviderError(e) {
		return nil
	}
	merr, ok := e.(*multierr.Errors)
	if !ok {
		return e
	}

	filtered := merr.Filter(func(e error) bool {
		return plugin.IsUnsupportedProviderError(e)
	})
	if len(filtered.Errors) == 0 {
		return nil
	}
	return filtered
}

// DataWithLabel prints RawData into a string
func (print *Printer) DataWithLabel(r *llx.RawData, checksum string, bundle *llx.CodeBundle, indent string) string {
	b := strings.Builder{}
	errs := filterErrors(r.Error)
	if errs != nil {
		b.WriteString(print.Error(strings.TrimSpace(errs.Error())))
		b.WriteByte('\n')
	}

	b.WriteString(print.label(checksum, bundle))
	b.WriteString(print.Data(r.Type, r.Value, checksum, bundle, indent))
	return b.String()
}

func (print *Printer) CompilerStats(stats mqlc.CompilerStats) string {
	var res strings.Builder
	res.WriteString("Resources and Fields used:\n")

	stats.WalkSorted(func(resource string, field string, info mqlc.FieldStat) {
		if field == "" {
			res.WriteString("- " + print.Yellow(resource) + "\n")
			return
		}

		autoexpand := ""
		if info.AutoExpand {
			autoexpand = print.Disabled(" [auto-expand]")
		}

		res.WriteString("  - " + print.Primary(field) + "  type=" + info.Type.Label() + autoexpand + "\n")
	})

	return res.String()
}

// CodeBundle prints the contents of the MQL query
func (print *Printer) CodeBundle(bundle *llx.CodeBundle) string {
	var res strings.Builder

	res.WriteString(print.CodeV2(bundle.CodeV2, bundle, ""))

	for idx := range bundle.Suggestions {
		info := bundle.Suggestions[idx]
		res.WriteString(print.Yellow("- suggestion: "+info.Field) + "\n")
	}

	return res.String()
}

func (print *Printer) CodeV2(code *llx.CodeV2, bundle *llx.CodeBundle, indent string) string {
	var res strings.Builder
	indent += "   "
	cache := newPrintCache(bundle)

	for i := range code.Blocks {
		block := code.Blocks[i]

		res.WriteString(print.Secondary(fmt.Sprintf("-> block %d\n", i+1)))

		res.WriteString(indent)
		res.WriteString("entrypoints: [")
		for idx, ep := range block.Entrypoints {
			res.WriteString(fmt.Sprintf("<%d,%d>", ep>>32, ep&0xFFFFFFFF))
			if idx != len(block.Entrypoints)-1 {
				res.WriteString(" ")
			}
		}
		if len(block.Datapoints) != 0 {
			res.WriteString("] datapoints: [")
			for idx, ep := range block.Datapoints {
				res.WriteString(fmt.Sprintf("<%d,%d>", ep>>32, ep&0xFFFFFFFF))
				if idx != len(block.Datapoints)-1 {
					res.WriteString(" ")
				}
			}
		}
		res.WriteString("]\n")

		for j := range block.Chunks {
			res.WriteString("   ")
			print.chunkV2(j, block.Chunks[j], block, indent, cache, &res)
		}
	}

	return res.String()
}

func (print *Printer) chunkV2(idx int, chunk *llx.Chunk, block *llx.Block, indent string, cache *printCache, w *strings.Builder) {
	sidx := strconv.Itoa(idx+1) + ": "

	switch chunk.Call {
	case llx.Chunk_FUNCTION:
		w.WriteString(print.Primary(sidx))
	case llx.Chunk_PRIMITIVE:
		w.WriteString(print.Secondary(sidx))
	default:
		w.WriteString(print.Error(sidx))
	}

	if chunk.Id != "" {
		w.WriteString(chunk.Id + " ")
	}

	if chunk.Primitive != nil {
		primitive := chunk.Primitive

		// special case: function arguments are supplied by the caller, so we fill
		// in the context for all resource references since they cannot have default values
		if idx < int(block.Parameters) && len(primitive.Value) == 0 && types.Type(primitive.Type).IsResource() {
			primitive = &llx.Primitive{Type: primitive.Type, Value: []byte("context")}
		}

		// FIXME: this is definitely the wrong ID
		w.WriteString(print.primitive(primitive, "", indent, cache))
	}

	if chunk.Function != nil {
		w.WriteString(print.functionV2(chunk.Function, "", indent, cache))
	}

	w.WriteString("\n")
}

func (print *Printer) functionV2(f *llx.Function, checksum string, indent string, cache *printCache) string {
	argsStr := ""
	if len(f.Args) > 0 {
		argsStr = " (" + strings.TrimSpace(print.primitives(f.Args, checksum, indent, cache)) + ")"
	}

	return "bind: <" + strconv.Itoa(int(f.Binding>>32)) + "," +
		strconv.Itoa(int(f.Binding&0xFFFFFFFF)) +
		"> type:" + types.Type(f.Type).Label() +
		argsStr
}

func (print *Printer) primitives(list []*llx.Primitive, checksum string, indent string, cache *printCache) string {
	if len(list) == 0 {
		return ""
	}
	var res strings.Builder

	if len(list) >= 1 {
		res.WriteString(print.primitive(list[0], checksum, indent, cache))
	}

	for i := 1; i < len(list); i++ {
		res.WriteString(", ")
		res.WriteString(strings.TrimLeft(print.primitive(list[i], checksum, indent, cache), " "))
	}
	return res.String()
}

// Primitive prints the llx primitive to a string
func (print *Printer) Primitive(primitive *llx.Primitive, checksum string, bundle *llx.CodeBundle, indent string) string {
	return print.primitive(primitive, checksum, indent, newPrintCache(bundle))
}

// Primitive prints the llx primitive to a string
func (print *Printer) primitive(primitive *llx.Primitive, checksum string, indent string, cache *printCache) string {
	if primitive == nil {
		return "?"
	}

	if primitive.Value == nil && primitive.Array == nil && primitive.Map == nil {
		return "_"
	}
	raw := primitive.RawData()
	return print.data(raw.Type, raw.Value, checksum, indent, cache)
}

func indentBlock(text string, indent string) string {
	return strings.ReplaceAll(text, "\n", "\n"+indent)
}
