package tests

import (
	"io"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"netcat/models"
	. "netcat/utils"
)

func TestLogToFile(t *testing.T) {
	// Test when LogFile is nil
	t.Run("LogFile is nil", func(t *testing.T) {
		models.LogFile = nil
		LogToFile("test message")
		// Should not panic or error
	})

	// Test normal logging
	t.Run("Normal logging", func(t *testing.T) {
		// Create temporary file
		tmpFile, err := ioutil.TempFile("", "test_log")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())
		defer tmpFile.Close()

		models.LogFile = tmpFile
		testMsg := "test message\n"
		LogToFile(testMsg)

		// Read back the content
		tmpFile.Seek(0, 0)
		content, err := ioutil.ReadAll(tmpFile)
		if err != nil {
			t.Fatalf("Failed to read temp file: %v", err)
		}

		if string(content) != testMsg {
			t.Errorf("Expected %q, got %q", testMsg, string(content))
		}
	})
}

func TestSendChatHistory(t *testing.T) {
	// Test when file doesn't exist
	t.Run("File doesn't exist", func(t *testing.T) {
		server, client := net.Pipe()
		defer server.Close()
		defer client.Close()

		// Use a WaitGroup to ensure we wait for SendChatHistory to complete
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			SendChatHistory(server, "nonexistent.txt")
		}()

		// Set read deadline to prevent hanging
		client.SetReadDeadline(time.Now().Add(2 * time.Second))
		buffer := make([]byte, 1024)
		n, err := client.Read(buffer)
		if err != nil {
			t.Fatalf("Failed to read from connection: %v", err)
		}
		response := string(buffer[:n])

		if !strings.Contains(response, "[No chat history available]") {
			t.Errorf("Expected no chat history message, got: %s", response)
		}

		wg.Wait()
	})

	// Test with existing file
	t.Run("File exists with history", func(t *testing.T) {
		// Create temporary file with content
		tmpFile, err := ioutil.TempFile("", "test_history")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())

		testContent := "line1\nline2\nline3\n"
		if _, err := tmpFile.WriteString(testContent); err != nil {
			t.Fatalf("Failed to write to temp file: %v", err)
		}
		tmpFile.Close()

		server, client := net.Pipe()
		defer server.Close()
		defer client.Close()

		// Use a WaitGroup to ensure we wait for SendChatHistory to complete
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			SendChatHistory(server, tmpFile.Name())
		}()

		// Read with timeout and accumulate all lines
		client.SetReadDeadline(time.Now().Add(2 * time.Second))
		var receivedContent strings.Builder
		buffer := make([]byte, 1024)

		for {
			n, err := client.Read(buffer)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					break // Break on timeout, we might have all the data
				}
				if err == io.EOF {
					break // Normal end of transmission
				}
				t.Fatalf("Error reading from connection: %v", err)
			}
			receivedContent.Write(buffer[:n])

			// Break if we've received all expected lines
			if strings.Count(receivedContent.String(), "\n") >= 3 {
				break
			}
		}

		response := receivedContent.String()
		expectedLines := []string{"line1", "line2", "line3"}
		for _, line := range expectedLines {
			if !strings.Contains(response, line) {
				t.Errorf("Expected response to contain %q, got: %s", line, response)
			}
		}

		wg.Wait()
	})
}

func TestNotifyClients(t *testing.T) {
	// Reset clients map before test
	models.Mu.Lock()
	models.Clients = make(map[net.Conn]string)
	models.Mu.Unlock()

	// Create test connections
	server1, client1 := net.Pipe()
	server2, client2 := net.Pipe()
	server3, client3 := net.Pipe()
	defer server1.Close()
	defer client1.Close()
	defer server2.Close()
	defer client2.Close()
	defer server3.Close()
	defer client3.Close()

	// Add clients to the map
	models.Mu.Lock()
	models.Clients[server1] = "User1"
	models.Clients[server2] = "User2"
	models.Clients[server3] = "User3"
	models.Mu.Unlock()

	testMessage := "Test broadcast message\n"

	// Notify all clients except server2
	go NotifyClients(server2, testMessage)

	// Check if server1 and server3 received the message
	checkMessage := func(client net.Conn, shouldReceive bool) {
		client.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		buffer := make([]byte, 1024)
		n, err := client.Read(buffer)

		if shouldReceive {
			if err != nil {
				t.Errorf("Expected to receive message, but got error: %v", err)
				return
			}
			if string(buffer[:n]) != testMessage {
				t.Errorf("Expected %q, got %q", testMessage, string(buffer[:n]))
			}
		} else {
			if err == nil {
				t.Errorf("Expected not to receive message, but got: %q", string(buffer[:n]))
			}
		}
	}

	go checkMessage(client1, true)  // Should receive
	go checkMessage(client3, true)  // Should receive
	go checkMessage(client2, false) // Should not receive

	time.Sleep(200 * time.Millisecond)
}
