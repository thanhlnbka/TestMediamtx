// Package websocket provides WebSocket connectivity.
package websocket

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"
	"strings"

	"github.com/gorilla/websocket"
)

var (
	pingInterval = 30 * time.Second
	pingTimeout  = 5 * time.Second
	writeTimeout = 2 * time.Second
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// ServerConn is a server-side WebSocket connection with automatic, periodic ping / pong.
type ServerConn struct {
	wc *websocket.Conn

	// in
	terminate chan struct{}
	write     chan []byte

	// out
	writeErr chan error

	// number connections
	num_conn int 
	url string
}

// NewServerConn allocates a ServerConn.
func NewServerConn(w http.ResponseWriter, req *http.Request) (*ServerConn, error) {
	wc, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		return nil, err
	}

	c := &ServerConn{
		wc:        wc,
		terminate: make(chan struct{}),
		write:     make(chan []byte),
		writeErr:  make(chan error),
		num_conn: 0,
		url: req.URL.String(),
	}

	go c.run()

	return c, nil
}

// Close closes a ServerConn.
func (c *ServerConn) Close() {
	c.wc.Close()
	close(c.terminate)
}

// RemoteAddr returns the remote address.
func (c *ServerConn) RemoteAddr() net.Addr {
	return c.wc.RemoteAddr()
}

func (c *ServerConn) run() {
	c.wc.SetReadDeadline(time.Now().Add(pingInterval + pingTimeout))

	c.wc.SetPongHandler(func(string) error {
		c.wc.SetReadDeadline(time.Now().Add(pingInterval + pingTimeout))
		return nil
	})

	pingTicker := time.NewTicker(pingInterval)
	defer pingTicker.Stop()

	for {
		select {
		case byts := <-c.write:
			c.wc.SetWriteDeadline(time.Now().Add(writeTimeout))
			err := c.wc.WriteMessage(websocket.TextMessage, byts)
			c.writeErr <- err

		case <-pingTicker.C:
			c.wc.SetWriteDeadline(time.Now().Add(writeTimeout))
			c.wc.WriteMessage(websocket.PingMessage, nil)

		case <-c.terminate:
			return
		
		default:
			if c.num_conn == 0 {
				c.wc.SetReadDeadline(time.Now().Add(pingInterval + pingTimeout))

				// Check for incoming messages
				_, message, err := c.wc.ReadMessage()
				if err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
						fmt.Printf("error: %v\n", err)
					}
					c.Close()
					return
				}
				// Handle the incoming message
				fmt.Printf("Url: %s\n", c.url)

				// Extract "stream" from "/stream/ws"
				substr := strings.TrimPrefix(c.url, "/")
				s := strings.Split(substr, "/")
				if len(s) > 0 {
					fmt.Printf("ID %s\n",s[0])
				} else {
					fmt.Println("No match found")
				}

				fmt.Printf("received: %s\n", message)
				// Verify token connection and retry max = 3 if connection refused
				url := "http://localhost:5000/verify_token"
				jsonStr := []byte(fmt.Sprintf(`{"token": "%s", "id": "%s" }`, message, s[0])) // Use the received message as the token
				maxRetries := 3 // maximum number of retries
				retries := 0    // current number of retries

				for retries < maxRetries {
					req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
					if err != nil {
						panic(err)
					}
					req.Header.Set("Content-Type", "application/json")

					client := &http.Client{}
					resp, err := client.Do(req)
					if err != nil {
						fmt.Printf("Error: %v\n", err)
						fmt.Printf("Retrying in 2 seconds...\n")
						time.Sleep(2 * time.Second) // sleep for 2 seconds before retrying
						retries++
						continue
					}

					defer resp.Body.Close()

					if resp.StatusCode == http.StatusOK {
						fmt.Println("Status: OK")
						// Process response here
						break
					} else {
						fmt.Println("Status: Unauthorized")
						// Process error response here
						c.Close()
						break
					}
				}

				if retries == maxRetries {
					fmt.Println("Maximum retries verify reached")
					// Handle maximum retries reached error here
					c.Close()
					return
				}

				c.num_conn = 1
			}
			time.Sleep(1000000 * time.Nanosecond)
			
		}
	}
}

// ReadJSON reads a JSON object.
func (c *ServerConn) ReadJSON(in interface{}) error {
	return c.wc.ReadJSON(in)
}

// WriteJSON writes a JSON object.
func (c *ServerConn) WriteJSON(in interface{}) error {
	byts, err := json.Marshal(in)
	if err != nil {
		return err
	}

	select {
	case c.write <- byts:
		return <-c.writeErr
	case <-c.terminate:
		return fmt.Errorf("terminated")
	}
}
