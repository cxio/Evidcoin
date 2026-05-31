package blockchain

import (
	"bytes"
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

func testGenesisID() types.BlockID {
	return types.MustBlockID(bytes.Repeat([]byte{0x5A}, 48))
}

// TestChainIdentityContainsFields 校验身份材料按序拼接 ProtocolID、ChainID、GenesisID。
func TestChainIdentityContainsFields(t *testing.T) {
	id := ChainIdentity{
		ProtocolID: types.ProtocolID,
		ChainID:    "mainnet",
		GenesisID:  testGenesisID(),
	}
	got := id.Bytes()

	var want []byte
	want = append(want, types.ProtocolID...)
	want = append(want, "mainnet"...)
	want = append(want, testGenesisID().Bytes()...)

	if !bytes.Equal(got, want) {
		t.Fatalf("身份材料不匹配\n got=%x\nwant=%x", got, want)
	}
}

// TestChainIdentityBoundPresenceDiffers 校验 BoundID absent/present 编码不同，
// 且 present 比 absent 多出 24 字节（4 字节高度 + 20 字节 BlockID 前缀）。
func TestChainIdentityBoundPresenceDiffers(t *testing.T) {
	base := ChainIdentity{
		ProtocolID: types.ProtocolID,
		ChainID:    "mainnet",
		GenesisID:  testGenesisID(),
	}
	absent := base.Bytes()

	bound := NewBoundID(20, types.MustBlockID(bytes.Repeat([]byte{0xC3}, 48)))
	withBound := base
	withBound.Bound = &bound
	present := withBound.Bytes()

	if bytes.Equal(absent, present) {
		t.Fatal("BoundID absent 与 present 编码相同")
	}
	if len(present)-len(absent) != 24 {
		t.Fatalf("BoundID 编码增量 = %d, 期望 24", len(present)-len(absent))
	}
	if !bytes.HasPrefix(present, absent) {
		t.Fatal("present 应以 absent 材料为前缀")
	}
}

// TestNewBoundIDPrefix 校验 BoundID 取 BlockID 前 20 字节。
func TestNewBoundIDPrefix(t *testing.T) {
	blk := types.MustBlockID(bytes.Repeat([]byte{0x77}, 48))
	bound := NewBoundID(42, blk)
	if bound.Height != 42 {
		t.Fatalf("Height = %d, 期望 42", bound.Height)
	}
	if !bytes.Equal(bound.BlockPrefix[:], blk.Bytes()[:20]) {
		t.Fatalf("BlockPrefix 不是 BlockID 前 20 字节, got=%x", bound.BlockPrefix)
	}
}

// TestChainIdentityStable 校验身份材料稳定可复现，供签名调用方多次取得一致字节。
func TestChainIdentityStable(t *testing.T) {
	build := func() ChainIdentity {
		bound := NewBoundID(7, types.MustBlockID(bytes.Repeat([]byte{0x01}, 48)))
		return ChainIdentity{
			ProtocolID: types.ProtocolID,
			ChainID:    "testnet",
			GenesisID:  testGenesisID(),
			Bound:      &bound,
		}
	}
	if !bytes.Equal(build().Bytes(), build().Bytes()) {
		t.Fatal("身份材料不稳定")
	}
}
