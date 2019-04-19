package nix

import (
   "blockbook/bchain/coins/btc"
   "blockbook/bchain"
   "bytes"
   "encoding/binary"
   "encoding/hex"
   "encoding/json"
   "math/big"

   vlq "github.com/bsm/go-vlq"
   "github.com/btcsuite/btcd/blockchain"
   "github.com/btcsuite/btcd/chaincfg/chainhash"
   "github.com/btcsuite/btcd/wire"
   "github.com/jakm/btcutil"
   "github.com/jakm/btcutil/chaincfg"
   "github.com/jakm/btcutil/txscript"
)

const (
   // Net Magics
   MainnetMagic wire.BitcoinNet = 0xa3d0cfb6
   TestnetMagic wire.BitcoinNet = 0xc4a7d1a8

   // Dummy TxId for zerocoin
   ZERO_INPUT = "0000000000000000000000000000000000000000000000000000000000000000"

   // Zerocoin op codes
   OP_ZEROCOINMINT  = 0xc1
   OP_ZEROCOINSPEND  = 0xc2

   // Dummy Internal Addresses
   STAKE_ADDR_INT = 0xf7

   // Labels
   ZEROCOIN_LABEL = "Zerocoin Accumulator"
   STAKE_LABEL = "Proof of Stake TX"
)

var (
   MainNetParams chaincfg.Params
   TestNetParams chaincfg.Params
)

func init() {
   // Nix mainnet Address encoding magics
   MainNetParams = chaincfg.MainNetParams
   MainNetParams.Net = MainnetMagic
   MainNetParams.PubKeyHashAddrID = []byte{38}
   MainNetParams.ScriptHashAddrID = []byte{53}
   MainNetParams.PrivateKeyID = []byte{128}
   MainNetParams.Bech32HRPSegwit = "nix"

   // Nix testnet Address encoding magics
   TestNetParams = chaincfg.TestNet3Params
   TestNetParams.Net = TestnetMagic
   TestNetParams.PubKeyHashAddrID = []byte{1}
   TestNetParams.ScriptHashAddrID = []byte{3}
   TestNetParams.PrivateKeyID = []byte{128}
   TestNetParams.Bech32HRPSegwit = "tnix"
}

// NixParser handle
type NixParser struct {
   *btc.BitcoinParser
}

// NewNixParser returns new NixParser instance
func NewNixParser(params *chaincfg.Params, c *btc.Configuration) *NixParser {
   p := &NixParser{BitcoinParser: btc.NewBitcoinParser(params, c)}
   p.OutputScriptToAddressesFunc = p.outputScriptToAddresses
      return p
}

// GetChainParams contains network parameters for the main and test Nix network
func GetChainParams(chain string) *chaincfg.Params {
   if !chaincfg.IsRegistered(&MainNetParams) {
      err := chaincfg.Register(&MainNetParams)
      if err == nil {
         err = chaincfg.Register(&TestNetParams)
      }
      if err != nil {
         panic(err)
      }
   }
   switch chain {
   case "test":
      return &TestNetParams
   default:
      return &MainNetParams
   }
}

// GetAddrDescFromVout returns internal address representation (descriptor) of given transaction output
func (p *NixParser) GetAddrDescFromVout(output *bchain.Vout) (bchain.AddressDescriptor, error) {

   if output.ScriptPubKey.Hex == "" {
      // Stake first output
      return bchain.AddressDescriptor{STAKE_ADDR_INT}, nil
  	}

   // zerocoin mint output
   if len(output.ScriptPubKey.Hex) > 1 && output.ScriptPubKey.Hex[:2] == hex.EncodeToString([]byte{OP_ZEROCOINMINT}) {
      return bchain.AddressDescriptor{OP_ZEROCOINMINT}, nil
	}
   // P2PK/P2PKH outputs
   ad, err := hex.DecodeString(output.ScriptPubKey.Hex)
   if err != nil {
      return ad, err
   }
   // convert possible P2PK script to P2PKH
   // so that all transactions by given public key are indexed together
   return txscript.ConvertP2PKtoP2PKH(ad)
}

// GetAddrDescFromAddress returns internal address representation (descriptor) of given address
func (p *NixParser) GetAddrDescFromAddress(address string) (bchain.AddressDescriptor, error) {
   return p.addressToOutputScript(address)
}

// GetAddressesFromAddrDesc returns addresses for given address descriptor with flag if the addresses are searchable
func (p *NixParser) GetAddressesFromAddrDesc(addrDesc bchain.AddressDescriptor) ([]string, bool, error) {
   return p.OutputScriptToAddressesFunc(addrDesc)
}

// addressToOutputScript converts Nix address to ScriptPubKey
func (p *NixParser) addressToOutputScript(address string) ([]byte, error) {
   // dummy address for stake output
   if address == STAKE_LABEL {
      return bchain.AddressDescriptor{STAKE_ADDR_INT}, nil
	}
   // dummy address for zerocoin mint output
   if address == ZEROCOIN_LABEL {
      return bchain.AddressDescriptor{OP_ZEROCOINMINT}, nil
   }

   // regular address
   da, err := btcutil.DecodeAddress(address, p.Params)
   if err != nil {
      return nil, err
   }
   script, err := txscript.PayToAddrScript(da)
   if err != nil {
      return nil, err
   }
   return script, nil
}

// outputScriptToAddresses converts ScriptPubKey to addresses with a flag that the addresses are searchable
func (p *NixParser) outputScriptToAddresses(script []byte) ([]string, bool, error) {
   // empty script --> newly generated coins
   if len(script) == 0 {
      return nil, false, nil
   }

   // coinstake tx output
   if len(script) > 0 && script[0] == STAKE_ADDR_INT {
      return []string{STAKE_LABEL}, false, nil
   }

   // zerocoin mint output
   ozm := TryParseOPZerocoinMint(script)
   if ozm != "" {
      return []string{ozm}, false, nil
   }

   // basecoin tx output
   sc, addresses, _, err := txscript.ExtractPkScriptAddrs(script, p.Params)

   if err != nil {
      return nil, false, err
   }
   rv := make([]string, len(addresses))

   for i, a := range addresses {
      rv[i] = a.EncodeAddress()
   }
   var s bool

   if sc == txscript.PubKeyHashTy || sc == txscript.WitnessV0PubKeyHashTy ||
   sc == txscript.ScriptHashTy || sc == txscript.WitnessV0ScriptHashTy {
      s = true
   } else if len(addresses) == 0 {
      or := btc.TryParseOPReturn(script)
      if or != "" {
         rv = []string{or}
      }
   }
   return rv, s, nil
}

// TxFromMsgTx returns the transaction from wire msg
func (p *NixParser) TxFromMsgTx(t *wire.MsgTx, tx *Tx, parseAddresses bool) bchain.Tx {
   // Tx Inputs
   vin := make([]bchain.Vin, len(t.TxIn))
   for i, in := range t.TxIn {
      if blockchain.IsCoinBaseTx(t) {
         vin[i] = bchain.Vin{
            Coinbase: hex.EncodeToString(in.SignatureScript),
            Sequence: in.Sequence,
         }
         break
      }

      s := bchain.ScriptSig{
         Hex: hex.EncodeToString(in.SignatureScript),
      }

      vin[i] = bchain.Vin{
         Sequence:  in.Sequence,
         ScriptSig: s,
      }

      // zerocoin spends have no PreviousOutPoint
      if in.SignatureScript[0] != OP_ZEROCOINSPEND {
         vin[i].Txid = in.PreviousOutPoint.Hash.String()
         vin[i].Vout = in.PreviousOutPoint.Index
      }
   }
   // Tx Outputs
   vout := make([]bchain.Vout, len(t.TxOut))
   for i, out := range t.TxOut {
      addrs := []string{}
      if parseAddresses {
         if len(out.PkScript) > 0 {
            addrs, _, _ = p.OutputScriptToAddressesFunc(out.PkScript)
         } else {
               addrs = []string{STAKE_LABEL}
         }
      }

      s := bchain.ScriptPubKey{
         Hex:       hex.EncodeToString(out.PkScript),
         Addresses: addrs,
      }

      var vs big.Int
      vs.SetInt64(out.Value)
      vout[i] = bchain.Vout{
         N:            uint32(i),
         ScriptPubKey: s,
         ValueSat:	  vs,
         Type:         tx.outTypes[i],
      }
   }

   bchaintx := bchain.Tx{
      Txid:     TxHash(t, tx).String(),
      Version:  t.Version,
      LockTime: t.LockTime,
      Vin:      vin,
      Vout:     vout,
   }

   return bchaintx
}

// ParseTx parses byte array containing transaction and returns Tx struct
func (p *NixParser) ParseTx(b []byte) (*bchain.Tx, error) {
   tm := wire.MsgTx{}
   tx := Tx{}
   r := bytes.NewReader(b)
   if err := UnserializeTx(&tm, &tx, r); err != nil {
      return nil, err
   }
   bchaintx := p.TxFromMsgTx(&tm, &tx, true)
   bchaintx.Hex = hex.EncodeToString(b)
   return &bchaintx, nil
}

// ParseBlock parses raw block to our Block struct
func (p *NixParser) ParseBlock(b []byte) (*bchain.Block, error) {
   w := wire.MsgBlock{}
   r := bytes.NewReader(b)
   blk := TxBlock{}
   hashWitnessMerkleRoot := chainhash.Hash{}
   hashAccumulators := chainhash.Hash{}

   if err := UnserializeBlock(&w, &blk, &hashWitnessMerkleRoot, &hashAccumulators,
         r); err != nil {
      return nil, err
   }

   bchaintxs := make([]bchain.Tx, len(w.Transactions))

   for ti, t := range w.Transactions {
      bchaintxs[ti] = p.TxFromMsgTx(t, blk.txs[ti], false)
   }

   return &bchain.Block{
      BlockHeader: bchain.BlockHeader{
         Size: len(b),
         Time: w.Header.Timestamp.Unix(),
      },
      Txs: bchaintxs,
   }, nil
}

// PackTx packs transaction to byte array
func (p *NixParser) PackTx(tx *bchain.Tx, height uint32, blockTime int64) ([]byte, error) {
   buf := make([]byte, 4+vlq.MaxLen64+len(tx.Hex)/2)
   binary.BigEndian.PutUint32(buf[0:4], height)
   vl := vlq.PutInt(buf[4:4+vlq.MaxLen64], blockTime)
   hl, err := hex.Decode(buf[4+vl:], []byte(tx.Hex))
   return buf[0 : 4+vl+hl], err
}

// UnpackTx unpacks transaction from byte array
func (p *NixParser) UnpackTx(buf []byte) (*bchain.Tx, uint32, error) {
   height := binary.BigEndian.Uint32(buf)
   bt, l := vlq.Int(buf[4:])
   tx, err := p.ParseTx(buf[4+l:])
   if err != nil {
      return nil, 0, err
   }
   tx.Blocktime = bt

   return tx, height, nil
}

// TryParseOPZerocoinMint tries to process
// OP_ZEROCOINMINT script and returns its string representation
func TryParseOPZerocoinMint(script []byte) string {
   if len(script) > 0 && script[0] == OP_ZEROCOINMINT {
      return ZEROCOIN_LABEL
   }
   return ""
}

// ParseTxFromJson parses JSON message containing transaction and returns Tx struct
func (p *NixParser) ParseTxFromJson(msg json.RawMessage) (*bchain.Tx, error) {
	var tx bchain.Tx
	err := json.Unmarshal(msg, &tx)
	if err != nil {
		return nil, err
	}
   // fix input (convert to big.Int and clear it)
   for i := range tx.Vin {
      vin := &tx.Vin[i]
      if vin.Denom != "" {
         vin.DenomSat, _ = p.AmountToBigInt(vin.Denom)
         vin.Denom = ""
      }
   }

	for i := range tx.Vout {
		vout := &tx.Vout[i]
		// convert vout.JsonValue to big.Int and clear it, it is only temporary value used for unmarshal
		vout.ValueSat, err = p.AmountToBigInt(vout.JsonValue)
      	// convert type string to number
      	switch vout.Type_str {
      	case "null" :
      		vout.Type = OUTPUT_NULL
      	default:
         	vout.Type = OUTPUT_STANDARD
      	}
		vout.JsonValue = ""
	}

	return &tx, nil
}

// PackTxid packs txid to byte array
func (p *NixParser) PackTxid(txid string) ([]byte, error) {
   if txid == "" || txid == ZERO_INPUT {
      return nil, bchain.ErrTxidMissing
	}
	return p.BitcoinParser.PackTxid(txid)
}
