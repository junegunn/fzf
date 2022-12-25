package fzf

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	crlf             = "\r\n"
	httpPattern      = "^(GET|POST) (/[^ ]*) HTTP"
	httpOk           = "HTTP/1.1 200 OK" + crlf
	httpBadRequest   = "HTTP/1.1 400 Bad Request" + crlf
	httpReadTimeout  = 10 * time.Second
	maxContentLength = 1024 * 1024
)

var (
	httpRegexp *regexp.Regexp
)

func startHttpServer(port int, requestChan chan []*action, responseChan chan string) error {
	if port == 0 {
		return nil
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("port not available: %d", port)
	}

	httpRegexp = regexp.MustCompile(httpPattern)
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
			conn.Write([]byte(handleHttpRequest(conn, requestChan, responseChan)))
			conn.Close()
		}
		listener.Close()
	}()

	return nil
}

// Here we are writing a simplistic HTTP server without using net/http
// package to reduce the size of the binary.
//
// * No --listen:            2.8MB
// * --listen with net/http: 5.7MB
// * --listen w/o net/http:  3.3MB
func handleHttpRequest(conn net.Conn, requestChan chan []*action, responseChan chan string) string {
	contentLength := 0
	body := ""
	response := func(header string, message string) string {
		return header + fmt.Sprintf("Content-Length: %d%s", len(message), crlf+crlf+message)
	}
	bad := func(message string) string {
		return response(httpBadRequest, strings.TrimSpace(message)+"\n")
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
			httpMatch := httpRegexp.FindStringSubmatch(text)
			if len(httpMatch) != 3 {
				return bad("invalid HTTP request: " + text)
			}
			if httpMatch[1] == "GET" {
				requestChan <- []*action{{t: actEvaluate, a: httpMatch[2][1:]}}
				return response(httpOk, <-responseChan)
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
			if len(pair) == 2 && strings.ToLower(pair[0]) == "content-length" {
				length, err := strconv.Atoi(strings.TrimSpace(pair[1]))
				if err != nil || length <= 0 || length > maxContentLength {
					return bad("invalid content length")
				}
				contentLength = length
			}
		case 2:
			body += text
		}
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
	requestChan <- actions
	return httpOk
}
