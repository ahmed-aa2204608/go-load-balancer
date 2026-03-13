package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"
)

type simpleServer struct {
	addr  string
	proxy *httputil.ReverseProxy
}

func newSimpleServer(addr string) *simpleServer {
	u, err := url.Parse(addr)
	handleErr(err)
	return &simpleServer{addr: addr, proxy: httputil.NewSingleHostReverseProxy(u)}
}

func (s *simpleServer) address() string                             { return s.addr }
func (s *simpleServer) serve(w http.ResponseWriter, r *http.Request) { s.proxy.ServeHTTP(w, r) }

func handleErr(err error) {
	if err != nil {
		fmt.Printf("Error is: %v\n", err)
		os.Exit(1)
	}
}

type LoadBalancer struct {
	roundRobinCount int
	servers         []*simpleServer
	mu              sync.Mutex
}

func newLoadBalancer(servers []*simpleServer) *LoadBalancer {
	return &LoadBalancer{servers: servers}
}

func (lb *LoadBalancer) getNextAvailableServer() *simpleServer {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	s := lb.servers[lb.roundRobinCount%len(lb.servers)]
	lb.roundRobinCount++
	return s
}

func (lb *LoadBalancer) serverProxy(w http.ResponseWriter, r *http.Request) {
	target := lb.getNextAvailableServer()
	fmt.Printf("forwarding request to: %q\n", target.address())
	target.serve(w, r)
}

func main() {
	algorithm := os.Getenv("LB_ALGORITHM")
	port := "8000"
	addrs := []string{
		"http://server1:3030",
		"http://server2:3030",
		"http://server3:3030",
	}

	var handler func(http.ResponseWriter, *http.Request)

	switch algorithm {
	case "lc":
		servers := make([]*LCserver, len(addrs))
		for i, addr := range addrs {
			servers[i] = newLCServer(addr)
		}
		lb := newLCLB(servers)
		handler = lb.serverProxy
	case "lrt":
		servers := make([]*LRTserver, len(addrs))
		for i, addr := range addrs {
			servers[i] = newLRTServer(addr)
		}
		lb := newLRTLB(servers)
		handler = lb.serverProxy
	default:
		servers := make([]*simpleServer, len(addrs))
		for i, addr := range addrs {
			servers[i] = newSimpleServer(addr)
		}
		lb := newLoadBalancer(servers)
		handler = lb.serverProxy
	}

	http.HandleFunc("/", handler)
	fmt.Println("Started serving at :" + port)
	http.ListenAndServe(":"+port, nil)
}
