package consensus

import (
	"testing"
	"time"

	"github.com/cxio/evidcoin/pkg/types"
)

func TestBlockTimeAt(t *testing.T) {
	// 测试依据：BlockTime(genesis, h) = genesis + h × 6min（DEC-0302）。
	// 创世时间戳为 C-9 占位（0），测试只验证推导逻辑，不依赖具体 mainnet 值。
	genesis := time.Unix(GenesisTimestamp(), 0).UTC()

	cases := []struct {
		height   types.BlockHeight
		wantDiff time.Duration
	}{
		{0, 0},
		{1, 6 * time.Minute},
		{10, 60 * time.Minute},
		{types.BlocksPerYear, time.Duration(types.BlocksPerYear) * 6 * time.Minute},
	}

	for _, tc := range cases {
		got := BlockTimeAt(tc.height)
		want := genesis.Add(tc.wantDiff)
		if !got.Equal(want) {
			t.Errorf("BlockTimeAt(%d): got %v, want %v", tc.height, got, want)
		}
	}
}

func TestBlockTimeAtUnix(t *testing.T) {
	genesis := time.Unix(GenesisTimestamp(), 0).UTC()

	cases := []types.BlockHeight{0, 1, 100, 87661}
	for _, h := range cases {
		got := BlockTimeAtUnix(h)
		want := genesis.Add(time.Duration(h) * types.BlockInterval).Unix()
		if got != want {
			t.Errorf("BlockTimeAtUnix(%d): got %d, want %d", h, got, want)
		}
	}
}

func TestRedundantPublishTime(t *testing.T) {
	// 共约：第 0 位在 BlockTime(h)+30s 发布，后续每位再加 15s。
	base := BlockTimeAt(0).Add(FirstBlockExtraDelay())
	cases := []struct {
		rank int
		want time.Time
	}{
		{0, base},
		{1, base.Add(15 * time.Second)},
		{2, base.Add(30 * time.Second)},
		{5, base.Add(75 * time.Second)},
	}

	for _, tc := range cases {
		got := RedundantPublishTime(0, tc.rank)
		if !got.Equal(tc.want) {
			t.Errorf("RedundantPublishTime(0, %d): got %v, want %v", tc.rank, got, tc.want)
		}
	}
}

func TestRedundantBroadcastDelay(t *testing.T) {
	if d := RedundantBroadcastDelay(); d != 15*time.Second {
		t.Errorf("RedundantBroadcastDelay: got %v, want 15s", d)
	}
}

func TestFirstBlockExtraDelay(t *testing.T) {
	if d := FirstBlockExtraDelay(); d != 30*time.Second {
		t.Errorf("FirstBlockExtraDelay: got %v, want 30s", d)
	}
}
