package validation

import (
	"bytes"
	"testing"

	"github.com/cxio/evidcoin/internal/blockchain"
	"github.com/cxio/evidcoin/internal/consensus"
	"github.com/cxio/evidcoin/internal/tx"
	"github.com/cxio/evidcoin/pkg/hashtree"
	"github.com/cxio/evidcoin/pkg/types"
)

// buildValidProofPackage 构造一个可通过步骤 1–7 的合法 ProofPackage，
// 铸造者签名步骤（9）需由调用方提供 CheckRootSigVerifier 自行控制。
func buildValidProofPackage(t *testing.T, mintPubKey []byte) (ProofPackage, LocalState) {
	t.Helper()

	// 构造铸造凭证
	txID := types.TxID{0x01}
	mintHash := types.MintHash{0x02}
	mintProof := consensus.MintProof{
		TxHeight:   1,
		TxID:       txID,
		MintPubKey: mintPubKey,
		MintHash:   mintHash,
		Signature:  []byte{0x03},
	}
	minterBytes := mintProof.CanonicalBytes()

	// 构造 CoinbaseHeader（高度 1，含 Minter）
	coinbaseHdr := tx.CoinbaseHeader{
		BlockHeight: 1,
		Minter:      minterBytes,
		BurnCoin:    0,
	}

	// 计算 Coinbase TxID
	coinbaseTxID, err := coinbaseHdr.TxID()
	if err != nil {
		t.Fatalf("coinbaseHdr.TxID: %v", err)
	}

	// 构造单叶交易树
	leaf := hashtree.LeafHash(hashtree.OrderedLeaf3(0, coinbaseTxID.Bytes()))
	ht, err := hashtree.BuildTree([][]byte{leaf})
	if err != nil {
		t.Fatalf("BuildTree: %v", err)
	}
	treeRoot := ht.Root()
	proof, err := ht.Proof(0)
	if err != nil {
		t.Fatalf("Proof(0): %v", err)
	}

	// 构造 UTXO/UTCO 状态指纹
	utxoRoot := types.TreeHash{0x11}
	utcoRoot := types.TreeHash{0x22}

	// 计算 CheckRoot
	checkRoot := blockchain.ComputeCheckRoot(treeRoot, utxoRoot, utcoRoot)

	// 构造 BlockHeader（高度 1，PrevBlock 指向"创世"）
	prevBlock := types.BlockID{0xAA}
	bh := blockchain.BlockHeader{
		Height:    1,
		PrevBlock: prevBlock,
		CheckRoot: checkRoot,
		Stakes:    0,
	}

	pp := ProofPackage{
		BlockHeader:              bh,
		CoinbaseTx:               coinbaseHdr,
		CoinbaseTxIndex:          0,
		CoinbaseMerklePath:       proof,
		TreeRoot:                 treeRoot,
		UTXORoot:                 utxoRoot,
		UTCORoot:                 utcoRoot,
		MinterCheckRootSignature: []byte{0xFF}, // 桩签名，由测试控制 verifier
	}
	local := LocalState{
		TipBlockID: prevBlock,
		UTXORoot:   utxoRoot,
		UTCORoot:   utcoRoot,
	}
	return pp, local
}

// alwaysInPoolChecker 所有公钥均视为在择优池中。
type alwaysInPoolChecker struct{}

func (a alwaysInPoolChecker) IsBestPoolMember(_ []byte) bool { return true }

// neverInPoolChecker 所有公钥均不在择优池中。
type neverInPoolChecker struct{}

func (n neverInPoolChecker) IsBestPoolMember(_ []byte) bool { return false }

// stubSigVerifier 根据 ok 字段决定签名是否有效。
type stubSigVerifier struct{ ok bool }

func (s stubSigVerifier) VerifyCheckRootSig(_, _ []byte, _ types.CheckRoot) (bool, error) {
	return s.ok, nil
}

// TestPreVerify_Valid 全部 9 步通过。
func TestPreVerify_Valid(t *testing.T) {
	pubKey := []byte{0xAB, 0xCD}
	pp, local := buildValidProofPackage(t, pubKey)
	err := PreVerify(pp, local, alwaysInPoolChecker{}, stubSigVerifier{ok: true})
	if err != nil {
		t.Fatalf("PreVerify valid: unexpected error %v", err)
	}
}

// TestPreVerify_PrevBlockMismatch 步骤 1 失败。
func TestPreVerify_PrevBlockMismatch(t *testing.T) {
	pp, local := buildValidProofPackage(t, []byte{0x01})
	local.TipBlockID = types.BlockID{0xFF} // 不匹配
	err := PreVerify(pp, local, alwaysInPoolChecker{}, stubSigVerifier{ok: true})
	if err != ErrPrevBlockMismatch {
		t.Fatalf("want ErrPrevBlockMismatch, got %v", err)
	}
}

// TestPreVerify_NoMinter 步骤 2 失败（Minter 字段空）。
func TestPreVerify_NoMinter(t *testing.T) {
	pp, local := buildValidProofPackage(t, []byte{0x01})
	pp.CoinbaseTx.Minter = nil // 清空 Minter
	err := PreVerify(pp, local, alwaysInPoolChecker{}, stubSigVerifier{ok: true})
	if err != ErrNoMinterField {
		t.Fatalf("want ErrNoMinterField, got %v", err)
	}
}

// TestPreVerify_MinterNotInPool 步骤 2 失败（公钥不在池中）。
func TestPreVerify_MinterNotInPool(t *testing.T) {
	pp, local := buildValidProofPackage(t, []byte{0x01})
	err := PreVerify(pp, local, neverInPoolChecker{}, stubSigVerifier{ok: true})
	if err != ErrMinterNotInPool {
		t.Fatalf("want ErrMinterNotInPool, got %v", err)
	}
}

// TestPreVerify_StateRootMismatch 步骤 3 失败。
func TestPreVerify_StateRootMismatch(t *testing.T) {
	pp, local := buildValidProofPackage(t, []byte{0x01})
	local.UTXORoot = types.TreeHash{0xFF} // 不匹配
	err := PreVerify(pp, local, alwaysInPoolChecker{}, stubSigVerifier{ok: true})
	if err != ErrStateRootMismatch {
		t.Fatalf("want ErrStateRootMismatch, got %v", err)
	}
}

// TestPreVerify_CoinbaseTxIndexNot0 步骤 4 失败。
func TestPreVerify_CoinbaseTxIndexNot0(t *testing.T) {
	pp, local := buildValidProofPackage(t, []byte{0x01})
	pp.CoinbaseTxIndex = 1
	err := PreVerify(pp, local, alwaysInPoolChecker{}, stubSigVerifier{ok: true})
	if err != ErrCoinbaseTxIndexNot0 {
		t.Fatalf("want ErrCoinbaseTxIndexNot0, got %v", err)
	}
}

// TestPreVerify_TreeRootMismatch_WrongLeaf 步骤 6a 失败（叶哈希不匹配）。
func TestPreVerify_TreeRootMismatch_WrongLeaf(t *testing.T) {
	pp, local := buildValidProofPackage(t, []byte{0x01})
	// 用错误数据替换叶哈希
	pp.CoinbaseMerklePath.LeafHash = bytes.Repeat([]byte{0x00}, len(pp.CoinbaseMerklePath.LeafHash))
	err := PreVerify(pp, local, alwaysInPoolChecker{}, stubSigVerifier{ok: true})
	if err != ErrTreeRootMismatch {
		t.Fatalf("want ErrTreeRootMismatch, got %v", err)
	}
}

// TestPreVerify_CheckRootMismatch 步骤 7 失败。
func TestPreVerify_CheckRootMismatch(t *testing.T) {
	pp, local := buildValidProofPackage(t, []byte{0x01})
	// 篡改 BlockHeader.CheckRoot 使其与重算值不符
	pp.BlockHeader.CheckRoot = types.CheckRoot{0xDE, 0xAD}
	err := PreVerify(pp, local, alwaysInPoolChecker{}, stubSigVerifier{ok: true})
	if err != ErrCheckRootMismatch {
		t.Fatalf("want ErrCheckRootMismatch, got %v", err)
	}
}

// TestPreVerify_MinterSigInvalid 步骤 9 失败。
func TestPreVerify_MinterSigInvalid(t *testing.T) {
	pp, local := buildValidProofPackage(t, []byte{0x01})
	err := PreVerify(pp, local, alwaysInPoolChecker{}, stubSigVerifier{ok: false})
	if err != ErrMinterSigInvalid {
		t.Fatalf("want ErrMinterSigInvalid, got %v", err)
	}
}
