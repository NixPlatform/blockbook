package nix

import (
	"blockbook/bchain"
	"blockbook/bchain/coins/btc"
	"encoding/json"

	"github.com/golang/glog"
)

// NixRPC is an interface to JSON-RPC bitcoind service.
type NixRPC struct {
	*btc.BitcoinRPC
	BitcoinGetBlockInfo func(hash string) (*bchain.BlockInfo, error)
}

// NewNixRPC returns new NixRPC instance.
func NewNixRPC(config json.RawMessage, pushHandler func(bchain.NotificationType)) (bchain.BlockChain, error) {
	b, err := btc.NewBitcoinRPC(config, pushHandler)
	if err != nil {
		return nil, err
	}

	s := &NixRPC{
		b.(*btc.BitcoinRPC),
		b.GetBlockInfo,
	}
	s.RPCMarshaler = btc.JSONMarshalerV1{}
	s.ChainConfig.SupportsEstimateFee = true
	s.ChainConfig.SupportsEstimateSmartFee = false

	return s, nil
}

// Initialize initializes NixRPC instance.
func (b *NixRPC) Initialize() error {
	chainName, err := b.GetChainInfoAndInitializeMempool(b)
	if err != nil {
		return err
	}

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

// getBlockInfo extends GetBlockInfo for NixRPC
func (g *NixRPC) GetBlockInfo(hash string) (*bchain.BlockInfo, error) {
	bi, err := g.BitcoinGetBlockInfo(hash)
	if err != nil {
		return nil, err
	}

	header, err := g.GetBlockHeader(hash)
	if err != nil {
		return nil, err
	}
	bi.BlockHeader.MoneySupply = header.MoneySupply

	// block is PoS (type=2) when height is greater than 52999
	var height uint32
	height = bi.Height

	if height >= 53000 {
		bi.BlockHeader.Type = 2
	} else {
		bi.BlockHeader.Type = 1
	}
	// get zerocoin Supply
	var zcsupply []bchain.ZCsupply
	zcsupply, err = g.GetZerocoinSupply(bi.BlockHeader.Height)
	if err != nil {
		glog.Errorf("Unable to get zerocoin supply at height %v", bi.BlockHeader.Height)
	}
	bi.ZerocoinSupply = zcsupply

	return bi, nil
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
			if isMissingTx(err) {
				glog.Errorf("rpc: getblock: skipping missing tx %s in block %s", txid, hash)
				continue
			}
			return nil, err
		}
		txs = append(txs, *tx)
	}

	block := &bchain.Block{
		BlockHeader: bi.BlockHeader,
		Txs:         txs,
	}
	return block, nil
}



// getblockchaininfo
type CmdGetBlockChainInfo struct {
	Method string `json:"method"`
}

type ResGetBlockChainInfo struct {
	Error  *bchain.RPCError `json:"error"`
	Result struct {
		Chain         string      `json:"chain"`
		Blocks        int         `json:"blocks"`
		MoneySupply   json.Number `json:"moneysupply"`
		ZerocoinSupply  []bchain.ZCsupply    `json:"zerocoinsupply"`
		Headers       int         `json:"headers"`
		Bestblockhash string      `json:"bestblockhash"`
		PoWDiff       json.Number  `json:"difficulty_pow"`
		PoSDiff       json.Number  `json:"difficulty_pos"`
		SizeOnDisk    int64       `json:"size_on_disk"`
		Warnings      string      `json:"warnings"`
	} `json:"result"`
}

// getnetworkinfo
type CmdGetNetworkInfo struct {
	Method string `json:"method"`
}

type ResGetNetworkInfo struct {
	Error  *bchain.RPCError `json:"error"`
	Result struct {
		Version         json.Number `json:"version"`
		Subversion      json.Number `json:"subversion"`
		ProtocolVersion json.Number `json:"protocolversion"`
		Timeoffset      float64     `json:"timeoffset"`
		Warnings        string      `json:"warnings"`
	} `json:"result"`
}

// GetNextSuperBlock returns the next superblock height after nHeight
func (b *NixRPC) GetNextSuperBlock(nHeight int) int {
	if b.Testnet {
		if nHeight == 0 {
			return 1
		}
		return nHeight - nHeight % nBlocksPerPeriod + 20000
	}
	return nHeight - nHeight % nBlocksPerPeriod + nBlocksPerPeriod
}

// GetChainInfo returns information about the connected backend
func (b *NixRPC) GetChainInfo() (*bchain.ChainInfo, error) {
	glog.V(1).Info("rpc: getblockchaininfo")

	resCi := ResGetBlockChainInfo{}
	err := b.Call(&CmdGetBlockChainInfo{Method: "getblockchaininfo"}, &resCi)
	if err != nil {
		return nil, err
	}
	if resCi.Error != nil {
		return nil, resCi.Error
	}

	glog.V(1).Info("rpc: getnetworkinfo")
	resNi := ResGetNetworkInfo{}
	err = b.Call(&CmdGetNetworkInfo{Method: "getnetworkinfo"}, &resNi)
	if err != nil {
		return nil, err
	}
	if resNi.Error != nil {
		return nil, resNi.Error
	}

	nextSuperBlock := b.GetNextSuperBlock(resCi.Result.Headers)

	rv := &bchain.ChainInfo{
		Bestblockhash: resCi.Result.Bestblockhash,
		Blocks:        resCi.Result.Blocks,
		Chain:         resCi.Result.Chain,
		Headers:       resCi.Result.Headers,
		SizeOnDisk:    resCi.Result.SizeOnDisk,
		Subversion:    string(resNi.Result.Subversion),
		Timeoffset:    resNi.Result.Timeoffset,
		PoWDiff:       resCi.Result.PoWDiff,
		PoSDiff:       resCi.Result.PoSDiff,
		MoneySupply:   resCi.Result.MoneySupply,
		ZerocoinSupply: resCi.Result.ZerocoinSupply,
		NextSuperBlock: nextSuperBlock,
	}
	rv.Version = string(resNi.Result.Version)
	rv.ProtocolVersion = string(resNi.Result.ProtocolVersion)
	if len(resCi.Result.Warnings) > 0 {
		rv.Warnings = resCi.Result.Warnings + " "
	}
	if resCi.Result.Warnings != resNi.Result.Warnings {
		rv.Warnings += resNi.Result.Warnings
	}
	return rv, nil
}


// GetTransactionForMempool returns a transaction by the transaction ID
func (b *NixRPC) GetTransactionForMempool(txid string) (*bchain.Tx, error) {
	return b.GetTransaction(txid)
}

func isMissingTx(err error) bool {
	return err == bchain.ErrTxNotFound
}


// getzerocoinsupply
type CmdGetZerocoinSupply struct {
	Method string `json:"method"`
	Params struct {
		Height uint32 `json:"height"`
	} `json:"params"`
}

type ResGetZerocoinSupply struct {
	Error  *bchain.RPCError `json:"error"`
	Result []bchain.ZCsupply `json:"result"`
}

// GetZerocoinSupply returns hash of block in best-block-chain at given height.
func (b *NixRPC) GetZerocoinSupply(height uint32) ([]bchain.ZCsupply, error) {
	glog.V(1).Info("rpc: getblockhash ", height)

	res := ResGetZerocoinSupply{}
	req := CmdGetZerocoinSupply{Method: "ghostprivacysets"}
	req.Params.Height = height
	err := b.Call(&req, &res)

	if err != nil {
		return nil, err
	}

	return res.Result, nil
}
