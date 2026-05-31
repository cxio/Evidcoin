package blockchain

import (
	"bytes"
	"testing"

	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// fakeStateProvider 是测试用状态指纹提供者，记录被查询的高度。
type fakeStateProvider struct {
	utxo       types.TreeHash
	utco       types.TreeHash
	queried    uint32
	queriedSet bool
}

func (p *fakeStateProvider) StateFingerprint(h uint32) (types.TreeHash, types.TreeHash, error) {
	p.queried = h
	p.queriedSet = true
	return p.utxo, p.utco, nil
}

func h32(t *testing.T, b byte) types.TreeHash {
	t.Helper()
	h, err := types.NewTreeHash(bytes.Repeat([]byte{b}, 32))
	if err != nil {
		t.Fatalf("NewTreeHash: %v", err)
	}
	return h
}

// TestComputeCheckRootLengthAndDomain 校验输出 48 字节，且组合等于带 checkroot 域标签的哈希。
func TestComputeCheckRootLengthAndDomain(t *testing.T) {
	treeRoot := bytes.Repeat([]byte{0x01}, 32)
	utxo := h32(t, 0x02)
	utco := h32(t, 0x03)

	got := ComputeCheckRoot(treeRoot, utxo, utco)
	if len(got.Bytes()) != 48 {
		t.Fatalf("CheckRoot 长度 = %d, 期望 48", len(got.Bytes()))
	}

	var pre []byte
	pre = append(pre, treeRoot...)
	pre = append(pre, utxo.Bytes()...)
	pre = append(pre, utco.Bytes()...)
	want := crypto.HashCheckRoot(pre)
	if got != want {
		t.Fatal("ComputeCheckRoot 与 crypto.HashCheckRoot(TreeRoot||UTXO||UTCO) 不一致")
	}
}

// TestComputeCheckRootChangesWithEachInput 校验改变任一输入都会改变结果。
func TestComputeCheckRootChangesWithEachInput(t *testing.T) {
	treeRoot := bytes.Repeat([]byte{0x01}, 32)
	utxo := h32(t, 0x02)
	utco := h32(t, 0x03)
	base := ComputeCheckRoot(treeRoot, utxo, utco)

	if ComputeCheckRoot(bytes.Repeat([]byte{0xAA}, 32), utxo, utco) == base {
		t.Error("改变 TreeRoot 未改变 CheckRoot")
	}
	if ComputeCheckRoot(treeRoot, h32(t, 0xBB), utco) == base {
		t.Error("改变 UTXORoot 未改变 CheckRoot")
	}
	if ComputeCheckRoot(treeRoot, utxo, h32(t, 0xCC)) == base {
		t.Error("改变 UTCORoot 未改变 CheckRoot")
	}
}

// TestComputeCheckRootUTXOUTCOOrderMatters 校验 UTXO 与 UTCO 顺序调换会改变结果。
func TestComputeCheckRootUTXOUTCOOrderMatters(t *testing.T) {
	treeRoot := bytes.Repeat([]byte{0x01}, 32)
	utxo := h32(t, 0x02)
	utco := h32(t, 0x03)
	if ComputeCheckRoot(treeRoot, utxo, utco) == ComputeCheckRoot(treeRoot, utco, utxo) {
		t.Fatal("UTXO/UTCO 顺序调换未改变 CheckRoot")
	}
}

// TestComputeCheckRootAtGenesis 校验 h==0 使用空状态指纹，结果稳定可复现且不查询提供者。
func TestComputeCheckRootAtGenesis(t *testing.T) {
	treeRoot := bytes.Repeat([]byte{0x01}, 32)
	provider := &fakeStateProvider{utxo: h32(t, 0xEE), utco: h32(t, 0xFF)}

	got1, err := ComputeCheckRootAt(0, treeRoot, provider)
	if err != nil {
		t.Fatalf("ComputeCheckRootAt(0): %v", err)
	}
	if provider.queriedSet {
		t.Fatal("h==0 不应查询状态指纹提供者")
	}
	want := ComputeCheckRoot(treeRoot, crypto.EmptyUTXORoot(), crypto.EmptyUTCORoot())
	if got1 != want {
		t.Fatal("创世 CheckRoot 未使用空状态指纹")
	}
	got2, _ := ComputeCheckRootAt(0, treeRoot, provider)
	if got1 != got2 {
		t.Fatal("创世 CheckRoot 不稳定")
	}
}

// TestComputeCheckRootAtReadsPreviousHeight 校验 H>0 读取上一高度 H-1 的状态指纹。
func TestComputeCheckRootAtReadsPreviousHeight(t *testing.T) {
	treeRoot := bytes.Repeat([]byte{0x01}, 32)
	provider := &fakeStateProvider{utxo: h32(t, 0x22), utco: h32(t, 0x33)}

	got, err := ComputeCheckRootAt(5, treeRoot, provider)
	if err != nil {
		t.Fatalf("ComputeCheckRootAt(5): %v", err)
	}
	if !provider.queriedSet || provider.queried != 4 {
		t.Fatalf("应读取上一高度 4 的状态指纹, queried=%d set=%v", provider.queried, provider.queriedSet)
	}
	want := ComputeCheckRoot(treeRoot, provider.utxo, provider.utco)
	if got != want {
		t.Fatal("CheckRoot 未采用 H-1 状态指纹组合")
	}
}
