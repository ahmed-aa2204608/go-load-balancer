package main

import (
	"math/rand"
	"net/http"
	"net/http/httputil"
	"sync"
)

type LCserver struct {
	Addr            string
	NoOfConnections int
	Proxy           *httputil.ReverseProxy
}

type LeastConnectionsLoadBalancer struct {
	servers []Server
	mu      sync.Mutex
}

func newLCLB(servers []Server) *LeastConnectionsLoadBalancer {
	return &LeastConnectionsLoadBalancer{
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
		randomInt := rand.Intn(len(targetServers))
		return targetServers[randomInt]
	}
	return targerServer
}

func (lb *LeastConnectionsLoadBalancer) serverProxy(w http.ResponseWriter, r *http.Request) {
	lb.mu.Lock()
	target := lb.getNextAvaliableServer()
	target.changeNoOfConnections(1)
	lb.mu.Unlock()
	target.Serve(w, r)
	lb.mu.Lock()
	target.changeNoOfConnections(-1)
	lb.mu.Unlock()

}
