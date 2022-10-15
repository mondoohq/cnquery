package printer

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/types"
)

// Results prints a full query with all data points
// NOTE: ensure that results only contains results that match the bundle!
func (print *Printer) Results(bundle *llx.CodeBundle, results map[string]*llx.RawResult) string {
	assessment := llx.Results2Assessment(bundle, results)

	if assessment != nil {
		return print.Assessment(bundle, assessment)
	}

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

	// ident all sub-results
	for i := range assessment.Results {
		cur := print.assessment(bundle, assessment.Results[i], indent)
		res.WriteString(indent)
		res.WriteString(cur)
		res.WriteString("\n")
	}

	return res.String()
}

func isBooleanOp(op string) bool {
	return op == "&&" || op == "||"
}

func (print *Printer) assessment(bundle *llx.CodeBundle, assessment *llx.AssessmentItem, indent string) string {
	var codeID string
	if bundle.CodeV2 != nil {
		codeID = bundle.CodeV2.Id
	} else {
		return "error: no code id"
	}

	nextIndent := indent + "  "
	if assessment.Error != "" {
		var res strings.Builder
		res.WriteString(print.Failed("[failed] "))
		if bundle != nil && bundle.Labels != nil {
			label := bundle.Labels.Labels[assessment.Checksum]
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
			data[i] = print.Primitive(assessment.Data[i], codeID, bundle, indent)
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
		res.WriteString(print.Primitive(assessment.Actual, codeID, bundle, nextIndent))
		res.WriteString(" " + assessment.Operation + " ")
		res.WriteString(print.Primitive(assessment.Expected, codeID, bundle, nextIndent))
		return res.String()
	}

	if assessment.Success {
		if assessment.Actual == nil {
			return print.Secondary("[ok] ") + " (missing result)"
		}

		return print.Secondary("[ok]") +
			" value: " +
			print.Primitive(assessment.Actual, codeID, bundle, indent)
	}

	var res strings.Builder
	res.WriteString(print.Failed("[failed] "))
	if bundle != nil && bundle.Labels != nil {
		label, ok := bundle.Labels.Labels[assessment.Checksum]
		if ok {
			res.WriteString(label)
		}
	}
	res.WriteString("\n")

	if assessment.Expected != nil {
		res.WriteString(nextIndent)
		res.WriteString("expected: " + assessment.Operation + " ")
		res.WriteString(print.Primitive(assessment.Expected, codeID, bundle, nextIndent))
		res.WriteString("\n")
	}

	if assessment.Actual != nil {
		res.WriteString(nextIndent)
		res.WriteString("actual:   ")
		res.WriteString(print.Primitive(assessment.Actual, codeID, bundle, nextIndent))
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

func (print *Printer) label(ref string, bundle *llx.CodeBundle, isResource bool) string {
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

func (print *Printer) array(typ types.Type, data []interface{}, codeID string, bundle *llx.CodeBundle, indent string) string {
	if len(data) == 0 {
		return "[]"
	}

	var res strings.Builder
	res.WriteString("[\n")

	for i := range data {
		res.WriteString(fmt.Sprintf(
			indent+"  %d: %s\n",
			i,
			print.Data(typ.Child(), data[i], codeID, bundle, indent+"  "),
		))
	}

	res.WriteString(indent + "]")
	return res.String()
}

func mapKeys(m map[string]interface{}) []string {
	keys := make([]string, len(m))
	var i int
	for k := range m {
		keys[i] = k
		i++
	}
	return keys
}

func isRefMap(v interface{}) bool {
	smap, ok := v.(map[string]interface{})
	if !ok {
		return false
	}
	for _, vv := range smap {
		_, ok = vv.(*llx.RawData)
		return ok
	}
	return false
}

func (print *Printer) assessmentTemplate(template string, data []string) string {
	res := template
	for i := range data {
		r := "$" + strconv.Itoa(i)
		res = strings.ReplaceAll(res, r, data[i])
	}
	return res
}

func (print *Printer) dataAssessment(codeID string, data map[string]interface{}, bundle *llx.CodeBundle) string {
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
		data, ok = raw.Value.(map[string]interface{})
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

func (print *Printer) refMap(typ types.Type, data map[string]interface{}, codeID string, bundle *llx.CodeBundle, indent string) string {
	if len(data) == 0 {
		return "{}"
	}

	var res strings.Builder
	res.WriteString("{\n")

	keys := mapKeys(data)
	sort.Strings(keys)

	// we need to separate entries that are unlabelled (eg part of an assertion)
	labeledKeys := []string{}
	for i := range keys {
		if _, ok := bundle.Labels.Labels[keys[i]]; ok {
			labeledKeys = append(labeledKeys, keys[i])
		}
	}

	for _, k := range labeledKeys {
		if k == "_" {
			continue
		}

		v := data[k]
		label := print.label(k, bundle, true)
		val := v.(*llx.RawData)

		if val.Error != nil {
			res.WriteString(indent + "  " + label + print.Error(val.Error.Error()) + "\n")
			continue
		}

		if truthy, _ := val.IsTruthy(); !truthy {
			assertion := print.dataAssessment(k, data, bundle)
			if assertion != "" {
				assertion = print.Failed("[failed]") + " " + strings.Trim(assertion, "\n\t ")
				assertion = indentBlock(assertion, indent+"  ")
				res.WriteString(indent + "  " + assertion + "\n")
				continue
			}
		}

		data := print.Data(val.Type, val.Value, k, bundle, indent+"  ")
		res.WriteString(indent + "  " + label + data + "\n")
	}

	res.WriteString(indent + "}")
	return res.String()
}

func (print *Printer) stringMap(typ types.Type, data map[string]interface{}, codeID string, bundle *llx.CodeBundle, indent string) string {
	if len(data) == 0 {
		return "{}"
	}

	var res strings.Builder
	res.WriteString("{\n")

	keys := mapKeys(data)
	sort.Strings(keys)

	for _, k := range keys {
		v := data[k]
		val := print.Data(typ.Child(), v, k, bundle, indent+"  ")
		res.WriteString(fmt.Sprintf(indent+"  %s: %s\n", k, val))
	}

	res.WriteString(indent + "}")
	return res.String()
}

func (print *Printer) dict(typ types.Type, raw interface{}, codeID string, bundle *llx.CodeBundle, indent string) string {
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

	case []interface{}:
		if len(data) == 0 {
			return "[]"
		}

		var res strings.Builder
		res.WriteString("[\n")

		for i := range data {
			res.WriteString(fmt.Sprintf(
				indent+"  %d: %s\n",
				i,
				print.dict(typ, data[i], "", bundle, indent+"  "),
			))
		}

		res.WriteString(indent + "]")
		return res.String()

	case map[string]interface{}:
		if len(data) == 0 {
			return "{}"
		}

		var res strings.Builder
		res.WriteString("{\n")

		keys := mapKeys(data)
		sort.Strings(keys)

		for _, k := range keys {
			s := print.dict(typ, data[k], "", bundle, indent+"  ")
			res.WriteString(fmt.Sprintf(indent+"  %s: %s\n", k, s))
		}

		res.WriteString(indent + "}")
		return res.String()

	default:
		return print.Secondary(fmt.Sprintf("%+v", raw))
	}
}

func (print *Printer) intMap(typ types.Type, data map[int]interface{}, codeID string, bundle *llx.CodeBundle, indent string) string {
	var res strings.Builder
	res.WriteString("{\n")

	for i := range data {
		value := data[i]
		res.WriteString(fmt.Sprintf(indent+"  %d: "+print.Secondary("%#v")+"\n", i, value))
	}

	res.WriteString(indent + "}")
	return res.String()
}

var codeBlockIds = map[string]struct{}{
	"{}": {},
	"if": {},
}

func isCodeBlock(codeID string, bundle *llx.CodeBundle) bool {
	if bundle == nil {
		return false
	}

	_, ok := bundle.Labels.Labels[codeID]
	return ok
}

func (print *Printer) Data(typ types.Type, data interface{}, codeID string, bundle *llx.CodeBundle, indent string) string {
	if typ.IsEmpty() {
		return "no data available"
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

		seconds := llx.TimeToDuration(time)
		minutes := seconds / 60
		hours := minutes / 60
		days := hours / 24

		var res strings.Builder
		if days > 0 {
			res.WriteString(fmt.Sprintf("%d days ", days))
		}
		if hours%24 != 0 {
			res.WriteString(fmt.Sprintf("%d hours ", hours%24))
		}
		if minutes%24 != 0 {
			res.WriteString(fmt.Sprintf("%d minutes ", minutes%60))
		}
		// if we haven't printed any of the other pieces (days/hours/minutes) then print this
		// if we have, then check if this is non-zero
		if minutes == 0 || seconds%60 != 0 {
			res.WriteString(fmt.Sprintf("%d seconds", seconds%60))
		}

		return print.Secondary(res.String())
	case types.Dict:
		return print.dict(typ, data, codeID, bundle, indent)

	case types.Score:
		return llx.ScoreString(data.([]byte))

	case types.Block:
		return print.refMap(typ, data.(map[string]interface{}), codeID, bundle, indent)

	case types.ArrayLike:
		if data == nil {
			return print.Secondary("null")
		}
		return print.array(typ, data.([]interface{}), codeID, bundle, indent)

	case types.MapLike:
		if data == nil {
			return print.Secondary("null")
		}
		if typ.Key() == types.String {
			return print.stringMap(typ, data.(map[string]interface{}), codeID, bundle, indent)
		}
		if typ.Key() == types.Int {
			return print.intMap(typ, data.(map[int]interface{}), codeID, bundle, indent)
		}
		return "unable to render map, its type is not supported: " + typ.Label() + ", raw: " + fmt.Sprintf("%#v", data)

	case types.ResourceLike:
		if data == nil {
			return print.Secondary("null")
		}

		r := data.(resources.ResourceType)
		i := r.MqlResource()
		idline := i.Name
		if i.Id != "" {
			idline += " id = " + i.Id
		}

		return idline
	case types.FunctionLike:
		if d, ok := data.(uint64); ok {
			return indent + fmt.Sprintf("=> <%d,0>", d>>32)
		} else {
			return indent + fmt.Sprintf("=> %#v", data)
		}
	default:
		return indent + fmt.Sprintf("🤷 %#v", data)
	}
}

// DataWithLabel prints RawData into a string
func (print *Printer) DataWithLabel(r *llx.RawData, codeID string, bundle *llx.CodeBundle, indent string) string {
	b := strings.Builder{}
	if r.Error != nil {
		b.WriteString(print.Error(
			fmt.Sprintf("Query encountered errors:\n%s\n",
				strings.TrimSpace(r.Error.Error()))))
	}

	b.WriteString(print.label(codeID, bundle, r.Type.IsResource()))
	b.WriteString(print.Data(r.Type, r.Value, codeID, bundle, indent))
	return b.String()
}

// CodeBundle prints a bundle to a string
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
		res.WriteString("]\n")

		for j := range block.Chunks {
			res.WriteString("   ")
			print.chunkV2(j, block.Chunks[j], block, bundle, indent, &res)
		}
	}

	return res.String()
}

func (print *Printer) chunkV2(idx int, chunk *llx.Chunk, block *llx.Block, bundle *llx.CodeBundle, indent string, w *strings.Builder) {
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
		w.WriteString(print.Primitive(primitive, bundle.CodeV2.Id, bundle, indent))
	}

	if chunk.Function != nil {
		w.WriteString(print.functionV2(chunk.Function, bundle.CodeV2.Id, bundle, indent))
	}

	w.WriteString("\n")
}

func (print *Printer) functionV2(f *llx.Function, codeID string, bundle *llx.CodeBundle, indent string) string {
	argsStr := ""
	if len(f.Args) > 0 {
		argsStr = " (" + strings.TrimSpace(print.primitives(f.Args, codeID, bundle, indent)) + ")"
	}

	return "bind: <" + strconv.Itoa(int(f.Binding>>32)) + "," +
		strconv.Itoa(int(f.Binding&0xFFFFFFFF)) +
		"> type:" + types.Type(f.Type).Label() +
		argsStr
}

func (print *Printer) primitives(list []*llx.Primitive, codeID string, bundle *llx.CodeBundle, indent string) string {
	if len(list) == 0 {
		return ""
	}
	var res strings.Builder

	if len(list) >= 1 {
		res.WriteString(print.Primitive(list[0], codeID, bundle, indent))
	}

	for i := 1; i < len(list); i++ {
		res.WriteString(", ")
		res.WriteString(strings.TrimLeft(print.Primitive(list[i], codeID, bundle, indent), " "))
	}
	return res.String()
}

// Primitive prints the llx primitive to a string
func (print *Printer) Primitive(primitive *llx.Primitive, codeID string, bundle *llx.CodeBundle, indent string) string {
	if primitive == nil {
		return "?"
	}

	if primitive.Value == nil && primitive.Array == nil && primitive.Map == nil {
		return "_"
	}
	raw := primitive.RawData()
	return print.Data(raw.Type, raw.Value, codeID, bundle, indent)
}

func indentBlock(text string, indent string) string {
	return strings.ReplaceAll(text, "\n", "\n"+indent)
}
