package models

import (
	"net"
	"os"
	"sync"
)

var (
	Clients   = make(map[net.Conn]string)
	Broadcast = make(chan string)
	Mu        sync.Mutex
	LogFile   *os.File
)
