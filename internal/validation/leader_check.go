package validation

import (
	"time"

	"github.com/cxio/evidcoin/internal/tx"
	"github.com/cxio/evidcoin/pkg/types"
)

// LeaderCheckWindow 是首领校验黑名单的默认冻结时长（约 24 小时，第 13 章 §2）。
// 冻结时长为本地策略参数，区块验证不得因首领输入处于本地黑名单而拒绝合法区块。
const LeaderCheckWindow = 24 * time.Hour

// CoinStakeLookup 提供首领校验所需的币金币权查询接口。
// 查询结果必须来自已验证的本地 UTXO 集，不能信任外部未验证数据。
type CoinStakeLookup interface {
	// CoinStake 返回指定 OutPoint 对应的币权（Amount × 持有时长，单位聪时）。
	// 若 OutPoint 不存在或已花费，返回 0, false。
	CoinStake(ref tx.OutPoint) (stake uint64, found bool)
}

// CheckLeadInput 执行首领校验（第 13 章 §2）：
// 仅验证交易首笔输入，合法即通过，用于加快全网传播。
//
// 约束（须同时满足）：
//  1. 首笔输入由定义即为币金（InputCoin），无需额外类别检查。
//  2. 首笔输入在全部币金输入中必须具有最大币权。
//
// lookup 由调用方提供并基于本地 UTXO 集；bl 为 nil 时跳过黑名单检查。
// 返回 nil 表示首领校验通过。
func CheckLeadInput(lead tx.LeadInput, rest []tx.RestInput, lookup CoinStakeLookup, bl *Blacklist) error {
	if bl != nil && bl.Contains(lead.Ref) {
		return ErrBlacklisted
	}
	leadStake, found := lookup.CoinStake(lead.Ref)
	if !found {
		return ErrLeadInputNotFound
	}
	// 检查所有其余币金输入，确认没有比首领更大的币权。
	for _, r := range rest {
		if r.Kind != tx.InputCoin {
			continue
		}
		s, ok := lookup.CoinStake(r.Ref)
		if !ok {
			continue
		}
		if s > leadStake {
			return ErrLeadInputNotMaxStake
		}
	}
	return nil
}

// outPointKey 将 tx.OutPoint 短引用编码为可作 map 键的字符串。
// tx.OutPoint.TxIDPart 是切片，不可直接作 map 键；故以规范变长整数编码序列化后转字符串。
func outPointKey(ref tx.OutPoint) string {
	key := make([]byte, 0, 24)
	key = types.AppendVarUint(key, ref.Year)
	key = types.AppendBytes(key, ref.TxIDPart)
	key = types.AppendVarUint(key, ref.OutIndex)
	return string(key)
}

// BlacklistEntry 是黑名单中的单条冻结记录。
type BlacklistEntry struct {
	// FrozenAt 是该 OutPoint 进入黑名单的时刻。
	FrozenAt time.Time
}

// Blacklist 是首领校验黑名单（第 13 章 §2）。
// 记录通过首领校验但最终完整验证失败的交易首笔输入，临时冻结约 24 小时，
// 以限制洪流消耗攻击的资源可用性。
//
// 注意：黑名单为本地策略，区块验证不得因此拒绝合法区块（不属于共识规则）。
type Blacklist struct {
	window  time.Duration
	entries map[string]BlacklistEntry
}

// NewBlacklist 创建空黑名单，冻结窗口为 window；传 0 使用默认时长（LeaderCheckWindow）。
func NewBlacklist(window time.Duration) *Blacklist {
	if window <= 0 {
		window = LeaderCheckWindow
	}
	return &Blacklist{
		window:  window,
		entries: make(map[string]BlacklistEntry),
	}
}

// Add 将指定 OutPoint 加入黑名单并记录冻结时刻。
// 若已存在则刷新冻结时刻（重置冻结窗口）。
func (bl *Blacklist) Add(ref tx.OutPoint, now time.Time) {
	bl.entries[outPointKey(ref)] = BlacklistEntry{FrozenAt: now}
}

// Contains 报告指定 OutPoint 当前是否在有效冻结期内（未过期）。
// 过期记录视为不冻结，但不自动清理——可调用 Purge 定期清理。
func (bl *Blacklist) Contains(ref tx.OutPoint) bool {
	e, ok := bl.entries[outPointKey(ref)]
	if !ok {
		return false
	}
	return time.Since(e.FrozenAt) < bl.window
}

// Purge 清理所有已过期的黑名单记录。调用方可按需定期执行。
func (bl *Blacklist) Purge(now time.Time) {
	for k, e := range bl.entries {
		if now.Sub(e.FrozenAt) >= bl.window {
			delete(bl.entries, k)
		}
	}
}

// Len 返回黑名单中的记录总数（含已过期但未清理的条目）。
func (bl *Blacklist) Len() int { return len(bl.entries) }
