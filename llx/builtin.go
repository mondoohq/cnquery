package llx

import (
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/types"
)

type chunkHandler struct {
	Compiler func(types.Type, types.Type) (string, error)
	f        func(*LeiseExecutor, *RawData, *Chunk, int32) (*RawData, int32, error)
	Label    string
}

// BuiltinFunctions for all builtin types
var BuiltinFunctions map[types.Type]map[string]chunkHandler

func init() {
	BuiltinFunctions = map[types.Type]map[string]chunkHandler{
		types.Bool: map[string]chunkHandler{
			string("==" + types.Bool):                chunkHandler{f: boolCmpBool, Label: "=="},
			string("!=" + types.Bool):                chunkHandler{f: boolNotBool, Label: "!="},
			string("==" + types.String):              chunkHandler{f: boolCmpString, Label: "=="},
			string("!=" + types.String):              chunkHandler{f: boolNotString, Label: "!="},
			string("==" + types.Regex):               chunkHandler{f: boolCmpRegex, Label: "=="},
			string("!=" + types.Regex):               chunkHandler{f: boolNotRegex, Label: "!="},
			string("==" + types.Array(types.Bool)):   chunkHandler{f: boolCmpBoolarray, Label: "=="},
			string("!=" + types.Array(types.Bool)):   chunkHandler{f: boolNotBoolarray, Label: "!="},
			string("==" + types.Array(types.String)): chunkHandler{f: boolCmpStringarray, Label: "=="},
			string("!=" + types.Array(types.String)): chunkHandler{f: boolNotStringarray, Label: "!="},
		},
		types.Int: map[string]chunkHandler{
			string("==" + types.Int):                 chunkHandler{f: intCmpInt, Label: "=="},
			string("!=" + types.Int):                 chunkHandler{f: intNotInt, Label: "!="},
			string("==" + types.String):              chunkHandler{f: intCmpString, Label: "=="},
			string("!=" + types.String):              chunkHandler{f: intNotString, Label: "!="},
			string("==" + types.Regex):               chunkHandler{f: intCmpRegex, Label: "=="},
			string("!=" + types.Regex):               chunkHandler{f: intNotRegex, Label: "!="},
			string("==" + types.Array(types.Int)):    chunkHandler{f: intCmpIntarray, Label: "=="},
			string("!=" + types.Array(types.Int)):    chunkHandler{f: intNotIntarray, Label: "!="},
			string("==" + types.Array(types.String)): chunkHandler{f: intCmpStringarray, Label: "=="},
			string("!=" + types.Array(types.String)): chunkHandler{f: intNotStringarray, Label: "!="},
			string("<" + types.Int):                  chunkHandler{f: intLTInt, Label: "<"},
			string("<=" + types.Int):                 chunkHandler{f: intLTEInt, Label: "<="},
			string(">" + types.Int):                  chunkHandler{f: intGTInt, Label: ">"},
			string(">=" + types.Int):                 chunkHandler{f: intGTEInt, Label: ">="},
			string("<" + types.Float):                chunkHandler{f: intLTFloat, Label: "<"},
			string("<=" + types.Float):               chunkHandler{f: intLTEFloat, Label: "<="},
			string(">" + types.Float):                chunkHandler{f: intGTFloat, Label: ">"},
			string(">=" + types.Float):               chunkHandler{f: intGTEFloat, Label: ">="},
			string("<" + types.String):               chunkHandler{f: intLTString, Label: "<"},
			string("<=" + types.String):              chunkHandler{f: intLTEString, Label: "<="},
			string(">" + types.String):               chunkHandler{f: intGTString, Label: ">"},
			string(">=" + types.String):              chunkHandler{f: intGTEString, Label: ">="},
		},
		types.Float: map[string]chunkHandler{
			string("==" + types.Float):               chunkHandler{f: floatCmpFloat, Label: "=="},
			string("!=" + types.Float):               chunkHandler{f: floatNotFloat, Label: "!="},
			string("==" + types.String):              chunkHandler{f: floatCmpString, Label: "=="},
			string("!=" + types.String):              chunkHandler{f: floatNotString, Label: "!="},
			string("==" + types.Regex):               chunkHandler{f: floatCmpRegex, Label: "=="},
			string("!=" + types.Regex):               chunkHandler{f: floatNotRegex, Label: "!="},
			string("==" + types.Array(types.Float)):  chunkHandler{f: floatCmpFloatarray, Label: "=="},
			string("!=" + types.Array(types.Float)):  chunkHandler{f: floatNotFloatarray, Label: "!="},
			string("==" + types.Array(types.String)): chunkHandler{f: floatCmpStringarray, Label: "=="},
			string("!=" + types.Array(types.String)): chunkHandler{f: floatNotStringarray, Label: "!="},
			string("<" + types.Float):                chunkHandler{f: floatLTFloat, Label: "<"},
			string("<=" + types.Float):               chunkHandler{f: floatLTEFloat, Label: "<="},
			string(">" + types.Float):                chunkHandler{f: floatGTFloat, Label: ">"},
			string(">=" + types.Float):               chunkHandler{f: floatGTEFloat, Label: ">="},
			string("<" + types.Int):                  chunkHandler{f: floatLTInt, Label: "<"},
			string("<=" + types.Int):                 chunkHandler{f: floatLTEInt, Label: "<="},
			string(">" + types.Int):                  chunkHandler{f: floatGTInt, Label: ">"},
			string(">=" + types.Int):                 chunkHandler{f: floatGTEInt, Label: ">="},
			string("<" + types.String):               chunkHandler{f: floatLTString, Label: "<"},
			string("<=" + types.String):              chunkHandler{f: floatLTEString, Label: "<="},
			string(">" + types.String):               chunkHandler{f: floatGTString, Label: ">"},
			string(">=" + types.String):              chunkHandler{f: floatGTEString, Label: ">="},
		},
		types.String: map[string]chunkHandler{
			string("==" + types.String):              chunkHandler{f: stringCmpString, Label: "=="},
			string("!=" + types.String):              chunkHandler{f: stringNotString, Label: "!="},
			string("==" + types.Regex):               chunkHandler{f: stringCmpRegex, Label: "=="},
			string("!=" + types.Regex):               chunkHandler{f: stringNotRegex, Label: "!="},
			string("==" + types.Bool):                chunkHandler{f: stringCmpBool, Label: "=="},
			string("!=" + types.Bool):                chunkHandler{f: stringNotBool, Label: "!="},
			string("==" + types.Int):                 chunkHandler{f: stringCmpInt, Label: "=="},
			string("!=" + types.Int):                 chunkHandler{f: stringNotInt, Label: "!="},
			string("==" + types.Float):               chunkHandler{f: stringCmpFloat, Label: "=="},
			string("!=" + types.Float):               chunkHandler{f: stringNotFloat, Label: "!="},
			string("==" + types.Array(types.String)): chunkHandler{f: stringCmpStringarray, Label: "=="},
			string("!=" + types.Array(types.String)): chunkHandler{f: stringNotStringarray, Label: "!="},
			string("==" + types.Array(types.Bool)):   chunkHandler{f: stringCmpBoolarray, Label: "=="},
			string("!=" + types.Array(types.Bool)):   chunkHandler{f: stringNotBoolarray, Label: "!="},
			string("==" + types.Array(types.Int)):    chunkHandler{f: stringCmpIntarray, Label: "=="},
			string("!=" + types.Array(types.Int)):    chunkHandler{f: stringNotIntarray, Label: "!="},
			string("==" + types.Array(types.Float)):  chunkHandler{f: stringCmpFloatarray, Label: "=="},
			string("!=" + types.Array(types.Float)):  chunkHandler{f: stringNotFloatarray, Label: "!="},
			string("<" + types.String):               chunkHandler{f: stringLTString, Label: "<"},
			string("<=" + types.String):              chunkHandler{f: stringLTEString, Label: "<="},
			string(">" + types.String):               chunkHandler{f: stringGTString, Label: ">"},
			string(">=" + types.String):              chunkHandler{f: stringGTEString, Label: ">="},
			string("<" + types.Int):                  chunkHandler{f: stringLTInt, Label: "<"},
			string("<=" + types.Int):                 chunkHandler{f: stringLTEInt, Label: "<="},
			string(">" + types.Int):                  chunkHandler{f: stringGTInt, Label: ">"},
			string(">=" + types.Int):                 chunkHandler{f: stringGTEInt, Label: ">="},
			string("<" + types.Float):                chunkHandler{f: stringLTFloat, Label: "<"},
			string("<=" + types.Float):               chunkHandler{f: stringLTEFloat, Label: "<="},
			string(">" + types.Float):                chunkHandler{f: stringGTFloat, Label: ">"},
			string(">=" + types.Float):               chunkHandler{f: stringGTEFloat, Label: ">="},
		},
		types.Regex: map[string]chunkHandler{
			string("==" + types.Regex):               chunkHandler{f: stringCmpString, Label: "=="},
			string("!=" + types.Regex):               chunkHandler{f: stringNotString, Label: "!="},
			string("==" + types.Bool):                chunkHandler{f: regexCmpBool, Label: "=="},
			string("!=" + types.Bool):                chunkHandler{f: regexNotBool, Label: "!="},
			string("==" + types.Int):                 chunkHandler{f: regexCmpInt, Label: "=="},
			string("!=" + types.Int):                 chunkHandler{f: regexNotInt, Label: "!="},
			string("==" + types.Float):               chunkHandler{f: regexCmpFloat, Label: "=="},
			string("!=" + types.Float):               chunkHandler{f: regexNotFloat, Label: "!="},
			string("==" + types.String):              chunkHandler{f: regexCmpString, Label: "=="},
			string("!=" + types.String):              chunkHandler{f: regexNotString, Label: "!="},
			string("==" + types.Array(types.Regex)):  chunkHandler{f: stringCmpStringarray, Label: "=="},
			string("!=" + types.Array(types.Regex)):  chunkHandler{f: stringNotStringarray, Label: "!="},
			string("==" + types.Array(types.Bool)):   chunkHandler{f: regexCmpBoolarray, Label: "=="},
			string("!=" + types.Array(types.Bool)):   chunkHandler{f: regexNotBoolarray, Label: "!="},
			string("==" + types.Array(types.Int)):    chunkHandler{f: regexCmpIntarray, Label: "=="},
			string("!=" + types.Array(types.Int)):    chunkHandler{f: regexNotIntarray, Label: "!="},
			string("==" + types.Array(types.Float)):  chunkHandler{f: regexCmpFloatarray, Label: "=="},
			string("!=" + types.Array(types.Float)):  chunkHandler{f: regexNotFloatarray, Label: "!="},
			string("==" + types.Array(types.String)): chunkHandler{f: regexCmpStringarray, Label: "=="},
			string("!=" + types.Array(types.String)): chunkHandler{f: regexNotStringarray, Label: "!="},
		},
		types.ArrayLike: map[string]chunkHandler{
			"[]":     chunkHandler{f: arrayGetIndex},
			"{}":     chunkHandler{f: arrayBlockList},
			"length": chunkHandler{f: arrayLength},
			"==":     chunkHandler{Compiler: compileArrayOpArray("==")},
			"!=":     chunkHandler{Compiler: compileArrayOpArray("!=")},
			// []T -- []T
			string(types.Bool + "==" + types.Array(types.Bool)):     chunkHandler{f: boolarrayCmpBoolarray, Label: "=="},
			string(types.Bool + "!=" + types.Array(types.Bool)):     chunkHandler{f: boolarrayNotBoolarray, Label: "!="},
			string(types.Int + "==" + types.Array(types.Int)):       chunkHandler{f: intarrayCmpIntarray, Label: "=="},
			string(types.Int + "!=" + types.Array(types.Int)):       chunkHandler{f: intarrayNotIntarray, Label: "!="},
			string(types.Float + "==" + types.Array(types.Float)):   chunkHandler{f: floatarrayCmpFloatarray, Label: "=="},
			string(types.Float + "!=" + types.Array(types.Float)):   chunkHandler{f: floatarrayNotFloatarray, Label: "!="},
			string(types.String + "==" + types.Array(types.String)): chunkHandler{f: stringarrayCmpStringarray, Label: "=="},
			string(types.String + "!=" + types.Array(types.String)): chunkHandler{f: stringarrayNotStringarray, Label: "!="},
			string(types.Regex + "==" + types.Array(types.Regex)):   chunkHandler{f: stringarrayCmpStringarray, Label: "=="},
			string(types.Regex + "!=" + types.Array(types.Regex)):   chunkHandler{f: stringarrayNotStringarray, Label: "!="},
			// []T -- T
			string(types.Bool + "==" + types.Bool):     chunkHandler{f: boolarrayCmpBool, Label: "=="},
			string(types.Bool + "!=" + types.Bool):     chunkHandler{f: boolarrayNotBool, Label: "!="},
			string(types.Int + "==" + types.Int):       chunkHandler{f: intarrayCmpInt, Label: "=="},
			string(types.Int + "!=" + types.Int):       chunkHandler{f: intarrayNotInt, Label: "!="},
			string(types.Float + "==" + types.Float):   chunkHandler{f: floatarrayCmpFloat, Label: "=="},
			string(types.Float + "!=" + types.Float):   chunkHandler{f: floatarrayNotFloat, Label: "!="},
			string(types.String + "==" + types.String): chunkHandler{f: stringarrayCmpString, Label: "=="},
			string(types.String + "!=" + types.String): chunkHandler{f: stringarrayNotString, Label: "!="},
			string(types.Regex + "==" + types.Regex):   chunkHandler{f: stringarrayCmpString, Label: "=="},
			string(types.Regex + "!=" + types.Regex):   chunkHandler{f: stringarrayNotString, Label: "!="},
			// []string -- T
			string(types.String + "==" + types.Bool):  chunkHandler{f: stringarrayCmpBool, Label: "=="},
			string(types.String + "!=" + types.Bool):  chunkHandler{f: stringarrayNotBool, Label: "!="},
			string(types.String + "==" + types.Int):   chunkHandler{f: stringarrayCmpInt, Label: "=="},
			string(types.String + "!=" + types.Int):   chunkHandler{f: stringarrayNotInt, Label: "!="},
			string(types.String + "==" + types.Float): chunkHandler{f: stringarrayCmpFloat, Label: "=="},
			string(types.String + "!=" + types.Float): chunkHandler{f: stringarrayNotFloat, Label: "!="},
			// []T -- string
			string(types.Bool + "==" + types.String):  chunkHandler{f: boolarrayCmpString, Label: "=="},
			string(types.Bool + "!=" + types.String):  chunkHandler{f: boolarrayNotString, Label: "!="},
			string(types.Int + "==" + types.String):   chunkHandler{f: intarrayCmpString, Label: "=="},
			string(types.Int + "!=" + types.String):   chunkHandler{f: intarrayNotString, Label: "!="},
			string(types.Float + "==" + types.String): chunkHandler{f: floatarrayCmpString, Label: "=="},
			string(types.Float + "!=" + types.String): chunkHandler{f: floatarrayNotString, Label: "!="},
			// []T -- regex
			string(types.Bool + "==" + types.Regex):   chunkHandler{f: boolarrayCmpRegex, Label: "=="},
			string(types.Bool + "!=" + types.Regex):   chunkHandler{f: boolarrayNotRegex, Label: "!="},
			string(types.Int + "==" + types.Regex):    chunkHandler{f: intarrayCmpRegex, Label: "=="},
			string(types.Int + "!=" + types.Regex):    chunkHandler{f: intarrayNotRegex, Label: "!="},
			string(types.Float + "==" + types.Regex):  chunkHandler{f: floatarrayCmpRegex, Label: "=="},
			string(types.Float + "!=" + types.Regex):  chunkHandler{f: floatarrayNotRegex, Label: "!="},
			string(types.String + "==" + types.Regex): chunkHandler{f: stringarrayCmpRegex, Label: "=="},
			string(types.String + "!=" + types.Regex): chunkHandler{f: stringarrayNotRegex, Label: "!="},
		},
		types.MapLike: map[string]chunkHandler{
			"[]":     chunkHandler{f: mapGetIndex},
			"length": chunkHandler{f: mapLength},
		},
		types.ResourceLike: map[string]chunkHandler{
			"where":  chunkHandler{f: resourceWhere},
			"length": chunkHandler{f: resourceLength},
			"{}": chunkHandler{f: func(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
				return c.runBlock(bind, chunk.Function.Args[0], ref)
			}},
		},
	}
}

func runResourceFunction(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	// ugh something is wrong here.... fix it later
	rr, ok := bind.Value.(lumi.ResourceType)
	if !ok {
		// TODO: can we get rid of this fmt call
		return nil, 0, fmt.Errorf("cannot cast resource to resource type: %+v", bind.Value)
	}

	info := rr.LumiResource()
	// resource := c.runtime.Registry.Resources[bind.Type]
	if info == nil {
		return nil, 0, errors.New("Cannot retrieve resource from the binding to run the raw function")
	}

	resource, ok := c.runtime.Registry.Resources[info.Name]
	if !ok || resource == nil {
		return nil, 0, errors.New("Cannot retrieve resource definition for resource '" + info.Name + "'")
	}

	// record this watcher on the executors watcher IDs
	wid := c.watcherUID(ref)
	log.Debug().Str("wid", wid).Msg("exec> add watcher id ")
	c.watcherIds.Store(wid)

	// watch this field in the resource
	err := c.runtime.WatchAndUpdate(rr, chunk.Id, wid, func(fieldData interface{}, fieldError error) {
		if fieldError != nil {
			c.callback(errorResult(fieldError, c.entrypoints[ref]))
			return
		}

		c.cache.Store(ref, &stepCache{Result: &RawData{
			Type:  types.Type(resource.Fields[chunk.Id].Type),
			Value: fieldData,
			Error: fieldError,
		}})
		c.triggerChain(ref)
	})

	// we are done executing this chain
	return nil, 0, err
}

// BuiltinFunction provides the handler for this type's function
func BuiltinFunction(typ types.Type, name string) (*chunkHandler, error) {
	h, ok := BuiltinFunctions[typ.Underlying()]
	if !ok {
		return nil, errors.New("cannot find functions for type '" + typ.Label() + "' (called '" + name + "')")
	}
	fh, ok := h[name]
	if !ok {
		return nil, errors.New("cannot find function '" + name + "' for type '" + typ.Label() + "'")
	}
	return &fh, nil
}

// this is called for objects that call a function
func (c *LeiseExecutor) runBoundFunction(bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	log.Debug().Int32("ref", ref).Str("id", chunk.Id).Msg("exec> run bound function")

	fh, err := BuiltinFunction(bind.Type, chunk.Id)
	if err == nil {
		res, dref, err := fh.f(c, bind, chunk, ref)
		if res != nil {
			c.cache.Store(ref, &stepCache{Result: res})
		}
		return res, dref, err
	}

	if bind.Type.IsResource() {
		return runResourceFunction(c, bind, chunk, ref)
	}
	return nil, 0, err
}
