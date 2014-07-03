package main

import (
	"log"

	"code.google.com/p/go.net/websocket"
)

// A single websocket object. This handles all the logic for a single connection
// out to the web.
type conn struct {
	ws *websocket.Conn
	// Each sending chan gets a 50k buffer
	send chan []byte
}

func (c *conn) writer() {
	for {
		var err error
		select {
		case msg := <-c.send:
			_, err = c.ws.Write(msg)
		}
		if err != nil {
			break
		}
	}
}

func (c *conn) reader() {
	for {
		resp := make([]byte, 256)
		n, err := c.ws.Read(resp)
		if err != nil {
			break
		}
		if n > 0 {
			break
		}
		log.Printf("Received: %x\n", resp[:n])

	}
	log.Println("Closing socket")
	c.ws.Close()
}

func wsHandlerClosure(h *hub) func(*websocket.Conn) {
	wsHandler := func(ws *websocket.Conn) {
		log.Printf("Creating socket for: %s\n", ws.Request().RemoteAddr)
		c := &conn{
			ws:   ws,
			send: make(chan []byte, 5e4),
		}
		h.register <- c
		defer func() { h.unregister <- c }()
		go c.writer()
		c.reader()
	}
	return wsHandler
}
