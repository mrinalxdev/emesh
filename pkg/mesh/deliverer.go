package mesh

import (
	"bytes"
	"log"
	"strings"
	"sync"

	"eventmesh/pkg/protocol"
	"eventmesh/pkg/store"
)


type deliverer struct {
	self string
	vc   protocol.Vector
	mu   sync.Mutex
	q    []entry
	db   store.KV
}

type entry struct {
	hdr     protocol.Header
	payload []byte
}

func newDeliverer(self string) *deliverer {
	return &deliverer{
		self: self,
		vc:   make(protocol.Vector),
		db:   store.NewMemKV(), // local copy
	}
}

func (d *deliverer) copyVC() protocol.Vector {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.vc.Copy()
}

func (d *deliverer) submit(hdr protocol.Header, payload []byte) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.q = append(d.q, entry{hdr, payload})
	d.tryDeliver()
}

func (d *deliverer) tryDeliver() {
	for {
		progress := false
		for i := 0; i < len(d.q); i++ {
			e := d.q[i]
			if d.ready(e.hdr.Clock) {
				d.apply(e.hdr, e.payload)
				// remove
				d.q = append(d.q[:i], d.q[i+1:]...)
				i--
				progress = true
			}
		}
		if !progress {
			break
		}
	}
}

func (d *deliverer) ready(clk protocol.Vector) bool {
	for id, ts := range clk {
		if id == d.self {
			continue
		}
		if ts > d.vc[id]+1 {
			return false
		}
	}
	return true
}

func (d *deliverer) apply(hdr protocol.Header, payload []byte) {
	from := string(bytes.TrimRight(hdr.From[:], "\x00"))
	//update own clock
	d.vc[from]++
	for id, ts := range hdr.Clock {
		if ts > d.vc[id] {
			d.vc[id] = ts
		}
	}
	//parse PUT
	parts := strings.SplitN(string(payload), ":", 3)
	if len(parts) == 3 && parts[0] == "PUT" {
		d.db.Put(parts[1], parts[2])
		log.Printf("delivered %s=%s (vc=%v)", parts[1], parts[2], d.vc)
	}
}