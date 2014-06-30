/* This example pushes non pay to public key hash transactions into a sqlite database.
   The sql is defined in create.sql, but there is a sample db already instantiated for
   you in sample.db.
*/
package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/NSkelsey/watchtower"
	"github.com/conformal/btcscript"
	"github.com/conformal/btcwire"
)

var conn *sql.DB

func main() {
	addr := "127.0.0.1:18333"
	testnet := btcwire.TestNet3
	var err error
	conn, err = sql.Open("sqlite3", "sample.db")
	if err != nil {
		log.Fatal(err)
	}

	watchtower.Create(addr, testnet, txParser, blockParser)
}

// example parsers

// The block parser only logs stores blocks as they come in
func blockParser(now time.Time, block *btcwire.MsgBlock) {
	_hash, _ := block.BlockSha()
	hash := _hash.String()
	log.Println(hash)
	prevhash := block.Header.PrevBlock.String()
	_, err := conn.Exec("INSERT INTO blocks VALUES ($1, $2)", hash, prevhash)
	if err != nil {
		log.Fatal(err)
	}
}

// This tx parser writes transactions into a sqlite3 db
func txParser(meta *watchtower.TxMeta) {
	sha, _ := meta.MsgTx.TxSha()
	log.Println(sha.String())
	if filter(meta.MsgTx) {
		args := dbArgs(meta)
		_, err := conn.Exec("INSERT OR REPLACE INTO transactions VALUES ($1, $2, $3)",
			args.Txid, args.Blockid, args.Raw)
		if err != nil {
			log.Fatal(err)
		}
	}
}

type TxArgs struct {
	Txid    string
	Blockid string
	Raw     []byte
}

// get the database args into a storable state
func dbArgs(meta *watchtower.TxMeta) TxArgs {
	txid, _ := meta.MsgTx.TxSha()
	var blockid string
	if meta.BlockSha != nil {
		blockid = fmt.Sprintf("%x", meta.BlockSha)
	} else {
		blockid = "nil"
	}
	buf := bytes.NewBuffer(make([]byte, 0, meta.MsgTx.SerializeSize()))
	meta.MsgTx.Serialize(buf)
	raw := buf.Bytes()
	return TxArgs{
		Txid:    txid.String(),
		Blockid: blockid,
		Raw:     raw,
	}
}

// Only log transactions that are not of type "pubkeyhash"
func filter(tx *btcwire.MsgTx) bool {
	for _, txout := range tx.TxOut {
		script := txout.PkScript
		scriptclass := btcscript.GetScriptClass(script).String()
		if scriptclass != "pubkeyhash" {
			return true
		}
	}
	return true
}
