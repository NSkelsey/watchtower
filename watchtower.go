/*
	This program connects to a bitcoin node and streams all txs and blocks through
	configurable parsers that do interesting things with that output.
	The data flow looks like this:
    nodeHandler->------------> blockParser->----
		          \		    					\
                   ------------------------------->txParser
*/

package watchtower

import (
	"log"
	"net"
	"os"
	"time"

	"github.com/conformal/btcwire"
)

var btcnet = btcwire.TestNet3
var logger = log.New(os.Stdout, "", log.Ltime)

type TxMeta struct {
	MsgTx    *btcwire.MsgTx
	BlockSha []byte
}

func init() {
	logger.Println("Processing config...")
}

func nodeHandler(addr string, txStream chan<- *TxMeta, blockStream chan<- *btcwire.MsgBlock) {
	logger.Println("Connecting to: ", addr)
	conn := setupConn(addr)
	pver := btcwire.ProtocolVersion
	read, write := composeConnOuts(conn, pver, btcnet, logger)

	// Initial handshake
	ver_m, _ := btcwire.NewMsgVersionFromConn(conn, genNonce(), 0)
	ver_m.AddUserAgent("Watchtower", "0.0.0")
	write(ver_m)
	// Wait for responses
	acked, responded := false, false
	for {
		var msg btcwire.Message
		msg = read()
		logger.Println(msg.Command())
		switch msg := msg.(type) {
		case *btcwire.MsgVersion:
			responded = true
			ack := btcwire.NewMsgVerAck()
			nodeVer := uint32(msg.ProtocolVersion)
			if nodeVer < pver {
				read, write = composeConnOuts(conn, nodeVer, btcnet, logger)
			}
			write(ack)
		case *btcwire.MsgVerAck:
			acked = true
		}
		if responded && acked {
			break
		}
	}
	logger.Println("Conn Negotiated")
	// listen for txs + blocks then pushes into streams
	for {
		msg := read()
		logger.Println(msg.Command())
		switch msg := msg.(type) {
		case *btcwire.MsgInv:
			want := btcwire.NewMsgGetData()
			invVec := msg.InvList
			for i := range invVec {
				chunk := invVec[i]
				want.AddInvVect(chunk)
			}
			write(want)
		case *btcwire.MsgTx:
			var empt []byte // evaluates to nil
			meta := TxMeta{MsgTx: msg, BlockSha: empt}
			txStream <- &meta
		case *btcwire.MsgBlock:
			blockStream <- msg
		case *btcwire.MsgPing:
			pong := btcwire.NewMsgPong(msg.Nonce)
			write(pong)
		}
	}
}

func Create(txParser func(*TxMeta), blockParser func(*btcwire.MsgBlock)) {
	addr := "127.0.0.1:18333"

	txStream := make(chan *TxMeta, 5e3)
	blockStream := make(chan *btcwire.MsgBlock)

	go blockParserWrapper(blockParser, blockStream, txStream)

	go txParserWrapper(txParser, txStream)

	nodeHandler(addr, txStream, blockStream)
}

// Utility functions
func setupConn(addr string) net.Conn {
	conn, err := net.DialTimeout("tcp", addr, time.Millisecond*500)
	if err != nil {
		logger.Fatal(err, "Could not connect to "+addr)
	}
	return conn
}

func composeConnOuts(conn net.Conn, pver uint32, btcnet btcwire.BitcoinNet, logger *log.Logger) (func() btcwire.Message, func(btcwire.Message)) {
	read := func() btcwire.Message {
		msg, _, err := btcwire.ReadMessage(conn, pver, btcnet)
		if err != nil {
			logger.Fatal(err)
		}
		return msg
	}
	write := func(msg btcwire.Message) {
		err := btcwire.WriteMessage(conn, msg, pver, btcnet)
		if err != nil {
			logger.Fatal(err)
		}
	}
	return read, write
}

func genNonce() uint64 {
	n, _ := btcwire.RandomUint64()
	return n
}
