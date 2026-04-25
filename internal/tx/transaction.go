package tx

import (
	"errors"

	"github.com/cxio/evidcoin/pkg/types"
)

// Transaction 标准交易结构。
type Transaction struct {
	// Header 交易头（TxID 由此计算）
	Header TxHeader
	// Lead 首领输入
	Lead LeadInput
	// Rests 非首领输入列表
	Rests []RestInput
	// Outputs 输出项列表
	Outputs []Output
	// Unlocks 各输入对应的解锁数据（按输入顺序，不参与 TxID 计算）
	Unlocks []UnlockData
}

// ID 返回该交易的 TxID。
func (tx *Transaction) ID() types.Hash384 {
	return ComputeTxID(&tx.Header)
}

// InputCount 返回输入项总数（首领 + 非首领）。
func (tx *Transaction) InputCount() int {
	return 1 + len(tx.Rests)
}

// CoinFee 计算交易手续费。
// 手续费 = inputTotal - 所有 Coin 类型非销毁输出之和。
// 注意：该函数需要外部传入查询结果，tx 包本身不查询 UTXO。
func CoinFee(inputTotal int64, outputs []Output) (int64, error) {
	var outputTotal int64
	for _, o := range outputs {
		if o.Type() == types.OutTypeCoin && !o.IsDestroy() {
			outputTotal += o.Amount
		}
	}
	fee := inputTotal - outputTotal
	if fee < 0 {
		return 0, errors.New("negative fee: outputs exceed inputs")
	}
	return fee, nil
}

// CoinbaseTx 铸币交易（每个区块的第一笔交易，索引 [0]，无输入）。
type CoinbaseTx struct {
	// Header 标准交易头
	Header TxHeader
	// BlockHeight 区块高度
	BlockHeight int32
	// MeritProof 择优凭证字节（铸造者证明）
	MeritProof []byte
	// TotalReward 收益总额（聪）
	TotalReward int64
	// SelfData 自由数据（最大 255 字节）
	SelfData []byte
	// Outputs 收益分配输出列表
	Outputs []CoinbaseOutput
}

// CoinbaseOutput Coinbase 专用输出（配置字节低 4 位为目标类型）。
type CoinbaseOutput struct {
	// Target 奖励目标类型
	Target CoinbaseOutputTarget
	// Amount 金额
	Amount int64
	// Address 接收地址
	Address types.PubKeyHash
	// LockScript 锁定脚本（包含 SYS_AWARD 指令）
	LockScript []byte
	// IsBurn 是否为销毁输出（手续费中 50% 销毁）
	IsBurn bool
}

// ID 返回 Coinbase 交易的 TxID。
func (cb *CoinbaseTx) ID() types.Hash384 {
	return ComputeTxID(&cb.Header)
}

// ValidateSize 验证交易大小是否超限（不含解锁数据）。
func ValidateSize(serializedSize int) error {
	if serializedSize > types.MaxTxSize {
		return errors.New("transaction size exceeds limit")
	}
	return nil
}
