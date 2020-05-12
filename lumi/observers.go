package lumi

import (
	"errors"
	"sync"

	"github.com/rs/zerolog/log"
	motor "go.mondoo.io/mondoo/motor/motoros"
	"go.mondoo.io/mondoo/types"
)

// Callback is a function without arguments
type Callback func()

// Callbacks is a map of callbacks
type Callbacks struct{ sync.Map }

// Store a callback
func (c *Callbacks) Store(key string, callback Callback) {
	c.Map.Store(key, callback)
}

// Load a callback by ID
func (c *Callbacks) Load(key string) (Callback, bool) {
	res, ok := c.Map.Load(key)
	if !ok {
		return nil, ok
	}
	return res.(Callback), ok
}

// List all keys with callbacks
func (c *Callbacks) List() []string {
	res := []string{}
	c.Range(func(k string, _ Callback) bool {
		res = append(res, k)
		return true
	})
	return res
}

// Delete a callback
func (c *Callbacks) Delete(key string) {
	c.Map.Delete(key)
}

// Range cycles over all callbacks
func (c *Callbacks) Range(cb func(string, Callback) bool) {
	c.Map.Range(func(k, v interface{}) bool {
		return cb(k.(string), v.(Callback))
	})
}

// CallbacksList a list of all callback maps
type CallbacksList struct{ sync.Map }

// List all key-value lists of ID associations
func (c *CallbacksList) List() map[string][]string {
	res := make(map[string][]string)
	c.Map.Range(func(k, v interface{}) bool {
		res[k.(string)] = v.(*Callbacks).List()
		return true
	})
	return res
}

// Store a key-value combination and return if it is the initial for this key
// and if this value already exists.
// 1. true = this is the first time this key is stored
// 2. true = this key-value combination already existed
func (c *CallbacksList) Store(key, value string, cb Callback) (bool, bool) {
	v, ok := c.Map.Load(key)
	var callbacks *Callbacks
	if !ok {
		callbacks = &Callbacks{}
		c.Map.Store(key, callbacks)
	} else {
		callbacks = v.(*Callbacks)
	}

	_, exists := callbacks.Load(value)
	callbacks.Store(value, cb)

	return !ok, exists
}

// Delete a key-value callback
// Return true if the key is now empty
func (c *CallbacksList) Delete(key, value string) bool {
	callbacks, ok := c.Load(key)
	if !ok {
		return true
	}

	callbacks.Delete(value)
	isEmpty := true
	callbacks.Range(func(_ string, _ Callback) bool {
		isEmpty = false
		return false
	})
	if isEmpty {
		c.Map.Delete(key)
	}

	return isEmpty
}

// Load a key
func (c *CallbacksList) Load(key string) (*Callbacks, bool) {
	v, ok := c.Map.Load(key)
	if !ok {
		return nil, false
	}
	return v.(*Callbacks), true
}

// Hooks is a map of func
type Hooks struct{ sync.Map }

func (c *Hooks) Store(k string, v func()) {
	c.Map.Store(k, v)
}

func (c *Hooks) Load(k string) (func(), bool) {
	res, ok := c.Map.Load(k)
	if !ok {
		return nil, ok
	}
	return res.(func()), ok
}

// Observers manages all the observers
type Observers struct {
	list        CallbacksList
	reverseList types.StringToStrings
	hooks       Hooks
	motor       *motor.Motor
}

// NewObservers creates an observers instance
func NewObservers(motor *motor.Motor) *Observers {
	return &Observers{
		motor: motor,
	}
}

// List out all observers
func (ctx *Observers) List() (map[string][]string, map[string][]string) {
	return ctx.list.List(), ctx.reverseList.List()
}

// Watch a UID for any changes to it and call the watcher via callback if anything changes
// we return a boolean to indicate if this is the first watcher the resource field receives
// true => first time this resource is watched
// false => not the first watcher
func (ctx *Observers) Watch(resourceFieldUID string, watcherUID string, callback Callback) (bool, bool, error) {
	if watcherUID == "" {
		return false, false, errors.New("Cannot register observer with empty watcher UID")
	}

	initial, exists := ctx.list.Store(resourceFieldUID, watcherUID, callback)
	ctx.reverseList.Store(watcherUID, resourceFieldUID)

	return initial, exists, nil
}

// Unwatch a watcher from a resource field
// returns a boolean if the ist of watcher is now empty
// true => this was the last watcher, no-one is watching this resource field
// false => there are still watchers on this resource field
func (ctx *Observers) Unwatch(resourceFieldUID string, watcherUID string) (bool, error) {
	if watcherUID == "" {
		return false, errors.New("Cannot unwatch observer with empty watcher UID")
	}

	listIsEmpty := ctx.list.Delete(resourceFieldUID, watcherUID)
	ctx.reverseList.Delete(watcherUID, resourceFieldUID)

	return listIsEmpty, nil
}

// OnUnwatch will trigger the given handler once the watcher is removed
func (ctx *Observers) OnUnwatch(watcherUID string, f func()) {
	ctx.hooks.Store("unwatch\x00"+watcherUID, f)
}

// UnwatchAll the references that this watcher is looking at. If it finds
// any resourceFieldUID that isn't watched by anyone anymore it will recursively
// do the same to this watcher.
func (ctx *Observers) UnwatchAll(watcherUID string) error {
	log.Debug().Str("watcher", watcherUID).Msg("observer> unwatch all")

	h, ok := ctx.hooks.Load("unwatch\x00" + watcherUID)
	if ok {
		h()
	}

	cbs := ctx.reverseList.ListKey(watcherUID)
	for _, key := range cbs {
		last, err := ctx.Unwatch(key, watcherUID)
		if err != nil {
			return err
		}
		if last {
			err = ctx.UnwatchAll(key)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Trigger a change to an observed ID. This means that all watchers on this
// field will be called
func (ctx *Observers) Trigger(resourceFieldUID string) error {
	f, ok := ctx.list.Load(resourceFieldUID)
	if !ok {
		return errors.New("Cannot find field " + resourceFieldUID + " to trigger its change.")
	}
	f.Range(func(_ string, cb Callback) bool {
		cb()
		return true
	})
	return nil
}
