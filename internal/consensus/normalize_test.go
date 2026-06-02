package consensus

import (
	"bytes"
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// mockBlockID 构造固定宽度的测试用 BlockID：前缀 b 填充至 48 字节。
func mockBlockID(b byte) types.BlockID {
	var id types.BlockID
	id[0] = b
	return id
}

func TestNormalizeMultiSig(t *testing.T) {
	minter1 := [32]byte{1}
	minter2 := [32]byte{2}

	cases := []struct {
		name  string
		input []ForkBlock
		// 期望输出中各铸造者的 BlockID（用于识别保留了哪一块）
		wantByMinter map[[32]byte]types.BlockID
	}{
		{
			name:         "空输入",
			input:        nil,
			wantByMinter: map[[32]byte]types.BlockID{},
		},
		{
			name: "单块直接返回",
			input: []ForkBlock{
				{BlockID: mockBlockID(1), MintPKHash: minter1, MinterReward: 100},
			},
			wantByMinter: map[[32]byte]types.BlockID{minter1: mockBlockID(1)},
		},
		{
			name: "同铸造者保留低收益块",
			input: []ForkBlock{
				{BlockID: mockBlockID(0x10), MintPKHash: minter1, MinterReward: 200, TotalFee: 50},
				{BlockID: mockBlockID(0x11), MintPKHash: minter1, MinterReward: 100, TotalFee: 80},
			},
			wantByMinter: map[[32]byte]types.BlockID{minter1: mockBlockID(0x11)},
		},
		{
			name: "同铸造者收益相同保留低总费",
			input: []ForkBlock{
				{BlockID: mockBlockID(0x10), MintPKHash: minter1, MinterReward: 100, TotalFee: 200},
				{BlockID: mockBlockID(0x11), MintPKHash: minter1, MinterReward: 100, TotalFee: 50},
			},
			wantByMinter: map[[32]byte]types.BlockID{minter1: mockBlockID(0x11)},
		},
		{
			name: "同铸造者收益和总费均相同保留小 BlockID",
			input: []ForkBlock{
				{BlockID: mockBlockID(0x02), MintPKHash: minter1, MinterReward: 100, TotalFee: 50},
				{BlockID: mockBlockID(0x01), MintPKHash: minter1, MinterReward: 100, TotalFee: 50},
			},
			wantByMinter: map[[32]byte]types.BlockID{minter1: mockBlockID(0x01)},
		},
		{
			name: "不同铸造者均保留",
			input: []ForkBlock{
				{BlockID: mockBlockID(0x01), MintPKHash: minter1, MinterReward: 100},
				{BlockID: mockBlockID(0x02), MintPKHash: minter2, MinterReward: 200},
			},
			wantByMinter: map[[32]byte]types.BlockID{
				minter1: mockBlockID(0x01),
				minter2: mockBlockID(0x02),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := NormalizeMultiSig(tc.input)
			// 构造实际 by-minter 映射
			got := make(map[[32]byte]types.BlockID, len(out))
			for _, b := range out {
				got[b.MintPKHash] = b.BlockID
			}
			if len(got) != len(tc.wantByMinter) {
				t.Fatalf("len mismatch: got %d, want %d", len(got), len(tc.wantByMinter))
			}
			for minter, wantID := range tc.wantByMinter {
				gotID, ok := got[minter]
				if !ok {
					t.Errorf("minter %v missing in output", minter)
					continue
				}
				if !bytes.Equal(gotID[:], wantID[:]) {
					t.Errorf("minter %v: got BlockID[0]=%x, want %x", minter, gotID[0], wantID[0])
				}
			}
		})
	}
}

func TestNormalizeTxVolume(t *testing.T) {
	// 制造具有不同 PoolRank 的候选块
	cases := []struct {
		name       string
		candidates []ForkBlock
		wantRank   int // 期望胜出的 PoolRank（-1=empty）
		wantOK     bool
	}{
		{
			name:       "空输入",
			candidates: nil,
			wantRank:   -1,
			wantOK:     false,
		},
		{
			name: "单候选直接胜出",
			candidates: []ForkBlock{
				{PoolRank: 0, Stakes: 100, TxCount: 10},
			},
			wantRank: 0,
			wantOK:   true,
		},
		{
			name: "后位未超 2x，停止于 rank0",
			candidates: []ForkBlock{
				{PoolRank: 0, Stakes: 100, TxCount: 10},
				{PoolRank: 1, Stakes: 200, TxCount: 20}, // Stakes=2x 但非严格 >2x
			},
			wantRank: 0,
			wantOK:   true,
		},
		{
			name: "后位严格超 2x，替换 winner",
			candidates: []ForkBlock{
				{PoolRank: 0, Stakes: 100, TxCount: 10},
				{PoolRank: 1, Stakes: 201, TxCount: 21}, // 严格 >2x
			},
			wantRank: 1,
			wantOK:   true,
		},
		{
			name: "连续多次超越",
			candidates: []ForkBlock{
				{PoolRank: 0, Stakes: 10, TxCount: 5},
				{PoolRank: 1, Stakes: 21, TxCount: 11},  // >2x → 替换
				{PoolRank: 2, Stakes: 43, TxCount: 23},  // >2x → 替换
				{PoolRank: 3, Stakes: 87, TxCount: 47},  // >2x → 替换
				{PoolRank: 4, Stakes: 100, TxCount: 48}, // TxCount 未超 2x，停止
			},
			wantRank: 3,
			wantOK:   true,
		},
		{
			name: "winner.Stakes==0，后位 Stakes>0 即满足 Stakes 条件",
			candidates: []ForkBlock{
				{PoolRank: 0, Stakes: 0, TxCount: 5},
				{PoolRank: 1, Stakes: 1, TxCount: 11}, // TxCount >2x，Stakes>0 满足
			},
			wantRank: 1,
			wantOK:   true,
		},
		{
			name: "winner.TxCount==0，后位 TxCount>0 满足",
			candidates: []ForkBlock{
				{PoolRank: 0, Stakes: 10, TxCount: 0},
				{PoolRank: 1, Stakes: 21, TxCount: 1}, // Stakes >2x，TxCount>0 满足
			},
			wantRank: 1,
			wantOK:   true,
		},
		{
			name: "TxCount 相等不算超越",
			candidates: []ForkBlock{
				{PoolRank: 0, Stakes: 10, TxCount: 10},
				{PoolRank: 1, Stakes: 21, TxCount: 10}, // TxCount 相等 ≠ >2x
			},
			wantRank: 0,
			wantOK:   true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			winner, ok := NormalizeTxVolume(tc.candidates)
			if ok != tc.wantOK {
				t.Fatalf("ok: got %v, want %v", ok, tc.wantOK)
			}
			if !ok {
				return
			}
			if winner.PoolRank != tc.wantRank {
				t.Errorf("winner.PoolRank: got %d, want %d", winner.PoolRank, tc.wantRank)
			}
		})
	}
}

func TestSelectCandidate(t *testing.T) {
	// 集成测试：多签归一化后的 2 倍交易量归一化
	minter1 := [32]byte{1}
	minter2 := [32]byte{2}

	// 同铸造者1 有两块（高收益/低收益），铸造者2 有一块
	candidates := []ForkBlock{
		{BlockID: mockBlockID(0xAA), MintPKHash: minter1, MinterReward: 500, Stakes: 100, TxCount: 10, PoolRank: 0},
		{BlockID: mockBlockID(0xAB), MintPKHash: minter1, MinterReward: 200, Stakes: 100, TxCount: 10, PoolRank: 0},
		{BlockID: mockBlockID(0xBB), MintPKHash: minter2, MinterReward: 300, Stakes: 201, TxCount: 21, PoolRank: 1},
	}

	// 多签归一化后：minter1 → MinterReward=200 的块，minter2 保留自身
	// 然后 minter2 的 Stakes=201>100×2，TxCount=21>10×2 → 替换 winner
	winner, ok := SelectCandidate(candidates)
	if !ok {
		t.Fatal("SelectCandidate: expected ok=true")
	}
	if winner.MintPKHash != minter2 {
		t.Errorf("winner should be minter2, got MintPKHash=%v", winner.MintPKHash)
	}
}
