// Package utxo 维护已确认交易的未花费币金输出（UTXO）集，提供 entry 结构、
// 内存状态集、局部引用解析、Coin 状态转移、五层宽成员状态指纹与快照回滚
// （第 09 章、DEC-0201）。本包属 Layer 2，仅依赖 pkg/types、pkg/crypto、
// pkg/hashtree 与 internal/tx，不依赖脚本 VM 具体执行器（通过 ScriptVerifier
// 接口注入），亦不处理 Credit 语义。
package utxo

import "github.com/cxio/evidcoin/pkg/types"

// OutPoint 是 UTXO 集内一个币金输出的完整定位三元组：交易年度、完整 TxID、
// 输出序位。它与 internal/tx 的短引用 OutPoint（仅含 TxIDPart 前缀）不同——
// 这里使用完整 TxID，可作为状态集映射键（TxID 为定长数组，类型可比较）。
type OutPoint struct {
	// Year 是来源交易的真实年度（按时间戳计），UTC 自然年。
	Year uint64
	// TxID 是来源交易的完整 TxID（48 字节）。
	TxID types.TxID
	// OutIndex 是来源交易输出集中的序位下标。
	OutIndex uint64
}

// Entry 是一条 UTXO 记录：一个币金输出在状态集中的完整快照。
// 其中 Receiver/LockScript/Amount 等详情属缓存集（检索优化），不参与状态指纹；
// 状态指纹只取 TxID、有效输出数与各序位的有效位（见 fingerprint.go）。
type Entry struct {
	// Year 是来源交易年度。
	Year uint64
	// TxID 是来源交易的完整 TxID。
	TxID types.TxID
	// OutIndex 是输出序位下标。
	OutIndex uint64
	// Amount 是币金数量（chx）。
	Amount types.Amount
	// Receiver 是接收者公钥哈希或自定义验证序列。
	Receiver []byte
	// LockScript 是锁定脚本。
	LockScript []byte
	// CreatedHeight 是产生该输出的交易所在区块高度。
	CreatedHeight uint32
	// Spent 是有效位的反面：true 表示已花费（状态位记为 0）。
	Spent bool
}

// OutPoint 返回该 entry 的完整定位三元组，供状态集映射键使用。
func (e Entry) OutPoint() OutPoint {
	return OutPoint{Year: e.Year, TxID: e.TxID, OutIndex: e.OutIndex}
}
