package store

import "sync"

type KV interface {
	Put(key, value string)
	Get(key string) (string, bool)
}

type MemKV struct {
	mu sync.RWMutex
	m map[string]string
}


func NewMemKV() *MemKV {
	return &MemKV{m : make(map[string]string)}
}


func (kv *MemKV) Put(k, v string) {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	kv.m[k] = v
}

func (kv *MemKV) Get(k string) (string, bool) {
	kv.mu.RLock()
	defer kv.mu.RUnlock()
	v, ok := kv.m[k]
	return v, ok
}