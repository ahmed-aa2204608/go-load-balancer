package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"
)

type Server interface {
	Address() string
	isAlive() bool
	Serve(w http.ResponseWriter, r *http.Request)
}

type simpleServer struct {
	addr  string
	proxy *httputil.ReverseProxy
}

func newSimpleServer(addr string) *simpleServer {
	serverUrl, err := url.Parse(addr)
	handleErr(err)

	return &simpleServer{
		addr:  addr,
		proxy: httputil.NewSingleHostReverseProxy(serverUrl),
	}
}

func handleErr(err error) {
	if err != nil {
		fmt.Printf("Error is: %v\n", err)
		os.Exit(1)
	}
}

type LoadBalancer struct {
	port            string
	roundRobinCount int
	servers         []Server
	mu              sync.Mutex
}

func newLoadBalancer(port string, servers []Server) *LoadBalancer {
	return &LoadBalancer{
		port:            port,
		roundRobinCount: 0,
		servers:         servers,
	}
}

func (s *simpleServer) isAlive() bool {
	return true
}

func (s *simpleServer) Address() string {
	return s.addr
}

func (s *simpleServer) Serve(w http.ResponseWriter, r *http.Request) {
	s.proxy.ServeHTTP(w, r)
}

func (lb *LoadBalancer) getNextAvaliableServer() Server {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	server := lb.servers[lb.roundRobinCount%len(lb.servers)]
	for !server.isAlive() {
		lb.roundRobinCount++
		server = lb.servers[lb.roundRobinCount%len(lb.servers)]
	}
	lb.roundRobinCount++
	return server
}

func (lb *LoadBalancer) serverProxy(w http.ResponseWriter, r *http.Request) {
	target := lb.getNextAvaliableServer()
	fmt.Printf("foward requests to address: %q\n", target.Address())
	target.Serve(w, r)
}

func main() {
	severs := []Server{
		newSimpleServer("http://server1:3030"),
		newSimpleServer("http://server2:3030"),
		newSimpleServer("http://server3:3030"),
	}

	lb := newLoadBalancer("8000", severs)
	handleRedirect := func(w http.ResponseWriter, r *http.Request) {
		lb.serverProxy(w, r)
	}
	http.HandleFunc("/", handleRedirect)
	fmt.Println("Started Serving at")
	http.ListenAndServe(":"+lb.port, nil)
}
