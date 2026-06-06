package utxo

import (
	"context"

	"github.com/cxio/evidcoin/internal/tx"
	"github.com/cxio/evidcoin/pkg/types"
)

// ScriptVerifier 抽象解锁脚本/签名验证，由调用方（脚本层或更高层）注入，
// 避免状态层反向依赖 internal/script 具体执行器（防止循环依赖，第 09 章）。
//
// 注：plan 草案签名为 VerifyCoinSpend(ctx, entry, input tx.Input)，但 internal/tx
// 并无统一的 Input 类型（实为 LeadInput/RestInput）。此处适配为传入待花费 entry
// 与该输入的解锁脚本字节——验证器据此结合 entry.LockScript 执行栈式校验。
type ScriptVerifier interface {
	// VerifyCoinSpend 校验对 entry 的一次币金花费是否被 unlockScript 合法解锁。
	VerifyCoinSpend(ctx context.Context, entry Entry, unlockScript []byte) error
}

// SpendRef 描述批次内一次币金消费请求：来源类别、短引用与解锁脚本。
type SpendRef struct {
	// Kind 是输入来源类别，UTXO apply 仅接受 tx.InputCoin。
	Kind tx.InputKind
	// Ref 是来源输出短引用（Year + TxIDPart 前缀 + OutIndex）。
	Ref tx.OutPoint
	// UnlockScript 是解锁脚本字节，交给 ScriptVerifier。
	UnlockScript []byte
}

// NewOutput 描述批次内一个新产生的输出及其定位信息。仅币金类输出
// 进入 UTXO；存证/凭信被跳过（第 06〉09 章）。摘要标记不影响状态归属。
type NewOutput struct {
	// Year 是产生该输出的交易年度。
	Year uint64
	// TxID 是产生该输出的交易完整 TxID。
	TxID types.TxID
	// Height 是产生该输出的区块高度。
	Height uint32
	// Output 是输出 envelope（提供 Serial、Type、LockScript）。
	Output tx.Output
	// Coin 是币金详情，仅当 Output 为币金类型时有意义。
	Coin tx.Coin
}

// Batch 是一个区块对 UTXO 集的状态变更请求：先消费输入，再插入新输出。
// 两阶段顺序保证输入只能引用批次开始前（历史区块）已确认的 UTXO，
// 无法引用同批次新产生的输出（交易独立性，第 09 章 §1）。
type Batch struct {
	// Spends 是本批次消费的币金输入集合。
	Spends []SpendRef
	// Outputs 是本批次新产生的输出集合。
	Outputs []NewOutput
}

// Apply 将一个 Batch 应用到 UTXO 集（Coin 状态转移，第 09 章）。
//
// 阶段一（消费）：逐个解析并花费输入。非币金来源类别拒绝；引用解析仅针对批次
// 开始前的有效状态，故同批次新输出不可被引用（解析为 ErrNotFound）；同一输出在
// 批次内重复消费会因首次花费后不再可解析而被拒绝；脚本验证失败立即返回且不标记花费。
//
// 阶段二（插入）：仅币金且非自定义类输出进入 UTXO，其余（自定义类、存证、凭信）跳过；
// 普通交易无销毁位，销毁仅由 Coinbase BurnCoin 表达，不产出可花费项。
//
// 失败时可能留下部分变更，原子回滚由快照层（Task 8）承载。
func Apply(ctx context.Context, s *Store, v ScriptVerifier, b Batch) error {
	for _, sp := range b.Spends {
		if sp.Kind != tx.InputCoin {
			return ErrInputKindInvalid
		}
		e, err := s.Resolve(sp.Ref)
		if err != nil {
			return err
		}
		if v != nil {
			if err := v.VerifyCoinSpend(ctx, e, sp.UnlockScript); err != nil {
				return err
			}
		}
		if err := s.Spend(e.OutPoint()); err != nil {
			return err
		}
	}
	for i := range b.Outputs {
		o := b.Outputs[i]
		// 仅币金类输出进入 UTXO；存证/凭信跳过。摘要标记不影响归属。
		if o.Output.Type != tx.TypeCoin {
			continue
		}
		entry := Entry{
			Year:          o.Year,
			TxID:          o.TxID,
			OutIndex:      uint64(o.Output.Serial),
			Amount:        o.Coin.Amount,
			Receiver:      o.Coin.Receiver,
			LockScript:    o.Output.LockScript,
			CreatedHeight: o.Height,
		}
		if err := s.Put(entry); err != nil {
			return err
		}
	}
	return nil
}
