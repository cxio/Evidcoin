package utxo

import "github.com/cxio/evidcoin/pkg/types"

// UTXOEntry 表示一个未花费的 Coin 输出项。
type UTXOEntry struct {
	TxID       types.Hash384 // 所在交易 ID
	OutIndex   int           // 输出序位
	TxYear     int           // 交易所在年度（UTC 年份）
	Amount     int64         // 币金数量（最小单位 chx）
	Address    [32]byte      // 接收地址（公钥哈希，32 字节）
	LockScript []byte        // 锁定脚本
	CoinAge    int64         // 已持有整小时数（用于币权计算）
	Spent      bool          // 是否已花费
}
