package consensus

import (
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

func TestFilterRecyclable(t *testing.T) {
	makeTxID := func(b byte) types.TxID {
		var id types.TxID
		id[0] = b
		return id
	}

	cases := []struct {
		name       string
		candidates []RecycleCandidate
		wantLen    int
		wantIDs    []byte // 期望出现的 TxID[0] 值（简化检查）
	}{
		{name: "空输入", candidates: nil, wantLen: 0},
		{
			name: "全部有效新币输入失效",
			candidates: []RecycleCandidate{
				{TxID: makeTxID(1), HasInvalidNewCoin: true},
				{TxID: makeTxID(2), HasInvalidNewCoin: true},
			},
			wantLen: 0,
		},
		{
			name: "全部可回收",
			candidates: []RecycleCandidate{
				{TxID: makeTxID(0xAA), HasInvalidNewCoin: false},
				{TxID: makeTxID(0xBB), HasInvalidNewCoin: false},
			},
			wantLen: 2,
			wantIDs: []byte{0xAA, 0xBB},
		},
		{
			name: "混合，过滤失效",
			candidates: []RecycleCandidate{
				{TxID: makeTxID(0x01), HasInvalidNewCoin: true},
				{TxID: makeTxID(0x02), HasInvalidNewCoin: false},
				{TxID: makeTxID(0x03), HasInvalidNewCoin: true},
				{TxID: makeTxID(0x04), HasInvalidNewCoin: false},
			},
			wantLen: 2,
			wantIDs: []byte{0x02, 0x04},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := FilterRecyclable(tc.candidates)
			if len(got) != tc.wantLen {
				t.Errorf("len: got %d, want %d", len(got), tc.wantLen)
			}
			for i, id := range got {
				if id[0] != tc.wantIDs[i] {
					t.Errorf("got[%d][0]=%x, want %x", i, id[0], tc.wantIDs[i])
				}
			}
		})
	}
}

func TestIsTxExpired(t *testing.T) {
	// 每块 6 分钟（360 秒），过期阈值 > 240 块 = > 86400 秒
	blockSec := int64(types.BlockInterval.Seconds()) // 360

	cases := []struct {
		name       string
		txTS       int64
		blockTS    int64
		wantExpire bool
	}{
		{"刚好 240 块（未超过）", 0, 240 * blockSec, false},
		{"241 块（超过 240）", 0, 241 * blockSec, true},
		{"0 块差", 1000, 1000, false},
		{"未来交易（负差）", 2000, 1000, false},
		{"1 块差", 0, blockSec, false},
		{"239 块", 0, 239 * blockSec, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsTxExpired(tc.txTS, tc.blockTS)
			if got != tc.wantExpire {
				t.Errorf("IsTxExpired(%d, %d): got %v, want %v",
					tc.txTS, tc.blockTS, got, tc.wantExpire)
			}
		})
	}
}

func TestIsFutureTx(t *testing.T) {
	cases := []struct {
		name       string
		txTS       int64
		blockTS    int64
		wantFuture bool
	}{
		{"txTS == blockTS（边界，不是未来）", 1000, 1000, false},
		{"txTS < blockTS（正常）", 999, 1000, false},
		{"txTS > blockTS（未来）", 1001, 1000, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsFutureTx(tc.txTS, tc.blockTS)
			if got != tc.wantFuture {
				t.Errorf("IsFutureTx(%d, %d): got %v, want %v",
					tc.txTS, tc.blockTS, got, tc.wantFuture)
			}
		})
	}
}

func TestMinFeeThreshold(t *testing.T) {
	cases := []struct {
		name string
		fees []uint64
		want uint64
	}{
		{"空列表", nil, 0},
		{"单笔费 1000", []uint64{1000}, 250},
		{"平均 2000，阈值 500", []uint64{1000, 3000}, 500},
		{"全零", []uint64{0, 0, 0}, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := MinFeeThreshold(tc.fees)
			if got != tc.want {
				t.Errorf("MinFeeThreshold: got %d, want %d", got, tc.want)
			}
		})
	}
}

func TestEndpointConstants(t *testing.T) {
	if MinFeeWindowSize != 6000 {
		t.Errorf("MinFeeWindowSize: got %d, want 6000", MinFeeWindowSize)
	}
	if NewCoinConfirmations != 31 {
		t.Errorf("NewCoinConfirmations: got %d, want 31", NewCoinConfirmations)
	}
}
