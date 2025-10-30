// Package serverpool provides utilities for managing a pool of backend servers.
// It is designed for concurrent use and tracks server health and performance
// metrics to facilitate load balancing, such as selecting the server
// with the lowest average response time.
package serverpool

import (
	"net/http/httputil"
	"net/url"
	"sync"
	"time"
)

// alpha is the smoothing factor for the EWMA calculation.
const alpha = 0.25

type Server struct {
	mu sync.RWMutex
	URL *url.URL
	ReverseProxy *httputil.ReverseProxy
	Alive bool
	AvgRespTime time.Duration
}

type ServerPool struct {
	mu sync.RWMutex
	Servers []*Server
}

// ServerPool methods
func (sp *ServerPool) Add(server *Server) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.Servers = append(sp.Servers, server)
}

func (sp *ServerPool) GetBestServer() *Server {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	var bestServer *Server
	var minResponseTime = time.Duration(1<<63 - 1)

	for _, s := range sp.Servers {
		s.mu.RLock()
		if s.Alive && s.AvgRespTime < minResponseTime {
			minResponseTime = s.AvgRespTime
			bestServer = s
		}
		s.mu.RUnlock()
	}
	return bestServer
}

// Server method
func (s *Server) UpdateRespTime(sample time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.AvgRespTime == 0 {
		s.AvgRespTime = sample
	} else {
		// using EWMA formula
		newAvg := alpha*float64(sample) + (1-alpha)*float64(s.AvgRespTime)
		s.AvgRespTime = time.Duration(newAvg)
	}
}

func (s *Server) SetAlive(alive bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Alive = alive
}

// NewServer creates and initializes a new Server instance from a raw URL string.
// It also creates a httputil.ReverseProxy for the server.
func NewServer(rawUrl string) (*Server, error) {
	url, err := url.Parse(rawUrl)
	if err != nil {
		return  nil, err
	}
	proxy := httputil.NewSingleHostReverseProxy(url)

	return &Server{
		URL: url,
		ReverseProxy: proxy,
		Alive: true,
		AvgRespTime: 0,
	}, nil
}