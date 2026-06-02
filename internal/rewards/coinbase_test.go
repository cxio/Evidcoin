package rewards

import (
	"testing"

	"github.com/cxio/evidcoin/internal/tx"
	"github.com/cxio/evidcoin/pkg/types"
)

func TestCoinbase(t *testing.T) {
	t.Run("IsPreHundredDay", func(t *testing.T) {
		cases := []struct {
			height uint32
			want   bool
		}{
			{0, true},
			{24000, true},
			{24001, false},
			{100000, false},
		}
		for _, c := range cases {
			if got := IsPreHundredDay(c.height); got != c.want {
				t.Errorf("IsPreHundredDay(%d) = %v, want %v", c.height, got, c.want)
			}
		}
	})

	t.Run("CoinbaseOutputCount", func(t *testing.T) {
		cases := []struct {
			height uint32
			want   int
		}{
			{0, 2},
			{24000, 2},
			{24001, 5},
			{1000000, 5},
		}
		for _, c := range cases {
			if got := CoinbaseOutputCount(c.height); got != c.want {
				t.Errorf("CoinbaseOutputCount(%d) = %d, want %d", c.height, got, c.want)
			}
		}
	})

	t.Run("RewardBase", func(t *testing.T) {
		// RewardBase = issuance + unburnedFee + reclaimedAward
		got := RewardBase(3000, 1000, 200)
		if got != 4200 {
			t.Errorf("RewardBase(3000,1000,200) = %d, want 4200", got)
		}
		// 全零。
		if got := RewardBase(0, 0, 0); got != 0 {
			t.Errorf("RewardBase(0,0,0) = %d, want 0", got)
		}
	})

	t.Run("BuildCoinbaseOutputs_pre", func(t *testing.T) {
		// 百日前：2 输出。
		rewards := DistributeReward(1000, 0)
		receivers := [][]byte{
			{0x01, 0x02}, // 铸凭者
			{0x03, 0x04}, // 校验组
		}
		outputs, err := BuildCoinbaseOutputs(rewards, receivers)
		if err != nil {
			t.Fatalf("BuildCoinbaseOutputs: %v", err)
		}
		if len(outputs) != 2 {
			t.Fatalf("got %d outputs, want 2", len(outputs))
		}
		for i, o := range outputs {
			if o.Type != tx.TypeCoin {
				t.Errorf("outputs[%d].Type = %v, want TypeCoin", i, o.Type)
			}
			if o.Serial != uint32(i) {
				t.Errorf("outputs[%d].Serial = %d, want %d", i, o.Serial, i)
			}
			// Payload 必须以 Amount varint 开头。
			amt, _, err := types.ReadVarUint(o.Payload)
			if err != nil {
				t.Errorf("outputs[%d]: failed to read amount varint: %v", i, err)
			}
			if types.Amount(amt) != rewards[i].Amount {
				t.Errorf("outputs[%d].amount = %d, want %d", i, amt, rewards[i].Amount)
			}
		}
	})

	t.Run("BuildCoinbaseOutputs_post", func(t *testing.T) {
		// 百日后：5 输出。
		rewards := DistributeReward(10000, 24001)
		receivers := [][]byte{
			{0x11}, {0x22}, {0x33}, {0x44}, {0x55},
		}
		outputs, err := BuildCoinbaseOutputs(rewards, receivers)
		if err != nil {
			t.Fatalf("BuildCoinbaseOutputs: %v", err)
		}
		if len(outputs) != 5 {
			t.Fatalf("got %d outputs, want 5", len(outputs))
		}
		for i, o := range outputs {
			if o.Serial != uint32(i) {
				t.Errorf("outputs[%d].Serial = %d, want %d", i, o.Serial, i)
			}
		}
	})

	t.Run("BuildCoinbaseOutputs_receiver_mismatch", func(t *testing.T) {
		rewards := DistributeReward(1000, 0)
		_, err := BuildCoinbaseOutputs(rewards, [][]byte{{0x01}}) // 1 地址，2 奖励 → 错误
		if err != ErrReceiverCountMismatch {
			t.Errorf("expected ErrReceiverCountMismatch, got %v", err)
		}
	})
}
