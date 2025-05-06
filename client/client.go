package client

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"netcat/models"
	"netcat/utils"
)

func HandleClient(conn net.Conn, fileName string) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	logo, _ := os.ReadFile("logo.txt")

	conn.Write([]byte(string(logo) + "\n"))
	conn.Write([]byte("[ENTER YOUR NAME]: "))
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)

	if name == "" {
		conn.Write([]byte("Invalid name. Connection closed.\n"))
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	models.Mu.Lock()
	models.Clients[conn] = name
	models.Mu.Unlock()

	utils.SendChatHistory(conn, fileName)

	joinMsg := fmt.Sprintf("%s has joined our chat...\n", name)
	utils.NotifyClients(conn, joinMsg)

	nameTag := "[" + name + "]"

	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		msg = strings.TrimSpace(msg)
		if msg == "" {
			continue
		}

		if msg == "/exit" {
			break
		} else if strings.HasPrefix(msg, "/change ") {
			newName := strings.TrimPrefix(msg, "/change ")

			if newName == "" {
				conn.Write([]byte("Invalid name. Usage: /change <new_name>\n"))
				continue
			}

			models.Mu.Lock()
			oldName := models.Clients[conn]
			models.Clients[conn] = newName
			models.Mu.Unlock()

			utils.NotifyClients(conn, fmt.Sprintf("%s has changed their name to %s\n", oldName, newName))
			nameTag = "[" + newName + "]"
		}

		models.Broadcast <- fmt.Sprintf("[%s]%s: %s\n", timestamp, nameTag, msg)
	}

	models.Mu.Lock()
	delete(models.Clients, conn)
	models.Mu.Unlock()

	leaveMsg := fmt.Sprintf("%s has left our chat.\n", name)
	utils.NotifyClients(conn, leaveMsg)
}
