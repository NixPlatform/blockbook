//build unittest

package nix

import (
	//"blockbook/bchain"
	//"blockbook/bchain/coins/btc"
	"bytes"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	//"math/big"
	"path/filepath"
	//"reflect"
	"testing"
)

type testBlock struct {
	size int
	time int64
	txs  []string
}

var testParseBlockTxs = map[int]testBlock{
	1337: {
		size: 295,
		time: 1530239907,
		txs: []string{
			"a914c247a37256e27a70d3440735d1852f1efe569d5d87",
			"a9146a27966d76edfdfb3c53dc37dd5471bdeac7d46c87",
			"a9147150055215791b779dddb1d112541a3fce71061b87",
			"76a914a5494a7646ceffc2c1b60226258409074f326c5c88ac",
		},
	},
	// last block without special transactions, valid for bitcoin parser
	25000: {
		size: 1175,
		time: 1533251989,
		txs: []string{
			"0070632c0286d61b23049a4a70d334149378636491d5c8e7c26d393ccd692747",
			"e2591f10aab89ecd06a84aa3a41451fd6cb04e0655e5468240cdd51c1aeee22d",
			"e6d9bfb39d8d0c6d9b32994686729ce43a782daf592f45b0a2a809ab0095fd44",
		},
	},
	// first block with special transactions, invalid for bitcoin parser
	// 1028160: {
	// 	size: 295,
	// 	time: 1530239907,
	// 	txs: []string{
	// 		"",
	// 	},
	// },
}

func helperLoadBlock(t *testing.T, height int) []byte {
	name := fmt.Sprintf("block_dump.%d", height)
	path := filepath.Join("testdata", name)

	d, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	d = bytes.TrimSpace(d)

	b := make([]byte, hex.DecodedLen(len(d)))
	_, err = hex.Decode(b, d)
	if err != nil {
		t.Fatal(err)
	}

	return b
}

//func TestParseBlock(t *testing.T) {
//	p := newNixParser(GetChainParams("main"), &btc.Configuration{})
//
//	for height, tb := range testParseBlockTxs {
//		b := helperLoadBlock(t, height)
//
//		blk, err := p.ParseBlock(b)
//		if err != nil {
//			t.Errorf("ParseBlock() error %v", err)
//		}
//
//		if blk.Size != tb.size {
//			t.Errorf("ParseBlock() block size: got %d, want %d", blk.Size, tb.size)
//		}
//
//		if blk.Time != tb.time {
//			t.Errorf("ParseBlock() block time: got %d, want %d", blk.Time, tb.time)
//		}
//
//		if len(blk.Txs) != len(tb.txs) {
//			t.Errorf("ParseBlock() number of transactions: got %d, want %d", len(blk.Txs), len(tb.txs))
//		}
//
//		for ti, tx := range tb.txs {
//			if blk.Txs[ti].Txid != tx {
//				t.Errorf("ParseBlock() transaction %d: got %s, want %s", ti, blk.Txs[ti].Txid, tx)
//			}
//		}
//	}
//}

//var (
//	testTx1 = bchain.Tx{
//		Blocktime:     1551246710,
//		Confirmations: 0,
//		Hex:           "0100000001f85264d11a747bdba77d411e5e4a3d35e3aeb5843b34a95234a2121ac65496bd000000006b483045022100dfa158fbd9773fab4f6f329c807e040af0c3a40967cbe01667169b914ed5ad960220061c5876364caa3e3c9c990ad2b4cc8b1a53d4f954dbda8434b0e67cc8348ff6012103093865e1e132b33a2a5ed01c79d2edba3473826a66cb26b8311bfa42749c2190ffffffff02ec3f8a2a010000001976a91470dcef2a22575d7a8f0779fb1d6cdd48135bd22788ac3116491d000000001976a91471348f7780e955a2a60eba17ecc4c826ebc23a9888ac00000000",
//		LockTime:      0,
//		Time:          1551246710,
//		Txid:          "88fde87182a7772c5103e6539dfe0262878f8869a2ca7739797b14463b732056",
//		Version:       1,
//		Vin: []bchain.Vin{
//			{
//				Txid: "bd9654c61a12a23452a9343b84b5aee3353d4a5e1e417da7db7b741ad16452f8",
//				Vout: 0,
//				ScriptSig: bchain.ScriptSig{
//					Hex: "483045022100dfa158fbd9773fab4f6f329c807e040af0c3a40967cbe01667169b914ed5ad960220061c5876364caa3e3c9c990ad2b4cc8b1a53d4f954dbda8434b0e67cc8348ff6012103093865e1e132b33a2a5ed01c79d2edba3473826a66cb26b8311bfa42749c2190",
//				},
//				Sequence: 4294967295,
//			},
//		},
//		Vout: []bchain.Vout{
//			{
//				N: 0,
//				ScriptPubKey: bchain.ScriptPubKey{
//					Addresses: []string{"XkycBX1ykVXXs92pAi6ZQwZPEre9kSHHKH"},
//					Hex:       "76a91470dcef2a22575d7a8f0779fb1d6cdd48135bd22788ac",
//				},
//				ValueSat: *big.NewInt(5008670700),
//			},
//			{
//				N: 1,
//				ScriptPubKey: bchain.ScriptPubKey{
//					Addresses: []string{"Xm1R9thKBm2EZKZevXsmMX4DVwQQuTohZu"},
//					Hex:       "76a91471348f7780e955a2a60eba17ecc4c826ebc23a9888ac",
//				},
//				ValueSat: *big.NewInt(491329073),
//			},
//		},
//	}
//	testTxPacked1 = "0a20ed732a404cdfd4e0475a7a016200b7eef191f2c9de0ffdef8a20091c0499299c12e2010100000001f85264d11a747bdba77d411e5e4a3d35e3aeb5843b34a95234a2121ac65496bd000000006b483045022100dfa158fbd9773fab4f6f329c807e040af0c3a40967cbe01667169b914ed5ad960220061c5876364caa3e3c9c990ad2b4cc8b1a53d4f954dbda8434b0e67cc8348ff6012103093865e1e132b33a2a5ed01c79d2edba3473826a66cb26b8311bfa42749c2190ffffffff02ec3f8a2a010000001976a91470dcef2a22575d7a8f0779fb1d6cdd48135bd22788ac3116491d000000001976a91471348f7780e955a2a60eba17ecc4c826ebc23a9888ac0000000018f6cad8e305200028c0e03e3299010a001220bd9654c61a12a23452a9343b84b5aee3353d4a5e1e417da7db7b741ad16452f81800226b483045022100dfa158fbd9773fab4f6f329c807e040af0c3a40967cbe01667169b914ed5ad960220061c5876364caa3e3c9c990ad2b4cc8b1a53d4f954dbda8434b0e67cc8348ff6012103093865e1e132b33a2a5ed01c79d2edba3473826a66cb26b8311bfa42749c219028ffffffff0f3a480a05012a8a3fec10001a1976a91470dcef2a22575d7a8f0779fb1d6cdd48135bd22788ac2222586b7963425831796b565858733932704169365a51775a50457265396b5348484b483a470a041d49163110011a1976a91471348f7780e955a2a60eba17ecc4c826ebc23a9888ac2222586d31523974684b426d32455a4b5a657658736d4d5834445677515175546f685a754001"
//
//	testTx2 = bchain.Tx{
//		Blocktime:     1551246710,
//		Confirmations: 0,
//		Hex:           "03000500010000000000000000000000000000000000000000000000000000000000000000ffffffff170340b00f1291af3c09542bc8349901000000002f4e614effffffff024181f809000000001976a9146a341485a9444b35dc9cb90d24e7483de7d37e0088ac3581f809000000001976a9140d1156f6026bf975ea3553b03fb534d0959c294c88ac0000000026010040b00f000000000000000000000000000000000000000000000000000000000000000000",
//		LockTime:      0,
//		Time:          1551246710,
//		Txid:          "71d6975e3b79b52baf26c3269896a34f3bedfb04561c692ffa31f64dada1f9c4",
//		Version:       3,
//		Vin: []bchain.Vin{
//			{
//				Coinbase: "0340b00f1291af3c09542bc8349901000000002f4e614e",
//				Sequence: 4294967295,
//			},
//		},
//		Vout: []bchain.Vout{
//			{
//				N: 0,
//				ScriptPubKey: bchain.ScriptPubKey{
//					Addresses: []string{"XkNPrBSJtrHZUvUqb3JF4g5rMB3uzaJfEL"},
//					Hex:       "76a9146a341485a9444b35dc9cb90d24e7483de7d37e0088ac",
//				},
//				ValueSat: *big.NewInt(167280961),
//			},
//			{
//				N: 1,
//				ScriptPubKey: bchain.ScriptPubKey{
//					Addresses: []string{"XbswPXhcLqm5AN5gwcTTyiUGSP2YndWwk9"},
//					Hex:       "76a9140d1156f6026bf975ea3553b03fb534d0959c294c88ac",
//				},
//				ValueSat: *big.NewInt(167280949),
//			},
//		},
//	}
//
//	testTxPacked2 = "0a2071d6975e3b79b52baf26c3269896a34f3bedfb04561c692ffa31f64dada1f9c412b50103000500010000000000000000000000000000000000000000000000000000000000000000ffffffff170340b00f1291af3c09542bc8349901000000002f4e614effffffff024181f809000000001976a9146a341485a9444b35dc9cb90d24e7483de7d37e0088ac3581f809000000001976a9140d1156f6026bf975ea3553b03fb534d0959c294c88ac0000000026010040b00f00000000000000000000000000000000000000000000000000000000000000000018f6cad8e305200028c0e03e32380a2e30333430623030663132393161663363303935343262633833343939303130303030303030303266346536313465180028ffffffff0f3a470a0409f8814110001a1976a9146a341485a9444b35dc9cb90d24e7483de7d37e0088ac2222586b4e507242534a7472485a5576557162334a46346735724d4233757a614a66454c3a470a0409f8813510011a1976a9140d1156f6026bf975ea3553b03fb534d0959c294c88ac222258627377505868634c716d35414e35677763545479695547535032596e6457776b394003"
//)
//
//func TestBaseParser_ParseTxFromJson(t *testing.T) {
//	p := NewDashParser(GetChainParams("main"), &btc.Configuration{})
//	tests := []struct {
//		name    string
//		msg     string
//		want    *bchain.Tx
//		wantErr bool
//	}{
//		{
//			name: "normal tx",
//			msg:  `{"hex":"0100000001f85264d11a747bdba77d411e5e4a3d35e3aeb5843b34a95234a2121ac65496bd000000006b483045022100dfa158fbd9773fab4f6f329c807e040af0c3a40967cbe01667169b914ed5ad960220061c5876364caa3e3c9c990ad2b4cc8b1a53d4f954dbda8434b0e67cc8348ff6012103093865e1e132b33a2a5ed01c79d2edba3473826a66cb26b8311bfa42749c2190ffffffff02ec3f8a2a010000001976a91470dcef2a22575d7a8f0779fb1d6cdd48135bd22788ac3116491d000000001976a91471348f7780e955a2a60eba17ecc4c826ebc23a9888ac00000000","txid":"ed732a404cdfd4e0475a7a016200b7eef191f2c9de0ffdef8a20091c0499299c","size":226,"version":1,"type":0,"locktime":0,"vin":[{"txid":"bd9654c61a12a23452a9343b84b5aee3353d4a5e1e417da7db7b741ad16452f8","vout":0,"scriptSig":{"asm":"3045022100dfa158fbd9773fab4f6f329c807e040af0c3a40967cbe01667169b914ed5ad960220061c5876364caa3e3c9c990ad2b4cc8b1a53d4f954dbda8434b0e67cc8348ff6[ALL]03093865e1e132b33a2a5ed01c79d2edba3473826a66cb26b8311bfa42749c2190","hex":"483045022100dfa158fbd9773fab4f6f329c807e040af0c3a40967cbe01667169b914ed5ad960220061c5876364caa3e3c9c990ad2b4cc8b1a53d4f954dbda8434b0e67cc8348ff6012103093865e1e132b33a2a5ed01c79d2edba3473826a66cb26b8311bfa42749c2190"},"value":55.00000000,"valueSat":5500000000,"address":"Xgcv4bKAXaWf5sjX9KR49L98jeMwNgeXWh","sequence":4294967295}],"vout":[{"value":50.08670700,"valueSat":5008670700,"n":0,"scriptPubKey":{"asm":"OP_DUPOP_HASH16070dcef2a22575d7a8f0779fb1d6cdd48135bd227OP_EQUALVERIFYOP_CHECKSIG","hex":"76a91470dcef2a22575d7a8f0779fb1d6cdd48135bd22788ac","reqSigs":1,"type":"pubkeyhash","addresses":["XkycBX1ykVXXs92pAi6ZQwZPEre9kSHHKH"]}},{"value":4.91329073,"valueSat":491329073,"n":1,"scriptPubKey":{"asm":"OP_DUPOP_HASH16071348f7780e955a2a60eba17ecc4c826ebc23a98OP_EQUALVERIFYOP_CHECKSIG","hex":"76a91471348f7780e955a2a60eba17ecc4c826ebc23a9888ac","reqSigs":1,"type":"pubkeyhash","addresses":["Xm1R9thKBm2EZKZevXsmMX4DVwQQuTohZu"]}}],"blockhash":"000000000000002099caaf1a877911d99a5980ede9b981280eecb291afedf87b","height":1028160,"confirmations":0,"time":1551246710,"blocktime":1551246710,"instantlock":false}`,
//			want: &testTx1,
//		},
//		{
//			name: "special tx - DIP2",
//			msg:  `{"hex":"03000500010000000000000000000000000000000000000000000000000000000000000000ffffffff170340b00f1291af3c09542bc8349901000000002f4e614effffffff024181f809000000001976a9146a341485a9444b35dc9cb90d24e7483de7d37e0088ac3581f809000000001976a9140d1156f6026bf975ea3553b03fb534d0959c294c88ac0000000026010040b00f000000000000000000000000000000000000000000000000000000000000000000","txid":"71d6975e3b79b52baf26c3269896a34f3bedfb04561c692ffa31f64dada1f9c4","size":181,"version":3,"type":5,"locktime":0,"vin":[{"coinbase":"0340b00f1291af3c09542bc8349901000000002f4e614e","sequence":4294967295}],"vout":[{"value":1.67280961,"valueSat":167280961,"n":0,"scriptPubKey":{"asm":"OP_DUPOP_HASH1606a341485a9444b35dc9cb90d24e7483de7d37e00OP_EQUALVERIFYOP_CHECKSIG","hex":"76a9146a341485a9444b35dc9cb90d24e7483de7d37e0088ac","reqSigs":1,"type":"pubkeyhash","addresses":["XkNPrBSJtrHZUvUqb3JF4g5rMB3uzaJfEL"]}},{"value":1.67280949,"valueSat":167280949,"n":1,"scriptPubKey":{"asm":"OP_DUPOP_HASH1600d1156f6026bf975ea3553b03fb534d0959c294cOP_EQUALVERIFYOP_CHECKSIG","hex":"76a9140d1156f6026bf975ea3553b03fb534d0959c294c88ac","reqSigs":1,"type":"pubkeyhash","addresses":["XbswPXhcLqm5AN5gwcTTyiUGSP2YndWwk9"]}}],"extraPayloadSize":38,"extraPayload":"010040b00f000000000000000000000000000000000000000000000000000000000000000000","cbTx":{"version":1,"height":1028160,"merkleRootMNList":"0000000000000000000000000000000000000000000000000000000000000000"},"blockhash":"000000000000002099caaf1a877911d99a5980ede9b981280eecb291afedf87b","height":1028160,"confirmations":0,"time":1551246710,"blocktime":1551246710,"instantlock":false}`,
//			want: &testTx2,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			got, err := p.ParseTxFromJson([]byte(tt.msg))
//			if (err != nil) != tt.wantErr {
//				t.Errorf("DashParser.ParseTxFromJson() error = %v, wantErr %v", err, tt.wantErr)
//				return
//			}
//			if !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("DashParser.ParseTxFromJson() = %+v, want %+v", got, tt.want)
//			}
//		})
//	}
//}
//
//func Test_PackTx(t *testing.T) {
//	type args struct {
//		tx        bchain.Tx
//		height    uint32
//		blockTime int64
//		parser    *DashParser
//	}
//	tests := []struct {
//		name    string
//		args    args
//		want    string
//		wantErr bool
//	}{
//		{
//			name: "dash-1",
//			args: args{
//				tx:        testTx1,
//				height:    1028160,
//				blockTime: 1551246710,
//				parser:    NewDashParser(GetChainParams("main"), &btc.Configuration{}),
//			},
//			want:    testTxPacked1,
//			wantErr: false,
//		},
//		{
//			name: "dash-2",
//			args: args{
//				tx:        testTx2,
//				height:    1028160,
//				blockTime: 1551246710,
//				parser:    NewDashParser(GetChainParams("main"), &btc.Configuration{}),
//			},
//			want:    testTxPacked2,
//			wantErr: false,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			got, err := tt.args.parser.PackTx(&tt.args.tx, tt.args.height, tt.args.blockTime)
//			if (err != nil) != tt.wantErr {
//				t.Errorf("packTx() error = %v, wantErr %v", err, tt.wantErr)
//				return
//			}
//			h := hex.EncodeToString(got)
//			if !reflect.DeepEqual(h, tt.want) {
//				t.Errorf("packTx() = %v, want %v", h, tt.want)
//			}
//		})
//	}
//}
//
//func Test_UnpackTx(t *testing.T) {
//	type args struct {
//		packedTx string
//		parser   *DashParser
//	}
//	tests := []struct {
//		name    string
//		args    args
//		want    *bchain.Tx
//		want1   uint32
//		wantErr bool
//	}{
//		{
//			name: "dash-1",
//			args: args{
//				packedTx: testTxPacked1,
//				parser:   NewDashParser(GetChainParams("main"), &btc.Configuration{}),
//			},
//			want:    &testTx1,
//			want1:   1028160,
//			wantErr: false,
//		},
//		{
//			name: "dash-2",
//			args: args{
//				packedTx: testTxPacked2,
//				parser:   NewDashParser(GetChainParams("main"), &btc.Configuration{}),
//			},
//			want:    &testTx2,
//			want1:   1028160,
//			wantErr: false,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			b, _ := hex.DecodeString(tt.args.packedTx)
//			got, got1, err := tt.args.parser.UnpackTx(b)
//			if (err != nil) != tt.wantErr {
//				t.Errorf("unpackTx() error = %v, wantErr %v", err, tt.wantErr)
//				return
//			}
//			if !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("unpackTx() got = %v, want %v", got, tt.want)
//			}
//			if got1 != tt.want1 {
//				t.Errorf("unpackTx() got1 = %v, want %v", got1, tt.want1)
//			}
//		})
//	}
//}
