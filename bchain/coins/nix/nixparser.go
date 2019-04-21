package nix

import (
   "blockbook/bchain/coins/btc"

   "github.com/martinboehm/btcd/wire"
   "github.com/martinboehm/btcutil/chaincfg"
)

const (
   MainnetMagic wire.BitcoinNet = 0xb9b4bef9
   TestnetMagic wire.BitcoinNet = 0x09070b11
   RegtestMagic wire.BitcoinNet = 0xfabfb5da

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
   return &NixParser{BitcoinParser: btc.NewBitcoinParser(params, c)}
}

// GetChainParams contains network parameters for the main and test Nix network
func GetChainParams(chain string) *chaincfg.Params {
   //if !chaincfg.IsRegistered(&MainNetParams) {
   //   err := chaincfg.Register(&MainNetParams)
   //   if err == nil {
   //      err = chaincfg.Register(&TestNetParams)
   //   }
   //   if err != nil {
   //      panic(err)
   //   }
   //}
   switch chain {
   case "test":
      return &TestNetParams
   default:
      return &MainNetParams
   }
}