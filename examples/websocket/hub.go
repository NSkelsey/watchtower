package main

type hub struct {
	// The upstream sources of txs and blocks to send to connected sockets
	btcpipe chan []byte

	// Adds new websockets
	register chan *conn

	// Closes old websockets
	unregister chan *conn

	// The live connection map -- maps to the empty struct
	connections map[*conn]struct{}
}

func (h *hub) run() {
	for {
		select {
		case c := <-h.register:
			h.connections[c] = struct{}{}
		case c := <-h.unregister:
			delete(h.connections, c)
			c.ws.Close()
		case msg := <-h.btcpipe:
			for c := range h.connections {
				select {
				case c.send <- msg:
				default:
					delete(h.connections, c)
					c.ws.Close()
				}
			}
		}
	}
}
