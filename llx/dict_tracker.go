package llx

import "sync"

// DictTracker for tracking simple keys in a map
type DictTracker struct {
	mu   sync.Mutex
	data map[string]struct{}
}

// NewDictTracker initializes a new dict tracker
func NewDictTracker() *DictTracker {
	return &DictTracker{
		data: map[string]struct{}{},
	}
}

// Set the key in the tracker
func (c *DictTracker) Set(k string) {
	c.mu.Lock()
	c.data[k] = struct{}{}
	c.mu.Unlock()
}

// Exists checks if key exists in the map
func (c *DictTracker) Exists(k string) bool {
	c.mu.Lock()
	_, ok := c.data[k]
	c.mu.Unlock()
	return ok
}

// CheckOrSet a key, returns true if the key already exists
func (c *DictTracker) CheckOrSet(k string) bool {
	c.mu.Lock()
	_, ok := c.data[k]
	if !ok {
		c.data[k] = struct{}{}
	}
	c.mu.Unlock()
	return ok
}

// Unset a key in the tracker
func (c *DictTracker) Unset(k string) {
	c.mu.Lock()
	delete(c.data, k)
	c.mu.Unlock()
}

// GetAndClear retrieves all keys and clears out the dict
func (c *DictTracker) GetAndClear() []string {
	c.mu.Lock()
	res := make([]string, len(c.data))
	var i int
	for k := range c.data {
		res[i] = k
		i++
	}
	c.data = map[string]struct{}{}
	c.mu.Unlock()

	return res
}

type DictGroupTracker struct {
	mutex sync.Mutex
	data  map[string]map[string]struct{}
}

func (g *DictGroupTracker) Clear() {
	g.mutex.Lock()
	g.data = map[string]map[string]struct{}{}
	g.mutex.Unlock()
}

func (g *DictGroupTracker) ClearGroup(name string) {
	g.mutex.Lock()
	g.data[name] = map[string]struct{}{}
	g.mutex.Unlock()
}

// CheckOrSet a key, returns true if the key already exists
func (g *DictGroupTracker) CheckOrSet(group string, key string) bool {
	g.mutex.Lock()
	grp, ok := g.data[group]
	if !ok {
		grp = map[string]struct{}{}
		g.data[group] = grp
	}

	_, ok = grp[key]
	if !ok {
		grp[key] = struct{}{}
	}
	g.mutex.Unlock()

	return ok
}
