package fzf

import (
	"bufio"
	"bytes"
	"crypto/subtle"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	crlf             = "\r\n"
	httpOk           = "HTTP/1.1 200 OK" + crlf
	httpBadRequest   = "HTTP/1.1 400 Bad Request" + crlf
	httpUnauthorized = "HTTP/1.1 401 Unauthorized" + crlf
	httpReadTimeout  = 10 * time.Second
	maxContentLength = 1024 * 1024
)

type httpServer struct {
	apiKey  []byte
	channel chan []*action
}

func startHttpServer(port int, channel chan []*action) (error, int) {
	if port < 0 {
		return nil, port
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		return fmt.Errorf("port not available: %d", port), port
	}
	if port == 0 {
		addr := listener.Addr().String()
		parts := strings.SplitN(addr, ":", 2)
		if len(parts) < 2 {
			return fmt.Errorf("cannot extract port: %s", addr), port
		}
		var err error
		port, err = strconv.Atoi(parts[1])
		if err != nil {
			return err, port
		}
	}

	server := httpServer{
		apiKey:  []byte(os.Getenv("FZF_API_KEY")),
		channel: channel,
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					break
				} else {
					continue
				}
			}
			conn.Write([]byte(server.handleHttpRequest(conn)))
			conn.Close()
		}
		listener.Close()
	}()

	return nil, port
}

// Here we are writing a simplistic HTTP server without using net/http
// package to reduce the size of the binary.
//
// * No --listen:            2.8MB
// * --listen with net/http: 5.7MB
// * --listen w/o net/http:  3.3MB
func (server *httpServer) handleHttpRequest(conn net.Conn) string {
	contentLength := 0
	apiKey := ""
	body := ""
	unauthorized := func(message string) string {
		message += "\n"
		return httpUnauthorized + fmt.Sprintf("Content-Length: %d%s", len(message), crlf+crlf+message)
	}
	bad := func(message string) string {
		message += "\n"
		return httpBadRequest + fmt.Sprintf("Content-Length: %d%s", len(message), crlf+crlf+message)
	}
	conn.SetReadDeadline(time.Now().Add(httpReadTimeout))
	scanner := bufio.NewScanner(conn)
	scanner.Split(func(data []byte, atEOF bool) (int, []byte, error) {
		found := bytes.Index(data, []byte(crlf))
		if found >= 0 {
			token := data[:found+len(crlf)]
			return len(token), token, nil
		}
		if atEOF || len(body)+len(data) >= contentLength {
			return 0, data, bufio.ErrFinalToken
		}
		return 0, nil, nil
	})

	section := 0
	for scanner.Scan() {
		text := scanner.Text()
		switch section {
		case 0:
			if !strings.HasPrefix(text, "POST / HTTP") {
				return bad("invalid request method")
			}
			section++
		case 1:
			if text == crlf {
				if contentLength == 0 {
					return bad("content-length header missing")
				}
				section++
				continue
			}
			pair := strings.SplitN(text, ":", 2)
			if len(pair) == 2 {
				switch strings.ToLower(pair[0]) {
				case "content-length":
					length, err := strconv.Atoi(strings.TrimSpace(pair[1]))
					if err != nil || length <= 0 || length > maxContentLength {
						return bad("invalid content length")
					}
					contentLength = length
				case "x-api-key":
					apiKey = strings.TrimSpace(pair[1])
				}
			}
		case 2:
			body += text
		}
	}

	if len(server.apiKey) != 0 && subtle.ConstantTimeCompare([]byte(apiKey), server.apiKey) != 1 {
		return unauthorized("invalid api key")
	}

	if len(body) < contentLength {
		return bad("incomplete request")
	}
	body = body[:contentLength]

	errorMessage := ""
	actions := parseSingleActionList(strings.Trim(string(body), "\r\n"), func(message string) {
		errorMessage = message
	})
	if len(errorMessage) > 0 {
		return bad(errorMessage)
	}
	if len(actions) == 0 {
		return bad("no action specified")
	}

	server.channel <- actions
	return httpOk
}
