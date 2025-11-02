package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/alireza-aliabadi/least-response-time-go-lb/internal/serverPool"
	"github.com/alireza-aliabadi/least-response-time-go-lb/internal/urls"
)

var serverPool serverpool.ServerPool

func loadBalancer(w http.ResponseWriter, r *http.Request) {
	server := serverPool.GetBestServer()
	if server == nil {
		http.Error(w, "Servers aren't available", http.StatusServiceUnavailable)
		return
	}
	
	start := time.Now()
	server.ReverseProxy.ServeHTTP(w, r)
	duration := time.Since(start)

	go server.UpdateRespTime(duration)

	log.Printf("Forwarded request to %s, took %s, new avg: %s\n", server.URL, duration, server.AvgRespTime)
}

func HealthCheck(pool *serverpool.ServerPool) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		log.Println("Running health check ...")
		for _, s := range pool.Servers {
			connection, err := net.DialTimeout("tcp", s.URL.Host, 2 * time.Second)
			if err != nil {
				if s.Alive {
					log.Printf("Server %s is down\n", s.URL)
					s.SetAlive(false)
				}
				continue
			}
			connection.Close()
			if !s.Alive {
				log.Printf("Server %s is backed up\n", s.URL)
				s.SetAlive(true)
			}
		}
	}
}

func main() {
	serversUrls, err := urls.ReadUrlsFromFile("hosts")
	if err != nil {
		log.Fatalf("Failed to read servers url file: %v", err)
	}

	log.Printf("Found %d servers in file.", len(serversUrls))

	for _, url := range serversUrls {
		server, err := serverpool.NewServer(url)
		if err != nil {
			log.Printf("Error creating server for %s: %v. Skipping", url, err)
			continue
		}
		serverPool.Add(server)
		log.Printf("Added server: %s", url)
	}

	if len(serverPool.Servers) == 0 {
		log.Fatal("No valid servers were loaded. Existing")
	}

	go HealthCheck(&serverPool)

	lbPort := flag.String("port", "9000", "port that lb runs on")
	flag.Parse()

	service := http.Server{
		Addr: ":"+*lbPort,
		Handler: http.HandlerFunc(loadBalancer),
	}

	log.Printf("Load balancer started on port %s, serving %d servers", *lbPort, len(serverPool.Servers))
	if err := service.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}