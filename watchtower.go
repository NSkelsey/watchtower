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
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/conformal/btcwire"
)

var btcnet btcwire.BitcoinNet

// The meta information reported by watchtower about a transaction it has seen.
type TxMeta struct {
	MsgTx    *btcwire.MsgTx
	BlockSha []byte
	Time     time.Time
}

// The initial configuration for the watchtower instance.
type TowerCfg struct {
	Addr        string
	Net         btcwire.BitcoinNet
	StartHeight int
	// A list of messages to send the bitcoin peer
	ToSend []btcwire.Message
	Logger *log.Logger
}

// Params required to construct a connection to a bitcoin peer
type connParams struct {
	conn net.Conn
	// the protocol version to communicate at
	pver   uint32
	btcnet btcwire.BitcoinNet
	logger *log.Logger
	// indicates that conn is in handshake phase.
	insetup bool
}

func nodeHandler(cfg TowerCfg, txStream chan<- *TxMeta, blockStream chan<- *btcwire.MsgBlock) {

	conn := setupConn(cfg.Addr, cfg.Logger)

	connparams := connParams{
		conn:    conn,
		pver:    btcwire.ProtocolVersion,
		btcnet:  cfg.Net,
		logger:  cfg.Logger,
		insetup: true,
	}

	read, write := composeConnOuts(connparams)

	// Initial handshake
	ver_m, _ := btcwire.NewMsgVersionFromConn(conn, genNonce(), int32(cfg.StartHeight))
	ver_m.AddUserAgent("Watchtower", "0.0.0")

	write(ver_m)

	// Wait for responses
	acked, responded := false, false
	for {
		var msg btcwire.Message
		msg = read()
		cfg.Logger.Println(msg.Command())
		switch msg := msg.(type) {
		case *btcwire.MsgVersion:
			responded = true
			ack := btcwire.NewMsgVerAck()
			nodeVer := uint32(msg.ProtocolVersion)
			if nodeVer < connparams.pver {
				connparams.pver = nodeVer
				read, write = composeConnOuts(connparams)
			}
			write(ack)
		case *btcwire.MsgVerAck:
			acked = true
		}
		if responded && acked {
			break
		}
	}

	// We are through the initial handshake, assume functional channel from here
	// on out. If there any errors with the pipe logger.Fatal gets called.
	connparams.insetup = false
	read, write = composeConnOuts(connparams)
	cfg.Logger.Println("Conn Negotiated")

	// If there are msgs to send. Send them first
	if len(cfg.ToSend) > 0 {
		s := "Sending cmds:"
		for _, msg := range cfg.ToSend {
			s += " " + msg.Command()
		}
		cfg.Logger.Println(s)
	}
	for _, msg := range cfg.ToSend {
		write(msg)
	}

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
			// More fun than anything...
			write(pong)
		}
	}
}

func Create(cfg TowerCfg, txParser func(*TxMeta), blockParser func(time.Time, *btcwire.MsgBlock)) {
	btcnet = cfg.Net

	if cfg.Logger == nil {
		// if no logger is present, create one and pass it into our config
		cfg.Logger = log.New(os.Stdout, "", log.Ltime)
	}

	txStream := make(chan *TxMeta, 5e3)
	blockStream := make(chan *btcwire.MsgBlock)

	go blockParserWrapper(blockParser, blockStream, txStream)

	go txParserWrapper(txParser, txStream)

	nodeHandler(cfg, txStream, blockStream)
}

// Utility functions

// Dials a Bitcoin server over tcp, times out in .5 seconds
func setupConn(addr string, logger *log.Logger) net.Conn {

	logger.Println("Connecting to:", addr)
	conn, err := net.DialTimeout("tcp", addr, time.Millisecond*500)
	if err != nil {
		logger.Fatalf(fmt.Sprintf("%v, could not connect to %s", err, addr))
	}
	return conn
}

// Creates the reading and writing interfaces used to send and recieve messages
// from a Bitcoin peer. inSetup indicates the verbosity of the failure mode.
func composeConnOuts(params connParams) (func() btcwire.Message, func(btcwire.Message)) {

	conn := params.conn
	logger := params.logger

	failmsg := "The Bitcoin server has reached its max peers and does not want to talk"

	read := func() btcwire.Message {
		msg, _, err := btcwire.ReadMessage(conn, params.pver, params.btcnet)
		// a Bitcoin node will typically send a TCP RST if it doesn't want to talk
		if params.insetup && err != nil {
			logger.Printf(failmsg)
		}
		if err != nil {
			logger.Fatal(err)
		}
		return msg
	}
	write := func(msg btcwire.Message) {
		err := btcwire.WriteMessage(conn, msg, params.pver, params.btcnet)
		if params.insetup && err != nil {
			logger.Printf(failmsg)
		}
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
