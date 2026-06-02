package rewards

import (
	"github.com/cxio/evidcoin/internal/tx"
	"github.com/cxio/evidcoin/pkg/types"
)

// IsPreHundredDay 报告指定高度是否属于百日前阶段（height <= 24000，DEC-0401）。
func IsPreHundredDay(height uint32) bool {
	return height <= hundredDayBoundary
}

// CoinbaseOutputCount 返回指定高度的 Coinbase 输出数量（百日前 2，百日后 5，DEC-0401）。
func CoinbaseOutputCount(height uint32) int {
	if IsPreHundredDay(height) {
		return 2
	}
	return 5
}

// BuildCoinbaseOutputs 根据奖励分配结果与接收地址列表构建 Coinbase 输出集（DEC-0401）。
//
// rewards 是由 DistributeReward 返回的奖励列表（按配置值升序排列）。
// receivers 是与 rewards 一一对应的接收者公钥哈希，长度须与 rewards 相等。
// 返回的输出集 Serial 从 0 开始，与 rewards 顺序一致（配置值升序）。
// 公共服务输出（Config=3/4/5）的 SYS_AWARD 锁定脚本由脚本层负责，本层 LockScript 为空。
func BuildCoinbaseOutputs(rewards []RewardOutput, receivers [][]byte) ([]tx.Output, error) {
	if len(receivers) != len(rewards) {
		return nil, ErrReceiverCountMismatch
	}
	outputs := make([]tx.Output, len(rewards))
	for i, r := range rewards {
		coin := tx.Coin{
			Amount:   r.Amount,
			Receiver: receivers[i],
		}
		payload, err := coin.Payload()
		if err != nil {
			return nil, err
		}
		outputs[i] = tx.Output{
			Serial:  uint32(i),
			Type:    tx.TypeCoin,
			Payload: payload,
		}
	}
	return outputs, nil
}

// RewardBase 根据发行量、未销毁交易费与回收额计算奖励基数（第 14 章 §2、DEC-0401）：
//
//	RewardBase = issuance + unburned_tx_fee + reclaimed_award
func RewardBase(issuance, unburnedFee, reclaimedAward types.Amount) types.Amount {
	return issuance + unburnedFee + reclaimedAward
}
