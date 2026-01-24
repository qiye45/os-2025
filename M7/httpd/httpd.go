package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

const (
	BufferSize     = 4096
	DefaultPort    = 8080
	MaxConcurrency = 4
)

var (
	logMutex       sync.Mutex
	semaphore      chan struct{}
	requestCounter int
	requestMutex   sync.Mutex
)

type Request struct {
	Method      string
	Path        string
	QueryString string
	Order       int
}

func main() {
	port := DefaultPort
	if len(os.Args) > 1 {
		if p, err := strconv.Atoi(os.Args[1]); err == nil {
			port = p
		}
	}

	semaphore = make(chan struct{}, MaxConcurrency)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to listen: %v\n", err)
		os.Exit(1)
	}
	defer listener.Close()

	fmt.Printf("Server listening on port %d...\n", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Accept failed: %v\n", err)
			continue
		}

		requestMutex.Lock()
		requestCounter++
		order := requestCounter
		requestMutex.Unlock()

		go handleConnection(conn, order)
	}
}

func handleConnection(conn net.Conn, order int) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	requestLine, err := reader.ReadString('\n')
	if err != nil {
		return
	}
	fmt.Println(requestLine)

	// TODO: Parse HTTP request
	// TODO: Handle CGI execution
	// TODO: Send response
	// TODO: Log request

	// Placeholder response
	response := "HTTP/1.1 200 OK\r\n"
	response += "Content-Type: text/plain\r\n"
	response += "Content-Length: 18\r\n"
	response += "Connection: close\r\n"
	response += "\r\n"
	response += "Under construction"

	conn.Write([]byte(response))
}

func logRequest(method, path string, statusCode int) {
	logMutex.Lock()
	defer logMutex.Unlock()

	now := time.Now()
	timestamp := now.Format("2006-01-02 15:04:05")
	fmt.Printf("[%s] [%s] [%s] [%d]\n", timestamp, method, path, statusCode)
	os.Stdout.Sync()
}
