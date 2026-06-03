package consensus

import (
	"bytes"
	"testing"

	"github.com/cxio/evidcoin/internal/consensus/equix"
	"github.com/cxio/evidcoin/internal/tx"
	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/hashtree"
	"github.com/cxio/evidcoin/pkg/types"
)

// fakeMintSigVerifier 是测试替身：仅当 sig 等于约定的 "valid" 标记时通过。
type fakeMintSigVerifier struct {
	wantMessage []byte
}

func (f fakeMintSigVerifier) VerifyMintSignature(pubKey, message, sig []byte) (bool, error) {
	if f.wantMessage != nil && !bytes.Equal(message, f.wantMessage) {
		return false, nil
	}
	return bytes.Equal(sig, []byte("valid-sig")), nil
}

// fakeDataSource 是测试替身数据源。
type fakeDataSource struct {
	header    RetrievedMintTx
	headerErr error
	checkRoot types.CheckRoot
	crErr     error
	proof     hashtree.Proof
	proofErr  error
}

func (d *fakeDataSource) MintTransactionHeader(id types.TxID) (RetrievedMintTx, error) {
	return d.header, d.headerErr
}
func (d *fakeDataSource) CheckRootAt(height uint32) (types.CheckRoot, error) {
	return d.checkRoot, d.crErr
}
func (d *fakeDataSource) InclusionPath(id types.TxID) (hashtree.Proof, error) {
	return d.proof, d.proofErr
}

// testEquiXHashList 是测试中使用的固定 Equi-X 哈希列表（代替真实算法输出）。
var testEquiXHashList = [][]byte{
	bytes.Repeat([]byte{0xAA}, 32),
	bytes.Repeat([]byte{0xBB}, 32),
}

// fakeEquiXSolver 是测试替身：对任意输入返回 testEquiXHashList 并认为 solution 有效。
type fakeEquiXSolver struct {
	// returnInvalid 为 true 时模拟 solution 验证失败。
	returnInvalid bool
}

func (f fakeEquiXSolver) Solve(_ []byte, startNonce uint64) ([][]byte, []byte, uint64, error) {
	return testEquiXHashList, []byte{0x01, 0x02}, startNonce, nil
}

func (f fakeEquiXSolver) Verify(_ []byte, _ uint64, _ []byte) ([][]byte, bool, error) {
	if f.returnInvalid {
		return nil, false, nil
	}
	return testEquiXHashList, true, nil
}

// 确认 fakeEquiXSolver 满足 equix.Solver 接口（编译期检查）。
var _ equix.Solver = fakeEquiXSolver{}

// buildPKHashTxScenario 构造一个「含 MintPKHash」的合法铸凭交易场景，
// 返回 MintProof、数据源与组合 CheckRoot 所需的状态根。
func buildPKHashTxScenario(t *testing.T, currentHeight, txHeight uint32) (MintProof, *fakeDataSource, []byte) {
	t.Helper()
	pubKey := []byte{0x11, 0x22, 0x33, 0x44}
	pkHash := crypto.AddressHashSingle(pubKey)

	// 重算交易头：含 MintPKHash。
	hdr := &tx.TxHeader{
		Version:     1,
		HashInputs:  types.Hash32{0x01},
		HashOutputs: types.Hash32{0x02},
		Timestamp:   1_700_000_000_000,
		MintPKHash:  pkHash.Bytes(),
	}
	txID, err := hdr.TxID()
	if err != nil {
		t.Fatalf("TxID: %v", err)
	}

	// 构造交易树（单叶：txID 作为 payload）并得到验证路径与树根。
	tree, err := hashtree.BuildFromPayloads([][]byte{txID.Bytes()})
	if err != nil {
		t.Fatalf("BuildFromPayloads: %v", err)
	}
	proof, err := tree.Proof(0)
	if err != nil {
		t.Fatalf("Proof: %v", err)
	}
	treeRoot := tree.Root()

	// 组合 CheckRoot：用空状态根作为占位（验证只比对 CheckRoot 是否一致）。
	utxoRoot := crypto.EmptyUTXORoot()
	utcoRoot := crypto.EmptyUTCORoot()
	checkRoot := computeCheckRootForTest(treeRoot, utxoRoot, utcoRoot)

	// 计算铸凭哈希（使用两阶段算法，BlockHeight 为当前待铸区块高度）。
	// 在测试中使用 fakeEquiXSolver 返回的 testEquiXHashList 作为 hashList。
	_ = MintHashPreimage{
		MintPubKey:  pubKey,
		MintTxID:    txID,
		Stakes:      500,
		RefMintHash: types.MintHash{},
		BlockHeight: currentHeight,
	}
	mintHash := ComputeMintHash(testEquiXHashList)

	proofMP := MintProof{
		TxHeight:   txHeight,
		TxID:       txID,
		Nonce:      uint64(currentHeight),
		Solution:   []byte{0x01, 0x02},
		MintPubKey: pubKey,
		MintHash:   mintHash,
		Signature:  []byte("valid-sig"),
	}

	ds := &fakeDataSource{
		header: RetrievedMintTx{
			Version:     hdr.Version,
			HashInputs:  hdr.HashInputs,
			HashOutputs: hdr.HashOutputs,
			Timestamp:   hdr.Timestamp,
			MintPKHash:  pkHash.Bytes(),
			Stakes:      500,
			RefMintHash: types.MintHash{},
		},
		checkRoot: checkRoot,
		proof:     proof,
	}
	return proofMP, ds, treeRoot
}

// computeCheckRootForTest 复制 blockchain.ComputeCheckRoot 的前像规则用于测试断言，
// 避免在 Layer 4 测试中反向依赖 blockchain 的内部细节。
func computeCheckRootForTest(treeRoot []byte, utxoRoot, utcoRoot types.TreeHash) types.CheckRoot {
	pre := make([]byte, 0, len(treeRoot)+32+32)
	pre = append(pre, treeRoot...)
	pre = append(pre, utxoRoot.Bytes()...)
	pre = append(pre, utcoRoot.Bytes()...)
	return crypto.HashCheckRoot(pre)
}

// TestVerifyMinterSuccess 断言含 MintPKHash 路径下三段验证全通过。
func TestVerifyMinterSuccess(t *testing.T) {
	const current = uint32(100000)
	const txHeight = uint32(50000) // h=50000，在 [240,80000] 内
	mp, ds, _ := buildPKHashTxScenario(t, current, txHeight)

	cfg := MinterVerifyConfig{
		CurrentHeight: current,
		StateUTXORoot: crypto.EmptyUTXORoot(),
		StateUTCORoot: crypto.EmptyUTCORoot(),
		SigVerifier:   fakeMintSigVerifier{},
		EquiXSolver:   fakeEquiXSolver{},
	}
	if err := VerifyMinter(ds, mp, cfg); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

// TestVerifyMinterHeightOutOfWindow 断言高度不在窗口内被拒绝（第一段）。
func TestVerifyMinterHeightOutOfWindow(t *testing.T) {
	const current = uint32(100000)
	const txHeight = uint32(99999) // h=1，太近，不合格
	mp, ds, _ := buildPKHashTxScenario(t, current, txHeight)

	cfg := MinterVerifyConfig{
		CurrentHeight: current,
		StateUTXORoot: crypto.EmptyUTXORoot(),
		StateUTCORoot: crypto.EmptyUTCORoot(),
		SigVerifier:   fakeMintSigVerifier{},
		EquiXSolver:   fakeEquiXSolver{},
	}
	if err := VerifyMinter(ds, mp, cfg); err != ErrMintHeightOutOfWindow {
		t.Fatalf("expected ErrMintHeightOutOfWindow, got %v", err)
	}
}

// TestVerifyMinterTxIDMismatch 断言重算交易头哈希与 MintProof.TxID 不符被拒绝。
func TestVerifyMinterTxIDMismatch(t *testing.T) {
	const current = uint32(100000)
	const txHeight = uint32(50000)
	mp, ds, _ := buildPKHashTxScenario(t, current, txHeight)
	// 篡改检索到的交易头字段，使重算 TxID 不再匹配。
	ds.header.Timestamp += 1

	cfg := MinterVerifyConfig{
		CurrentHeight: current,
		StateUTXORoot: crypto.EmptyUTXORoot(),
		StateUTCORoot: crypto.EmptyUTCORoot(),
		SigVerifier:   fakeMintSigVerifier{},
		EquiXSolver:   fakeEquiXSolver{},
	}
	if err := VerifyMinter(ds, mp, cfg); err != ErrMintIdentityMismatch {
		t.Fatalf("expected ErrMintIdentityMismatch, got %v", err)
	}
}

// TestVerifyMinterCheckRootMismatch 断言目标区块 CheckRoot 不符被拒绝（第二段）。
func TestVerifyMinterCheckRootMismatch(t *testing.T) {
	const current = uint32(100000)
	const txHeight = uint32(50000)
	mp, ds, _ := buildPKHashTxScenario(t, current, txHeight)
	// 篡改数据源返回的 CheckRoot。
	ds.checkRoot[0] ^= 0xFF

	cfg := MinterVerifyConfig{
		CurrentHeight: current,
		StateUTXORoot: crypto.EmptyUTXORoot(),
		StateUTCORoot: crypto.EmptyUTCORoot(),
		SigVerifier:   fakeMintSigVerifier{},
		EquiXSolver:   fakeEquiXSolver{},
	}
	if err := VerifyMinter(ds, mp, cfg); err != ErrCheckRootMismatch {
		t.Fatalf("expected ErrCheckRootMismatch, got %v", err)
	}
}

// TestVerifyMinterEquiXUnavailable 断言未注入 Equi-X 求解器时返回 ErrEquiXUnavailable（第三段）。
func TestVerifyMinterEquiXUnavailable(t *testing.T) {
	const current = uint32(100000)
	const txHeight = uint32(50000)
	mp, ds, _ := buildPKHashTxScenario(t, current, txHeight)

	cfg := MinterVerifyConfig{
		CurrentHeight: current,
		StateUTXORoot: crypto.EmptyUTXORoot(),
		StateUTCORoot: crypto.EmptyUTCORoot(),
		SigVerifier:   fakeMintSigVerifier{},
		EquiXSolver:   nil, // 未注入
	}
	if err := VerifyMinter(ds, mp, cfg); err != ErrEquiXUnavailable {
		t.Fatalf("expected ErrEquiXUnavailable, got %v", err)
	}
}

// TestVerifyMinterEquiXSolutionInvalid 断言 Equi-X solution 验证失败被拒绝（第三段）。
func TestVerifyMinterEquiXSolutionInvalid(t *testing.T) {
	const current = uint32(100000)
	const txHeight = uint32(50000)
	mp, ds, _ := buildPKHashTxScenario(t, current, txHeight)

	cfg := MinterVerifyConfig{
		CurrentHeight: current,
		StateUTXORoot: crypto.EmptyUTXORoot(),
		StateUTCORoot: crypto.EmptyUTCORoot(),
		SigVerifier:   fakeMintSigVerifier{},
		EquiXSolver:   fakeEquiXSolver{returnInvalid: true},
	}
	if err := VerifyMinter(ds, mp, cfg); err != ErrEquiXSolutionInvalid {
		t.Fatalf("expected ErrEquiXSolutionInvalid, got %v", err)
	}
}

// TestVerifyMinterMintHashMismatch 断言重算铸凭哈希与凭证不符被拒绝（第三段）。
func TestVerifyMinterMintHashMismatch(t *testing.T) {
	const current = uint32(100000)
	const txHeight = uint32(50000)
	mp, ds, _ := buildPKHashTxScenario(t, current, txHeight)
	// 篡改 MintProof.MintHash。
	mp.MintHash[0] ^= 0xFF

	cfg := MinterVerifyConfig{
		CurrentHeight: current,
		StateUTXORoot: crypto.EmptyUTXORoot(),
		StateUTCORoot: crypto.EmptyUTCORoot(),
		SigVerifier:   fakeMintSigVerifier{},
		EquiXSolver:   fakeEquiXSolver{},
	}
	if err := VerifyMinter(ds, mp, cfg); err != ErrMintHashMismatch {
		t.Fatalf("expected ErrMintHashMismatch, got %v", err)
	}
}

// TestVerifyMinterBadSignature 断言铸造者签名无效被拒绝（第三段）。
func TestVerifyMinterBadSignature(t *testing.T) {
	const current = uint32(100000)
	const txHeight = uint32(50000)
	mp, ds, _ := buildPKHashTxScenario(t, current, txHeight)
	mp.Signature = []byte("bad-sig")

	cfg := MinterVerifyConfig{
		CurrentHeight: current,
		StateUTXORoot: crypto.EmptyUTXORoot(),
		StateUTCORoot: crypto.EmptyUTCORoot(),
		SigVerifier:   fakeMintSigVerifier{},
		EquiXSolver:   fakeEquiXSolver{},
	}
	if err := VerifyMinter(ds, mp, cfg); err != ErrMintSignatureInvalid {
		t.Fatalf("expected ErrMintSignatureInvalid, got %v", err)
	}
}

// TestYearStartFallbackHeights 断言年初回退检索：高度在真实年初 1 天内时
// 返回包含当年与上一年度的两次检索高度集；否则只返回当年。
func TestYearStartFallbackHeights(t *testing.T) {
	tests := []struct {
		name      string
		height    uint32
		wantCount int
	}{
		{"year zero start has no previous year", 100, 1},
		{"second year start within 1 day falls back", types.BlocksPerYear + 50, 2},
		{"deep into second year no fallback", types.BlocksPerYear + 5000, 1},
		{"first year deep no fallback", 50000, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := YearSearchHeights(tt.height)
			if len(got) != tt.wantCount {
				t.Fatalf("YearSearchHeights(%d) = %v (len %d), want %d entries",
					tt.height, got, len(got), tt.wantCount)
			}
		})
	}
}
