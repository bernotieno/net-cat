package tests

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	cl "netcat/client"
	"netcat/models"
	"os"
	"strings"
	"testing"
	"time"
)

const testUserName = "TestUser"

func TestHandleClientBasicFlow(t *testing.T) {
	// Setup
	models.Mu.Lock()
	models.Clients = make(map[net.Conn]string)
	models.Broadcast = make(chan string, 10)
	models.Mu.Unlock()

	// Create test logo file
	logoFile, err := ioutil.TempFile("", "logo.txt")
	if err != nil {
		t.Fatalf("Failed to create logo file: %v", err)
	}
	defer os.Remove(logoFile.Name())
	logoContent := "Welcome to Chat Server!"
	logoFile.WriteString(logoContent)
	logoFile.Close()

	// HandleClient is expected to read "logo.txt" from the current working directory.
	const logoFileName = "logo.txt"
	err = ioutil.WriteFile(logoFileName, []byte(logoContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create %s: %v", logoFileName, err)
	}
	defer os.Remove(logoFileName)

	// Create chat history file
	historyFile, err := ioutil.TempFile("", "history.txt")
	if err != nil {
		t.Fatalf("Failed to create history file: %v", err)
	}
	defer os.Remove(historyFile.Name())
	historyContent := "Previous message 1\nPrevious message 2\n"
	historyFile.WriteString(historyContent)
	historyFile.Close()

	// Create connection pair
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	// Start client handler in goroutine
	done := make(chan bool)
	go func() {
		cl.HandleClient(server, historyFile.Name())
		done <- true
	}()

	// Client side interactions
	clientReader := bufio.NewReader(client)

	// Set initial read deadline
	client.SetReadDeadline(time.Now().Add(2 * time.Second))

	// Read logo
	logoResponse, err := clientReader.ReadString('\n')
	if err != nil {
		t.Fatalf("Failed to read logo: %v. Got: %s", err, logoResponse)
	}
	if !strings.Contains(logoResponse, logoContent) {
		t.Errorf("Expected logo in response, got: %q, expected to contain: %q", logoResponse, logoContent)
	}

	// Read name prompt
	namePrompt, err := clientReader.ReadString(':')
	if err != nil {
		t.Fatalf("Failed to read name prompt: %v. Got: %s", err, namePrompt)
	}
	if !strings.Contains(namePrompt, "[ENTER YOUR NAME]:") {
		t.Errorf("Expected name prompt to contain '[ENTER YOUR NAME]:', got: %q", namePrompt)
	}

	// Reset deadline before write
	client.SetReadDeadline(time.Time{})

	// Send name
	_, err = client.Write([]byte(testUserName + "\n"))
	if err != nil {
		t.Fatalf("Failed to write username: %v", err)
	}

	// Read chat history and join message with timeout
	var initialServerOutput strings.Builder
	readBuffer := make([]byte, 256)
	expectedPromptSuffix := fmt.Sprintf("[%s]: ", testUserName)

	client.SetReadDeadline(time.Now().Add(2 * time.Second))
	for {
		n, readErr := client.Read(readBuffer)
		if n > 0 {
			initialServerOutput.WriteString(string(readBuffer[:n]))
		}
		if readErr != nil {
			if readErr == io.EOF {
				t.Fatalf("Server closed connection unexpectedly while reading initial messages")
			}
			if netErr, ok := readErr.(net.Error); ok && netErr.Timeout() {
				break // Break on timeout, we might have enough data
			}
			t.Fatalf("Error reading from server: %v", readErr)
		}
		if strings.HasSuffix(initialServerOutput.String(), expectedPromptSuffix) {
			break
		}
		if initialServerOutput.Len() > 4096 {
			t.Fatalf("Read too much data without finding prompt")
		}
	}
	client.SetReadDeadline(time.Time{})

	responseAfterLogin := initialServerOutput.String()
	if !strings.Contains(responseAfterLogin, "Previous message 1") || !strings.Contains(responseAfterLogin, "Previous message 2") {
		t.Errorf("Expected chat history in response, got: %q", responseAfterLogin)
	}

	// Send a message
	clientMessage := "Hello everyone!\n"
	_, err = client.Write([]byte(clientMessage))
	if err != nil {
		t.Fatalf("Failed to write client message: %v", err)
	}

	// Check if message was broadcasted with timeout
	select {
	case broadcastMsg := <-models.Broadcast:
		if !strings.Contains(broadcastMsg, testUserName) || !strings.Contains(broadcastMsg, strings.TrimSpace(clientMessage)) {
			t.Errorf("Expected broadcast message to contain user %q and message %q, got: %q", testUserName, strings.TrimSpace(clientMessage), broadcastMsg)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for message to be broadcasted")
	}

	// Verify client was added to map
	models.Mu.Lock()
	if name, exists := models.Clients[server]; !exists || name != testUserName {
		t.Errorf("Expected client to be in map with name %q, got: name=%q, exists=%v", testUserName, name, exists)
	}
	models.Mu.Unlock()

	// Send quit command to clean up
	_, err = client.Write([]byte("/quit\n"))
	if err != nil {
		t.Logf("Failed to send quit command during cleanup: %v", err)
	}

	// Wait for handler to finish with timeout
	select {
	case <-done:
		// Good, handler finished
	case <-time.After(2 * time.Second):
		t.Log("Warning: Handler did not finish after quit command during cleanup")
	}
}

func TestHandleClientEmptyName(t *testing.T) {
	// Create logo file
	logoContent := "Welcome!"
	err := ioutil.WriteFile("logo.txt", []byte(logoContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create logo.txt: %v", err)
	}
	defer os.Remove("logo.txt")

	server, client := net.Pipe()
	defer client.Close()

	go cl.HandleClient(server, "nonexistent.txt")

	reader := bufio.NewReader(client)

	// Read logo and prompt
	reader.ReadString('\n')
	reader.ReadString(']')

	// Send empty name
	client.Write([]byte("\n"))

	// Should receive invalid name message
	response, _ := reader.ReadString('\n')
	if !strings.Contains(response, "Invalid name") {
		t.Errorf("Expected invalid name message, got: %s", response)
	}

	// Connection should be closed by server
	// We can test this by trying to write and expecting an error
	time.Sleep(100 * time.Millisecond)
	_, err = client.Write([]byte("test"))
	if err == nil {
		t.Error("Expected connection to be closed after invalid name")
	}
}

func TestHandleClientRename(t *testing.T) {
	// Setup
	models.Mu.Lock()
	models.Clients = make(map[net.Conn]string)
	models.Broadcast = make(chan string, 10)
	models.Mu.Unlock()

	// Create logo file
	logoContent := "Welcome!"
	err := ioutil.WriteFile("logo.txt", []byte(logoContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create logo.txt: %v", err)
	}
	defer os.Remove("logo.txt")

	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	go cl.HandleClient(server, "nonexistent.txt")

	reader := bufio.NewReader(client)

	// Read logo and prompt
	reader.ReadString('\n')
	reader.ReadString(']')

	// Send name
	client.Write([]byte("OriginalName\n"))

	// Read initial messages
	client.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	buffer := make([]byte, 1024)
	client.Read(buffer)

	// Send rename command
	client.Write([]byte("/rename NewName\n"))

	// Check if name was updated in clients map
	time.Sleep(100 * time.Millisecond)
	models.Mu.Lock()
	if name, exists := models.Clients[server]; !exists || name != "NewName" {
		t.Errorf("Expected client name to be updated to 'NewName', got: %s, exists: %v", name, exists)
	}
	models.Mu.Unlock()

	// Test invalid rename
	client.Write([]byte("/rename \n"))

	// Should get error response
	client.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	errorBuffer := make([]byte, 1024)
	n, err := client.Read(errorBuffer)
	if err == nil && strings.Contains(string(errorBuffer[:n]), "Invalid name") {
		// Good, we got the expected error message
	} else if err != nil {
		// This might be because the message went to broadcast instead
		// Let's check the map to make sure name wasn't changed
		models.Mu.Lock()
		if name := models.Clients[server]; name != "NewName" {
			t.Errorf("Client name should still be 'NewName' after invalid rename, got: %s", name)
		}
		models.Mu.Unlock()
	}
}

func TestHandleClientQuit(t *testing.T) {
	// Setup
	models.Mu.Lock()
	models.Clients = make(map[net.Conn]string)
	models.Broadcast = make(chan string, 10)
	models.Mu.Unlock()

	// Create logo file
	logoContent := "Welcome!"
	err := ioutil.WriteFile("logo.txt", []byte(logoContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create logo.txt: %v", err)
	}
	defer os.Remove("logo.txt")

	server, client := net.Pipe()
	defer client.Close()

	// Channel to know when HandleClient finishes
	done := make(chan bool)
	go func() {
		cl.HandleClient(server, "nonexistent.txt")
		done <- true
	}()

	reader := bufio.NewReader(client)

	// Set a read deadline to prevent hanging
	client.SetReadDeadline(time.Now().Add(2 * time.Second))

	// Read logo and prompt
	_, err = reader.ReadString('\n')
	if err != nil {
		t.Fatalf("Failed to read logo: %v", err)
	}
	_, err = reader.ReadString(':')
	if err != nil {
		t.Fatalf("Failed to read name prompt: %v", err)
	}

	// Reset deadline
	client.SetReadDeadline(time.Time{})

	// Send name
	_, err = client.Write([]byte("TestUser\n"))
	if err != nil {
		t.Fatalf("Failed to send name: %v", err)
	}

	// Wait for initial messages with timeout
	client.SetReadDeadline(time.Now().Add(2 * time.Second))
	buffer := make([]byte, 1024)
	_, err = client.Read(buffer)
	if err != nil && !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("Failed to read initial messages: %v", err)
	}
	client.SetReadDeadline(time.Time{})

	// Verify client is in map
	models.Mu.Lock()
	if _, exists := models.Clients[server]; !exists {
		t.Error("Client should be in map before quit")
	}
	models.Mu.Unlock()

	// Send quit command
	_, err = client.Write([]byte("/quit\n"))
	if err != nil {
		t.Fatalf("Failed to send quit command: %v", err)
	}

	// Wait for handler to finish with timeout
	select {
	case <-done:
		// Good, handler finished
	case <-time.After(5 * time.Second):
		t.Fatal("Handler did not finish after quit command")
	}

	// Verify client was removed from map
	models.Mu.Lock()
	if _, exists := models.Clients[server]; exists {
		t.Error("Client should be removed from map after quit")
	}
	models.Mu.Unlock()
}
