package main

import (
	"fmt"
	"log"
	"os"
	"netcat/server"
)

func main() {
	port := ":8989"

	// Get port from command line argument
	if len(os.Args) == 2 {
		port = ":" + os.Args[1]
	} else if len(os.Args) > 2 {
		fmt.Println("[USAGE]: ./TCPChat $port")
		return
	}

	// Start server
	err := server.InitServer(port)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
