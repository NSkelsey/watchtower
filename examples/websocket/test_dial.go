package main

import (
	"flag"
	"fmt"
	"log"

	"code.google.com/p/go.net/websocket"
)

var (
	host = flag.String("host", "localhost:1034/ws", "The websocket host + path")
)

func main() {
	flag.Parse()

	url := "ws://" + *host

	origin := "http://localhost/"
	ws, err := websocket.Dial(url, "", origin)
	if err != nil {
		log.Fatal(err)
	}

	ws.Write([]byte("Hello world!\n"))

	for {
		var msg = make([]byte, 2048)

		var n int
		if n, err = ws.Read(msg); err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Received: %s\n", msg[:n])
	}
}
