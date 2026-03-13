package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"
)

var algorithm string

type Server interface {
	Address() string
	Serve(w http.ResponseWriter, r *http.Request)
	getNoOfConnections() int
	changeNoOfConnections(n int)
}

type simpleServer struct {
	Addr            string
	NoOfConnections int
	Proxy           *httputil.ReverseProxy
}

func newServer(addr string) Server {
	serverUrl, err := url.Parse(addr)
	handleErr(err)
	if algorithm == "lc" {
		return &LCserver{
			Addr:            addr,
			NoOfConnections: 0,
			Proxy:           httputil.NewSingleHostReverseProxy(serverUrl)}
	} else {
		return &simpleServer{
			Addr:            addr,
			NoOfConnections: 0,
			Proxy:           httputil.NewSingleHostReverseProxy(serverUrl),
		}
	}
}

func handleErr(err error) {
	if err != nil {
		fmt.Printf("Error is: %v\n", err)
		os.Exit(1)
	}
}

func (s *simpleServer) getNoOfConnections() int {
	return s.NoOfConnections
}

func (s *simpleServer) changeNoOfConnections(n int) {
	s.NoOfConnections = s.NoOfConnections + n
}

type LoadBalancer struct {
	roundRobinCount int
	servers         []Server
	mu              sync.Mutex
}

func newLoadBalancer(servers []Server) *LoadBalancer {
	return &LoadBalancer{
		roundRobinCount: 0,
		servers:         servers,
	}
}

func (s *simpleServer) Address() string {
	return s.Addr
}

func (s *simpleServer) Serve(w http.ResponseWriter, r *http.Request) {
	s.Proxy.ServeHTTP(w, r)
}

func (lb *LoadBalancer) getNextAvaliableServer() Server {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	server := lb.servers[lb.roundRobinCount%len(lb.servers)]
	lb.roundRobinCount++
	return server
}

func (lb *LoadBalancer) serverProxy(w http.ResponseWriter, r *http.Request) {
	target := lb.getNextAvaliableServer()
	fmt.Printf("foward requests to address: %q\n", target.Address())
	target.Serve(w, r)
}

func main() {
	algorithm = os.Getenv("LB_ALGORITHM")
	severs := []Server{
		newServer("http://server1:3030"),
		newServer("http://server2:3030"),
		newServer("http://server3:3030"),
	}
	port := "8000"
	var handler func(http.ResponseWriter, *http.Request)

	if algorithm == "lc" {
		lb := newLCLB(severs)
		handler = lb.serverProxy
	} else {
		lb := newLoadBalancer(severs)
		handler = lb.serverProxy
	}
	http.HandleFunc("/", handler)
	fmt.Println("Started Serving at")
	http.ListenAndServe(":"+port, nil)
}
