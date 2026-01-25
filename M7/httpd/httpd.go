package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	DefaultPort    = 8080
	MaxConcurrency = 4
)

var (
	logMutex       sync.Mutex
	semaphore      chan struct{}
	requestCounter int64
	statusText     = map[int]string{
		200: "OK",
		404: "Not Found",
		500: "Internal Server Error",
	}
)
var (
	notFoundError = errors.New("CGI command not found")
)

type Request struct {
	Method         string
	Path           string
	QueryStringMap map[string]string
	QueryString    string
	Order          int64
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

	fmt.Printf("Server listening on http://localhost:%d\n", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Accept failed: %v\n", err)
			continue
		}

		atomic.AddInt64(&requestCounter, 1)
		order := requestCounter

		semaphore <- struct{}{}
		go handleConnection(conn, order)
	}
}

func handleConnection(conn net.Conn, order int64) {
	defer func() { <-semaphore }()
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// Parse HTTP request
	statusCode := 200
	request, err := parseRequest(reader, order)
	if err != nil {
		statusCode = 500
		sendResponse(conn, statusCode, "", request)
		return
	}
	fmt.Printf("request : %v", request)
	// Handle CGI execution
	cgiOutput, err := handleCGI(request.Method, request.Path, request.QueryString)
	if err != nil {
		if errors.Is(err, notFoundError) {
			statusCode = 404
		} else {
			statusCode = 500
		}
	}
	sendResponse(conn, statusCode, cgiOutput, request)
}

func handleCGI(method, path, query string) (string, error) {
	if !strings.HasPrefix(path, "/cgi-bin") {
		return "", fmt.Errorf("invalid CGI command")
	}
	path = path[1:]
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", notFoundError
	}
	env := os.Environ()
	env = append(env, "REQUEST_METHOD="+method)
	env = append(env, "QUERY_STRING="+query)
	// 启动进程
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path)
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func sendResponse(conn net.Conn, statusCode int, cgiOutput string, request *Request) {
	// Send response
	var response strings.Builder
	response.Write([]byte(fmt.Sprintf("HTTP/1.1 %d %s\r\n", statusCode, statusText[statusCode])))
	response.Write([]byte("Content-Type: text/plain\r\n"))
	response.Write([]byte(fmt.Sprintf("Content-Length: %d\r\n", len(cgiOutput))))
	response.Write([]byte("Connection: close\r\n"))
	response.Write([]byte("\r\n"))
	response.Write([]byte(cgiOutput))
	// Log request
	if request != nil {
		logRequest(request.Method, request.Path, statusCode)
	} else {
		logRequest("unknown", "unknown", statusCode)
	}

	conn.Write([]byte(response.String()))
}
func readRequestHeader(reader *bufio.Reader) (string, error) {
	var buf strings.Builder
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			return buf.String(), err
		}
		buf.Write(line)
		// 请求头结束
		if len(line) == 2 && line[0] == '\r' && line[1] == '\n' {
			break
		}
	}
	return buf.String(), nil
}
func parseRequest(reader *bufio.Reader, order int64) (*Request, error) {
	header, err := readRequestHeader(reader)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(header, "\r\n")
	// 请求行
	if len(lines) == 0 {
		return nil, fmt.Errorf("invalid request line")
	}
	requestLine := strings.Split(lines[0], " ")
	if len(requestLine) != 3 {
		return nil, fmt.Errorf("invalid request line")
	}
	method, path, protocol := requestLine[0], requestLine[1], requestLine[2]
	var queryStringMap map[string]string
	var queryString string
	parts := strings.SplitN(path, "?", 2)
	if len(parts) == 2 {
		path = parts[0]
		queryString = parts[1]
		queryStringMap = parseQueryString(parts[1])
	}
	// 请求头
	requestHeaders := parseRequestHeader(lines[1:])
	fmt.Printf("order: %d ,Method: %s, Path: %s, Protocol: %s, headerLen:%d\n", order, method, path, protocol, len(requestHeaders))

	// 请求消息体
	contentLength, ok := requestHeaders["Content-Length"]
	if !ok {
		return &Request{
			Method:         method,
			Path:           path,
			QueryStringMap: queryStringMap,
			QueryString:    queryString,
			Order:          order,
		}, nil
	}
	contentLengthInt, err := strconv.Atoi(contentLength)
	if err != nil {
		return nil, fmt.Errorf("invalid Content-Length header: %s", contentLength)
	}
	requestBody := make([]byte, contentLengthInt)
	_, err = io.ReadFull(reader, requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %v", err)
	}

	return &Request{
		Method:         method,
		Path:           path,
		QueryStringMap: queryStringMap,
		QueryString:    queryString,
		Order:          order,
	}, nil
}

func parseRequestHeader(lines []string) map[string]string {
	headers := make(map[string]string)
	for _, line := range lines {
		keyValue := strings.SplitN(line, ":", 2)
		if len(keyValue) != 2 {
			continue
		}
		key, value := keyValue[0], keyValue[1]
		headers[key] = strings.TrimSpace(value)
	}
	return headers
}

func parseQueryString(line string) map[string]string {
	params := strings.Split(line, "&")
	queryString := make(map[string]string)
	for _, param := range params {
		keyValue := strings.SplitN(param, "=", 2)
		if len(keyValue) != 2 {
			continue
		}
		queryString[keyValue[0]] = keyValue[1]
	}
	return queryString
}

func logRequest(method, path string, statusCode int) {
	logMutex.Lock()
	defer logMutex.Unlock()

	now := time.Now()
	timestamp := now.Format("2006-01-02 15:04:05")
	fmt.Printf("[%s] [%s] [%s] [%d]\n", timestamp, method, path, statusCode)
	os.Stdout.Sync()
}
