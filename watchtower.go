/*
	This program connects to a bitcoin node and streams all txs and blocks through
	configurable parsers that do interesting things with that output.
	The data flow looks like this:
    nodeHandler->------------> blockParser->----
                  \                              \
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

var btcnet btcwire.BitcoinNet
var logger = log.New(os.Stdout, "", log.Ltime)

type TxMeta struct {
	MsgTx    *btcwire.MsgTx
	BlockSha []byte
	Time     time.Time
}

func nodeHandler(addr string, height int64, txStream chan<- *TxMeta, blockStream chan<- *btcwire.MsgBlock) {
	logger.Println("Connecting to: ", addr)
	conn := setupConn(addr)
	pver := btcwire.ProtocolVersion
	read, write := composeConnOuts(conn, pver, btcnet, logger)

	// Initial handshake
	ver_m, _ := btcwire.NewMsgVersionFromConn(conn, genNonce(), int32(height))
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
			meta := TxMeta{MsgTx: msg, BlockSha: empt, Time: time.Now()}
			txStream <- &meta
		case *btcwire.MsgBlock:
			blockStream <- msg
		case *btcwire.MsgPing:
			pong := btcwire.NewMsgPong(msg.Nonce)
			write(pong)
		}
	}
}

type TowerCfg struct {
	Addr        string
	Net         btcwire.BitcoinNet
	StartHeight int
}

func Create(cfg TowerCfg, txParser func(*TxMeta), blockParser func(time.Time, *btcwire.MsgBlock)) {
	btcnet = cfg.Net

	txStream := make(chan *TxMeta, 5e3)
	blockStream := make(chan *btcwire.MsgBlock)

	go blockParserWrapper(blockParser, blockStream, txStream)

	go txParserWrapper(txParser, txStream)

	nodeHandler(cfg.Addr, int64(cfg.StartHeight), txStream, blockStream)
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
