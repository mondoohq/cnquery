package llx

//go:generate protoc --proto_path=. --go_out=. --go_opt=paths=source_relative --falcon_out=. llx.proto

import (
	"errors"
	"sort"
	"strconv"
	"sync"

	uuid "github.com/gofrs/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/types"
)

// ResultCallback function type
type ResultCallback func(*RawResult)

var emptyFunction = Function{}

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
type Calls struct {
	locker sync.Mutex
	calls  map[int32][]int32
}

// Store a new call connection.
// Returns true if this connection already exists.
// Returns false if this is a new connection.
func (c *Calls) Store(k int32, v int32) bool {
	c.locker.Lock()
	defer c.locker.Unlock()

	calls, ok := c.calls[k]
	if !ok {
		calls = []int32{}
	} else {
		for k := range calls {
			if calls[k] == v {
				return true
			}
		}
	}

	calls = append(calls, v)
	c.calls[k] = calls
	return false
}

// Load a call connection
func (c *Calls) Load(k int32) ([]int32, bool) {
	c.locker.Lock()
	v, ok := c.calls[k]
	c.locker.Unlock()
	return v, ok
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
	watcherIds     *types.StringSet
	blockExecutors []*LeiseExecutor
	runtime        *lumi.Runtime
	code           *Code
	entrypoints    map[int32]struct{}
	callbackPoints map[int32]string
	callback       ResultCallback
	cache          *Cache
	stepTracker    *Cache
	calls          *Calls
	starts         []int32
	props          map[string]*Primitive
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
func NewExecutor(code *Code, runtime *lumi.Runtime, props map[string]*Primitive, callback ResultCallback) (*LeiseExecutor, error) {
	if runtime == nil {
		return nil, errors.New("cannot exec leise without a runtime")
	}

	if code == nil {
		return nil, errors.New("cannot RunChunky without code")
	}

	res := &LeiseExecutor{
		id:             uuid.Must(uuid.NewV4()).String(),
		runtime:        runtime,
		entrypoints:    make(map[int32]struct{}),
		callbackPoints: make(map[int32]string),
		code:           code,
		callback:       callback,
		cache:          &Cache{},
		stepTracker:    &Cache{},
		calls: &Calls{
			locker: sync.Mutex{},
			calls:  map[int32][]int32{},
		},
		watcherIds: &types.StringSet{},
		props:      props,
	}

	for _, ref := range code.Entrypoints {
		id := code.Checksums[ref]
		if id == "" {
			return nil, errors.New("llx.executor> cannot execute with invalid ref ID in entrypoint")
		}
		if ref < 1 {
			return nil, errors.New("llx.executor> cannot execute with invalid ref number in entrypoint")
		}
		res.entrypoints[ref] = struct{}{}
		res.callbackPoints[ref] = id
	}

	for _, ref := range code.Datapoints {
		id := code.Checksums[ref]
		if id == "" {
			return nil, errors.New("llx.executor> cannot execute with invalid ref ID in datapoint")
		}
		if ref < 1 {
			return nil, errors.New("llx.executor> cannot execute with invalid ref number in datapoint")
		}
		res.callbackPoints[ref] = id
	}

	return res, nil
}

// Run code with a runtime and return results
func (c *LeiseExecutor) Run() {
	// work down all entrypoints
	refs := make([]int32, len(c.callbackPoints))
	i := 0
	for ref := range c.callbackPoints {
		refs[i] = ref
		i++
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i] > refs[j] })

	for _, ref := range refs {
		// if this entrypoint is already connected, don't add it again
		if _, ok := c.stepTracker.Load(ref); ok {
			continue
		}

		log.Trace().Int32("entrypoint", ref).Str("exec-ID", c.id).Msg("exec.Run>")
		c.runChain(ref)
	}
}

// Unregister an execution chain from receiving any further updates
func (c *LeiseExecutor) Unregister() error {
	log.Trace().Str("id", c.id).Msg("exec> unregister")
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
	executor, err := NewExecutor(code, c.runtime, c.props, cb)
	if err != nil {
		return err
	}
	c.blockExecutors = append(c.blockExecutors, executor)

	if bind != nil {
		executor.cache.Store(1, &stepCache{
			Result:   bind,
			IsStatic: true,
		})
	}

	executor.Run()
	return nil
}

func (c *LeiseExecutor) runBlock(bind *RawData, functionRef *Primitive, ref int32) (*RawData, int32, error) {

	if bind != nil && bind.Value == nil && bind.Type != types.Nil {
		return &RawData{Type: bind.Type, Value: nil}, 0, nil
	}

	typ := types.Type(functionRef.Type)
	if !typ.IsFunction() {
		return nil, 0, errors.New("called block with wrong function type")
	}
	fref, ok := functionRef.Ref()
	if !ok {
		return nil, 0, errors.New("cannot retrieve function reference on block call")
	}
	fun := c.code.Functions[fref-1]
	if fun == nil {
		return nil, 0, errors.New("block function is nil")
	}

	blockResult := map[string]interface{}{}

	var anyError error
	err := c.runFunctionBlock(bind, fun, func(res *RawResult) {
		if fun.SingleValue {
			c.cache.Store(ref, &stepCache{
				Result: res.Data,
			})
			c.triggerChain(ref, res.Data)
			return
		}

		if _, exists := blockResult[res.CodeID]; !exists && res.Data.Error != nil {
			anyError = multierror.Append(anyError, res.Data.Error)
		}
		blockResult[res.CodeID] = res.Data
		expectedCnt := len(fun.Entrypoints) + len(fun.Datapoints)
		if len(blockResult) == expectedCnt {
			if bind != nil && bind.Type.IsResource() {
				rr, ok := bind.Value.(lumi.ResourceType)
				if !ok {
					log.Warn().Msg("cannot cast resource to resource type")
				} else {
					blockResult["_"] = &RawData{
						Type:  bind.Type,
						Value: rr,
					}
				}
			}

			data := &RawData{
				Type:  types.Block,
				Value: blockResult,
				Error: anyError,
			}
			c.cache.Store(ref, &stepCache{
				Result:   data,
				IsStatic: true,
			})
			c.triggerChain(ref, data)
		}
	})

	return nil, 0, err
}

func (c *LeiseExecutor) createResource(name string, f *Function, ref int32) (*RawData, int32, error) {
	args, rref, err := args2resourceargs(c, ref, f.Args)
	if err != nil || rref != 0 {
		return nil, rref, err
	}

	resource, err := c.runtime.CreateResource(name, args...)
	if err != nil {
		// in case it's not something that requires later loading, store the error
		// so that consecutive steps can retrieve it cached
		if _, ok := err.(lumi.NotReadyError); !ok {
			res := stepCache{
				Result: &RawData{
					Type:  types.Resource(name),
					Value: nil,
					Error: err,
				},
				IsStatic: true,
			}
			c.cache.Store(ref, &res)
		}

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
		log.Trace().Msgf("exec> global: %s %+v = %#v", chunk.Id, f.Args, res)
		if res != nil {
			c.cache.Store(ref, &stepCache{Result: res})
		}
		return res, dref, err
	}

	return c.createResource(chunk.Id, f, ref)
}

// connect references, calling `dst` if `src` is updated
func (c *LeiseExecutor) connectRef(src int32, dst int32) (*RawData, int32, error) {
	// connect the ref. If it is already connected, someone else already made this
	// call, so we don't have to follow up anymore
	if exists := c.calls.Store(src, dst); exists {
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

	if res.Result.Error != nil {
		c.cache.Store(ref, &stepCache{Result: res.Result})
		return nil, 0, res.Result.Error
	}

	return c.runBoundFunction(res.Result, chunk, ref)
}

func (c *LeiseExecutor) runChunk(chunk *Chunk, ref int32) (*RawData, int32, error) {
	switch chunk.Call {
	case Chunk_PRIMITIVE:
		res, dref, err := c.resolveValue(chunk.Primitive, ref)
		if res != nil {
			c.cache.Store(ref, &stepCache{Result: res})
		} else if err != nil {
			c.cache.Store(ref, &stepCache{Result: &RawData{
				Error: err,
			}})
		}

		return res, dref, err
	case Chunk_FUNCTION:
		return c.runFunction(chunk, ref)

	case Chunk_PROPERTY:
		property, ok := c.props[chunk.Id]
		if !ok {
			return nil, 0, errors.New("cannot find property '" + chunk.Id + "'")
		}

		res, dref, err := c.resolveValue(property, ref)
		if dref != 0 || err != nil {
			return res, dref, err
		}
		c.cache.Store(ref, &stepCache{Result: res})
		return res, dref, err

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
	var remaining []int32

	for nextRef != 0 {
		curRef = nextRef
		c.stepTracker.Store(curRef, nil)
		// log.Trace().Int32("ref", curRef).Msg("exec> run chain")

		// Try to load the result from cache if it already exists. This was added
		// so that blocks that are called on top of a binding, where the results
		// for the binding are pre-loaded, are actually read from cache. Typically
		// follow-up calls would try to load from cache and would get the correct
		// value, however if there are no follow-up calls we still want to return
		// the correct value.
		// This may be optimized in a way that we don't have to check loading it
		// on every call.
		cached, ok := c.cache.Load(curRef)
		if ok {
			res = cached.Result
			nextRef = 0
			err = nil
		} else {
			res, nextRef, err = c.runRef(curRef)
		}

		// stop this chain of execution, if it didn't return anything
		// we need more data ie an event to provide info
		if res == nil && nextRef == 0 && err == nil {
			return
		}

		// if this is a result for a callback (entry- or datapoint) send it
		if res != nil {
			if codeID, ok := c.callbackPoints[curRef]; ok {
				c.callback(&RawResult{Data: res, CodeID: codeID})
			}
		} else if err != nil {
			if codeID, ok := c.callbackPoints[curRef]; ok {
				c.callback(errorResult(err, codeID))
			}
		}

		// get the next reference, if we are not directed anywhere
		if nextRef == 0 {
			// note: if the call cannot be retrieved it will use the
			// zero value, which is 0 in this case; i.e. if !ok => ref = 0
			nextRefs, _ := c.calls.Load(curRef)
			cnt := len(nextRefs)
			if cnt != 0 {
				nextRef = nextRefs[0]
				remaining = append(remaining, nextRefs[1:]...)
				continue
			}

			cnt = len(remaining)
			if cnt == 0 {
				break
			}
			nextRef = remaining[0]
			remaining = remaining[1:]
		}
	}
}

// triggerChain when a reference has a new value set
// unlike runChain this will not execute the ref chunk, but rather
// try to move to the next called chunk - or if it's not available
// handle the result
func (c *LeiseExecutor) triggerChain(ref int32, data *RawData) {
	// before we do anything else, we may have to provide the value from
	// this callback point
	if codeID, ok := c.callbackPoints[ref]; ok {
		c.callback(&RawResult{Data: data, CodeID: codeID})
	}

	nxt, ok := c.calls.Load(ref)
	if ok {
		if len(nxt) == 0 {
			panic("internal state error: cannot trigger next call on chain because it points to a zero ref")
		}
		for i := range nxt {
			c.runChain(nxt[i])
		}
		return
	}

	codeID := c.callbackPoints[ref]
	res, ok := c.cache.Load(ref)
	if !ok {
		c.callback(errorResultMsg("exec> cannot find results to chunk reference "+strconv.FormatInt(int64(ref), 10), codeID))
		return
	}

	log.Trace().Int32("ref", ref).Msgf("exec> trigger callback")
	c.callback(&RawResult{Data: res.Result, CodeID: codeID})
}

func (c *LeiseExecutor) triggerChainError(ref int32, err error) {
	cur := ref
	var remaining []int32
	for cur > 0 {
		if codeID, ok := c.callbackPoints[cur]; ok {
			c.callback(&RawResult{
				Data: &RawData{
					Error: err,
				},
				CodeID: codeID,
			})
		}

		nxt, ok := c.calls.Load(cur)
		if !ok {
			if len(remaining) == 0 {
				break
			}
			cur = remaining[0]
			remaining = remaining[1:]
		}
		if len(nxt) == 0 {
			panic("internal state error: cannot trigger next call on chain because it points to a zero ref")
		}
		cur = nxt[0]
		remaining = append(remaining, nxt[1:]...)
	}
}
