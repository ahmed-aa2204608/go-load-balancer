package main

import (
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
)

type LCserver struct {
	addr            string
	noOfConnections int
	proxy           *httputil.ReverseProxy
}

func newLCServer(addr string) *LCserver {
	u, err := url.Parse(addr)
	handleErr(err)
	return &LCserver{addr: addr, proxy: httputil.NewSingleHostReverseProxy(u)}
}

type LeastConnectionsLoadBalancer struct {
	servers []*LCserver
	mu      sync.Mutex
}

func newLCLB(servers []*LCserver) *LeastConnectionsLoadBalancer {
	return &LeastConnectionsLoadBalancer{servers: servers}
}

func (lb *LeastConnectionsLoadBalancer) getNextAvailableServer() *LCserver {
	min := lb.servers[0]
	for _, s := range lb.servers[1:] {
		if s.noOfConnections < min.noOfConnections {
			min = s
		}
	}
	var tied []*LCserver
	for _, s := range lb.servers {
		if s.noOfConnections == min.noOfConnections {
			tied = append(tied, s)
		}
	}
	return tied[rand.Intn(len(tied))]
}

func (lb *LeastConnectionsLoadBalancer) serverProxy(w http.ResponseWriter, r *http.Request) {
	lb.mu.Lock()
	target := lb.getNextAvailableServer()
	target.noOfConnections++
	lb.mu.Unlock()

	target.proxy.ServeHTTP(w, r)

	lb.mu.Lock()
	target.noOfConnections--
	lb.mu.Unlock()
}
