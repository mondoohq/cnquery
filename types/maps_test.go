package types

import (
	"testing"

	uuid "github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
)

const max = 1000

func RunStringSet(t *testing.T, s *StringSet) {
	id := uuid.Must(uuid.NewV4()).String()
	assert.False(t, s.Exist(id), "key that wasn't yet added doesn't exist")
	s.Store(id)
	assert.True(t, s.Exist(id), "key that was added exists")
	s.Delete(id)
	assert.False(t, s.Exist(id), "key that was deleted doesn't exist")
}

func TestMaps_StringSet(t *testing.T) {
	t.Run("all functions", func(t *testing.T) {
		s := StringSet{}
		id := uuid.Must(uuid.NewV4()).String()
		assert.False(t, s.Exist(id), "key that wasn't yet added doesn't exist")
		s.Store(id)
		assert.True(t, s.Exist(id), "key that was added exists")
		assert.Equal(t, []string{id}, s.List())
		s.Delete(id)
		assert.False(t, s.Exist(id), "key that was deleted doesn't exist")
		assert.Equal(t, []string{}, s.List())
	})

	t.Run("concurrent runs", func(t *testing.T) {
		s := StringSet{}
		for i := 0; i < max; i++ {
			go RunStringSet(t, &s)
		}
	})
}

func RunStringToStrings(t *testing.T, s *StringToStrings) {
	a := uuid.Must(uuid.NewV4()).String()
	b := uuid.Must(uuid.NewV4()).String()
	assert.False(t, s.Exist(a, b), "key that wasn't yet added doesn't exist")
	s.Store(a, b)
	assert.True(t, s.Exist(a, b), "key that was added exists")
	s.Delete(a, b)
	assert.False(t, s.Exist(a, b), "key that was deleted doesn't exist")
}

func TestMaps_StringToStrings(t *testing.T) {
	t.Run("all functions", func(t *testing.T) {
		s := StringToStrings{}
		a := uuid.Must(uuid.NewV4()).String()
		b := uuid.Must(uuid.NewV4()).String()
		assert.False(t, s.Exist(a, b), "key that wasn't yet added doesn't exist")
		s.Store(a, b)
		assert.True(t, s.Exist(a, b), "key that was added exists")
		assert.Equal(t, map[string][]string{
			a: []string{b},
		}, s.List())
		s.Delete(a, b)
		assert.False(t, s.Exist(a, b), "key that was deleted doesn't exist")
		assert.Equal(t, map[string][]string{}, s.List())
	})

	t.Run("concurrent runs", func(t *testing.T) {
		s := StringToStrings{}
		for i := 0; i < max; i++ {
			go RunStringToStrings(t, &s)
		}
	})
}
