package validation

import (
	"github.com/cxio/evidcoin/internal/tx"
	"github.com/cxio/evidcoin/pkg/hashtree"
	"github.com/cxio/evidcoin/pkg/types"
)

// MintRequest 是铸造者在铸造协作第一步向管理层发送的请求（第 13 章 §4）。
// 内容：铸造者的择优凭证字节（用于证明铸造资格）及其签名（防伪造）。
// P2P 线格式外包（C-10 待决），本类型只承载协议约定字段。
type MintRequest struct {
	// MintProofBytes 是铸造者持有的择优凭证（MintProof）规范编码字节（共识层产出）。
	MintProofBytes []byte
	// Signature 是铸造者对 MintProofBytes 的签名（供管理层验证真实性）。
	Signature []byte
}

// MintInfoResponse 是管理层对合法铸造者回复的区块构建所需信息（第 13 章 §4）。
// 铸造者据此构造 CoinbaseHeader 并签名 CheckRoot。
type MintInfoResponse struct {
	// TxFeeTotal 是本区块中全部普通交易的手续费合计（聪）。
	TxFeeTotal types.Amount
	// GroupRewardAddr 是校验组奖励地址原始字节（用于 Coinbase 输出）。
	GroupRewardAddr []byte
	// ServiceRewardAddr 是公共服务奖励地址原始字节（用于 Coinbase 输出）。
	ServiceRewardAddr []byte
	// MintAmount 是本区块的铸造奖励总量（聪），由激励层按协议规则计算。
	MintAmount types.Amount
	// AwardSlots 是公共服务兑奖槽固定 18 字节，由管理层按 DEC-0401 填充。
	AwardSlots [18]byte
}

// CoinbaseSubmission 是铸造者在铸造协作第二步向管理层提交的 Coinbase（第 13 章 §4）。
// MinterSig 是铸造者对 CheckRoot 的签名，它不计入 CoinbaseTxID 的哈希范围。
// 管理层收到后须验证：CoinbaseHeader 内容合规、MinterSig 对应本次 CheckRoot 有效。
type CoinbaseSubmission struct {
	// CoinbaseHeader 是铸造者构造的完整 Coinbase 交易头（tx.CoinbaseHeader）。
	CoinbaseHeader tx.CoinbaseHeader
	// MinterSig 是铸造者对 BlockHeader.CheckRoot 的签名（不计入 TxID 哈希范围）。
	MinterSig []byte
}

// InclusionResponse 是管理层向铸造者回复的纳入确认（第 13 章 §4）。
// 包含 Coinbase 在交易树中的 Merkle 证明、树根、UTXO/UTCO 指纹；
// 铸造者据此组装 ProofPackage 对外广播。
type InclusionResponse struct {
	// CoinbaseMerklePath 是 Coinbase 在交易树中的 Merkle 路径（含叶哈希与树根）。
	CoinbaseMerklePath hashtree.Proof
	// TreeRoot 是本次区块的完整交易树根字节。
	TreeRoot []byte
	// UTXORoot 是当前已验证 UTXO 集的状态指纹（DEC-0201）。
	UTXORoot types.TreeHash
	// UTCORoot 是当前已验证 UTCO 集的状态指纹（DEC-0201）。
	UTCORoot types.TreeHash
}

// BlockSignature 是铸造者在铸造协作第三步向管理层发送的 CheckRoot 签名（第 13 章 §4）。
// 管理层将 MinterCheckRootSignature 填入 ProofPackage 第 8 字段后对外广播。
type BlockSignature struct {
	// TxID 是 Coinbase 的交易 ID，用于与当前区块上下文绑定。
	TxID types.TxID
	// CheckRootSig 是铸造者对 BlockHeader.CheckRoot 的签名（DEC-0601 第 8 字段）。
	CheckRootSig []byte
}

// MintingManager 是铸造协作中管理层端的接口（第 13 章 §4）。
// 管理层按三步协议与铸造者交互，负责验证资格、提供构建信息、验证并广播区块证明包。
// 各方法调用顺序：HandleMintRequest → HandleCoinbaseSubmission → HandleBlockSignature。
type MintingManager interface {
	// HandleMintRequest 接收铸造者的请求，验证凭证与签名，
	// 返回 MintInfoResponse 供铸造者构造 CoinbaseHeader；若不合资格返回非 nil 错误。
	HandleMintRequest(req MintRequest) (MintInfoResponse, error)
	// HandleCoinbaseSubmission 接收铸造者提交的 Coinbase，验证结构合规并计算纳入证明，
	// 返回 InclusionResponse；若 Coinbase 非法或 MinterSig 无效返回非 nil 错误。
	HandleCoinbaseSubmission(sub CoinbaseSubmission) (InclusionResponse, error)
	// HandleBlockSignature 接收铸造者对 CheckRoot 的最终签名，组装 ProofPackage 并广播。
	// 广播操作由具体实现负责；线格式外包（C-10 待决）。
	HandleBlockSignature(sig BlockSignature) error
}

// MintingConductor 是铸造协作中铸造者端的接口（第 13 章 §4）。
// 铸造者按三步协议向管理层发起协作、获取构建信息、提交 Coinbase 并发送签名。
type MintingConductor interface {
	// SendMintRequest 向管理层发送铸造请求（步骤 1），返回管理层的构建信息。
	SendMintRequest(req MintRequest) (MintInfoResponse, error)
	// SendCoinbase 向管理层发送 Coinbase 提交（步骤 2），返回纳入证明。
	SendCoinbase(sub CoinbaseSubmission) (InclusionResponse, error)
	// SendBlockSignature 向管理层发送 CheckRoot 签名（步骤 3）。
	SendBlockSignature(sig BlockSignature) error
}
