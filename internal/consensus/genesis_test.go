package consensus

import (
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// TestInitialEvalRefHeight 断言初段评参区块高度规则（DEC-0302）：
// currentHeight < 8 取 0（创世块）；>= 8 取 currentHeight - 8。
func TestInitialEvalRefHeight(t *testing.T) {
	tests := []struct {
		name    string
		current uint32
		want    uint32
	}{
		{"height 0", 0, 0},
		{"height 1", 1, 0},
		{"height 2", 2, 0},
		{"height 7 still genesis", 7, 0},
		{"height 8 first normal ref", 8, 0},
		{"height 9", 9, 1},
		{"height 100", 100, 92},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := InitialEvalRefHeight(tt.current); got != tt.want {
				t.Fatalf("InitialEvalRefHeight(%d) = %d, want %d", tt.current, got, tt.want)
			}
		})
	}
}

// TestMintTxEligibleInitial 断言初段铸凭交易高度放宽（DEC-0302）：
// currentHeight < 480 时 txHeight < currentHeight（须已确认）；
// >= 480 用正常 h > 239 && h <= 80000。
func TestMintTxEligibleInitial(t *testing.T) {
	tests := []struct {
		name     string
		current  uint32
		txHeight uint32
		want     bool
	}{
		// 初段放宽区（current < 480）：任何已确认的更早交易均合格。
		{"h2 tx0 relaxed", 2, 0, true},
		{"h2 tx1 relaxed", 2, 1, true},
		{"h2 tx2 self not confirmed", 2, 2, false},
		{"h239 tx0 relaxed", 239, 0, true},
		{"h240 tx239 relaxed", 240, 239, true},
		{"h479 tx0 relaxed", 479, 0, true},
		{"h479 tx478 relaxed", 479, 478, true},
		{"h479 tx479 self not confirmed", 479, 479, false},
		// 边界 480 起切回正常窗口。
		{"h480 tx479 too recent under normal", 480, 479, false}, // h=1，太近
		{"h480 tx240 normal pass", 480, 240, true},              // h=240，下界合格
		{"h480 tx241 normal pass", 480, 241, false},             // h=239，仍太近
		{"h480 tx0 normal pass", 480, 0, true},                  // h=480，窗口内合格
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MintTxEligibleInitial(tt.current, tt.txHeight); got != tt.want {
				t.Fatalf("MintTxEligibleInitial(%d,%d) = %v, want %v",
					tt.current, tt.txHeight, got, tt.want)
			}
		})
	}
}

// TestMintTxEligibleInitialMatchesNormalAtBoundary 断言 currentHeight >= 480 时
// 初段判定与正常期判定完全一致（边界 479/480 切换）。
func TestMintTxEligibleInitialMatchesNormalAtBoundary(t *testing.T) {
	const current = uint32(480)
	for txHeight := uint32(0); txHeight < current; txHeight++ {
		init := MintTxEligibleInitial(current, txHeight)
		norm := MintTxEligibleNormal(current, txHeight)
		if init != norm {
			t.Fatalf("at current=%d txHeight=%d: initial=%v normal=%v", current, txHeight, init, norm)
		}
	}
}

// TestInitialStageActive 断言初段标志：高度 < 480 为初段放宽期。
func TestInitialStageActive(t *testing.T) {
	tests := []struct {
		current uint32
		want    bool
	}{
		{0, true},
		{1, true},
		{479, true},
		{480, false},
		{1000, false},
	}
	for _, tt := range tests {
		if got := InInitialMintWindow(tt.current); got != tt.want {
			t.Fatalf("InInitialMintWindow(%d) = %v, want %v", tt.current, got, tt.want)
		}
	}
}

// TestGenesisMintHashAllZero 断言创世块 MintHash 定义为 32 字节全零。
func TestGenesisMintHashAllZero(t *testing.T) {
	got := GenesisMintHash()
	var zero types.MintHash
	if got != zero {
		t.Fatalf("GenesisMintHash() = %x, want all-zero", got.Bytes())
	}
}

// TestGenesisArtifactStructure 断言创世工件结构含五项，且 C-9 占位字段为未冻结标志。
func TestGenesisArtifactStructure(t *testing.T) {
	art := GenesisArtifact{
		HeaderBytes:        []byte{0x01},
		CoinbaseBytes:      []byte{0x02},
		CoinbaseSignature:  []byte{0x03},
		CheckRootSignature: []byte{0x04},
		FreeData:           []byte("genesis declaration"),
	}
	// 五项工件均可独立访问。
	if len(art.HeaderBytes) == 0 || len(art.CoinbaseBytes) == 0 ||
		len(art.CoinbaseSignature) == 0 || len(art.CheckRootSignature) == 0 ||
		len(art.FreeData) == 0 {
		t.Fatal("genesis artifact must carry all five components")
	}

	// C-9 占位：GenesisID 与 Timestamp 未冻结，必须保持零/未定状态而非虚构值。
	if !GenesisParamsUnfrozen() {
		t.Fatal("C-9 genesis params must remain unfrozen (no fabricated mainnet values)")
	}
	if MainnetGenesisID() != (types.BlockID{}) {
		t.Fatal("C-9: mainnet Genesis-ID must not be fabricated before decision")
	}
	if GenesisTimestamp() != GenesisTimestampPlaceholder {
		t.Fatal("C-9: genesis timestamp must equal documented placeholder")
	}
}

// TestGenesisCoinbaseEligibilityBoundaries 断言 Coinbase 铸凭资格在初段从 #2 起、
// 显式 MintPKHash、已确认的边界场景（DEC-0302）。
func TestGenesisCoinbaseEligibilityBoundaries(t *testing.T) {
	var pk [32]byte
	for i := range pk {
		pk[i] = 0x5A
	}
	// 显式非零 MintPKHash → 资格判定通过（高度窗口另行判定）。
	if err := VerifyCoinbaseMintEligibility(pk); err != nil {
		t.Fatalf("explicit MintPKHash coinbase should be eligible: %v", err)
	}
	// #1（创世块自身）不可作铸凭交易竞争来源：高度 0 在初段放宽下要求 txHeight < current。
	if MintTxEligibleInitial(1, 1) {
		t.Fatal("#1 cannot reference itself (current block tx)")
	}
}
