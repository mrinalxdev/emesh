package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strconv"
	"strings"
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

	node, err := mesh.New(*selfID, reg, db)
	if err != nil {
		log.Fatal(err)
	}

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
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("type 'put key value' or 'get key' (commands are broadcast causally)")
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			fmt.Println("usage: put key value  OR  get key")
			continue
		}
		op := parts[0]
		switch op {
		case "put":
			if len(parts) < 3 {
				fmt.Println("usage: put key value")
				continue
			}
			key, val := parts[1], parts[2]
			if err := n.BroadcastPut(key, val); err != nil {
				log.Println("broadcast:", err)
			}
		case "get":
			key := parts[1]
			if val, ok := n.Store().Get(key); !ok {
				fmt.Println("key not found")
			} else {
				fmt.Println(key, "=", val)
			}
		default:
			fmt.Println("unknown command")
		}
	}
}