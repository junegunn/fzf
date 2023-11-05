package fzf

import (
	"bufio"
	"bytes"
	"crypto/subtle"
	"errors"
	"fmt"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var getRegex *regexp.Regexp

func init() {
	getRegex = regexp.MustCompile(`^GET /(?:\?([a-z0-9=&]+))? HTTP`)
}

type getParams struct {
	limit  int
	offset int
}

const (
	crlf             = "\r\n"
	httpOk           = "HTTP/1.1 200 OK" + crlf
	httpBadRequest   = "HTTP/1.1 400 Bad Request" + crlf
	httpUnauthorized = "HTTP/1.1 401 Unauthorized" + crlf
	httpReadTimeout  = 10 * time.Second
	maxContentLength = 1024 * 1024
)

type httpServer struct {
	apiKey          []byte
	actionChannel   chan []*action
	responseChannel chan string
}

type listenAddress struct {
	host string
	port int
}

func (addr listenAddress) IsLocal() bool {
	return addr.host == "localhost" || addr.host == "127.0.0.1"
}

var defaultListenAddr = listenAddress{"localhost", 0}

func parseListenAddress(address string) (error, listenAddress) {
	parts := strings.SplitN(address, ":", 3)
	if len(parts) == 1 {
		parts = []string{"localhost", parts[0]}
	}
	if len(parts) != 2 {
		return fmt.Errorf("invalid listen address: %s", address), defaultListenAddr
	}
	portStr := parts[len(parts)-1]
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 0 || port > 65535 {
		return fmt.Errorf("invalid listen port: %s", portStr), defaultListenAddr
	}
	if len(parts[0]) == 0 {
		parts[0] = "localhost"
	}
	return nil, listenAddress{parts[0], port}
}

func startHttpServer(address listenAddress, actionChannel chan []*action, responseChannel chan string) (error, int) {
	host := address.host
	port := address.port
	apiKey := os.Getenv("FZF_API_KEY")
	if !address.IsLocal() && len(apiKey) == 0 {
		return fmt.Errorf("FZF_API_KEY is required to allow remote access"), port
	}
	addrStr := fmt.Sprintf("%s:%d", host, port)
	listener, err := net.Listen("tcp", addrStr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s", addrStr), port
	}
	if port == 0 {
		addr := listener.Addr().String()
		parts := strings.Split(addr, ":")
		if len(parts) < 2 {
			return fmt.Errorf("cannot extract port: %s", addr), port
		}
		var err error
		port, err = strconv.Atoi(parts[len(parts)-1])
		if err != nil {
			return err, port
		}
	}

	server := httpServer{
		apiKey:          []byte(apiKey),
		actionChannel:   actionChannel,
		responseChannel: responseChannel,
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
	answer := func(code string, message string) string {
		message += "\n"
		return code + fmt.Sprintf("Content-Length: %d%s", len(message), crlf+crlf+message)
	}
	unauthorized := func(message string) string {
		return answer(httpUnauthorized, message)
	}
	bad := func(message string) string {
		return answer(httpBadRequest, message)
	}
	good := func(message string) string {
		return answer(httpOk+"Content-Type: application/json"+crlf, message)
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
			getMatch := getRegex.FindStringSubmatch(text)
			if len(getMatch) > 0 {
				server.actionChannel <- []*action{{t: actResponse, a: getMatch[1]}}
				response := <-server.responseChannel
				return good(response)
			} else if !strings.HasPrefix(text, "POST / HTTP") {
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

	server.actionChannel <- actions
	return httpOk
}

func parseGetParams(query string) getParams {
	params := getParams{limit: 100, offset: 0}
	for _, pair := range strings.Split(query, "&") {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			switch parts[0] {
			case "limit":
				val, err := strconv.Atoi(parts[1])
				if err == nil {
					params.limit = val
				}
			case "offset":
				val, err := strconv.Atoi(parts[1])
				if err == nil {
					params.offset = val
				}
			}
		}
	}
	return params
}
