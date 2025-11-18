// HTTP Server Example
//
// This example demonstrates a simple HTTP/1.1 server using the custom TCP implementation.
// The server can handle basic GET requests and serve static files.
//
// Usage:
//   sudo go run main.go -i eth0 -addr 192.168.1.100 -port 8080 -dir ./www
//
// Test with curl:
//   curl http://192.168.1.100:8080/
//   curl http://192.168.1.100:8080/test.html
//
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
	"github.com/therealutkarshpriyadarshi/network/pkg/tcp"
)

var (
	interfaceName = flag.String("i", "eth0", "Network interface name")
	listenAddr    = flag.String("addr", "192.168.1.100", "IP address to listen on")
	listenPort    = flag.Int("port", 8080, "Port to listen on")
	documentRoot  = flag.String("dir", "./www", "Document root directory")
)

const (
	// HTTP status codes
	StatusOK                  = 200
	StatusBadRequest          = 400
	StatusNotFound            = 404
	StatusMethodNotAllowed    = 405
	StatusInternalServerError = 500
)

var statusText = map[int]string{
	StatusOK:                  "OK",
	StatusBadRequest:          "Bad Request",
	StatusNotFound:            "Not Found",
	StatusMethodNotAllowed:    "Method Not Allowed",
	StatusInternalServerError: "Internal Server Error",
}

// HTTPRequest represents a parsed HTTP request.
type HTTPRequest struct {
	Method  string
	Path    string
	Version string
	Headers map[string]string
	Body    []byte
}

// HTTPResponse represents an HTTP response.
type HTTPResponse struct {
	StatusCode int
	Headers    map[string]string
	Body       []byte
}

func main() {
	flag.Parse()

	log.SetFlags(log.Ltime | log.Lshortfile)
	log.Printf("Starting HTTP server on %s:%d", *listenAddr, *listenPort)
	log.Printf("Document root: %s", *documentRoot)

	// Create document root if it doesn't exist
	if err := os.MkdirAll(*documentRoot, 0755); err != nil {
		log.Fatalf("Failed to create document root: %v", err)
	}

	// Create default index.html if it doesn't exist
	indexPath := filepath.Join(*documentRoot, "index.html")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		defaultHTML := `<!DOCTYPE html>
<html>
<head>
    <title>Custom TCP/IP Stack HTTP Server</title>
</head>
<body>
    <h1>Welcome to the Custom TCP/IP Stack HTTP Server</h1>
    <p>This server is running on a custom TCP/IP implementation!</p>
    <p>Current time: ` + time.Now().Format(time.RFC3339) + `</p>
</body>
</html>`
		if err := ioutil.WriteFile(indexPath, []byte(defaultHTML), 0644); err != nil {
			log.Printf("Failed to create default index.html: %v", err)
		}
	}

	// Parse listen address
	addr, err := common.ParseIPv4(*listenAddr)
	if err != nil {
		log.Fatalf("Invalid listen address: %v", err)
	}

	// Create TCP socket
	socket := tcp.NewSocket(addr, uint16(*listenPort))

	// Set up send function
	socket.SetSendFunc(func(seg *tcp.Segment, srcIP, dstIP common.IPv4Address) error {
		log.Printf("Sending segment: flags=%s seq=%d ack=%d len=%d",
			formatFlags(seg.Flags), seg.SequenceNumber, seg.AckNumber, len(seg.Data))
		// In a real implementation, this would send via the network stack
		return nil
	})

	// Listen for connections
	if err := socket.Listen(128); err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	log.Printf("HTTP server listening on http://%s:%d", *listenAddr, *listenPort)

	// Accept connections in a loop
	for {
		conn, err := socket.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}

		log.Printf("Accepted connection from %s:%d",
			conn.GetRemoteAddr(), conn.GetRemotePort())

		// Handle connection in a goroutine
		go handleHTTPConnection(conn)
	}
}

func handleHTTPConnection(conn *tcp.Socket) {
	defer func() {
		conn.Close()
		log.Printf("Closed connection from %s:%d",
			conn.GetRemoteAddr(), conn.GetRemotePort())
	}()

	buf := make([]byte, 8192)

	// Receive HTTP request
	n, err := conn.Recv(buf)
	if err != nil {
		log.Printf("Receive error: %v", err)
		return
	}

	requestData := buf[:n]
	log.Printf("Received %d bytes from %s:%d",
		n, conn.GetRemoteAddr(), conn.GetRemotePort())

	// Parse HTTP request
	req, err := parseHTTPRequest(requestData)
	if err != nil {
		log.Printf("Failed to parse HTTP request: %v", err)
		sendErrorResponse(conn, StatusBadRequest, err.Error())
		return
	}

	log.Printf("Request: %s %s %s", req.Method, req.Path, req.Version)

	// Handle request
	var resp *HTTPResponse
	switch req.Method {
	case "GET":
		resp = handleGET(req)
	case "HEAD":
		resp = handleGET(req)
		resp.Body = nil // HEAD doesn't include body
	default:
		resp = createErrorResponse(StatusMethodNotAllowed,
			fmt.Sprintf("Method %s not allowed", req.Method))
	}

	// Send response
	responseData := serializeHTTPResponse(resp)
	sent, err := conn.Send(responseData)
	if err != nil {
		log.Printf("Send error: %v", err)
		return
	}

	log.Printf("Sent %d bytes (status %d) to %s:%d",
		sent, resp.StatusCode, conn.GetRemoteAddr(), conn.GetRemotePort())
}

func parseHTTPRequest(data []byte) (*HTTPRequest, error) {
	req := &HTTPRequest{
		Headers: make(map[string]string),
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))

	// Parse request line
	if !scanner.Scan() {
		return nil, fmt.Errorf("empty request")
	}

	requestLine := scanner.Text()
	parts := strings.SplitN(requestLine, " ", 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid request line: %s", requestLine)
	}

	req.Method = parts[0]
	req.Path = parts[1]
	req.Version = parts[2]

	// Parse headers
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || line == "\r" {
			break // End of headers
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			req.Headers[key] = value
		}
	}

	return req, nil
}

func handleGET(req *HTTPRequest) *HTTPResponse {
	// Clean path to prevent directory traversal
	path := filepath.Clean(req.Path)
	if path == "." || path == "/" {
		path = "/index.html"
	}

	// Build file path
	filePath := filepath.Join(*documentRoot, path)

	// Check if file exists
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return createErrorResponse(StatusNotFound,
				fmt.Sprintf("File not found: %s", req.Path))
		}
		return createErrorResponse(StatusInternalServerError,
			fmt.Sprintf("Error accessing file: %v", err))
	}

	// If directory, try to serve index.html
	if fileInfo.IsDir() {
		filePath = filepath.Join(filePath, "index.html")
		if _, err := os.Stat(filePath); err != nil {
			return createErrorResponse(StatusNotFound,
				"Directory listing not allowed")
		}
	}

	// Read file
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return createErrorResponse(StatusInternalServerError,
			fmt.Sprintf("Error reading file: %v", err))
	}

	// Determine content type
	contentType := getContentType(filePath)

	// Create response
	resp := &HTTPResponse{
		StatusCode: StatusOK,
		Headers: map[string]string{
			"Content-Type":   contentType,
			"Content-Length": fmt.Sprintf("%d", len(content)),
			"Server":         "Custom-TCP-Stack/1.0",
			"Date":           time.Now().UTC().Format(time.RFC1123),
		},
		Body: content,
	}

	return resp
}

func createErrorResponse(statusCode int, message string) *HTTPResponse {
	body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>%d %s</title>
</head>
<body>
    <h1>%d %s</h1>
    <p>%s</p>
</body>
</html>`, statusCode, statusText[statusCode], statusCode, statusText[statusCode], message)

	return &HTTPResponse{
		StatusCode: statusCode,
		Headers: map[string]string{
			"Content-Type":   "text/html",
			"Content-Length": fmt.Sprintf("%d", len(body)),
			"Server":         "Custom-TCP-Stack/1.0",
			"Date":           time.Now().UTC().Format(time.RFC1123),
		},
		Body: []byte(body),
	}
}

func sendErrorResponse(conn *tcp.Socket, statusCode int, message string) {
	resp := createErrorResponse(statusCode, message)
	responseData := serializeHTTPResponse(resp)
	conn.Send(responseData)
}

func serializeHTTPResponse(resp *HTTPResponse) []byte {
	var builder strings.Builder

	// Status line
	builder.WriteString(fmt.Sprintf("HTTP/1.1 %d %s\r\n",
		resp.StatusCode, statusText[resp.StatusCode]))

	// Headers
	for key, value := range resp.Headers {
		builder.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}

	// End of headers
	builder.WriteString("\r\n")

	// Body
	result := []byte(builder.String())
	if resp.Body != nil {
		result = append(result, resp.Body...)
	}

	return result
}

func getContentType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".html", ".htm":
		return "text/html"
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
	case ".json":
		return "application/json"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".txt":
		return "text/plain"
	default:
		return "application/octet-stream"
	}
}

func formatFlags(flags uint8) string {
	var parts []string
	if flags&tcp.FlagFIN != 0 {
		parts = append(parts, "FIN")
	}
	if flags&tcp.FlagSYN != 0 {
		parts = append(parts, "SYN")
	}
	if flags&tcp.FlagRST != 0 {
		parts = append(parts, "RST")
	}
	if flags&tcp.FlagPSH != 0 {
		parts = append(parts, "PSH")
	}
	if flags&tcp.FlagACK != 0 {
		parts = append(parts, "ACK")
	}
	if flags&tcp.FlagURG != 0 {
		parts = append(parts, "URG")
	}
	if len(parts) == 0 {
		return "NONE"
	}
	return strings.Join(parts, "|")
}
