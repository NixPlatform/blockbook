package nix

import (
	"blockbook/bchain"
	"blockbook/bchain/coins/btc"
	"encoding/json"
	"errors"

	"github.com/golang/glog"
)

// NixRPC is an interface to JSON-RPC bitcoind service.
type NixRPC struct {
	*btc.BitcoinRPC
}

// NewNixRPC returns new NixRPC instance.
func NewNixRPC(config json.RawMessage, pushHandler func(bchain.NotificationType)) (bchain.BlockChain, error) {
	b, err := btc.NewBitcoinRPC(config, pushHandler)
	if err != nil {
		return nil, err
	}

	s := &NixRPC{
		b.(*btc.BitcoinRPC),
	}
	s.RPCMarshaler = btc.JSONMarshalerV2{}

	return s, nil
}

// Initialize initializes NixRPC instance.
func (b *NixRPC) Initialize() error {
	ci, err := b.GetChainInfo()
	if err != nil {
		return err
	}
	chainName := ci.Chain

	glog.Info("Chain name ", chainName)
	params := GetChainParams(chainName)

	// always create parser
	b.Parser = NewNixParser(params, b.ChainConfig)

	// parameters for getInfo request
	if params.Net == MainnetMagic {
		b.Testnet = false
		b.Network = "livenet"
	} else {
		b.Testnet = true
		b.Network = "testnet"
	}

	glog.Info("rpc: block chain ", params.Name)

	return nil
}

// GetBlock returns block with given hash.
func (g *NixRPC) GetBlock(hash string, height uint32) (*bchain.Block, error) {
   var err error
   if hash == "" && height > 0 {
      hash, err = g.GetBlockHash(height)
      if err != nil {
         return nil, err
      }
   }

   glog.V(1).Info("rpc: getblock (verbosity=1) ", hash)

   bi, err := g.GetBlockInfo(hash)
   if err != nil {
      return nil, err
   }

   txs := make([]bchain.Tx, 0, len(bi.Txids))
   for _, txid := range bi.Txids {
      tx, err := g.GetTransaction(txid)
      if err != nil {
		  if err == bchain.ErrTxNotFound {
			  glog.Errorf("rpc: getblock: skipping transanction in block %s due error: %s", hash, err)
			  continue
		  }
         return nil, err
      }
      txs = append(txs, *tx)
   }

   // block is PoS when nonce is zero
   //blocktype := 1

   block := &bchain.Block{
      BlockHeader: bi.BlockHeader,
      Txs:         txs,
      //Type:        blocktype,
   }
   return block, nil
}

//func isInvalidTx(err error) bool {
//   switch e1 := err.(type) {
//   case *errors.Err:
//      switch e2 := e1.Cause().(type) {
//      case *bchain.RPCError:
//         if e2.Code == -5 { // "No information available about transaction"
//            return true
//         }
//      }
//   }
//
//   return false
//}

// GetTransactionForMempool returns a transaction by the transaction ID.
// It could be optimized for mempool, i.e. without block time and confirmations
//func (p *NixRPC) GetTransactionForMempool(txid string) (*bchain.Tx, error) {
//   return p.GetTransaction(txid)
//}
// GetTransactionForMempool returns a transaction by the transaction ID.
// It could be optimized for mempool, i.e. without block time and confirmations
func (b *NixRPC) GetTransactionForMempool(txid string) (*bchain.Tx, error) {
	return b.GetTransaction(txid)
}

//// GetTransaction returns a transaction by the transaction ID.
//func (p *NixRPC) GetTransaction(txid string) (*bchain.Tx, error) {
//   if txid == ZERO_INPUT {
//      return nil, bchain.ErrTxidMissing
//   }
//	r, err := p.GetTransactionSpecific(txid)
//	if err != nil {
//		return nil, err
//	}
//	tx, err := p.Parser.ParseTxFromJson(r)
//	if err != nil {
//		return nil, errors.Annotatef(err, "txid %v", txid)
//	}
//	return tx, nil
//}
//
//// GetMempoolEntry returns mempool data for given transaction
//func (p *NixRPC) GetMempoolEntry(txid string) (*bchain.MempoolEntry, error) {
//   return nil, errors.New("GetMempoolEntry: not implemented")
//}
//
//func isErrBlockNotFound(err *bchain.RPCError) bool {
//   return err.Message == "Block not found" || err.Message == "Block height out of range"
//}
