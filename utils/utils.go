package utils

import (
	"bufio"
	"log"
	"net"
	"os"

	"netcat/models"
)

// LogToFile writes messages to the chat log file
func LogToFile(msg string) {
	if models.LogFile == nil {
		return
	}

	_, err := models.LogFile.WriteString(msg)
	if err != nil {
		log.Printf("Error writing to log file: %v", err)
	}
}

// SendChatHistory sends chat history to a newly connected client
func SendChatHistory(conn net.Conn, fileName string) {
	file, err := os.Open(fileName)
	if err != nil {
		conn.Write([]byte("[No chat history available]\n"))
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		conn.Write([]byte(scanner.Text() + "\n"))
	}
}

// NotifyClients sends a message to all clients except the excluded one
func NotifyClients(excludeConn net.Conn, message string) {
	models.Mu.Lock()
	defer models.Mu.Unlock()

	for conn := range models.Clients {
		if conn != excludeConn {
			conn.Write([]byte(message))
		}
	}
}
