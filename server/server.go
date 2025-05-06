package server

import (
	"fmt"
	"log"
	"net"
	"os"

	"netcat/broadcast"
	"netcat/client"
	"netcat/models"
)

// StartServer initializes the TCP chat server
func InitServer(port string) error {
	ln, err := net.Listen("tcp", port)
	if err != nil {
		return err
	}
	defer ln.Close()

	fmt.Println("Listening on the port " + port)

	portnum := port[1:]
	logfileName := fmt.Sprintf("logs/chat_log_%s.log", portnum)

	models.LogFile, err = os.OpenFile(logfileName, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer models.LogFile.Close()

	go broadcast.Broadcaster()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}

		if len(models.Clients) >= 10 {
			conn.Write([]byte("Chatroom full...\n"))
			conn.Close()
			continue
		}

		go client.HandleClient(conn, logfileName)
	}
}
