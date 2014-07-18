// This runs a small bitcoin node that acts as a websocket dumping transaction +
// block data into connected webbrowsers
package main

import (
	"flag"
	"log"
	"net/http"

	"code.google.com/p/go.net/websocket"

	"github.com/NSkelsey/btcbuilder"
	"github.com/NSkelsey/watchtower"
)

var (
	addr     = flag.String("addr", ":1034", "http service address")
	btcnet   = flag.String("net", "TestNet3", "Which network params to use")
	nodeaddr = flag.String("nodeaddr", "127.0.0.1:18333", "The node to connect to")
)

// Main does 3 things.
// It starts a hub that has txJson channels in and out for sending txs from
// the bitcoin network into connected websockets. It starts the bitcoin listening
// side by creating a watchtower and passing a reference to that hub. Finally it
// starts listening for websockect connections to pass to the hub.
func main() {
	flag.Parse()

	net, err := btcbuilder.NetParamsFromStr(*btcnet)
	if err != nil {
		log.Fatal(err)
	}

	cfg := watchtower.TowerCfg{
		Addr:        *nodeaddr,
		Net:         net.Net,
		StartHeight: 0,
	}

	hub := &hub{
		btcpipe:     make(chan []byte, 256),
		register:    make(chan *conn),
		unregister:  make(chan *conn),
		connections: make(map[*conn]struct{}),
	}

	go hub.run()
	go watchtower.Create(cfg, txClosure(hub), blockClosure(hub))

	http.Handle("/ws", websocket.Handler(wsHandlerClosure(hub)))
	log.Printf("Listening for conns at: %s%s\n", *addr, "/ws")
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal("Http service died: ", err.Error())
	}
}
