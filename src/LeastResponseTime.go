package main

import (
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"
)

type LRTserver struct {
	addr         string
	proxy        *httputil.ReverseProxy
	averageTime  time.Duration
	noOfRequests int
}

func newLRTServer(addr string) *LRTserver {
	u, err := url.Parse(addr)
	handleErr(err)
	return &LRTserver{addr: addr, proxy: httputil.NewSingleHostReverseProxy(u)}
}

type LeastResponseTimeLoadBalancer struct {
	servers []*LRTserver
	mu      sync.Mutex
}

func newLRTLB(servers []*LRTserver) *LeastResponseTimeLoadBalancer {
	return &LeastResponseTimeLoadBalancer{servers: servers}
}

func (lb *LeastResponseTimeLoadBalancer) getNextAvailableServer() *LRTserver {
	min := lb.servers[0]
	for _, s := range lb.servers[1:] {
		if s.averageTime < min.averageTime {
			min = s
		}
	}
	var tied []*LRTserver
	for _, s := range lb.servers {
		if s.averageTime == min.averageTime {
			tied = append(tied, s)
		}
	}
	return tied[rand.Intn(len(tied))]
}

func (lb *LeastResponseTimeLoadBalancer) serverProxy(w http.ResponseWriter, r *http.Request) {
	lb.mu.Lock()
	target := lb.getNextAvailableServer()
	lb.mu.Unlock()

	start := time.Now()
	target.proxy.ServeHTTP(w, r)
	elapsed := time.Since(start)

	lb.mu.Lock()
	total := target.averageTime*time.Duration(target.noOfRequests) + elapsed
	target.noOfRequests++
	target.averageTime = total / time.Duration(target.noOfRequests)
	lb.mu.Unlock()
}
