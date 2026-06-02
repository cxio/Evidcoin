package validation

import (
	"bytes"
	"testing"

	"github.com/cxio/evidcoin/internal/blockchain"
	"github.com/cxio/evidcoin/internal/tx"
	"github.com/cxio/evidcoin/pkg/hashtree"
	"github.com/cxio/evidcoin/pkg/types"
)

// TestProofPackageEncode_Genesis 对创世区块（无 Minter）执行 Encode 并检验字节布局。
func TestProofPackageEncode_Genesis(t *testing.T) {
	checkRoot := types.CheckRoot{0xCC}
	bh := blockchain.NewGenesisHeader(checkRoot)

	leaf := hashtree.LeafHash(hashtree.OrderedLeaf3(0, make([]byte, 48)))

	// 构造一个单叶 hashtree，取出 Proof
	ht, err := hashtree.BuildTree([][]byte{leaf})
	if err != nil {
		t.Fatalf("BuildTree: %v", err)
	}
	treeRoot := ht.Root()
	proof, err := ht.Proof(0)
	if err != nil {
		t.Fatalf("Proof(0): %v", err)
	}

	pp := ProofPackage{
		BlockHeader:              *bh,
		CoinbaseTx:               tx.CoinbaseHeader{BlockHeight: 0, BurnCoin: 0},
		CoinbaseTxIndex:          0,
		CoinbaseMerklePath:       proof,
		TreeRoot:                 treeRoot,
		UTXORoot:                 types.TreeHash{0x11},
		UTCORoot:                 types.TreeHash{0x22},
		MinterCheckRootSignature: []byte{0xDE, 0xAD},
	}

	enc, err := pp.Encode()
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if len(enc) == 0 {
		t.Fatal("Encode returned empty bytes")
	}

	// 检验尾部 32+32+2+1 字节的结构（UTXORoot||UTCORoot||varint(2)||sig）
	tail := enc[len(enc)-67:]

	// UTXORoot (32 bytes)
	if !bytes.Equal(tail[:32], pp.UTXORoot[:]) {
		t.Errorf("UTXORoot bytes mismatch in encoded output")
	}
	// UTCORoot (32 bytes)
	if !bytes.Equal(tail[32:64], pp.UTCORoot[:]) {
		t.Errorf("UTCORoot bytes mismatch in encoded output")
	}
	// MinterCheckRootSignature: varint(2)=0x02 || 0xDE 0xAD
	if tail[64] != 0x02 || tail[65] != 0xDE || tail[66] != 0xAD {
		t.Errorf("MinterCheckRootSignature encoding: got %x, want 02 DE AD", tail[64:])
	}
}

// TestProofPackageEncode_Deterministic 同一 ProofPackage 多次编码结果一致。
func TestProofPackageEncode_Deterministic(t *testing.T) {
	checkRoot := types.CheckRoot{0x01}
	bh := blockchain.NewGenesisHeader(checkRoot)

	leaf := hashtree.LeafHash(hashtree.OrderedLeaf3(0, make([]byte, 48)))
	ht, err := hashtree.BuildTree([][]byte{leaf})
	if err != nil {
		t.Fatalf("BuildTree: %v", err)
	}
	proof, err := ht.Proof(0)
	if err != nil {
		t.Fatalf("Proof(0): %v", err)
	}

	pp := ProofPackage{
		BlockHeader:              *bh,
		CoinbaseTx:               tx.CoinbaseHeader{BlockHeight: 0},
		CoinbaseTxIndex:          0,
		CoinbaseMerklePath:       proof,
		TreeRoot:                 ht.Root(),
		UTXORoot:                 types.TreeHash{0x11},
		UTCORoot:                 types.TreeHash{0x22},
		MinterCheckRootSignature: []byte{0x01},
	}

	enc1, err := pp.Encode()
	if err != nil {
		t.Fatalf("Encode 1: %v", err)
	}
	enc2, err := pp.Encode()
	if err != nil {
		t.Fatalf("Encode 2: %v", err)
	}
	if !bytes.Equal(enc1, enc2) {
		t.Error("Encode is not deterministic")
	}
}
