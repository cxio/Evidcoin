package tx

import (
	"bytes"
	"testing"
)

// genesisCoinbase 构造一个用于测试的创世 Coinbase 头（BlockHeight==0，Minter 省略）。
func genesisCoinbase() *CoinbaseHeader {
	var pk [32]byte
	for i := range pk {
		pk[i] = byte(i)
	}
	return &CoinbaseHeader{
		Version:     1,
		Timestamp:   1234567890,
		MintPKHash:  pk,
		BlockHeight: 0,
	}
}

// TestCoinbaseGenesisCanonicalLayout 校验创世 Coinbase 头规范编码字段顺序与长度，
// 省略 Minter，AwardSlots 全零且始终存在，无 HashInputs。
func TestCoinbaseGenesisCanonicalLayout(t *testing.T) {
	cb := genesisCoinbase()
	got, err := cb.CanonicalBytes()
	if err != nil {
		t.Fatalf("CanonicalBytes: %v", err)
	}
	// 期望长度：Version(2)+HashOutputs(32)+Timestamp(8)+MintPKHash(32)
	//   +BlockHeight(4)+[Minter 省略]+FreeData(varint(0)=1)+BurnCoin(8)+AwardSlots(18) = 105。
	if len(got) != 105 {
		t.Fatalf("创世 Coinbase 头长度应为 105，got=%d", len(got))
	}
	// 末 18 字节为 AwardSlots，创世恒为全零。
	tail := got[len(got)-18:]
	if !bytes.Equal(tail, make([]byte, 18)) {
		t.Fatalf("创世 AwardSlots 应全零，got=%x", tail)
	}
}

// TestCoinbaseGenesisMinterMustBeAbsent 校验创世（BlockHeight==0）携带 Minter 时被拒绝。
func TestCoinbaseGenesisMinterMustBeAbsent(t *testing.T) {
	cb := genesisCoinbase()
	cb.Minter = []byte{0x01}
	if _, err := cb.CanonicalBytes(); err == nil {
		t.Fatal("创世 Coinbase 携带 Minter 应被拒绝")
	}
}

// TestCoinbaseNonGenesisMinterRequired 校验非创世（BlockHeight>0）缺少 Minter 时被拒绝，
// 携带 Minter 时正常编码并将其原样置于 BlockHeight 与 FreeData 之间。
func TestCoinbaseNonGenesisMinterRequired(t *testing.T) {
	cb := genesisCoinbase()
	cb.BlockHeight = 100
	if _, err := cb.CanonicalBytes(); err == nil {
		t.Fatal("非创世缺少 Minter 应被拒绝")
	}
	cb.Minter = []byte{0xAA, 0xBB}
	got, err := cb.CanonicalBytes()
	if err != nil {
		t.Fatalf("CanonicalBytes: %v", err)
	}
	// Minter 原样出现在 BlockHeight(偏移 2+32+8+32=74，占 4 字节)之后。
	if !bytes.Contains(got, []byte{0xAA, 0xBB}) {
		t.Fatalf("Minter 字节应原样编码，got=%x", got)
	}
}

// TestCoinbaseTxID 校验 Coinbase 头能产出确定性 TxID。
func TestCoinbaseTxID(t *testing.T) {
	cb := genesisCoinbase()
	id1, err := cb.TxID()
	if err != nil {
		t.Fatalf("TxID: %v", err)
	}
	id2, _ := cb.TxID()
	if id1 != id2 {
		t.Fatal("相同 Coinbase 头 TxID 应一致")
	}
}

// TestCoinbaseOutputsOnlyCoin 校验 Coinbase 输出仅允许币金，其它类型被拒绝。
func TestCoinbaseOutputsOnlyCoin(t *testing.T) {
	coin := Output{Serial: 0, Type: TypeCoin}
	if err := ValidateCoinbaseOutputs([]Output{coin}); err != nil {
		t.Fatalf("纯币金输出应通过: %v", err)
	}
	cases := []Output{
		{Serial: 0, Type: TypeCredit},
		{Serial: 0, Type: TypeProof},
	}
	for _, bad := range cases {
		if err := ValidateCoinbaseOutputs([]Output{coin, bad}); err == nil {
			t.Fatalf("非币金 Coinbase 输出应被拒绝: %+v", bad)
		}
	}
}

// TestCoinbasePosition 校验 Coinbase 必须位于区块交易序列第 0 项。
func TestCoinbasePosition(t *testing.T) {
	if err := ValidateCoinbasePosition(0); err != nil {
		t.Fatalf("位置 0 应合法: %v", err)
	}
	if err := ValidateCoinbasePosition(1); err == nil {
		t.Fatal("非 0 位置应被拒绝")
	}
}

// TestCoinbaseFreeDataLimit 校验 FreeData 超过 255 字节被拒绝。
func TestCoinbaseFreeDataLimit(t *testing.T) {
	cb := genesisCoinbase()
	cb.FreeData = make([]byte, 256)
	if _, err := cb.CanonicalBytes(); err == nil {
		t.Fatal("FreeData 256 字节应被拒绝")
	}
}
