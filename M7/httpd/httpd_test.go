package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestBasicFunctionality tests basic HTTP server functionality
func TestBasicFunctionality(t *testing.T) {
	// Test port parsing
	port := 8081 // Use different port for testing

	// Start server in background
	go func() {
		os.Args = []string{"httpd", strconv.Itoa(port)}
		main()
	}()

	// Wait for server to start
	time.Sleep(200 * time.Millisecond)

	// Test basic connection
	conn, err := net.Dial("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	// Send basic HTTP request
	request := "GET / HTTP/1.1\r\nHost: localhost\r\n\r\n"
	_, err = conn.Write([]byte(request))
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Read response
	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if !strings.Contains(response, "HTTP/1.1") {
		t.Errorf("Expected HTTP/1.1 response, got: %s", response)
	}
}

// TestCGIExecution tests CGI program execution
func TestCGIExecution(t *testing.T) {
	// Create test CGI script
	cgiDir := "cgi-bin"
	os.MkdirAll(cgiDir, 0755)

	testScript := filepath.Join(cgiDir, "test")
	scriptContent := `#!/bin/bash
echo "HTTP/1.1 200 OK"
echo "Content-Type: text/plain"
echo "Content-Length: 5"
echo ""
echo "test"`

	err := os.WriteFile(testScript, []byte(scriptContent), 0755)
	if err != nil {
		t.Fatalf("Failed to create test CGI script: %v", err)
	}
	defer os.Remove(testScript)

	port := 8082
	go func() {
		os.Args = []string{"httpd", strconv.Itoa(port)}
		main()
	}()

	time.Sleep(200 * time.Millisecond)

	// Test CGI execution
	conn, err := net.Dial("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	request := "GET /cgi-bin/test HTTP/1.1\r\nHost: localhost\r\n\r\n"
	_, err = conn.Write([]byte(request))
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if !strings.Contains(response, "200") {
		t.Errorf("Expected 200 response, got: %s", response)
	}
}

// TestErrorHandling tests 404 and 500 error handling
func TestErrorHandling(t *testing.T) {
	port := 8083
	go func() {
		os.Args = []string{"httpd", strconv.Itoa(port)}
		main()
	}()

	time.Sleep(200 * time.Millisecond)

	// Test 404 - non-existent CGI script
	conn, err := net.Dial("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	request := "GET /cgi-bin/nonexistent HTTP/1.1\r\nHost: localhost\r\n\r\n"
	_, err = conn.Write([]byte(request))
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if !strings.Contains(response, "404") {
		t.Errorf("Expected 404 response, got: %s", response)
	}
}

// TestConcurrency tests concurrent request handling
func TestConcurrency(t *testing.T) {
	port := 8084
	go func() {
		os.Args = []string{"httpd", strconv.Itoa(port)}
		main()
	}()

	time.Sleep(200 * time.Millisecond)

	// Create test CGI script
	cgiDir := "cgi-bin"
	os.MkdirAll(cgiDir, 0755)

	testScript := filepath.Join(cgiDir, "concurrent")
	scriptContent := `#!/bin/bash
echo "HTTP/1.1 200 OK"
echo "Content-Type: text/plain"
echo "Content-Length: 2"
echo ""
echo "OK"`

	err := os.WriteFile(testScript, []byte(scriptContent), 0755)
	if err != nil {
		t.Fatalf("Failed to create test CGI script: %v", err)
	}
	defer os.Remove(testScript)

	// Send multiple concurrent requests
	var wg sync.WaitGroup
	numRequests := 5
	errors := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			conn, err := net.Dial("tcp", fmt.Sprintf(":%d", port))
			if err != nil {
				errors <- fmt.Errorf("request %d failed: %v", id, err)
				return
			}
			defer conn.Close()

			request := fmt.Sprintf("GET /cgi-bin/concurrent?id=%d HTTP/1.1\r\nHost: localhost\r\n\r\n", id)
			_, err = conn.Write([]byte(request))
			if err != nil {
				errors <- fmt.Errorf("request %d write failed: %v", id, err)
				return
			}

			reader := bufio.NewReader(conn)
			_, err = reader.ReadString('\n')
			if err != nil {
				errors <- fmt.Errorf("request %d read failed: %v", id, err)
				return
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}
}

// TestLogging tests log output format and ordering
func TestLogging(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	port := 8085
	go func() {
		os.Args = []string{"httpd", strconv.Itoa(port)}
		main()
	}()

	time.Sleep(200 * time.Millisecond)

	// Make a request
	conn, err := net.Dial("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}

	request := "GET / HTTP/1.1\r\nHost: localhost\r\n\r\n"
	conn.Write([]byte(request))
	conn.Close()

	time.Sleep(100 * time.Millisecond)

	// Restore stdout and read captured output
	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Check log format
	if !strings.Contains(output, "[") || !strings.Contains(output, "]") {
		t.Errorf("Expected log format [timestamp] [method] [path] [status], got: %s", output)
	}
}

// BenchmarkRequest measures request handling performance
func BenchmarkRequest(b *testing.B) {
	port := 8086
	go func() {
		os.Args = []string{"httpd", strconv.Itoa(port)}
		main()
	}()

	time.Sleep(200 * time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn, err := net.Dial("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			b.Fatalf("Failed to connect: %v", err)
		}

		request := "GET / HTTP/1.1\r\nHost: localhost\r\n\r\n"
		_, err = conn.Write([]byte(request))
		if err != nil {
			b.Fatalf("Failed to send: %v", err)
		}

		bufio.NewReader(conn).ReadString('\n')
		conn.Close()
	}
}

// TestCGIEnvironment tests CGI environment variables
func TestCGIEnvironment(t *testing.T) {
	// Create test CGI script that checks environment
	cgiDir := "cgi-bin"
	os.MkdirAll(cgiDir, 0755)

	testScript := filepath.Join(cgiDir, "envtest")
	scriptContent := `#!/bin/bash
if [ -n "$REQUEST_METHOD" ] && [ -n "$QUERY_STRING" ]; then
  echo "HTTP/1.1 200 OK"
  echo "Content-Type: text/plain"
  echo "Content-Length: 2"
  echo ""
  echo "OK"
else
  echo "HTTP/1.1 500 Internal Server Error"
  echo "Content-Type: text/plain"
  echo "Content-Length: 0"
  echo ""
fi`

	err := os.WriteFile(testScript, []byte(scriptContent), 0755)
	if err != nil {
		t.Fatalf("Failed to create test CGI script: %v", err)
	}
	defer os.Remove(testScript)

	port := 8087
	go func() {
		os.Args = []string{"httpd", strconv.Itoa(port)}
		main()
	}()

	time.Sleep(200 * time.Millisecond)

	// Test with query string
	conn, err := net.Dial("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	request := "GET /cgi-bin/envtest?key=value HTTP/1.1\r\nHost: localhost\r\n\r\n"
	_, err = conn.Write([]byte(request))
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if !strings.Contains(response, "200") {
		t.Errorf("Expected 200 response with proper environment, got: %s", response)
	}
}

// TestMaxConcurrencyLimit tests 4-request concurrency limit
func TestMaxConcurrencyLimit(t *testing.T) {
	// Create a CGI script that records its execution start and end times
	cgiDir := "cgi-bin"
	os.MkdirAll(cgiDir, 0755)

	// Create a shared log file to track concurrent execution
	logFile := "/tmp/concurrency_test.log"
	// Clean up any existing log file
	os.Remove(logFile)

	testScript := filepath.Join(cgiDir, "slow")
	scriptContent := `#!/bin/bash
# Record when this script starts executing
echo "START $1 $(date +%s%N)" >> ` + logFile + `
# Sleep for 200ms to ensure overlap
sleep 0.2
# Record when this script ends
echo "END $1 $(date +%s%N)" >> ` + logFile + `
echo "HTTP/1.1 200 OK"
echo "Content-Type: text/plain"
echo "Content-Length: 2"
echo ""
echo "OK"`

	err := os.WriteFile(testScript, []byte(scriptContent), 0755)
	if err != nil {
		t.Fatalf("Failed to create test CGI script: %v", err)
	}
	defer os.Remove(testScript)
	defer os.Remove(logFile)

	port := 8092
	go func() {
		os.Args = []string{"httpd", strconv.Itoa(port)}
		main()
	}()

	time.Sleep(200 * time.Millisecond)

	// Send 8 requests concurrently (more than the limit of 4)
	var wg sync.WaitGroup
	numRequests := 8
	errors := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			conn, err := net.Dial("tcp", fmt.Sprintf(":%d", port))
			if err != nil {
				errors <- fmt.Errorf("request %d failed to connect: %v", id, err)
				return
			}
			defer conn.Close()

			request := fmt.Sprintf("GET /cgi-bin/slow?id=%d HTTP/1.1\r\nHost: localhost\r\n\r\n", id)
			_, err = conn.Write([]byte(request))
			if err != nil {
				errors <- fmt.Errorf("request %d write failed: %v", id, err)
				return
			}

			// Read response
			reader := bufio.NewReader(conn)
			response, err := reader.ReadString('\n')
			if err != nil {
				errors <- fmt.Errorf("request %d read failed: %v", id, err)
				return
			}

			if !strings.Contains(response, "200") {
				errors <- fmt.Errorf("request %d expected 200, got: %s", id, response)
				return
			}
		}(i)
	}

	// Wait for all requests to complete
	wg.Wait()
	close(errors)

	// Check for any errors
	for err := range errors {
		t.Error(err)
	}

	// Wait a bit for all log entries to be written
	time.Sleep(100 * time.Millisecond)

	// Read and analyze the concurrency log
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read concurrency log: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	startTimes := make(map[int]int64)
	endTimes := make(map[int]int64)

	// Parse the log to extract start and end times
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) != 3 {
			continue
		}
		action, idStr, timestampStr := parts[0], parts[1], parts[2]
		id, err := strconv.Atoi(idStr)
		if err != nil {
			continue
		}
		timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
		if err != nil {
			continue
		}

		if action == "START" {
			startTimes[id] = timestamp
		} else if action == "END" {
			endTimes[id] = timestamp
		}
	}

	// Calculate maximum concurrent executions
	maxConcurrent := 0
	// Check at each millisecond interval how many scripts were running
	for i := 0; i < numRequests; i++ {
		if start, hasStart := startTimes[i]; hasStart {
			if _, hasEnd := endTimes[i]; hasEnd {
				concurrent := 0
				// Count how many scripts were running at the start time of script i
				for j := 0; j < numRequests; j++ {
					if otherStart, hasOtherStart := startTimes[j]; hasOtherStart {
						if otherEnd, hasOtherEnd := endTimes[j]; hasOtherEnd {
							// Script j is running at time start if it started before start and ended after start
							if otherStart <= start && otherEnd > start {
								concurrent++
							}
						}
					}
				}
				if concurrent > maxConcurrent {
					maxConcurrent = concurrent
				}
			}
		}
	}

	// Verify that no more than 4 requests were concurrent
	if maxConcurrent > 4 {
		t.Errorf("Expected max concurrent CGI executions to be <= 4, got %d", maxConcurrent)
	}

	t.Logf("Maximum concurrent CGI executions: %d (should be <= 4)", maxConcurrent)
	t.Logf("Total log entries: %d", len(lines))
}

// TestQueryStringParsing tests query string parsing
func TestQueryStringParsing(t *testing.T) {
	// Create test CGI script
	cgiDir := "cgi-bin"
	os.MkdirAll(cgiDir, 0755)

	testScript := filepath.Join(cgiDir, "querytest")
	scriptContent := `#!/bin/bash
echo "$QUERY_STRING"`

	err := os.WriteFile(testScript, []byte(scriptContent), 0755)
	if err != nil {
		t.Fatalf("Failed to create test CGI script: %v", err)
	}
	defer os.Remove(testScript)

	port := 8088
	go func() {
		os.Args = []string{"httpd", strconv.Itoa(port)}
		main()
	}()

	time.Sleep(200 * time.Millisecond)

	// Test with query string
	conn, err := net.Dial("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	request := "GET /cgi-bin/querytest?name=test&value=123 HTTP/1.1\r\nHost: localhost\r\n\r\n"
	_, err = conn.Write([]byte(request))
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	reader := bufio.NewReader(conn)
	// Read status line
	reader.ReadString('\n')
	// Read headers
	for {
		line, err := reader.ReadString('\n')
		if err != nil || line == "\r\n" {
			break
		}
	}

	// Read body
	body, _ := reader.ReadString('\n')
	if !strings.Contains(body, "name=test") {
		t.Errorf("Expected query string in body, got: %s", body)
	}
}

// TestMalformedRequests tests handling of malformed HTTP requests
func TestMalformedRequests(t *testing.T) {
	port := 8089
	go func() {
		os.Args = []string{"httpd", strconv.Itoa(port)}
		main()
	}()

	time.Sleep(200 * time.Millisecond)

	// Test malformed request
	conn, err := net.Dial("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	// Send invalid HTTP request
	request := "INVALID REQUEST\r\n\r\n"
	_, err = conn.Write([]byte(request))
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Server should handle gracefully (may close connection or return error)
	// Just verify it doesn't crash
	time.Sleep(100 * time.Millisecond)
}

// TestIntegration tests complete workflow
func TestIntegration(t *testing.T) {
	// Create multiple CGI scripts for integration test
	cgiDir := "cgi-bin"
	os.MkdirAll(cgiDir, 0755)

	// Script 1: Simple echo
	echoScript := filepath.Join(cgiDir, "integration_echo")
	echoContent := `#!/bin/bash
echo "HTTP/1.1 200 OK"
echo "Content-Type: text/plain"
echo "Content-Length: 4"
echo ""
echo "echo"`
	err := os.WriteFile(echoScript, []byte(echoContent), 0755)
	if err != nil {
		t.Fatalf("Failed to create echo script: %v", err)
	}
	defer os.Remove(echoScript)

	// Script 2: Error handler
	errorScript := filepath.Join(cgiDir, "integration_error")
	errorContent := `#!/bin/bash
echo "HTTP/1.1 403 Forbidden"
echo "Content-Type: text/plain"
echo "Content-Length: 9"
echo ""
echo "Forbidden"`
	err = os.WriteFile(errorScript, []byte(errorContent), 0755)
	if err != nil {
		t.Fatalf("Failed to create error script: %v", err)
	}
	defer os.Remove(errorScript)

	port := 8090
	go func() {
		os.Args = []string{"httpd", strconv.Itoa(port)}
		main()
	}()

	time.Sleep(200 * time.Millisecond)

	// Test 1: Successful CGI execution
	t.Run("Success", func(t *testing.T) {
		conn, err := net.Dial("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}
		defer conn.Close()

		request := "GET /cgi-bin/integration_echo HTTP/1.1\r\nHost: localhost\r\n\r\n"
		conn.Write([]byte(request))

		reader := bufio.NewReader(conn)
		response, _ := reader.ReadString('\n')
		if !strings.Contains(response, "200") {
			t.Errorf("Expected 200, got: %s", response)
		}
	})

	// Test 2: Non-existent CGI
	t.Run("NotFound", func(t *testing.T) {
		conn, err := net.Dial("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}
		defer conn.Close()

		request := "GET /cgi-bin/nonexistent HTTP/1.1\r\nHost: localhost\r\n\r\n"
		conn.Write([]byte(request))

		reader := bufio.NewReader(conn)
		response, _ := reader.ReadString('\n')
		if !strings.Contains(response, "404") {
			t.Errorf("Expected 404, got: %s", response)
		}
	})
}

// TestPortParsing tests command line port argument parsing
func TestPortParsing(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"8080", 8080},
		{"9000", 9000},
		{"", 8080}, // default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if tt.input == "" {
				// Test default port
				// This would require refactoring main() to be testable
				t.Skip("Default port test requires refactoring")
			} else {
				port, err := strconv.Atoi(tt.input)
				if err != nil {
					t.Errorf("Failed to parse port: %v", err)
				}
				if port != tt.expected {
					t.Errorf("Expected port %d, got %d", tt.expected, port)
				}
			}
		})
	}
}

// TestHTTPMethods tests different HTTP methods
func TestHTTPMethods(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "DELETE"}

	for i, method := range methods {
		t.Run(method, func(t *testing.T) {
			// Create CGI script that checks method
			cgiDir := "cgi-bin"
			os.MkdirAll(cgiDir, 0755)

			testScript := filepath.Join(cgiDir, fmt.Sprintf("method_%s", strings.ToLower(method)))
			scriptContent := fmt.Sprintf(`#!/bin/bash
echo "HTTP/1.1 200 OK"
echo "Content-Type: text/plain"
echo "Content-Length: %d"
echo ""
echo "%s"`, len(method), method)

			err := os.WriteFile(testScript, []byte(scriptContent), 0755)
			if err != nil {
				t.Fatalf("Failed to create test CGI script: %v", err)
			}
			defer os.Remove(testScript)

			port := 8100 + i
			go func() {
				os.Args = []string{"httpd", strconv.Itoa(port)}
				main()
			}()

			time.Sleep(200 * time.Millisecond)

			conn, err := net.Dial("tcp", fmt.Sprintf(":%d", port))
			if err != nil {
				t.Fatalf("Failed to connect: %v", err)
			}
			defer conn.Close()

			request := fmt.Sprintf("%s /cgi-bin/method_%s HTTP/1.1\r\nHost: localhost\r\n\r\n", method, strings.ToLower(method))
			conn.Write([]byte(request))

			reader := bufio.NewReader(conn)
			response, _ := reader.ReadString('\n')
			if !strings.Contains(response, "200") {
				t.Errorf("Expected 200 for %s, got: %s", method, response)
			}
		})
	}
}
