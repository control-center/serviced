package cache

import (
	"fmt"
	"sync"
)

type InstanceCache struct {
	data map[string]CurrentState
	mu   *sync.RWMutex
}

type CurrentState string

const (
	StatePulling    CurrentState = "pulling"
	StateStarting   CurrentState = "starting"
	StateRunning    CurrentState = "running"
	StateStopping   CurrentState = "stopping"
	StateStopped    CurrentState = "stopped"
	StateResuming   CurrentState = "resuming"
	StatePausing    CurrentState = "pausing"
	StatePaused     CurrentState = "paused"
	StateRestarting CurrentState = "restarting" // XXX: Needs to be implemented
)

func NewInstanceCache() *InstanceCache {
	cache := &InstanceCache{
		mu:   &sync.RWMutex{},
		data: make(map[string]CurrentState),
	}
	return cache
}

func (cache *InstanceCache) GetAll() map[string]CurrentState {
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	return cache.data
}

func (cache *InstanceCache) SetAll(states map[string]CurrentStates) {
	cache.data.set(key, value)
	return
}

// Retrieve an item from the cache
func (cache *InstanceCache) Get(serviceID string, instanceID int) (CurrentState, bool) {
	key := fmt.Sprintf("%s_%d", serviceID, instanceID)
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	item, ok := cache.data.get(key)
	return item, ok
}

// get() is not thread safe
func (cache *InstanceCache) get(key string) (item string, ok bool) {
	item, ok = cache.data[key]
	return
}

// SetKey Set an item into the cache
func (cache *InstanceCache) SetKey(key string, value string) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.set(key, value)
}
func (cache *InstanceCache) Set(serviceID string, instanceID int, value string) {
	key := fmt.Sprintf("%s_%d", serviceID, instanceID)
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.set(key, value)
}

// set() is not thread safe
func (cache *InstanceCache) set(key string, value string) {
	cache.data[key] = value
}
