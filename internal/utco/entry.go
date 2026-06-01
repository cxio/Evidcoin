// Package utco 维护已确认交易的未转移凭信输出（UTCO）集，提供 entry 结构、
// 内存状态集、局部引用解析、Credit 状态转移、到期清理、五层宽成员状态指纹与
// 快照回滚（第 09 章、DEC-0201）。本包属 Layer 2，仅依赖 pkg/types、pkg/crypto、
// pkg/hashtree 与 internal/tx，不依赖脚本 VM 具体执行器（通过 ScriptVerifier
// 接口注入），亦不处理 Coin 金额守恒。
//
// UTCO 与 UTXO 采用相同的状态指纹结构与算法，但作为各自独立服务运行，且域标签
// （utco.leaf/utco.empty）与 UTXO 隔离，类型与 API 不可混用。
package utco

import (
	"github.com/cxio/evidcoin/internal/tx"
	"github.com/cxio/evidcoin/pkg/types"
)

// OutPoint 是 UTCO 集内一个凭信输出的完整定位三元组：交易年度、完整 TxID、
// 输出序位。它与 internal/tx 的短引用 OutPoint（仅含 TxIDPart 前缀）不同。
type OutPoint struct {
	// Year 是来源交易的真实年度（按时间戳计），UTC 自然年。
	Year uint64
	// TxID 是来源交易的完整 TxID（48 字节）。
	TxID types.TxID
	// OutIndex 是来源交易输出集中的序位下标。
	OutIndex uint64
}

// Entry 是一条 UTCO 记录：一个凭信输出在状态集中的完整快照。
// Receiver/Creator/Title/Description/AttachmentID/LockScript 等详情属缓存集，
// 不参与状态指纹；状态指纹只取 TxID、有效输出数与各序位的有效位。
//
// Credit 的不可变字段为 Creator/Title/Description/AttachmentID；转移时这些字段
// 必须保持不变，仅可更换 Receiver（持有人），见 apply.go。
type Entry struct {
	// Year 是来源交易年度。
	Year uint64
	// TxID 是来源交易的完整 TxID。
	TxID types.TxID
	// OutIndex 是输出序位下标。
	OutIndex uint64
	// Receiver 是当前持有人（可在转移时更换）。
	Receiver []byte
	// Creator 是创建者或创建者引用（不可变）。
	Creator []byte
	// Title 是标题（不可变）。
	Title []byte
	// Description 是描述（不可变）。
	Description []byte
	// AttachmentID 是可选附件 ID 已编码字节（不可变）。
	AttachmentID []byte
	// LockScript 是锁定脚本。
	LockScript []byte
	// CreatedHeight 是产生该凭信输出的交易所在区块高度，用于计算币龄与过期。
	CreatedHeight uint32
	// Spent 是有效位的反面：true 表示已转出/消费（状态位记为 0）。
	Spent bool
}

// OutPoint 返回该 entry 的完整定位三元组，供状态集映射键使用。
func (e Entry) OutPoint() OutPoint {
	return OutPoint{Year: e.Year, TxID: e.TxID, OutIndex: e.OutIndex}
}

// Age 返回该凭信在给定当前区块高度下的币龄（区块高度差）。
// 当 currentHeight < CreatedHeight 时返回 0（不应出现于已确认状态）。
func (e Entry) Age(currentHeight uint32) uint64 {
	if currentHeight < e.CreatedHeight {
		return 0
	}
	return uint64(currentHeight) - uint64(e.CreatedHeight)
}

// Expired 报告该凭信在给定当前区块高度下是否已过期失效。
// 失效条件为 age > CreditMaxAge（31 × BlocksPerYear）；边界相等仍可被引用花销
// （DEC-0101，第 07 章 §5）。过期判定复用 internal/tx 的权威规则。
func (e Entry) Expired(currentHeight uint32) bool {
	return tx.CreditExpired(e.Age(currentHeight))
}
