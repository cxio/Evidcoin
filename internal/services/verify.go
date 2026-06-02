package services

import "github.com/cxio/evidcoin/pkg/types"

// VerificationAnchor 携带本地已知的链式锚点，所有服务响应必须依此独立验证
// （第 15 章 §5，DEC-0603）。
//
// 公共服务（Blockqs/Depots）仅作查询加速层，不是信任根；验证函数本地执行。
type VerificationAnchor struct {
	// BlockID 是本地已知的待验证区块标识。
	BlockID types.BlockID
	// CheckRoot 是本地已知的该区块 CheckRoot。
	// 它承诺交易树根、UTXO 根和 UTCO 根。
	CheckRoot types.CheckRoot
	// UTXORoot 是本地持有的 UTXO 状态根（来自已完成区块）。
	UTXORoot types.TreeHash
	// UTCORoot 是本地持有的 UTCO 状态根（来自已完成区块）。
	UTCORoot types.TreeHash
}

// ServiceKeyConstraint 记录独立服务密钥的协议级约束（DEC-0603）。
//
// 每个 Blockqs/Depots 节点可使用独立服务密钥对响应签名：
//   - 该签名仅证明响应的来源，**不证明数据真实**。
//   - 独立服务密钥在协议层不需与区块链收益地址绑定。
//   - 收益地址声明不作为响应真实性依据。
//   - 客户端应向多个节点**交叉查询**关键数据。
type ServiceKeyConstraint struct {
	// ProveSourceOnly 表示服务密钥仅证明响应来源，
	// 不证明数据真实性。
	ProveSourceOnly bool
	// NotBoundToRewardAddress 表示服务密钥在协议层
	// 不绑定到节点的链上收益地址。
	NotBoundToRewardAddress bool
	// CrossQueryRequired 表示客户端对关键数据必须
	// 向多个 Blockqs 节点交叉查询。
	CrossQueryRequired bool
}

// DefaultServiceKeyConstraint 是协议要求的独立服务密钥约束（DEC-0603）。
// 所有使用服务密钥的场景都必须满足该约束。
var DefaultServiceKeyConstraint = ServiceKeyConstraint{
	ProveSourceOnly:         true,
	NotBoundToRewardAddress: true,
	CrossQueryRequired:      true,
}

// ValidateRecentBlockProofs 检查 RecentBlockProofsResponse 是否满足最小区块证明包数量要求
// （至少 31 个，以覆盖分叉安全窗口，第 15 章 §6，DEC-0601）。
//
// 若数量不足 MinRecentBlockProofs（31）则返回 ErrRecentBlockProofsInsufficient；
// 满足要求则返回 nil。
func ValidateRecentBlockProofs(resp RecentBlockProofsResponse) error {
	if len(resp.ProofPackages) < MinRecentBlockProofs {
		return ErrRecentBlockProofsInsufficient
	}
	return nil
}

// VerifyBlockSummaryConsistency 检查区块概要的内部一致性（DEC-0602）：
//   - TxCount 不为零。
//   - TxIDPrefixes 数量与 TxCount 严格一致。
//
// 注意：此函数仅验证内部一致性；最终验证仍需用完整 TxID 序列重算交易树根（DEC-0602）。
// 不一致返回 ErrInvalidSummary；一致返回 nil。
func VerifyBlockSummaryConsistency(s *BlockSummary) error {
	if s == nil || s.TxCount == 0 || len(s.TxIDPrefixes) == 0 {
		return ErrInvalidSummary
	}
	if uint64(len(s.TxIDPrefixes)) != s.TxCount {
		return ErrInvalidSummary
	}
	return nil
}
