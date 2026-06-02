// Package validation 实现组队校验（Layer 5）的角色接口、协议约束与区块证明包。
//
// 本包属 Layer 5，可依赖 Layer 0-4 接口与稳定类型，不被 Layer 0-4 反向 import。
// 不实现各角色 App、组内 RPC 或组间 P2P 线格式（P2P 线格式外包，见 C-10 声明）。
// 校验组与公共服务不是新的共识信任源；区块证明包仅支持快速预验证与转播，
// 不能替代完整区块验证，不证明 UTXO/UTCO 状态真实性（第 13 章，DEC-0601）。
package validation

import "errors"

// 首领校验与黑名单相关错误（第 13 章 §2）。
var (
	// ErrBlacklisted 表示首领输入当前处于黑名单冻结期内。
	// 黑名单为本地策略，不得作为拒绝合法区块的依据。
	ErrBlacklisted = errors.New("validation: lead input is blacklisted")

	// ErrLeadInputNotFound 表示首领输入在本地 UTXO 集中未命中或已花费。
	ErrLeadInputNotFound = errors.New("validation: lead input not found in utxo")

	// ErrLeadInputNotMaxStake 表示首领输入不是所有币金输入中币权最大者。
	ErrLeadInputNotMaxStake = errors.New("validation: lead input does not have the highest coin stake")
)

// 冗余校验与复核相关错误（第 13 章 §3）。
var (
	// ErrRedundancyTooLow 表示结果数量低于最低冗余度要求（须 >= 2）。
	ErrRedundancyTooLow = errors.New("validation: redundancy below minimum (2 required)")
)

// 区块证明包与快速预验证相关错误（第 13 章 §5，DEC-0601）。
var (
	// ErrNoMinterField 表示 Coinbase 缺少 Minter 字段（非创世区块必须存在）。
	ErrNoMinterField = errors.New("validation: coinbase has no minter field")

	// ErrMinterNotInPool 表示铸造者公钥不是本地当前择优池成员。
	ErrMinterNotInPool = errors.New("validation: minter is not a current best pool member")

	// ErrPrevBlockMismatch 表示 BlockHeader.PrevBlock 与本地末端区块 ID 不衔接。
	ErrPrevBlockMismatch = errors.New("validation: prev block does not chain to local tip")

	// ErrStateRootMismatch 表示 UTXORoot 或 UTCORoot 与本地当前状态指纹不一致。
	ErrStateRootMismatch = errors.New("validation: utxo or utco root does not match local state")

	// ErrCoinbaseTxIndexNot0 表示 CoinbaseTxIndex 不为 0（违反 DEC-0601 规则）。
	ErrCoinbaseTxIndexNot0 = errors.New("validation: coinbase tx index must be 0")

	// ErrCoinbaseTxIDFailed 表示 Coinbase TxID 计算失败（结构非法）。
	ErrCoinbaseTxIDFailed = errors.New("validation: failed to compute coinbase txid")

	// ErrTreeRootMismatch 表示从 Coinbase 验证路径重算的交易树根与 TreeRoot 不匹配。
	ErrTreeRootMismatch = errors.New("validation: recomputed tree root does not match proof package tree root")

	// ErrCheckRootMismatch 表示重算的 CheckRoot 与 BlockHeader.CheckRoot 不匹配。
	ErrCheckRootMismatch = errors.New("validation: recomputed check root does not match block header")

	// ErrMinterSigInvalid 表示铸造者对 CheckRoot 的签名无效。
	ErrMinterSigInvalid = errors.New("validation: minter check root signature is invalid")

	// ErrInvalidProofPackage 表示证明包编码非法或截断（反序列化失败）。
	ErrInvalidProofPackage = errors.New("validation: proof package encoding is invalid or truncated")
)

// 铸造协作相关错误（第 13 章 §4）。
var (
	// ErrMinterNotEligible 表示铸造者不在当前择优池中（不具铸造资格）。
	ErrMinterNotEligible = errors.New("validation: minter is not in best pool")

	// ErrCoinbaseVerifyFailed 表示 Coinbase 结构校验失败（管理层拒绝）。
	ErrCoinbaseVerifyFailed = errors.New("validation: coinbase verification failed by manager")
)

// 交易量约束相关错误（第 13 章 §6，DEC-0303）。
var (
	// ErrStakesConstraintFail 表示挑战者区块的 Stakes 未达到前位区块的 3x 严格下限。
	ErrStakesConstraintFail = errors.New("validation: challenger block stakes do not strictly exceed 3x base")
)
