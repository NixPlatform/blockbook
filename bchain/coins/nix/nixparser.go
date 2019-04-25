package nix

import (
   "blockbook/bchain"
   "blockbook/bchain/coins/btc"
   "bytes"
   "encoding/binary"
   "encoding/hex"
   "encoding/json"
   "io"
   "log"
   "log/syslog"
   "math/big"

   "github.com/golang/glog"
   "github.com/martinboehm/btcd/wire"
   "github.com/martinboehm/btcutil"
   "github.com/martinboehm/btcutil/chaincfg"
   "github.com/martinboehm/btcutil/txscript"
)

const (
   MainnetMagic wire.BitcoinNet = 0xa3d0cfb6
   TestnetMagic wire.BitcoinNet = 0xa3d0cfb6
   RegtestMagic wire.BitcoinNet = 0xdab5bffc

   // Dummy TxId for zerocoin
   ZERO_INPUT = "0000000000000000000000000000000000000000000000000000000000000000"

   // Zerocoin op codes
   OP_ZEROCOINMINT  = 193
   OP_ZEROCOINSPEND  = 194
   OP_COINSTAKE = 184
   OP_HASH160 = 169
   OP_0 = 0

   // Number of blocks per budget cycle
   nBlocksPerPeriod = 43200

   // Labels
   ZCMINT_LABEL = "Zerocoin Mint"
   ZCSPEND_LABEL = "Zerocoin Spend"
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

type OutputScriptToAddressesFunc func(script []byte) ([]string, bool, error)

// NixParser handle
type NixParser struct {
   *btc.BitcoinParser
   baseparser                           *bchain.BaseParser
   OutputScriptToAddressesFunc           OutputScriptToAddressesFunc
   //BitcoinOutputScriptToAddressesFunc   btc.OutputScriptToAddressesFunc
   //BitcoinGetAddrDescFromAddress        func(address string) (bchain.AddressDescriptor, error)
}

// NewNixParser returns new NixParser instance
func NewNixParser(params *chaincfg.Params, c *btc.Configuration) *NixParser {
   bcp := btc.NewBitcoinParser(params, c)
   p := &NixParser{
      BitcoinParser:   bcp,
      baseparser:      &bchain.BaseParser{},
      //BitcoinGetAddrDescFromAddress: p.GetAddrDescFromAddress,
   }
   //p.BitcoinOutputScriptToAddressesFunc = p.OutputScriptToAddressesFunc
   p.OutputScriptToAddressesFunc = p.outputScriptToAddresses
   return p
   //return &NixParser{BitcoinParser: btc.NewBitcoinParser(params, c)}
}

// GetAddressesFromAddrDesc returns addresses for given address descriptor with flag if the addresses are searchable
func (p *NixParser) GetAddressesFromAddrDesc(addrDesc bchain.AddressDescriptor) ([]string, bool, error) {
   return p.OutputScriptToAddressesFunc(addrDesc)
}

// addressToOutputScript converts address to ScriptPubKey
func (p *NixParser) addressToOutputScript(address string) ([]byte, error) {
   //logwriter, e := syslog.New(syslog.LOG_NOTICE, "blockbook")
   //if e == nil {
   //   log.SetOutput(logwriter)
   //}
   //log.Print(address)
   //log.Print(p.Params)
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

func (p *NixParser) NixOutputScriptToAddresses(script []byte) ([]string, bool, error) {
   //need to add lpos extractions
   sc, addresses, _, err := txscript.ExtractPkScriptAddrs(script, p.Params)
   if err != nil {
      return nil, false, err
   }
   rv := make([]string, len(addresses))
   for i, a := range addresses {
      rv[i] = a.EncodeAddress()
   }
   var s bool
   if sc == txscript.PubKeyHashTy || sc == txscript.WitnessV0PubKeyHashTy || sc == txscript.ScriptHashTy || sc == txscript.WitnessV0ScriptHashTy {
      s = true
   } else if len(rv) == 0 {
      or := p.TryParseOPReturn(script)
      if or != "" {
         rv = []string{or}
      }
   }
   return rv, s, nil
}

// GetChainParams contains network parameters for the main and test Nix network
func GetChainParams(chain string) *chaincfg.Params {
   if !chaincfg.IsRegistered(&MainNetParams) {
     err := chaincfg.Register(&MainNetParams)
     //if err == nil {
     //   err = chaincfg.Register(&TestNetParams)
     //}
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

// PackTx packs transaction to byte array using protobuf
func (p *NixParser) PackTx(tx *bchain.Tx, height uint32, blockTime int64) ([]byte, error) {
   return p.baseparser.PackTx(tx, height, blockTime)
}


// UnpackTx unpacks transaction from protobuf byte array
func (p *NixParser) UnpackTx(buf []byte) (*bchain.Tx, uint32, error) {
   return p.baseparser.UnpackTx(buf)
}


// ParseTxFromJson parses JSON message containing transaction and returns Tx struct
func (p *NixParser) ParseTxFromJson(msg json.RawMessage) (*bchain.Tx, error) {
   var tx bchain.Tx
   err := json.Unmarshal(msg, &tx)
   if err != nil {
      return nil, err
   }

   for i := range tx.Vout {
      vout := &tx.Vout[i]
      // convert vout.JsonValue to big.Int and clear it, it is only temporary value used for unmarshal
      vout.ValueSat, err = p.AmountToBigInt(vout.JsonValue)
      if err != nil {
         return nil, err
      }
      vout.JsonValue = ""

      if vout.ScriptPubKey.Addresses == nil {
         vout.ScriptPubKey.Addresses = []string{}
      }

   }

   return &tx, nil
}

func (p *NixParser) outputScriptToAddresses(script []byte) ([]string, bool, error) {
   if isZeroCoinSpendScript(script) {
      return []string{ZCSPEND_LABEL}, false, nil
   }
   if isZeroCoinMintScript(script) {
      return []string{ZCMINT_LABEL}, false, nil
   }
   if isLeaseProofOfStakeScript(script) {
      logwriter, e := syslog.New(syslog.LOG_NOTICE, "blockbook")
      if e == nil {
         log.SetOutput(logwriter)
      }
      log.Print("Is Lease Proof Of Stake")
      script = script[26: 49 + 1]
      log.Print(script)
   }
   if isLeaseProofOfStakeScriptBech32(script) {
      logwriter, e := syslog.New(syslog.LOG_NOTICE, "blockbook")
      if e == nil {
         log.SetOutput(logwriter)
      }
      log.Print("Is Lease Proof Of Stake Bech32")
      script = script[25:47 + 1]
      log.Print(script)
   }

   rv, s, _ := p.NixOutputScriptToAddresses(script)
   return rv, s, nil
}

// GetAddrDescFromAddress returns internal address representation (descriptor) of given address
func (p *NixParser) GetAddrDescFromAddress(address string) (bchain.AddressDescriptor, error) {

   logwriter, e := syslog.New(syslog.LOG_NOTICE, "blockbook")
   if e == nil {
      log.SetOutput(logwriter)
   }
   //log.Print(p.addressToOutputScript(address))
   return p.addressToOutputScript(address)
}


func (p *NixParser) GetAddrDescForUnknownInput(tx *bchain.Tx, input int) bchain.AddressDescriptor {
   if len(tx.Vin) > input {
      scriptHex := tx.Vin[input].ScriptSig.Hex

      if scriptHex != "" {
         script, _ := hex.DecodeString(scriptHex)
         return script
      }
   }

   s := make([]byte, 10)
   return s
}

func (p *NixParser) GetValueSatForUnknownInput(tx *bchain.Tx, input int) *big.Int {
   if len(tx.Vin) > input {
      scriptHex := tx.Vin[input].ScriptSig.Hex

      if scriptHex != "" {
         script, _ := hex.DecodeString(scriptHex)
         if isZeroCoinSpendScript(script) {
            valueSat,  err := p.GetValueSatFromZerocoinSpend(script)
            if err != nil {
               glog.Warningf("tx %v: input %d unable to convert denom to big int", tx.Txid, input)
               return big.NewInt(0)
            }
            return valueSat
         }
      }
   }
   return big.NewInt(0)
}

// Decodes the amount from the zerocoin spend script
func (p *NixParser) GetValueSatFromZerocoinSpend(signatureScript []byte) (*big.Int, error) {
   r := bytes.NewReader(signatureScript)
   r.Seek(1, io.SeekCurrent)                       // skip opcode
   len, err := Uint8(r)                            // get serialized coinspend size
   if err != nil {
      return nil, err
   }
   r.Seek(int64(len), io.SeekCurrent)              // and skip its bytes
   r.Seek(2, io.SeekCurrent)                       // skip version and spendtype
   len,  err = Uint8(r)                            // get pubkey len
   if err != nil {
      return nil, err
   }
   r.Seek(int64(len), io.SeekCurrent)              // and skip its bytes
   len, err = Uint8(r)                             // get vchsig len
   if err != nil {
      return nil, err
   }
   r.Seek(int64(len), io.SeekCurrent)              // and skip its bytes
   // get denom
   denom, err := Uint32(r, binary.LittleEndian)    // get denomination
   if err != nil {
      return nil, err
   }

   return big.NewInt(int64(denom)*1e8), nil
}

// Checks if script is OP_ZEROCOINMINT
func isZeroCoinMintScript(signatureScript []byte) bool {
   return len(signatureScript) > 1 && signatureScript[0] == OP_ZEROCOINMINT
}

// Checks if script is OP_ZEROCOINSPEND
func isZeroCoinSpendScript(signatureScript []byte) bool {
   return len(signatureScript) >= 100 && signatureScript[0] == OP_ZEROCOINSPEND
}

// Checks if script is p2sh lpos contract
func isLeaseProofOfStakeScript(signatureScript []byte) bool {
   logwriter, e := syslog.New(syslog.LOG_NOTICE, "blockbook")
   if e == nil {
      log.SetOutput(logwriter)
   }
   log.Print(signatureScript[0])
   log.Print(signatureScript[2])
   return signatureScript[0] == OP_COINSTAKE && signatureScript[2] == OP_HASH160
}

// Checks if script bech32 lpos contract
func isLeaseProofOfStakeScriptBech32(signatureScript []byte) bool {
   logwriter, e := syslog.New(syslog.LOG_NOTICE, "blockbook")
   if e == nil {
      log.SetOutput(logwriter)
   }
   log.Print(signatureScript[0])
   log.Print(signatureScript[2])
   return signatureScript[0] == OP_COINSTAKE && signatureScript[2] == OP_0
}
