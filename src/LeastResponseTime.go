package main

import (
	"math/rand"
	"net/http"
	"net/http/httputil"
	"sync"
	"time"
)

type LRTserver struct {
	Addr            string
	NoOfConnections int
	Proxy           *httputil.ReverseProxy
	AverageTime     time.Duration
	NoOfRequests    int
}

type LeastResponseTimeLoadBalancer struct {
	servers []Server
	mu      sync.Mutex
}

func newLRTLB(servers []Server) *LeastResponseTimeLoadBalancer {
	return &LeastResponseTimeLoadBalancer{
		servers: servers,
	}
}

func (s *LRTserver) Address() string {
	return s.Addr
}

func (s *LRTserver) getAverageTime() time.Duration {
	return s.AverageTime
}

func (s *LRTserver) updateAverageTime(elapsed time.Duration) {
	if s.NoOfRequests == 0 {
		s.AverageTime = elapsed
		return
	}
	total := s.AverageTime*time.Duration(s.NoOfRequests) + elapsed
	s.AverageTime = total / time.Duration(s.NoOfRequests+1)
}

func (s *LRTserver) changeNoOfConnections(n int) {
	s.NoOfConnections = s.NoOfConnections + n
}

func (s *LRTserver) addNoOfRequests() {
	s.NoOfRequests++
}

func (s *LRTserver) Serve(w http.ResponseWriter, r *http.Request) {
	s.Proxy.ServeHTTP(w, r)
}

func (lb *LeastResponseTimeLoadBalancer) getNextAvaliableServer() Server {
	var targerServer Server
	targetServers := make([]Server, 0)
	targerServer = lb.servers[0]
	for _, server := range lb.servers {
		if targerServer.getAverageTime() > server.getAverageTime() {
			targerServer = server
		}
	}
	for _, ser := range lb.servers {
		if ser.getAverageTime() == targerServer.getAverageTime() {
			targetServers = append(targetServers, ser)
		}
	}
	if len(targetServers) > 1 {
		randomInt := rand.Intn(len(targetServers))
		return targetServers[randomInt]
	}
	return targerServer
}

func (lb *LeastResponseTimeLoadBalancer) serverProxy(w http.ResponseWriter, r *http.Request) {
	target := lb.getNextAvaliableServer()
	start := time.Now()
	target.Serve(w, r)
	elapsed := time.Since(start)
	lb.mu.Lock()
	target.updateAverageTime(elapsed)
	target.addNoOfRequests()
	lb.mu.Unlock()

}
