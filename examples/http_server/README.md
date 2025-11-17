# HTTP Server Example

A simple HTTP/1.1 server implementation using the custom TCP/IP stack.

## Features

- HTTP/1.1 protocol support
- GET and HEAD request methods
- Static file serving
- Automatic content type detection
- Error handling (404, 405, 500)
- Concurrent connection handling

## Usage

### Start the Server

```bash
# Create a directory for static files
mkdir -p www
echo "<h1>Hello World</h1>" > www/index.html

# Run the server (requires root for raw sockets)
sudo go run main.go -i eth0 -addr 192.168.1.100 -port 8080 -dir ./www
```

### Command Line Options

- `-i` : Network interface name (default: eth0)
- `-addr` : IP address to listen on (default: 192.168.1.100)
- `-port` : Port to listen on (default: 8080)
- `-dir` : Document root directory (default: ./www)

### Test the Server

```bash
# Using curl
curl http://192.168.1.100:8080/
curl http://192.168.1.100:8080/index.html

# Using wget
wget http://192.168.1.100:8080/

# Using a web browser
# Open http://192.168.1.100:8080/ in your browser
```

## Supported HTTP Methods

- **GET**: Retrieve a resource
- **HEAD**: Same as GET but returns only headers (no body)

## Supported Content Types

The server automatically detects content types based on file extensions:

- `.html`, `.htm` → text/html
- `.css` → text/css
- `.js` → application/javascript
- `.json` → application/json
- `.txt` → text/plain
- `.png` → image/png
- `.jpg`, `.jpeg` → image/jpeg
- `.gif` → image/gif
- `.svg` → image/svg+xml
- Others → application/octet-stream

## HTTP Response Status Codes

- **200 OK**: Request succeeded
- **400 Bad Request**: Malformed request
- **404 Not Found**: File not found
- **405 Method Not Allowed**: Unsupported HTTP method
- **500 Internal Server Error**: Server error

## Example Session

```bash
$ curl -v http://192.168.1.100:8080/

> GET / HTTP/1.1
> Host: 192.168.1.100:8080
> User-Agent: curl/7.68.0
> Accept: */*
>
< HTTP/1.1 200 OK
< Content-Type: text/html
< Content-Length: 234
< Server: Custom-TCP-Stack/1.0
< Date: Mon, 17 Nov 2025 10:00:00 GMT
<
<!DOCTYPE html>
<html>
<head>
    <title>Custom TCP/IP Stack HTTP Server</title>
</head>
<body>
    <h1>Welcome to the Custom TCP/IP Stack HTTP Server</h1>
    <p>This server is running on a custom TCP/IP implementation!</p>
</body>
</html>
```

## Architecture

```
┌─────────────┐
│   Client    │
│  (Browser)  │
└──────┬──────┘
       │ HTTP Request
       ▼
┌─────────────┐
│    HTTP     │
│   Parser    │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  Request    │
│  Handler    │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│    File     │
│   System    │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  Response   │
│  Builder    │
└──────┬──────┘
       │ HTTP Response
       ▼
┌─────────────┐
│     TCP     │
│   Socket    │
└─────────────┘
```

## Limitations

- HTTP/1.1 only (no HTTP/2 or HTTP/3)
- No HTTPS/TLS support
- No chunked transfer encoding
- No compression
- No keep-alive connection reuse
- No request body parsing (POST data)
- No CGI or dynamic content
- Directory listing not supported

## Security Considerations

- Path traversal prevention (uses `filepath.Clean`)
- No execution of server-side code
- Static files only
- No authentication or authorization

## Performance

The server handles each connection in a separate goroutine, allowing concurrent request processing. Performance depends on:

- TCP stack implementation efficiency
- File I/O performance
- Number of concurrent connections
- Network bandwidth

## Future Enhancements

- [ ] HTTP persistent connections (keep-alive)
- [ ] POST request support
- [ ] Request body parsing
- [ ] Chunked transfer encoding
- [ ] Compression (gzip)
- [ ] Range requests
- [ ] Virtual hosts
- [ ] CGI support
- [ ] HTTPS/TLS

## Testing

```bash
# Basic functionality
curl http://192.168.1.100:8080/

# HEAD request
curl -I http://192.168.1.100:8080/

# 404 error
curl http://192.168.1.100:8080/nonexistent

# Different file types
curl http://192.168.1.100:8080/style.css
curl http://192.168.1.100:8080/script.js

# Load testing with ab (Apache Bench)
ab -n 1000 -c 10 http://192.168.1.100:8080/
```

## References

- [RFC 2616](https://tools.ietf.org/html/rfc2616) - HTTP/1.1
- [RFC 7230](https://tools.ietf.org/html/rfc7230) - HTTP/1.1 Message Syntax
