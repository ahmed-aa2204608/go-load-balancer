package main

import (
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
)

type LCserver struct {
	Addr            string
	NoOfConnections int
	Proxy           *httputil.ReverseProxy
}

type LeastConnectionsLoadBalancer struct {
	port    string
	servers []Server
	mu      sync.Mutex
}

func createNewLCServer(Addr string) *LCserver {
	serverUrl, err := url.Parse(Addr)
	handleErr(err)
	return &LCserver{
		Addr:            Addr,
		NoOfConnections: 0,
		Proxy:           httputil.NewSingleHostReverseProxy(serverUrl),
	}
}

func newLCLB(Addr string, servers []Server) *LeastConnectionsLoadBalancer {
	return &LeastConnectionsLoadBalancer{
		port:    Addr,
		servers: servers,
	}
}

func (s *LCserver) Address() string {
	return s.Addr
}

func (s *LCserver) getNoOfConnections() int {
	return s.NoOfConnections
}

func (s *LCserver) changeNoOfConnections(n int) {
	s.NoOfConnections = s.NoOfConnections + n
}

func (s *LCserver) Serve(w http.ResponseWriter, r *http.Request) {
	s.Proxy.ServeHTTP(w, r)
}

func (lb *LeastConnectionsLoadBalancer) getNextAvaliableServer() Server {
	var targerServer Server
	targetServers := make([]Server, 0)
	targerServer = lb.servers[0]
	for _, server := range lb.servers {
		if targerServer.getNoOfConnections() > server.getNoOfConnections() {
			targerServer = server
		}
	}
	for _, ser := range lb.servers {
		if ser.getNoOfConnections() == targerServer.getNoOfConnections() {
			targetServers = append(targetServers, ser)
		}
	}
	if len(targetServers) > 1 {
		randomInt := rand.Intn(len(targetServers)) + 1
		return targetServers[randomInt]
	}
	return targerServer
}

func (lb *LeastConnectionsLoadBalancer) serverProxy(w http.ResponseWriter, r *http.Request) {
	target := lb.getNextAvaliableServer()
	lb.mu.Lock()
	defer lb.mu.Unlock()
	target.changeNoOfConnections(1)
	defer func() { target.changeNoOfConnections(-1) }()
	target.Serve(w, r)
}
