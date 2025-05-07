# 🐾 Net-Cat: A Go-Powered TCP Chat Server

Net-Cat is a lightweight and efficient TCP-based chat server written in Go. Inspired by Netcat, it enhances the original utility with features like chat history, multi-user support, and activity logs. Built on a client-server model, it allows seamless, real-time communication between multiple connected clients.

---

## 🚀 Key Capabilities

- **Concurrent User Handling** – Supports multiple users chatting simultaneously  
- **Live Message Broadcast** – Instantly relays messages to all connected clients  
- **Username System** – Choose or update your display name during sessions  
- **Persistent Chat History** – New users receive prior messages upon joining  
- **Join/Leave Announcements** – All users are notified when someone connects or disconnects  
- **Automatic Logging** – Saves conversations and server activity to local files  
- **High Performance** – Uses Goroutines and channels for scalability and responsiveness  

---

## 🛠 Setup Guide

### ✅ Requirements

- Go (version 1.16 or newer)  
- `nc` (Netcat) for connecting as a client  

### 📥 Install & Build

```bash
git https://github.com/bernotieno/net-cat.git
cd net-cat
go build -o TCPChat
```

---

## 📡 Running the Application

### Start the Server

```bash
# On default port 8989
./TCPChat

# Or specify a different port
./TCPChat 2525
```

### Connect a Client

```bash
# Connect via netcat
nc localhost 8989
```
```bash
# Or connect on a custom port
nc localhost 2525
```

---

## 💬 How to Use

Once connected, users are prompted to input a username. Chat begins right after that!

### Available Commands

- `/exit` — Disconnect from the server  
- `/change <new_username>` — Update your current username  

---

## 🧾 Message Format

Standard messages:
```
[YYYY-MM-DD HH:MM:SS][username]: message
```

System messages:
```
[YYYY-MM-DD HH:MM:SS] username has joined the chat...
[YYYY-MM-DD HH:MM:SS] username has left the chat...
```

---

## 🗃 File Structure Overview

```
net-cat/
├── main.go              # Entry point
├── LICENSE              # MIT License
├── README.md            # Project documentation
└── server/
    ├── server.go        # Server setup and lifecycle
    ├── client.go        # Client connection and I/O handling
    ├── broadcaster.go   # Message distribution to clients
    ├── logger.go        # Chat and event logging
    ├── utils.go         # Helper utilities
```

---

## 🧪 Running Tests

To execute the test suite:

```bash
go test ./...
```

---

## 📝 Log Files

Each session is logged in a file named `chat_log_<port>.log` in the logs folder, which includes:

- Chat conversations  
- User join/leave events  
- Server startup/shutdown  
- Error reports  

---

## 📄 License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for full details.

