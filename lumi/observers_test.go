package lumi

import (
	"testing"

	uuid "github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/types"
)

func TestCallbacksList(t *testing.T) {
	o := CallbacksList{}
	a := uuid.Must(uuid.NewV4()).String()
	b := uuid.Must(uuid.NewV4()).String()
	cb := func() {}

	assert.Equal(t, map[string][]string{}, o.List())

	o.Store(a, b, cb)
	assert.Equal(t, map[string][]string{a: []string{b}}, o.List())

	last := o.Delete(a, b)
	assert.True(t, last)
	assert.Equal(t, map[string][]string{}, o.List())
}

func TestObservers(t *testing.T) {
	o := Observers{
		hooks:       &Hooks{},
		reverseList: &types.StringToStrings{},
	}
	a := uuid.Must(uuid.NewV4()).String()
	b := uuid.Must(uuid.NewV4()).String()

	_, err := o.Unwatch(a, b)
	assert.Nil(t, err, "no error on Unwatched id")

	err = o.UnwatchAll(a)
	assert.Nil(t, err, "no error on UnwatchedAll id")

	initial, exist, err := o.Watch(a, b, func() {})
	assert.True(t, initial, "initial watcher")
	assert.False(t, exist, "existing watcher")
	assert.Nil(t, err, "no error on Watch")

	assert.Equal(t, map[string][]string{
		a: []string{b}}, o.list.List())

	assert.Equal(t, map[string][]string{
		b: []string{a}}, o.reverseList.List())

	err = o.UnwatchAll(b)
	assert.Nil(t, err, "no error on UnwatchedAll id")

	assert.Equal(t, map[string][]string{}, o.list.List())
	assert.Equal(t, map[string][]string{}, o.reverseList.List())
}
