package tests

import (
	"io/ioutil"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	br "netcat/broadcast"
	"netcat/models"
)

func TestBroadcaster(t *testing.T) {
	// Setup
	models.Mu.Lock()
	models.Clients = make(map[net.Conn]string)
	models.Broadcast = make(chan string, 10)
	models.Mu.Unlock()

	// Create temporary log file
	tmpFile, err := ioutil.TempFile("", "test_broadcast_log")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()
	models.LogFile = tmpFile

	// Create test connections
	server1, client1 := net.Pipe()
	server2, client2 := net.Pipe()
	defer server1.Close()
	defer client1.Close()
	defer server2.Close()
	defer client2.Close()

	// Add clients
	models.Mu.Lock()
	models.Clients[server1] = "User1"
	models.Clients[server2] = "User2"
	models.Mu.Unlock()

	// Start broadcaster in goroutine with done channel
	broadcasterDone := make(chan bool)
	go func() {
		br.Broadcaster()
		close(broadcasterDone)
	}()

	// Send test message
	testMessage := "Test broadcast message\n"
	models.Broadcast <- testMessage

	// Use WaitGroup to ensure all checks complete
	var wg sync.WaitGroup
	wg.Add(2)

	// Check if both clients received the message
	checkClientMessage := func(client net.Conn, clientName string) {
		defer wg.Done()
		client.SetReadDeadline(time.Now().Add(2 * time.Second))
		buffer := make([]byte, 1024)
		n, err := client.Read(buffer)
		if err != nil {
			t.Errorf("Client %s failed to receive message: %v", clientName, err)
			return
		}

		if string(buffer[:n]) != testMessage {
			t.Errorf("Client %s expected %q, got %q", clientName, testMessage, string(buffer[:n]))
		}
	}

	go checkClientMessage(client1, "User1")
	go checkClientMessage(client2, "User2")

	// Wait for all client checks to complete
	wg.Wait()

	// Check if message was logged
	tmpFile.Seek(0, 0)
	logContent, err := ioutil.ReadAll(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if !strings.Contains(string(logContent), testMessage) {
		t.Errorf("Expected log to contain %q, got %q", testMessage, string(logContent))
	}

	// Clean up
	close(models.Broadcast)

	// Wait for broadcaster to finish with timeout
	select {
	case <-broadcasterDone:
		// Good, broadcaster finished
	case <-time.After(2 * time.Second):
		t.Error("Broadcaster did not finish after channel close")
	}
}

func TestBroadcasterWithFailedConnection(t *testing.T) {
	// Setup
	models.Mu.Lock()
	models.Clients = make(map[net.Conn]string)
	models.Broadcast = make(chan string, 10)
	models.Mu.Unlock()

	// Create temporary log file
	tmpFile, err := ioutil.TempFile("", "test_broadcast_log")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()
	models.LogFile = tmpFile

	// Create test connections
	server1, client1 := net.Pipe()
	server2, client2 := net.Pipe()
	defer client1.Close()
	defer client2.Close()

	// Add clients
	models.Mu.Lock()
	models.Clients[server1] = "User1"
	models.Clients[server2] = "User2"
	models.Mu.Unlock()

	// Close one server connection to simulate failure
	server1.Close()

	// Start broadcaster with done channel
	broadcasterDone := make(chan bool)
	go func() {
		br.Broadcaster()
		close(broadcasterDone)
	}()

	// Send test message
	testMessage := "Test message\n"
	models.Broadcast <- testMessage

	// Use WaitGroup for client check
	var wg sync.WaitGroup
	wg.Add(1)

	// Check that the working client received the message
	go func() {
		defer wg.Done()
		client2.SetReadDeadline(time.Now().Add(2 * time.Second))
		buffer := make([]byte, 1024)
		n, err := client2.Read(buffer)
		if err != nil {
			t.Errorf("Working client should have received message: %v", err)
		} else if string(buffer[:n]) != testMessage {
			t.Errorf("Expected %q, got %q", testMessage, string(buffer[:n]))
		}
	}()

	// Wait for client check to complete
	wg.Wait()

	// Check that failed connection was removed from clients
	models.Mu.Lock()
	if _, exists := models.Clients[server1]; exists {
		t.Error("Failed connection should have been removed from clients map")
	}
	if _, exists := models.Clients[server2]; !exists {
		t.Error("Working connection should still exist in clients map")
	}
	models.Mu.Unlock()

	// Clean up
	server2.Close()
	close(models.Broadcast)

	// Wait for broadcaster to finish with timeout
	select {
	case <-broadcasterDone:
		// Good, broadcaster finished
	case <-time.After(2 * time.Second):
		t.Error("Broadcaster did not finish after channel close")
	}
}
