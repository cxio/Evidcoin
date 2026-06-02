package services

import "github.com/cxio/evidcoin/pkg/types"

// VerificationAnchor 携带本地已知的链式锚点，所有服务响应必须依此独立验证
// （第 15 章 §5，DEC-0603）。
//
// 公共服务（Blockqs/Depots）仅作查询加速层，不是信任根；验证函数本地执行。
type VerificationAnchor struct {
	// BlockID is the locally-known block identifier for the block being verified.
	BlockID types.BlockID
	// CheckRoot is the locally-known CheckRoot for the block at BlockID.
	// It commits to the transaction tree root, UTXO root, and UTCO root.
	CheckRoot types.CheckRoot
	// UTXORoot is the locally-held UTXO state root (from the completed block).
	UTXORoot types.TreeHash
	// UTCORoot is the locally-held UTCO state root (from the completed block).
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
	// ProveSourceOnly documents that the service key proves only the response source,
	// not data authenticity.
	ProveSourceOnly bool
	// NotBoundToRewardAddress documents that the service key is not protocol-bound
	// to the node's blockchain reward address.
	NotBoundToRewardAddress bool
	// CrossQueryRequired documents that clients must cross-query multiple Blockqs nodes
	// for critical data.
	CrossQueryRequired bool
}

// DefaultServiceKeyConstraint 是协议要求的独立服务密钥约束（DEC-0603）。
// 所有使用服务密钥的场景必须符合此约束。
var DefaultServiceKeyConstraint = ServiceKeyConstraint{
	ProveSourceOnly:         true,
	NotBoundToRewardAddress: true,
	CrossQueryRequired:      true,
}

// ValidateRecentBlockProofs 检查 RecentBlockProofsResponse 是否满足最小区块证明包数量要求
// （至少 31 个，以覆盖分叉安全窗口，第 15 章 §6，DEC-0601）。
//
// 返回 ErrRecentBlockProofsInsufficient 若数量不足 MinRecentBlockProofs（31）；
// 返回 nil 若满足要求。
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
// 注意：此函数只验证内部一致性；最终验证仍须用完整 TxID 序列重算交易树根（DEC-0602）。
// 返回 ErrInvalidSummary 若不一致；返回 nil 若一致。
func VerifyBlockSummaryConsistency(s *BlockSummary) error {
	if s == nil || s.TxCount == 0 || len(s.TxIDPrefixes) == 0 {
		return ErrInvalidSummary
	}
	if uint64(len(s.TxIDPrefixes)) != s.TxCount {
		return ErrInvalidSummary
	}
	return nil
}
