package llx

import (
	"errors"
	"sort"
	"strconv"
	"sync"

	uuid "github.com/gofrs/uuid"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/types"
)

// ResultCallback function type
type ResultCallback func(*RawResult)

var emptyFunction = Function{}
var blockType = types.Map(types.String, types.Any)

// RawResult wraps RawData to code and refs
type RawResult struct {
	Data   *RawData
	CodeID string
}

type stepCache struct {
	Result   *RawData
	IsStatic bool
}

// Calls is a map connecting call-refs with each other
type Calls struct{ sync.Map }

// Store a new call connection
func (c *Calls) Store(k int32, v int32) { c.Map.Store(k, v) }

// Load a call connection
func (c *Calls) Load(k int32) (int32, bool) {
	res, ok := c.Map.Load(k)
	if !ok {
		return 0, ok
	}
	return res.(int32), ok
}

// Cache is a map containing stepCache values
type Cache struct{ sync.Map }

// Store a new call connection
func (c *Cache) Store(k int32, v *stepCache) { c.Map.Store(k, v) }

// Load a call connection
func (c *Cache) Load(k int32) (*stepCache, bool) {
	res, ok := c.Map.Load(k)
	if res == nil {
		return nil, ok
	}
	return res.(*stepCache), ok
}

// LeiseExecutor is the runtime of a leise/llx codestructure
type LeiseExecutor struct {
	id             string
	watcherIds     types.StringSet
	blockExecutors []*LeiseExecutor
	runtime        *lumi.Runtime
	code           *Code
	entrypoints    map[int32]string
	callback       ResultCallback
	cache          Cache
	calls          Calls
	starts         []int32
}

func (c *LeiseExecutor) watcherUID(ref int32) string {
	return c.id + "\x00" + strconv.FormatInt(int64(ref), 10)
}

func errorResult(err error, codeID string) *RawResult {
	return &RawResult{
		Data:   &RawData{Error: err},
		CodeID: codeID,
	}
}

func errorResultMsg(msg string, codeID string) *RawResult {
	return &RawResult{
		Data:   &RawData{Error: errors.New(msg)},
		CodeID: codeID,
	}
}

// NewExecutor will create a code runner from code, running in a runtime, calling
// callback whenever we get a result
func NewExecutor(code *Code, runtime *lumi.Runtime, callback ResultCallback) (*LeiseExecutor, error) {
	if runtime == nil {
		return nil, errors.New("cannot exec leise without a runtime")
	}

	if code == nil {
		return nil, errors.New("cannot RunChunky without code")
	}

	res := &LeiseExecutor{
		id:          uuid.Must(uuid.NewV4()).String(),
		runtime:     runtime,
		entrypoints: make(map[int32]string),
		code:        code,
		callback:    callback,
	}

	for _, ref := range code.Entrypoints {
		id := code.Checksums[ref]
		if id == "" {
			return nil, errors.New("llx.executor> cannot execute with invalid ref ID")
		}
		if ref < 1 {
			return nil, errors.New("llx.executor> cannot execute with invalid ref number")
		}
		res.entrypoints[ref] = id
	}

	return res, nil
}

// Run code with a runtime and return results
func (c *LeiseExecutor) Run() {
	// work down all entrypoints
	entrypoints := make([]int32, len(c.entrypoints))
	i := 0
	for ref := range c.entrypoints {
		entrypoints[i] = ref
		i++
	}
	sort.Slice(entrypoints, func(i, j int) bool { return entrypoints[i] < entrypoints[j] })

	for _, ref := range entrypoints {
		// if this entrypoint is already connected, don't add it again
		if _, ok := c.calls.Load(ref); ok {
			continue
		}

		log.Debug().Int32("entrypoint", ref).Str("exec-ID", c.id).Msg("exec.Run>")
		c.runChain(ref)
	}
}

// Unregister an execution chain from receiving any further updates
func (c *LeiseExecutor) Unregister() error {
	log.Debug().Str("id", c.id).Msg("exec> unregister")
	// clear out the callback, we don't want it to be called now anymore
	c.callback = func(_ *RawResult) {
		log.Warn().Str("id", c.id).Msg("exec> Decomissioned callback called on exec.LeiseExecutor")
	}

	errorList := []error{}

	for idx := range c.blockExecutors {
		if err := c.blockExecutors[idx].Unregister(); err != nil {
			log.Error().Err(err).Msg("exec> block unregister error")
			errorList = append(errorList, err)
		}
	}

	c.watcherIds.Range(func(key string) bool {
		if err := c.runtime.Unregister(key); err != nil {
			log.Error().Err(err).Msg("exec> unregister error")
			errorList = append(errorList, err)
		}
		return true
	})

	if len(errorList) > 0 {
		return errors.New("multiple errors unregistering")
	}
	return nil
}

func (c *LeiseExecutor) registerPrimitive(val *Primitive) {
	// TODO: not yet implemented?
}

func (c *LeiseExecutor) runFunctionBlock(bind *RawData, code *Code, cb ResultCallback) error {
	executor, err := NewExecutor(code, c.runtime, cb)
	if err != nil {
		return err
	}
	c.blockExecutors = append(c.blockExecutors, executor)
	executor.cache.Store(1, &stepCache{
		Result:   bind,
		IsStatic: true,
	})
	executor.Run()
	return nil
}

func (c *LeiseExecutor) runBlock(bind *RawData, functionRef *Primitive, ref int32) (*RawData, int32, error) {
	typ := types.Type(functionRef.Type)
	if !typ.IsFunction() {
		return nil, 0, errors.New("Called block with wrong function type")
	}
	fref, ok := functionRef.Ref()
	if !ok {
		return nil, 0, errors.New("Cannot retrieve function reference on block call")
	}
	fun := c.code.Functions[fref-1]
	if fun == nil {
		return nil, 0, errors.New("Block function is nil")
	}

	blockResult := map[string]interface{}{}
	err := c.runFunctionBlock(bind, fun, func(res *RawResult) {
		blockResult[res.CodeID] = res.Data
		if len(blockResult) == len(fun.Entrypoints) {
			c.cache.Store(ref, &stepCache{
				Result: &RawData{
					Type:  blockType,
					Value: blockResult,
				},
				IsStatic: true,
			})
			c.triggerChain(ref)
		}
	})

	return nil, 0, err
}

func (c *LeiseExecutor) createResource(name string, f *Function, ref int32) (*RawData, int32, error) {
	args, err := args2resourceargs(f.Args)
	if err != nil {
		return nil, 0, err
	}

	resource, err := c.runtime.CreateResource(name, args...)
	if err != nil {
		return nil, 0, err
	}

	res := stepCache{
		Result: &RawData{
			Type:  types.Resource(name),
			Value: resource,
		},
		IsStatic: true,
	}
	c.cache.Store(ref, &res)
	return res.Result, 0, nil
}

func (c *LeiseExecutor) runGlobalFunction(chunk *Chunk, f *Function, ref int32) (*RawData, int32, error) {
	h, ok := handleGlobal(chunk.Id)
	if ok {
		if h == nil {
			return nil, 0, errors.New("found function " + chunk.Id + " but no handler. this should not happen and points to an implementation error")
		}

		res, dref, err := h(c, f, ref)
		log.Debug().Msgf("exec> global: %s %+v = %#v", chunk.Id, f.Args, res)
		if res != nil {
			c.cache.Store(ref, &stepCache{Result: res})
		}
		return res, dref, err
	}

	return c.createResource(chunk.Id, f, ref)
}

// connect references, calling `dst` if `src` is updated
func (c *LeiseExecutor) connectRef(src int32, dst int32) (*RawData, int32, error) {
	// check if the ref is connected and connect it if not
	_, ok := c.calls.LoadOrStore(src, dst)
	if ok {
		return nil, 0, nil
	}
	// if the ref was not yet connected, we must run the src ref after we connected it
	return nil, src, nil
}

func (c *LeiseExecutor) runFunction(chunk *Chunk, ref int32) (*RawData, int32, error) {
	f := chunk.Function
	if f == nil {
		f = &emptyFunction
	}

	// global functions, for now only resources
	if f.Binding == 0 {
		return c.runGlobalFunction(chunk, f, ref)
	}

	// check if the bound value exists, otherwise connect it
	res, ok := c.cache.Load(f.Binding)
	if !ok {
		return c.connectRef(f.Binding, ref)
	}
	return c.runBoundFunction(res.Result, chunk, ref)
}

func (c *LeiseExecutor) runPrimitive(chunk *Chunk, ref int32) (*RawData, int32, error) {
	return chunk.Primitive.RawData(), 0, nil
}

func (c *LeiseExecutor) runChunk(chunk *Chunk, ref int32) (*RawData, int32, error) {
	switch chunk.Call {
	case Chunk_PRIMITIVE:
		res, dref, err := c.runPrimitive(chunk, ref)
		if dref != 0 || err != nil {
			return res, dref, err
		}
		c.cache.Store(ref, &stepCache{Result: res})
		return res, dref, err
	case Chunk_FUNCTION:
		return c.runFunction(chunk, ref)
	default:
		return nil, 0, errors.New("Tried to run a chunk which has an unknown type: " + chunk.Call.String())
	}
}

func (c *LeiseExecutor) runRef(ref int32) (*RawData, int32, error) {
	chunk := c.code.Code[ref-1]
	if chunk == nil {
		return nil, 0, errors.New("Called a chunk that doesn't exist, ref = " + strconv.FormatInt(int64(ref), 10))
	}
	return c.runChunk(chunk, ref)
}

// runChain starting at a ref of the code, follow it down and report
// jever result it has at the end of its execution. this will register
// async callbacks against referenced chunks too
func (c *LeiseExecutor) runChain(start int32) {
	var res *RawData
	var err error
	nextRef := start
	var curRef int32

	for nextRef != 0 {
		curRef = nextRef
		// log.Debug().Int32("ref", curRef).Msg("exec> run chain")

		res, nextRef, err = c.runRef(curRef)

		// back out of errors directly
		if err != nil {
			c.callback(errorResult(err, c.entrypoints[start]))
			return
		}

		// stop this chain of execution, if it didn't return anything
		// we need more data ie an event to provide info
		if res == nil && nextRef == 0 {
			return
		}

		// if this is a result for an existing entrypoint send it
		if res != nil {
			if codeID, ok := c.entrypoints[curRef]; ok {
				// log.Debug().Int32("ref", curRef).Msgf("exec> chain callback")
				c.callback(&RawResult{Data: res, CodeID: codeID})
			}
		}

		// get the next reference, if we are not directed anywhere
		if nextRef == 0 {
			// note: if the call cannot be retrieved it will use the
			// zero value, which is 0 in this case; i.e. if !ok => ref = 0
			nextRef, _ = c.calls.Load(curRef)
		}
	}
}

// triggerChain when a reference has a new value set
// unlike runChain this will not execute the ref chunk, but rather
// try to move to the next called chunk - or if it's not available
// handle the result
func (c *LeiseExecutor) triggerChain(ref int32) {
	nxt, ok := c.calls.Load(ref)
	if ok {
		if nxt < 1 {
			panic("internal state error: cannot trigger next call on chain because it points to a zero ref")
		}
		c.runChain(nxt)
		return
	}

	codeID := c.entrypoints[ref]
	res, ok := c.cache.Load(ref)
	if !ok {
		c.callback(errorResultMsg("exec> Cannot find results to chunk reference "+strconv.FormatInt(int64(ref), 10), codeID))
		return
	}

	log.Debug().Int32("ref", ref).Msgf("exec> trigger callback")
	c.callback(&RawResult{Data: res.Result, CodeID: codeID})
}

func (c *LeiseExecutor) triggerChainError(ref int32, err error) {
	cur := ref
	for cur > 0 {
		if codeID, ok := c.entrypoints[cur]; ok {
			c.callback(&RawResult{
				Data: &RawData{
					Error: err,
				},
				CodeID: codeID,
			})
		}

		nxt, ok := c.calls.Load(cur)
		if !ok {
			break
		}
		if nxt < 1 {
			panic("internal state error: cannot trigger next call on chain because it points to a zero ref")
		}
		cur = nxt
	}
}
