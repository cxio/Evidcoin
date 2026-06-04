package utco

import (
	"context"

	"github.com/cxio/evidcoin/internal/tx"
	"github.com/cxio/evidcoin/pkg/types"
)

// ScriptVerifier 抽象凭信转移的解锁脚本/签名验证，由调用方（脚本层或更高层）
// 注入，避免状态层反向依赖 internal/script 具体执行器（防止循环依赖，第 09 章）。
//
// 注：plan 草案签名为 VerifyCreditTransfer(ctx, entry, input tx.Input)，但 internal/tx
// 并无统一的 Input 类型（实为 LeadInput/RestInput）。此处适配为传入待转出 entry
// 与该输入的解锁脚本字节——验证器据此结合 entry.LockScript 执行栈式校验。
//
// 转移后输出字段的约束（如哪些字段可变、是否要求产生新输出）由锁定脚本决定；
// 状态层不施加硬性约束（第 07 章 §5）。
type ScriptVerifier interface {
	// VerifyCreditTransfer 校验对 entry 的一次凭信转移是否被 unlockScript 合法解锁。
	VerifyCreditTransfer(ctx context.Context, entry Entry, unlockScript []byte) error
}

// NewOutput 描述批次内一个新产生的输出及其定位信息。仅凭信（TypeCredit）且非
// 自定义类的输出进入 UTCO；自定义类、币金、存证类被跳过（第 06、09 章）。
type NewOutput struct {
	// Year 是产生该输出的交易年度。
	Year uint64
	// TxID 是产生该输出的交易完整 TxID。
	TxID types.TxID
	// Height 是产生该输出的区块高度，用于计算币龄与过期。
	Height uint32
	// Output 是输出 envelope（提供 Serial、IsCustom、Type、LockScript）。
	Output tx.Output
	// Credit 是凭信详情，仅当 Output 为凭信类型时有意义。
	Credit tx.Credit
}

// Transfer 描述批次内一次凭信转移请求：来源类别、短引用、解锁脚本与可选的新
// 凭信输出。凭信为一次性消费：消费旧 UTCO 后，若 NewOutput 非 nil 则插入新 UTCO，
// 若 NewOutput 为 nil 则视为凭信销毁（第 07 章 §5）。输出字段约束由锁定脚本决定。
type Transfer struct {
	// Kind 是输入来源类别，UTCO apply 仅接受 tx.InputCredit。
	Kind tx.InputKind
	// Ref 是被转移凭信的短引用（Year + TxIDPart 前缀 + OutIndex）。
	Ref tx.OutPoint
	// UnlockScript 是解锁脚本字节，交给 ScriptVerifier。
	UnlockScript []byte
	// NewOutput 是转移后新产生的凭信输出；nil 表示销毁，不产生新 UTCO。
	NewOutput *NewOutput
}

// Batch 是一个区块对 UTCO 集的状态变更请求：先消费转移输入，再插入新输出。
// 两阶段顺序保证转移只能引用批次开始前（历史区块）已确认且未过期的 UTCO，
// 无法引用同批次新产生的凭信（交易独立性，第 09 章 §1）。
type Batch struct {
	// CurrentHeight 是本批次所在区块高度，用于转移解析时的过期判定。
	CurrentHeight uint32
	// Transfers 是本批次的凭信转移集合。
	Transfers []Transfer
	// Creations 是本批次新建（非转移）的输出集合。
	Creations []NewOutput
}

// Apply 将一个 Batch 应用到 UTCO 集（Credit 状态转移，第 09 章）。
//
// 阶段一（消费转移）：逐个处理转移。非凭信来源类别拒绝；引用解析仅针对批次开始
// 前的有效（未转出且未过期）状态，故同批次新输出不可被引用（解析为 ErrNotFound）；
// 脚本验证失败立即返回且不标记转出；通过后消费旧凭信，新输出延迟到阶段二插入。
// Transfer.NewOutput 为 nil 时视为凭信销毁，不产生新 UTCO（第 07 章 §5）。
//
// 阶段二（插入）：先插入各转移的新凭信，再插入批次新建输出；仅凭信且非自定义类
// 进入 UTCO，其余（自定义类、币金、存证）跳过。
//
// 失败时可能留下部分变更，原子回滚由快照层（Task 8）承载。
func Apply(ctx context.Context, s *Store, v ScriptVerifier, b Batch) error {
	// 阶段一：消费转移，收集待插入的新凭信输出（nil 表示销毁）。
	pending := make([]*NewOutput, 0, len(b.Transfers))
	for i := range b.Transfers {
		t := b.Transfers[i]
		if t.Kind != tx.InputCredit {
			return ErrInputKindInvalid
		}
		old, err := s.Resolve(t.Ref, b.CurrentHeight)
		if err != nil {
			return err
		}
		if v != nil {
			if err := v.VerifyCreditTransfer(ctx, old, t.UnlockScript); err != nil {
				return err
			}
		}
		if err := s.Spend(old.OutPoint()); err != nil {
			return err
		}
		pending = append(pending, t.NewOutput)
	}
	// 阶段二：插入新凭信（转移产物在前，新建在后）；nil 跳过（销毁）。
	for i := range pending {
		if pending[i] == nil {
			continue
		}
		if err := insertCredit(s, *pending[i]); err != nil {
			return err
		}
	}
	for i := range b.Creations {
		if err := insertCredit(s, b.Creations[i]); err != nil {
			return err
		}
	}
	return nil
}

// insertCredit 将一个新产生的输出按规则插入 UTCO：仅凭信且非自定义类进入状态集，
// 其余（自定义类、币金、存证）跳过且不报错。
func insertCredit(s *Store, o NewOutput) error {
	if o.Output.IsCustom || o.Output.Type != tx.TypeCredit {
		return nil
	}
	entry := Entry{
		Year:          o.Year,
		TxID:          o.TxID,
		OutIndex:      uint64(o.Output.Serial),
		Receiver:      o.Credit.Receiver,
		Creator:       o.Credit.Creator,
		Title:         o.Credit.Title,
		Description:   o.Credit.Description,
		AttachmentID:  o.Credit.AttachmentID,
		LockScript:    o.Output.LockScript,
		CreatedHeight: o.Height,
	}
	return s.Put(entry)
}


