package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

// simpleServer represents a simple HTTP server.
type simpleServer struct {
	addr  string                 // Address of the server
	proxy *httputil.ReverseProxy // Reverse proxy for the server
}

// newSimpleServer creates a new instance of simpleServer.
func newSimpleServer(addr string) *simpleServer {
	serverURL, err := url.Parse(addr)
	handleErr(err)

	return &simpleServer{
		addr:  addr,
		proxy: httputil.NewSingleHostReverseProxy(serverURL),
	}
}

// Server defines the methods a server should implement.
type Server interface {
	Address() string                               // Address method returns the server's address
	IsAlive() bool                                 // IsAlive method checks if the server is alive
	Serve(rw http.ResponseWriter, r *http.Request) // Serve method serves HTTP requests
}

// Address returns the server's address.
func (s *simpleServer) Address() string { return s.addr }

// IsAlive checks if the server is alive.
func (s *simpleServer) IsAlive() bool { return true }

// Serve serves HTTP requests using the reverse proxy.
func (s *simpleServer) Serve(rw http.ResponseWriter, r *http.Request) {
	s.proxy.ServeHTTP(rw, r)
}

// LoadBalancer represents a simple round-robin load balancer.
type LoadBalancer struct {
	port            string   // Port on which the load balancer listens
	roundRobinCount int      // Counter for round-robin load balancing
	servers         []Server // Slice of servers to balance load between
}

// NewLoadBalancer creates a new instance of LoadBalancer.
func NewLoadBalancer(port string, servers []Server) *LoadBalancer {
	return &LoadBalancer{
		roundRobinCount: 0,
		port:            port,
		servers:         servers,
	}
}

// handleErr handles errors.
func handleErr(err error) {
	if err != nil {
		fmt.Printf("error : %v\n", err)
		os.Exit(1)
	}
}

// getNextAvailableServer returns the next available server in the round-robin fashion.
func (lb *LoadBalancer) getNextAvailableServer() Server {
	server := lb.servers[lb.roundRobinCount%len(lb.servers)]
	for !server.IsAlive() {
		lb.roundRobinCount++
		server = lb.servers[lb.roundRobinCount%len(lb.servers)]
	}
	lb.roundRobinCount++
	return server
}

// serveProxy forwards HTTP requests to the next available server.
func (lb *LoadBalancer) serveProxy(rw http.ResponseWriter, r *http.Request) {
	targetServer := lb.getNextAvailableServer()
	fmt.Printf("Forwarding request to address %q\n", targetServer.Address())
	targetServer.Serve(rw, r)
}

func main() {
	// Define a list of servers to load balance between
	servers := []Server{
		newSimpleServer("https://www.facebook.com"),
		newSimpleServer("https://www.google.com"),
		newSimpleServer("https://www.duckduckgo.com"),
	}
	// Create a new instance of LoadBalancer
	lb := NewLoadBalancer("8080", servers)

	// Define a handler function to handle requests and forward them to appropriate servers
	handleRedirect := func(rw http.ResponseWriter, r *http.Request) {
		lb.serveProxy(rw, r)
	}

	// Register the handler function for the root URL path "/"
	http.HandleFunc("/", handleRedirect)

	// Start the HTTP server
	fmt.Printf("Serving requests at 'localhost:%v'\n", lb.port)
	http.ListenAndServe(":"+lb.port, nil)
}
