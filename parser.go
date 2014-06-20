package watchtower

import (
	"time"

	"code.google.com/p/go-sqlite/go1/sqlite3"
	"github.com/conformal/btcwire"
)

var conn *sqlite3.Conn

func blockParserWrapper(blockParser func(*btcwire.MsgBlock), blockStream <-chan *btcwire.MsgBlock, txStream chan<- *TxMeta) {
	for {
		block := <-blockStream
		blockParser(block)
		hash, _ := block.BlockSha()
		bytes := hash.Bytes()
		for _, tx := range block.Transactions {
			meta := TxMeta{
				MsgTx:    tx,
				BlockSha: bytes,
				Time:     time.Now()}
			txStream <- &meta
		}
	}
}

func txParserWrapper(txParser func(*TxMeta), txStream <-chan *TxMeta) {
	for {
		txMeta := <-txStream
		txParser(txMeta)
	}
}
