package tests

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"netcat/models"
	"netcat/server"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestInitServer(t *testing.T) {
	originalLogFile := models.LogFile
	originalClients := make(map[net.Conn]string)
	if models.Clients != nil {
		models.Mu.Lock()
		for k, v := range models.Clients {
			originalClients[k] = v
		}
		models.Mu.Unlock()
	}
	baseDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	testLogsPath := filepath.Join(baseDir, "logs")

	originalStdLogOutput := log.Writer()
	log.SetOutput(io.Discard)
	defer log.SetOutput(originalStdLogOutput)

	defer func() {
		models.LogFile = originalLogFile
		models.Mu.Lock()
		models.Clients = originalClients
		models.Mu.Unlock()
		os.RemoveAll(testLogsPath)
	}()

	tests := []struct {
		name         string
		port         string
		setup        func(t *testing.T) (cleanup func(t *testing.T))
		expectErr    bool
		validateFunc func(t *testing.T, port string, logDir string)
	}{
		{
			name: "Successful server startup",
			port: ":38081",
			setup: func(t *testing.T) func(t *testing.T) {
				if err := os.MkdirAll(testLogsPath, os.ModePerm); err != nil {
					t.Fatalf(" Setup: Failed to create test logs directory: %v", err)
				}

				// Create logo file for test
				err := os.WriteFile("logo.txt", []byte("Welcome to TCP Chat!\n"), 0644)
				if err != nil {
					t.Fatalf("Failed to create logo file: %v", err)
				}

				return func(t *testing.T) {
					if models.LogFile != nil {
						models.LogFile.Close()
						models.LogFile = nil
					}
					os.Remove("logo.txt")
				}
			},
			expectErr: false,
			validateFunc: func(t *testing.T, port string, logDir string) {
				portNum := strings.TrimPrefix(port, ":")
				expectedLogFile := filepath.Join(logDir, fmt.Sprintf("chat_log_%s.log", portNum))

				time.Sleep(100 * time.Millisecond) // Give server a moment

				if _, err := os.Stat(expectedLogFile); os.IsNotExist(err) {
					t.Errorf("Expected log file %s to be created, but it wasn't", expectedLogFile)
				}

				conn, err := net.DialTimeout("tcp", "localhost"+port, 1*time.Second)
				if err != nil {
					t.Fatalf("Failed to connect to server on port %s: %v", port, err)
				}
				defer conn.Close()

				// Create a buffered reader to read line by line
				reader := bufio.NewReader(conn)
				conn.SetReadDeadline(time.Now().Add(2 * time.Second))

				// Read and verify logo
				logo, err := reader.ReadString('\n')
				if err != nil {
					t.Fatalf("Error reading logo: %v", err)
				}
				if len(logo) == 0 {
					t.Error("Expected logo, but got empty response")
				}

				// Read and verify name prompt
				namePrompt, err := reader.ReadString(':')
				if err != nil {
					t.Fatalf("Error reading name prompt: %v", err)
				}
				if !strings.Contains(namePrompt, "[ENTER YOUR NAME]:") {
					t.Errorf("Expected name prompt, got: %q", namePrompt)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Reset global state that InitServer modifies
			models.Mu.Lock()
			models.Clients = make(map[net.Conn]string)
			models.Mu.Unlock()
			if models.LogFile != nil { // Close any previously opened global log file
				models.LogFile.Close()
			}
			models.LogFile = nil

			cleanup := tc.setup(t)
			defer cleanup(t)

			errChan := make(chan error, 1)
			serverExitedChan := make(chan struct{})

			go func() {
				defer close(serverExitedChan)
				err := server.InitServer(tc.port)
				if err != nil {
					errChan <- err
				}
			}()

			if !tc.expectErr {
				// Success case: InitServer should not return an error and should be running.
				select {
				case err := <-errChan:
					t.Fatalf("InitServer returned an error unexpectedly: %v", err)
				case <-time.After(250 * time.Millisecond): // Give server time to start up
					if tc.validateFunc != nil {
						tc.validateFunc(t, tc.port, testLogsPath)
					}
					// Server goroutine is still running. Test will end, OS cleans up port.
				case <-serverExitedChan:
					// This means server exited cleanly, which is not expected for a successful persistent server start
					t.Fatal("InitServer exited unexpectedly for a success case (should run indefinitely)")
				}
			} else {
				// Error case: InitServer should return an error.
				select {
				case err := <-errChan:
					if err == nil {
						t.Errorf("Expected an error from InitServer, but got nil")
					}
				case <-time.After(1 * time.Second):
					t.Errorf("Expected InitServer to return an error, but it timed out")
				case <-serverExitedChan:
					// If serverExitedChan is closed, it means InitServer() returned.
					// We need to check if an error was actually sent to errChan.
					select {
					case err := <-errChan:
						if err == nil {
							t.Errorf("InitServer exited cleanly, but an error was expected.")
						}
						// Error received as expected.
					default:
						// This case should ideally not be hit if serverExitedChan implies an error was sent or it exited cleanly.
						t.Errorf("InitServer exited, but no error was received on errChan, though an error was expected.")
					}
				}
			}
		})
	}
}
