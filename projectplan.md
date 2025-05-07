# 🛠️ TCPChat - A NetCat-Inspired Multi-Client TCP Chat in Go

## 📌 Objective

Recreate the core functionality of NetCat (`nc`) using a TCP server-client model in Go, enabling real-time group chat with concurrency, message history, and multiple client support.

---

## 📅 Project Timeline

| Phase                         | Duration | Description                                                                 |
|------------------------------|----------|-----------------------------------------------------------------------------|
| 1. Requirements & Planning   | 1 day    | Understand NetCat behavior, finalize feature set, and design architecture   |
| 2. Project Setup & Base Code | 1 day    | Initialize Go modules, set up directories, basic TCP server-client          |
| 3. Core Features             | 3-4 days | Implement core chat, broadcasting, name system, connection handling         |
| 4. Concurrency & Error Handling | 2 days | Add Goroutines, channels/Mutexes, and handle edge-case errors               |
| 5. History Buffer            | 1 day    | Store and send chat history to new clients                                  |
| 6. Testing & Logging         | 2 days   | Unit tests and saving messages to file                                      |


---

## 📂 Project Structure

```
tcpchat/
├── main.go
├── server/
│   ├── server.go
│   └── handler.go
├── client/
│   └── client.go
├── utils/
│   └── utils.go
├── tests/
│   ├── server_test.go
│   └── client_test.go
├── logs/
│   └── chat.log
├── go.mod
└── README.md
```

---

## 🚀 Features

### ✅ Core Server Features
- Start on port 9060`` by default or custom port
- Accept up to **10 clients**
- Welcome message with Linux ASCII art
- Ask for and validate client name (non-empty)
- Broadcast messages with:
  ```
  [YYYY-MM-DD HH:MM:SS][ClientName]: Message
  ```
- Notify other clients when a user **joins or leaves**
- Send **chat history** to new clients
- Ignore and don't broadcast **empty messages**
- Prevent disconnects from crashing other clients

### ✅ Core Client Features
- Connect using NetCat or custom client
- Prompt for name input
- Send and receive messages
- Receive notifications of join/leave
- Receive chat history

---

## ⚙️ Concurrency Model

- Goroutines to handle individual clients
- Use **channels or `sync.Mutex`** to:
  - Protect access to shared message buffer
  - Protect and manage connected clients list
  - Ensure synchronized broadcasting and cleanup

---

## 🚫 Validation & Error Handling

- Invalid port → show usage:
  ```
  [USAGE]: ./TCPChat $port
  ```
- No port → default to `9060`
- Max clients → reject new connections
- Empty name → request again
- Empty messages → ignore
- Graceful handling of client disconnection

---

## 🧪 Testing Plan

- **Unit Tests** for:
  - TCP connection (server-client)
  - Message formatting
  - Name validation
  - Broadcasting logic
- **Manual testing** using multiple terminal clients (`nc localhost 9060`)

---

## 📘 Documentation (README.md)

Include:
- Overview and features
- Setup & Installation
- Usage examples:
  ```bash
  $ go run .            # uses default port 9060
  $ go run . 2525       # custom port
  $ nc localhost 9060   # connect client
  ```
- Contribution guidelines
- Bonus commands (if implemented)

---