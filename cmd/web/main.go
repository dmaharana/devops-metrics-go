package main

import (
	"flag"
	"log"
	"devops-metrics/web"
)

func main() {
	// Parse command line flags
	var port string
	flag.StringVar(&port, "port", "8080", "Port to run the server on")
	flag.Parse()

	// Create and start the server
	server := web.NewServer()
	server.Start(port)
}