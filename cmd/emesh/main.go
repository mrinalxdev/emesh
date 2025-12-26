package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"eventmesh/pkg/mesh"
	"eventmesh/pkg/store"
	"eventmesh/pkg/transport"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	var (
		selfID   = flag.String("id", "", "unique node id (required)")
		bindAddr = flag.String("bind", ":9000", "UDP listen address")
		peers    = flag.String("peers", "", "comma-separated peer addrs, e.g. 127.0.0.1:9001,127.0.0.1:9002")
	)
	flag.Parse()
	if *selfID == "" {
		flag.Usage()
		os.Exit(1)
	}

	udp, err := transport.NewUDP(*bindAddr)
	if err != nil {
		log.Fatal(err)
	}
	reg := transport.NewRegistry()
	reg.Register(*selfID, udp)
	db := store.NewMemKV()

	// 3. causal broadcast layer
	node, err := mesh.New(*selfID, reg, db)
	if err != nil {
		log.Fatal(err)
	}

	// 4. connect to peers
	if *peers != "" {
		if err := node.DialPeers(*peers); err != nil {
			log.Fatal(err)
		}
	}

	go repl(node)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	fmt.Println("shutting down…")
	node.Close()
	
	
	go func() {
		port := 6060
		if portStr := os.Getenv("METRICS_PORT"); portStr != "" {
			if i, err := strconv.Atoi(portStr); err == nil {
				port = i
			}
		}
		
		addr := fmt.Sprintf("0.0.0.0%d", port)
		http.Handle("/metrics", promhttp.Handler())
		log.Printf("metrics exposed on http://%s/metrics", addr)
		log.Fatal(http.ListenAndServe(addr, nil))
	}()
}

func repl(n *mesh.Node) {
	fmt.Println("type  'put key value'  or  'get key'  (commands are broadcast causally)")
	for {
		var op, k, v string
		if _, err := fmt.Scanln(&op, &k, &v); err != nil {
			continue
		}
		switch op {
		case "put":
			if err := n.BroadcastPut(k, v); err != nil {
				log.Println("broadcast:", err)
			}
		case "get":
			val, ok := n.Store().Get(k)
			if !ok {
				fmt.Println("key not found")
			} else {
				fmt.Println(k, "=", val)   // ← this line was missing
			}
		default:
			fmt.Println("unknown command")
		}
	}
}