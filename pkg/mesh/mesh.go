package mesh

import (
	"bytes"
	"log"
	"net"
	"strings"
	"sync"

	"eventmesh/pkg/protocol"
	"eventmesh/pkg/store"
	"eventmesh/pkg/transport"
)

type Node struct {
	id      string
	reg     *transport.Registry
	store   store.KV
	dlv     *deliverer
	peerAdd map[string]net.Addr //id
	mu      sync.RWMutex
}

func New(id string, reg *transport.Registry, store store.KV) (*Node, error) {
	n := &Node{
		id:      id,
		reg:     reg,
		store:   store,
		peerAdd: make(map[string]net.Addr),
		dlv:     newDeliverer(id),
	}
	// start receive loop
	t, _ := reg.Get(id)
	go n.recvLoop(t)
	return n, nil
}

func (n *Node) DialPeers(list string) error {
	for _, p := range strings.Split(list, ",") {
		if p == "" {
			continue
		}
		addr, err := net.ResolveUDPAddr("udp", p)
		if err != nil {
			return err
		}
		id := p
		n.mu.Lock()
		n.peerAdd[id] = addr
		n.mu.Unlock()
	}
	return nil
}

func (n *Node) BroadcastPut(key, value string) error {
	n.mu.RLock()
	vc := n.dlv.copyVC()
	n.mu.RUnlock()

	vc[n.id]++
	hdr := protocol.Header{}
	copy(hdr.From[:], n.id)
	hdr.Clock = vc
	hdr.ClockLen = uint8(len(vc))

	payload := []byte("PUT:" + key + ":" + value)

	n.mu.RLock()
	defer n.mu.RUnlock()
	for id, addr := range n.peerAdd {
		t, ok := n.reg.Get(id)
		if !ok {
			continue 
		}
		if err := t.Send(addr, hdr, payload); err != nil {
			log.Printf("send to %s: %v", id, err)
		}
	}

	n.dlv.submit(hdr, payload)
	return nil
}

func (n *Node) recvLoop(tr transport.Transport) {
	for {
		hdr, payload, from, err := tr.Receive()
		if err != nil {
			log.Printf("recv: %v", err)
			return
		}

		id := string(bytes.TrimRight(hdr.From[:], "\x00"))
		n.mu.Lock()
		n.peerAdd[id] = from
		n.mu.Unlock()

		n.dlv.submit(hdr, payload)
	}
}

func (n *Node) Store() store.KV { return n.store }

func (n *Node) Close() error {
	t, _ := n.reg.Get(n.id)
	return t.Close()
}