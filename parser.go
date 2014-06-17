package watchtower

import (
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
			meta := TxMeta{MsgTx: tx, BlockSha: bytes}
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

//
//	// example parsers
//	func blockParser(block *btcwire.MsgBlock) {
//		conn, _ = sqlite3.Open("dat.db")
//		hash, _ := block.BlockSha()
//		logger.Println(hash.String())
//		args := sqlite3.NamedArgs{
//			"$hash":     hash.String(),
//			"$prevhash": block.Header.PrevBlock.String(),
//		}
//		err := conn.Exec("INSERT INTO blocks VALUES ($hash, $prevhash)", args)
//		if err != nil {
//			logger.Fatal(err)
//		}
//	}
//
//	func txParser(meta *TxMeta) {
//		sha, _ := meta.MsgTx.TxSha()
//		logger.Println(sha.String())
//		if filter(meta.MsgTx) {
//			args := meta.SqliteArgs()
//			err := conn.Exec("INSERT OR REPLACE INTO transactions VALUES ($txid, $block, $raw)", args)
//			if err != nil {
//				logger.Fatal(err)
//			}
//		}
//	}
//
//	func (meta *TxMeta) SqliteArgs() sqlite3.NamedArgs {
//		txid, _ := meta.MsgTx.TxSha()
//		var blockid string
//		if meta.BlockHash != nil {
//			blockid = meta.BlockHash.String()
//		} else {
//			blockid = "nil"
//		}
//		buf := bytes.NewBuffer(make([]byte, 0, meta.MsgTx.SerializeSize()))
//		meta.MsgTx.Serialize(buf)
//		raw := buf.Bytes()
//		return sqlite3.NamedArgs{
//			"$txid":  txid.String(),
//			"$block": blockid,
//			"$raw":   raw,
//		}
//	}
//
//	func filter(tx *btcwire.MsgTx) bool {
//		//var AhimsaMagic []byte = []byte{0xde, 0xad, 0xbe, 0xef}
//		script := tx.TxOut[0].PkScript
//		if len(script) == 26 { //  && script[3:7] == AhimsaMagic {
//			return true
//		} else {
//			return true
//		}
//	}
