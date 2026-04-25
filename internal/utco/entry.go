package utco

import "github.com/cxio/evidcoin/pkg/types"

// UTCOEntry 表示一个未转移的 Credential 输出项。
type UTCOEntry struct {
	TxID        types.Hash384 // 所在交易 ID
	OutIndex    int           // 输出序位
	TxYear      int           // 交易所在年度（UTC 年份）
	Address     [32]byte      // 持有者地址（公钥哈希）
	LockScript  []byte        // 锁定脚本
	TransferMax int           // 最大转移次数（0=无限制）
	TransferCnt int           // 已转移次数
	ExpireAt    int64         // 过期时间戳（0=不过期）
	Transferred bool          // 是否已转移（最终状态）
}
