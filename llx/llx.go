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
	Ref    uint64
}

type stepCache struct {
	Result   *RawData
	IsStatic bool
}

// Calls is a map connecting call-refs with each other
type Calls struct {
	locker sync.Mutex
	calls  map[uint64][]uint64
}

// Store a new call connection.
// Returns true if this connection already exists.
// Returns false if this is a new connection.
func (c *Calls) Store(k uint64, v uint64) bool {
	c.locker.Lock()
	defer c.locker.Unlock()

	calls, ok := c.calls[k]
	if !ok {
		calls = []uint64{}
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
func (c *Calls) Load(k uint64) ([]uint64, bool) {
	c.locker.Lock()
	v, ok := c.calls[k]
	c.locker.Unlock()
	return v, ok
}

// Cache is a map containing stepCache values
type Cache struct{ sync.Map }

// Store a new call connection
func (c *Cache) Store(k uint64, v *stepCache) { c.Map.Store(k, v) }

// Load a call connection
func (c *Cache) Load(k uint64) (*stepCache, bool) {
	res, ok := c.Map.Load(k)
	if res == nil {
		return nil, ok
	}
	return res.(*stepCache), ok
}

type blockExecutor struct {
	id             string
	blockRef       uint64
	entrypoints    map[uint64]struct{}
	callback       ResultCallback
	callbackPoints map[uint64]string
	cache          *Cache
	stepTracker    *Cache
	calls          *Calls
	block          *Block
	parent         *blockExecutor
	ctx            *LeiseExecutorV2
	watcherIds     *types.StringSet
}

// LeiseExecutor is the runtime of a leise/llx codestructure
type LeiseExecutorV2 struct {
	id             string
	blockExecutors []*blockExecutor
	runtime        *lumi.Runtime
	code           *CodeV2
	starts         []uint64
	props          map[string]*Primitive
}

func (c *blockExecutor) watcherUID(ref uint64) string {
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
func NewExecutorV2(code *CodeV2, runtime *lumi.Runtime, props map[string]*Primitive, callback ResultCallback) (*LeiseExecutorV2, error) {
	if runtime == nil {
		return nil, errors.New("cannot exec leise without a runtime")
	}

	if code == nil {
		return nil, errors.New("cannot run executor without code")
	}

	res := &LeiseExecutorV2{
		id:             uuid.Must(uuid.NewV4()).String(),
		runtime:        runtime,
		code:           code,
		props:          props,
		blockExecutors: []*blockExecutor{},
	}

	exec, err := res._newBlockExecutor(1<<32, callback, nil)
	if err != nil {
		return nil, err
	}

	res.blockExecutors = append(res.blockExecutors, exec)

	return res, nil
}

func (c *LeiseExecutorV2) _newBlockExecutor(blockRef uint64, callback ResultCallback, parent *blockExecutor) (*blockExecutor, error) {
	block := c.code.Block(blockRef)

	if block == nil {
		return nil, errors.New("cannot find block " + strconv.FormatUint(blockRef, 10))
	}

	callbackPoints := map[uint64]string{}

	res := &blockExecutor{
		id:             uuid.Must(uuid.NewV4()).String() + "/" + strconv.FormatUint(blockRef>>32, 10),
		blockRef:       blockRef,
		callback:       callback,
		callbackPoints: callbackPoints,
		cache:          &Cache{},
		stepTracker:    &Cache{},
		calls: &Calls{
			locker: sync.Mutex{},
			calls:  map[uint64][]uint64{},
		},
		block:       block,
		ctx:         c,
		parent:      parent,
		watcherIds:  &types.StringSet{},
		entrypoints: map[uint64]struct{}{},
	}

	for _, ref := range block.Entrypoints {
		id := c.code.Checksums[ref]
		if id == "" {
			return nil, errors.New("llx.executor> cannot execute with invalid ref ID in entrypoint")
		}
		if ref < 1 {
			return nil, errors.New("llx.executor> cannot execute with invalid ref number in entrypoint")
		}
		res.entrypoints[ref] = struct{}{}
		res.callbackPoints[ref] = id
	}

	for _, ref := range block.Datapoints {
		id := c.code.Checksums[ref]
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

// NoRun returns error for all callbacks and don't run code
func (c *LeiseExecutorV2) NoRun(err error) {
	callback := c.blockExecutors[0].callback

	for ref := range c.blockExecutors[0].callbackPoints {
		if codeID, ok := c.blockExecutors[0].callbackPoints[ref]; ok {
			callback(errorResult(err, codeID))
		}
	}
}

func (c *LeiseExecutorV2) Unregister() error {
	log.Trace().Str("id", c.id).Msg("exec> unregister")

	var errs []error
	for i := range c.blockExecutors {
		be := c.blockExecutors[i]
		errs = append(errs, be.unregister()...)
	}

	if len(errs) > 0 {
		return errors.New("multiple errors unregistering")
	}

	return nil
}

// Run a given set of code
func (c *LeiseExecutorV2) Run() error {
	if len(c.blockExecutors) == 0 {
		return errors.New("cannot find initial block executor for running this code")
	}

	core := c.blockExecutors[0]
	core.run()
	return nil
}

func (b *blockExecutor) newBlockExecutor(blockRef uint64, callback ResultCallback) (*blockExecutor, error) {
	return b.ctx._newBlockExecutor(blockRef, callback, b)
}

func (e *blockExecutor) unregister() []error {
	var errs []error

	e.watcherIds.Range(func(key string) bool {
		if err := e.ctx.runtime.Unregister(key); err != nil {
			log.Error().Err(err).Msg("exec> unregister error")
			errs = append(errs, err)
		}
		return true
	})

	return errs
}

func (b *blockExecutor) isInMyBlock(ref uint64) bool {
	return (ref >> 32) == (b.blockRef >> 32)
}

func (b *blockExecutor) mustLookup(ref uint64) *RawData {
	d, _, err := b.parent.lookupValue(ref)
	if err != nil {
		panic(err)
	}
	if d == nil {
		panic("did not lookup datapoint")
	}
	return d
}

// run code with a runtime and return results
func (b *blockExecutor) run() {
	for ref, codeID := range b.callbackPoints {
		if !b.isInMyBlock(ref) {
			v := b.mustLookup(ref)
			b.callback(&RawResult{
				CodeID: codeID,
				Data:   v,
			})
		}
	}
	// work down all entrypoints
	refs := make([]uint64, len(b.block.Entrypoints)+len(b.block.Datapoints))
	i := 0
	for _, ref := range b.block.Entrypoints {
		refs[i] = ref
		i++
	}
	for _, ref := range b.block.Datapoints {
		refs[i] = ref
		i++
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i] > refs[j] })

	for _, ref := range refs {
		// if this entrypoint is already connected, don't add it again
		if _, ok := b.stepTracker.Load(ref); ok {
			continue
		}

		log.Trace().Uint64("entrypoint", ref).Str("exec-ID", b.ctx.id).Msg("exec.Run>")
		b.runChain(ref)
	}
}

func (b *blockExecutor) ensureArgsResolved(args []*Primitive, ref uint64) (uint64, error) {
	for _, arg := range args {
		_, dref, err := b.resolveValue(arg, ref)
		if dref != 0 || err != nil {
			return dref, err
		}
	}
	return 0, nil
}

func (b *blockExecutor) runFunctionBlock(args []*RawData, blockRef uint64, cb ResultCallback) error {
	executor, err := b.newBlockExecutor(blockRef, cb)
	if err != nil {
		return err
	}

	b.ctx.blockExecutors = append(b.ctx.blockExecutors, executor)

	if len(args) < int(executor.block.Parameters) {
		panic("not enough arguments")
	}

	for i := int32(0); i < executor.block.Parameters; i++ {
		executor.cache.Store(blockRef|uint64(i+1), &stepCache{
			Result:   args[i],
			IsStatic: true,
		})
	}

	executor.run()
	return nil
}

func (b *blockExecutor) runBlock(bind *RawData, functionRef *Primitive, args []*Primitive, ref uint64) (*RawData, uint64, error) {

	if bind != nil && bind.Value == nil && bind.Type != types.Nil {
		return &RawData{Type: bind.Type, Value: nil}, 0, nil
	}

	typ := types.Type(functionRef.Type)
	if !typ.IsFunction() {
		return nil, 0, errors.New("called block with wrong function type")
	}
	fref, ok := functionRef.RefV2()
	if !ok {
		return nil, 0, errors.New("cannot retrieve function reference on block call")
	}

	block := b.ctx.code.Block(fref)
	if block == nil {
		return nil, 0, errors.New("block function is nil")
	}

	blockResult := map[string]interface{}{}

	fargs := []*RawData{}
	if bind != nil {
		fargs = append(fargs, bind)
	}
	for i := range args {
		a, b, c := b.resolveValue(args[i], ref)
		if c != nil || b != 0 {
			return a, b, c
		}
		fargs = append(fargs, a)
	}

	var anyError error
	err := b.runFunctionBlock(fargs, fref, func(res *RawResult) {
		if block.SingleValue {
			b.cache.Store(ref, &stepCache{
				Result: res.Data,
			})
			b.triggerChain(ref, res.Data)
			return
		}

		if _, exists := blockResult[res.CodeID]; !exists && res.Data.Error != nil {
			anyError = multierror.Append(anyError, res.Data.Error)
		}
		blockResult[res.CodeID] = res.Data
		expectedCnt := len(block.Entrypoints) + len(block.Datapoints)
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
			b.cache.Store(ref, &stepCache{
				Result:   data,
				IsStatic: true,
			})
			b.triggerChain(ref, data)
		}
	})

	return nil, 0, err
}

func (b *blockExecutor) createResource(name string, f *Function, ref uint64) (*RawData, uint64, error) {
	args, rref, err := args2resourceargsV2(b, ref, f.Args)
	if err != nil || rref != 0 {
		return nil, rref, err
	}

	resource, err := b.ctx.runtime.CreateResource(name, args...)
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
			b.cache.Store(ref, &res)
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
	b.cache.Store(ref, &res)
	return res.Result, 0, nil
}

func (b *blockExecutor) runGlobalFunction(chunk *Chunk, f *Function, ref uint64) (*RawData, uint64, error) {
	h, ok := handleGlobalV2(chunk.Id)
	if ok {
		if h == nil {
			return nil, 0, errors.New("found function " + chunk.Id + " but no handler. this should not happen and points to an implementation error")
		}

		res, dref, err := h(b, f, ref)
		log.Trace().Msgf("exec> global: %s %+v = %#v", chunk.Id, f.Args, res)
		if res != nil {
			b.cache.Store(ref, &stepCache{Result: res})
		}
		return res, dref, err
	}

	return b.createResource(chunk.Id, f, ref)
}

// connect references, calling `dst` if `src` is updated
func (b *blockExecutor) connectRef(src uint64, dst uint64) (*RawData, uint64, error) {
	if !b.isInMyBlock(src) || !b.isInMyBlock(dst) {
		panic("cannot connect refs across block boundaries")
	}
	// connect the ref. If it is already connected, someone else already made this
	// call, so we don't have to follow up anymore
	if exists := b.calls.Store(src, dst); exists {
		return nil, 0, nil
	}

	// if the ref was not yet connected, we must run the src ref after we connected it
	return nil, src, nil
}

func (e *blockExecutor) runFunction(chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	f := chunk.Function
	if f == nil {
		f = &emptyFunction
	}

	// global functions, for now only resources
	if f.Binding == 0 {
		return e.runGlobalFunction(chunk, f, ref)
	}

	// check if the bound value exists, otherwise connect it
	res, dref, err := e.resolveRef(f.Binding, ref)
	if res == nil {
		return res, dref, err
	}

	if res.Error != nil {
		e.cache.Store(ref, &stepCache{Result: res})
		return nil, 0, res.Error
	}

	return e.runBoundFunction(res, chunk, ref)
}

func (e *blockExecutor) runChunk(chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	switch chunk.Call {
	case Chunk_PRIMITIVE:
		res, dref, err := e.resolveValue(chunk.Primitive, ref)
		if res != nil {
			e.cache.Store(ref, &stepCache{Result: res})
		} else if err != nil {
			e.cache.Store(ref, &stepCache{Result: &RawData{
				Error: err,
			}})
		}

		return res, dref, err
	case Chunk_FUNCTION:
		return e.runFunction(chunk, ref)

	case Chunk_PROPERTY:
		property, ok := e.ctx.props[chunk.Id]
		if !ok {
			return nil, 0, errors.New("cannot find property '" + chunk.Id + "'")
		}

		res, dref, err := e.resolveValue(property, ref)
		if dref != 0 || err != nil {
			return res, dref, err
		}
		e.cache.Store(ref, &stepCache{Result: res})
		return res, dref, err

	default:
		return nil, 0, errors.New("Tried to run a chunk which has an unknown type: " + chunk.Call.String())
	}
}

func (e *blockExecutor) runRef(ref uint64) (*RawData, uint64, error) {
	chunk := e.ctx.code.Chunk(ref)
	if chunk == nil {
		return nil, 0, errors.New("Called a chunk that doesn't exist, ref = " + strconv.FormatInt(int64(ref), 10))
	}
	return e.runChunk(chunk, ref)
}

// runChain starting at a ref of the code, follow it down and report
// jever result it has at the end of its execution. this will register
// async callbacks against referenced chunks too
func (e *blockExecutor) runChain(start uint64) {
	var res *RawData
	var err error
	nextRef := start
	var curRef uint64
	var remaining []uint64

	for nextRef != 0 {
		curRef = nextRef
		e.stepTracker.Store(curRef, nil)
		// log.Trace().Uint64("ref", curRef).Msg("exec> run chain")

		// Try to load the result from cache if it already exists. This was added
		// so that blocks that are called on top of a binding, where the results
		// for the binding are pre-loaded, are actually read from cache. Typically
		// follow-up calls would try to load from cache and would get the correct
		// value, however if there are no follow-up calls we still want to return
		// the correct value.
		// This may be optimized in a way that we don't have to check loading it
		// on every call.
		cached, ok := e.cache.Load(curRef)
		if ok {
			res = cached.Result
			nextRef = 0
			err = nil
		} else {
			res, nextRef, err = e.runRef(curRef)
		}

		// stop this chain of execution, if it didn't return anything
		// we need more data ie an event to provide info
		if res == nil && nextRef == 0 && err == nil {
			return
		}

		// if this is a result for a callback (entry- or datapoint) send it
		if res != nil {
			if codeID, ok := e.callbackPoints[curRef]; ok {
				e.callback(&RawResult{Data: res, CodeID: codeID})
			}
		} else if err != nil {
			if codeID, ok := e.callbackPoints[curRef]; ok {
				e.callback(errorResult(err, codeID))
			}
			if _, isNotReadyError := err.(lumi.NotReadyError); !isNotReadyError {
				if sc, _ := e.cache.Load(curRef); sc == nil {
					e.cache.Store(curRef, &stepCache{
						Result: &RawData{
							Type:  types.Unset,
							Value: nil,
							Error: err,
						},
					})
				}
			}
		}

		// get the next reference, if we are not directed anywhere
		if nextRef == 0 {
			// note: if the call cannot be retrieved it will use the
			// zero value, which is 0 in this case; i.e. if !ok => ref = 0
			nextRefs, _ := e.calls.Load(curRef)
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
func (e *blockExecutor) triggerChain(ref uint64, data *RawData) {
	// before we do anything else, we may have to provide the value from
	// this callback point
	if codeID, ok := e.callbackPoints[ref]; ok {
		e.callback(&RawResult{Data: data, CodeID: codeID})
	}

	nxt, ok := e.calls.Load(ref)
	if ok {
		if len(nxt) == 0 {
			panic("internal state error: cannot trigger next call on chain because it points to a zero ref")
		}
		for i := range nxt {
			e.runChain(nxt[i])
		}
		return
	}

	codeID := e.callbackPoints[ref]
	res, ok := e.cache.Load(ref)
	if !ok {
		e.callback(errorResultMsg("exec> cannot find results to chunk reference "+strconv.FormatInt(int64(ref), 10), codeID))
		return
	}

	log.Trace().Uint64("ref", ref).Msgf("exec> trigger callback")
	e.callback(&RawResult{Data: res.Result, CodeID: codeID})
}

func (e *blockExecutor) triggerChainError(ref uint64, err error) {
	cur := ref
	var remaining []uint64
	for cur > 0 {
		if codeID, ok := e.callbackPoints[cur]; ok {
			e.callback(&RawResult{
				Data: &RawData{
					Error: err,
				},
				CodeID: codeID,
			})
		}

		nxt, ok := e.calls.Load(cur)
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
