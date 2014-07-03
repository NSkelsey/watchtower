package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/NSkelsey/btcbuilder"
	"github.com/NSkelsey/watchtower"
	"github.com/conformal/btcwire"
)

type TxJson struct {
	Type string `json:"type"`
	Kind string `json:"kind"`
	Size int    `json:"size"`
	Txid string `json:"txid"`
	Time int64  `json:"time"`
}

type BlockJson struct {
	Type string `json:"type"`
	Hash string `json:"hash"`
	Size int    `json:"size"`
	Time int64  `json:"time"`
}

func txClosure(h *hub) func(*watchtower.TxMeta) {
	txParser := func(meta *watchtower.TxMeta) {
		// This parser skips txs that are in blocks.
		if meta.BlockSha != nil {
			return
		}

		kind := btcbuilder.SelectKind(meta.MsgTx)
		txid, _ := meta.MsgTx.TxSha()

		log.Println("Saw tx: ", txid.String())
		txMsg := &TxJson{
			Type: "tx",
			Kind: kind,
			Size: meta.MsgTx.SerializeSize(),
			Txid: txid.String(),
			Time: meta.Time.Unix(),
		}

		bytes, err := json.Marshal(txMsg)
		if err != nil {
			log.Fatal(err)
		}
		h.btcpipe <- bytes

	}
	return txParser
}

func blockClosure(h *hub) func(time.Time, *btcwire.MsgBlock) {
	blockParser := func(now time.Time, block *btcwire.MsgBlock) {
		hash, _ := block.BlockSha()
		log.Println("Saw block: ", hash.String())

		blockMsg := &BlockJson{
			Type: "block",
			Hash: hash.String(),
			Size: block.SerializeSize(),
			Time: now.Unix(),
		}
		bytes, _ := json.Marshal(blockMsg)

		h.btcpipe <- bytes
	}
	return blockParser
}
