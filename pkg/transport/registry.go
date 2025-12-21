package transport

import (
	"net"
	"sync"

	"eventmesh/pkg/protocol"
)


type Registry struct {
	mu    sync.RWMutex
	trans map[string]Transport
}

type Transport interface {
	LocalAddr() net.Addr
	Send(net.Addr, protocol.Header, []byte) error
	Receive() (protocol.Header, []byte, net.Addr, error)
	Close() error
}

func NewRegistry() *Registry {
	return &Registry{trans: make(map[string]Transport)}
}

func (r *Registry) Register(id string, t Transport) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.trans[id] = t
}

func (r *Registry) Get(id string) (Transport, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.trans[id]
	return t, ok
}