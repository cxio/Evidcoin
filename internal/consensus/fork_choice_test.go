package consensus

import (
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// buildSeg 构造单铸造者单块链段（简化测试用），每高度恰好一个候选块。
// mintHashes 中每个字节作为对应高度候选块 MintHash 的首字节。
func buildSeg(mintHashes []byte) ForkSegment {
	seg := make(ForkSegment, len(mintHashes))
	for i, b := range mintHashes {
		var mh types.MintHash
		mh[0] = b
		seg[i] = []ForkBlock{
			{MintHash: mh, PoolRank: 0, Stakes: 1, TxCount: 1},
		}
	}
	return seg
}

// buildTieSeg 构造全高度 MintHash 相等的链段（用于测试平局场景）。
func buildTieSeg(length int, val byte) ForkSegment {
	seg := make(ForkSegment, length)
	for i := range length {
		var mh types.MintHash
		mh[0] = val
		seg[i] = []ForkBlock{
			{MintHash: mh, PoolRank: 0, Stakes: 1, TxCount: 1},
		}
	}
	return seg
}

func TestForkChoiceBasic(t *testing.T) {
	cases := []struct {
		name string
		segA []byte
		segB []byte
		want ForkSide
	}{
		{
			name: "A 全胜",
			segA: []byte{0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
				0x01, 0x01, 0x01, 0x01, 0x01, 0x01},
			segB: []byte{0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02,
				0x02, 0x02, 0x02, 0x02, 0x02, 0x02},
			want: ForkSideA,
		},
		{
			name: "B 全胜",
			segA: []byte{0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02,
				0x02, 0x02, 0x02, 0x02, 0x02, 0x02},
			segB: []byte{0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
				0x01, 0x01, 0x01, 0x01, 0x01, 0x01},
			want: ForkSideB,
		},
		{
			name: "A 恰 16 分提前胜出",
			// A=0x01 < B=0x02，A 连得 16 分
			segA: []byte{0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
				0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
				0x02, 0x02, 0x02, 0x02},
			segB: []byte{0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02,
				0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02,
				0x01, 0x01, 0x01, 0x01},
			want: ForkSideA,
		},
		{
			name: "MintHash 全部相等 => 平局",
			segA: []byte{0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
				0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
				0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
				0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01},
			segB: []byte{0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
				0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
				0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
				0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01},
			want: ForkSideNone,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CompareForkSegments(buildSeg(tc.segA), buildSeg(tc.segB))
			if got != tc.want {
				t.Errorf("got %d, want %d", got, tc.want)
			}
		})
	}
}

func TestForkChoiceMintHashEqual(t *testing.T) {
	// MintHash 相等高度双方都不得分；但 A 在几个高度更小 → A 胜出
	// 16 高度：前 16 位 A=0x01 < B=0x02（A 得 16 分，第 16 次达阈值）
	// 最后 5 位相等 → 不影响已结束的比较
	segA := buildSeg([]byte{0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
		0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
		0x05, 0x05, 0x05, 0x05, 0x05})
	segB := buildSeg([]byte{0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02,
		0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02,
		0x05, 0x05, 0x05, 0x05, 0x05})
	got := CompareForkSegments(segA, segB)
	if got != ForkSideA {
		t.Errorf("expected ForkSideA, got %d", got)
	}
}

func TestForkChoiceEqualMintHashNoScore(t *testing.T) {
	// 全 31 个高度 MintHash 完全相同 → 平局
	seg := buildTieSeg(31, 0xAA)
	got := CompareForkSegments(seg, seg)
	if got != ForkSideNone {
		t.Errorf("all equal: expected ForkSideNone, got %d", got)
	}
}

func TestForkChoiceTruncatesAt31(t *testing.T) {
	// 超过 31 个高度的链段仅取前 31 个比较；超出 31 的部分应被忽略。
	// A 前 16 高度更优 → 胜出；后续高度 B 更优，但 31 块内已决出
	aHashes := make([]byte, 40)
	bHashes := make([]byte, 40)
	for i := range 40 {
		if i < 16 {
			aHashes[i] = 0x01 // A 在前 16 高度更小
			bHashes[i] = 0x02
		} else {
			aHashes[i] = 0x02 // 后续 B 更小
			bHashes[i] = 0x01
		}
	}
	got := CompareForkSegments(buildSeg(aHashes), buildSeg(bHashes))
	if got != ForkSideA {
		t.Errorf("expected ForkSideA (truncated at 31), got %d", got)
	}
}

func TestValidateForkLength(t *testing.T) {
	cases := []struct {
		length  int
		wantErr bool
	}{
		{0, false},
		{1, false},
		{20, false}, // 恰好在接收上限
		{21, true},  // 超出上限
		{100, true},
	}
	for _, tc := range cases {
		err := ValidateForkLength(tc.length)
		if (err != nil) != tc.wantErr {
			t.Errorf("ValidateForkLength(%d): gotErr=%v, wantErr=%v", tc.length, err != nil, tc.wantErr)
		}
	}
}

func TestForkChoiceWith31Blocks(t *testing.T) {
	// 31 个高度，A 得 16 分 B 得 15 分 → A 胜出（刚好到达阈值）
	aHashes := make([]byte, 31)
	bHashes := make([]byte, 31)
	for i := range 31 {
		if i < 16 {
			aHashes[i] = 0x01
			bHashes[i] = 0x02
		} else {
			aHashes[i] = 0x02
			bHashes[i] = 0x01
		}
	}
	got := CompareForkSegments(buildSeg(aHashes), buildSeg(bHashes))
	if got != ForkSideA {
		t.Errorf("expected ForkSideA, got %d", got)
	}
}
