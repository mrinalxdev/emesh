package mesh

import (
	"bytes"
	"log"
	"strings"
	"sync"
	"time"

	"eventmesh/pkg/protocol"
	"eventmesh/pkg/store"

	"github.com/prometheus/client_golang/prometheus"
)


type deliverer struct {
	self string
	vc   protocol.Vector
	mu   sync.Mutex
	q    []entry
	db   store.KV
	
	//{comment}
	//adding prometheus metrics for the histogram and
	//the counter which would counter state variable for
	recvBytes prometheus.Counter
	reorderDepth prometheus.Histogram
	deliverLag prometheus.Histogram
}

type entry struct {
	hdr     protocol.Header
	payload []byte
	
	//{comment}
	//setting it monotonic
	recvT time.Time
}

var (
	
	reorderDepth = prometheus.NewHistogram(prometheus.HistogramOpts{Name : "emesh_reorder_depth"})
	
	deliverLag = prometheus.NewHistogram(prometheus.HistogramOpts{ Name : "emesh_deliver_lag_ms"})
)

// func newDeliverer(self string) *deliverer {
// 	// return &deliverer{
// 	// 	self: self,
// 	// 	vc:   make(protocol.Vector),
// 	// 	db:   store.NewMemKV(), // local copy
// 	// }
// 	// 
// 	// 
// 	// 
	
// 	d := &deliverer{
// 		self : self,
// 		vc : make(protocol.Vector),
// 		db: store.NewMemKV(),
// 		recvBytes: recvBytes,
// 		reorderDepth: reorderDepth,
// 		deliverLag: deliverLag,
// 	}
	
// 	prometheus.MustRegister(recvBytes, reorderDepth, deliverLag)
	
// 	return d
// }

func newDeliverer(self string) *deliverer {
	recvBytes := prometheus.NewCounter(prometheus.CounterOpts{
		Name : "emesh_recv_bytes_total",
		Help : "Total bytes received",
		ConstLabels : prometheus.Labels{"node" : self},
	})
	
	
	reorderDepth := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name : "emesh_deliver_lag_ms",
		Help : "Time messages spend in reorder buffer (ms)",
		ConstLabels: prometheus.Labels{"node" : self},
		Buckets : []float64{0, 1, 2, 5, 10, 20, 50},
	})
	
	
	deliverLag := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name : "emesh_deliver_lag_ms",
		Help : "Time messages spend in reorder buffer (ms)",
		ConstLabels: prometheus.Labels{"node" : self},
		Buckets: prometheus.DefBuckets,
	})
	
	prometheus.MustRegister(recvBytes, reorderDepth, deliverLag)
	
	
	return &deliverer{
		self: self,
		vc: make(protocol.Vector),
		db : store.NewMemKV(),
		recvBytes: recvBytes,
		reorderDepth: reorderDepth,
		deliverLag: deliverLag,
	}
}

func (d *deliverer) copyVC() protocol.Vector {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.vc.Copy()
}

func (d *deliverer) submit(hdr protocol.Header, payload []byte) {
	// d.mu.Lock()
	// defer d.mu.Unlock()
	// // d.q = append(d.q, entry{hdr, payload, time.Now()})
	// d.q = append(d.q, entry{hdr, payload, time.Now()})
	// d.tryDeliver()
	// 
	d.mu.Lock()
	defer d.mu.Unlock()
	
	estimatedHeaderBytes := 24 + 24*int(hdr.ClockLen)
	totalBytes := float64(estimatedHeaderBytes + len(payload))
	
	d.recvBytes.Add(totalBytes)
}

func (d *deliverer) tryDeliver() {
	for {
		progress := false
		for i := 0; i < len(d.q); i++ {
			e := d.q[i]
			
			depth := len(d.q) - i - 1
			if d.ready(e.hdr.Clock) {
				d.apply(e.hdr, e.payload)
				
				d.reorderDepth.Observe(float64(depth))
				d.deliverLag.Observe(float64(time.Since(e.recvT).Milliseconds()))
				
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