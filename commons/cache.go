package commons

import (
	"fmt"
	"reflect"
	"sync"
)

type CopyCache struct {
	data map[string]interface{}
	sync.Mutex
}

func NewCache() *CopyCache {
	return &CopyCache{data: make(map[string]interface{})}
}

func (c *CopyCache) Invalidate(key string) {
	c.Lock()
	defer c.Unlock()
	delete(c.data, key)
}

func (c *CopyCache) Add(key string, val interface{}) {
	c.Lock()
	defer c.Unlock()
	c.data[key] = val
	//// Make sure that we make a copy
	//obj := reflect.ValueOf(val)
	//cp := reflect.New(obj.Type())
	//cp.Elem().Set(obj)
	//c.data[key] = cp.Elem().Interface()
}

func (c *CopyCache) Get(key string) (interface{}, bool) {
	val, ok := c.data[key]
	return val, ok
}

// GetInto() gets a cached value at the specified target address.
func (c *CopyCache) GetInto(key string, target interface{}) (bool, error) {
	// Retrieve a cached value, if there is one
	val, ok := c.Get(key)
	if !ok {
		return false, nil
	}
	// Verify that the target address is in fact a pointer
	t := reflect.ValueOf(target)
	if t.Kind() != reflect.Ptr {
		return false, fmt.Errorf("GetInto() requires a pointer")
	}
	// Make sure that the value we retrieve from the cache is the object itself
	v := reflect.ValueOf(val)
	if v.Kind() == reflect.Interface || v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	// Write a copy of the cached value to the address passed in
	t.Elem().Set(v)
	return true, nil
}