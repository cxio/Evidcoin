package services

import (
	"github.com/cxio/evidcoin/internal/validation"
	"github.com/cxio/evidcoin/pkg/hashtree"
	"github.com/cxio/evidcoin/pkg/types"
)

// StateKind 标识状态证明条目的类型（UTXO 或 UTCO）。
type StateKind uint8

const (
	// StateKindUTXO 表示 UTXO（未花费币金输出）状态条目。
	StateKindUTXO StateKind = 1
	// StateKindUTCO 表示 UTCO（未转出凭信输出）状态条目。
	StateKindUTCO StateKind = 2
)

// TxLookupResponse 是 TxLookup 查询的响应（第 15 章 §3，DEC-0603）。
//
// 它返回指定年份与 TxID 对应的完整交易数据，
// 以及该交易所在区块高度和区块内序位。
// 返回的 TxData 必须可由 TxID 独立验证（由 TxData 重算 TxID，
// 并与查询 TxID 比对）。
type TxLookupResponse struct {
	// Year 是交易所属年份（UTC 日历年）。
	Year uint64
	// TxID 是被查询交易的标识（SHA3-384，48 字节）。
	TxID types.TxID
	// TxData 是规范编码下的完整序列化交易。
	TxData []byte
	// BlockHeight 是该交易被确认时的区块高度。
	BlockHeight uint32
	// TxIndex 是区块内序位（从 0 开始，Coinbase 为 0）。
	TxIndex uint32
}

// TxProofResponse 是 TxProof 查询的响应（第 15 章 §3，DEC-0603）。
//
// 它返回从交易叶子到区块交易树根的 Merkle 证明路径
// （见 proposal §4，DEC-0004）。
// 客户端必须在本地验证该证明：由 TxID 重算叶子哈希，
// 逐层合并兄弟节点，并将结果与已知树根比对。
type TxProofResponse struct {
	// TxID 是提供 Merkle 证明的目标交易。
	TxID types.TxID
	// BlockHeight 是包含该交易的区块高度。
	BlockHeight uint32
	// Proof 是从交易叶子哈希到树根的 Merkle 路径。
	Proof hashtree.Proof
}

// BlockTxListResponse 是 BlockTxList 查询的响应（第 15 章 §3，DEC-0603）。
//
// 它返回区块完整 TxID 序列，或网络区块概要二者之一。
// 当 IsSummary 为 false 时，TxIDs 携带完整列表，客户端可重算
// 交易树根进行验证。
// 当 IsSummary 为 true 时，Summary 携带紧凑表示；最终验证
// 仍需获取完整 TxID 序列。
type BlockTxListResponse struct {
	// BlockID 是被请求区块的标识（SHA3-384，48 字节）。
	BlockID types.BlockID
	// BlockHeight 是区块高度。
	BlockHeight uint32
	// TxIDs 在 IsSummary 为 false 时包含完整 TxID 序列。
	// 按区块内序位排序；Coinbase 位于索引 0。
	TxIDs []types.TxID
	// Summary 在 IsSummary 为 true 时携带网络区块概要。
	// IsSummary 为 false 时为 nil。
	Summary *BlockSummary
	// IsSummary 表示该响应是否携带概要而非完整 TxIDs。
	IsSummary bool
}

// StateProofEntry 是单条 UTXO 或 UTCO 状态证明项（第 15 章 §3，DEC-0603）。
//
// 每条记录携带状态位证明（到状态根的 Merkle 路径）和
// 输出载荷供参考。客户端必须基于已知 UTXO/UTCO 根（经 CheckRoot 承诺）
// 验证 Kind、TxID、OutIndex 与 IsValid。
type StateProofEntry struct {
	// Kind 指示该条目是 UTXO（StateKindUTXO=1）还是 UTCO（StateKindUTCO=2）。
	Kind StateKind
	// TxID 是来源交易标识（SHA3-384，48 字节）。
	TxID types.TxID
	// OutIndex 是来源交易中的输出序位索引。
	OutIndex uint64
	// IsValid 表示该输出当前是否未花费（在状态集中有效）。
	IsValid bool
	// Proof 是从状态叶子到 UTXO 或 UTCO 根的 Merkle 路径。
	// 客户端必须将该路径与本地持有的状态根进行验证。
	Proof hashtree.Proof
	// OutputData 是原始规范编码输出载荷，用于参考与验证。
	OutputData []byte
}

// StateProofResponse 是 StateProof 查询的响应（第 15 章 §3，DEC-0603）。
//
// 它返回 UTXO/UTCO 状态位证明及输出明细（见 proposal §9，DEC-0201）。
// 客户端必须将每条证明与本地持有的 UTXO/UTCO 根进行验证，
// 而这些根由区块 CheckRoot 承诺。
type StateProofResponse struct {
	// Entries 是状态证明条目列表。
	Entries []StateProofEntry
	// UTXORoot 是这些证明计算所基于的 UTXO 状态根。
	// 客户端必须确认其与本地已知 UTXO 根一致。
	UTXORoot types.TreeHash
	// UTCORoot 是这些证明计算所基于的 UTCO 状态根。
	// 客户端必须确认其与本地已知 UTCO 根一致。
	UTCORoot types.TreeHash
}

// RecentBlockProofsResponse 是 RecentBlockProofs 查询的响应
// (第 15 章 §3·§6，DEC-0601，DEC-0603).
//
// 它返回至少 MinRecentBlockProofs（31）个连续区块证明包，
// 以覆盖分叉安全窗口。初始节点同步依赖该响应的完整性
// （见 proposal §13）。
// 可使用 ValidateRecentBlockProofs 验证最小数量要求。
type RecentBlockProofsResponse struct {
	// ProofPackages 是区块证明包列表，按从旧到新排序。
	// 必须至少包含 MinRecentBlockProofs（31）项。
	ProofPackages []validation.ProofPackage
}

// AttachmentIndexResponse 是 AttachmentIndex 查询的响应
// (第 15 章 §3，DEC-0603).
//
// 对于小附件（< DataBoundaryBytes），Data 携带原始附件字节。
// 对于大附件（>= DataBoundaryBytes），FragmentIndex 携带序列化分片索引，
// 用于经 Depots 拉取数据。
// 客户端必须将数据与已知附件指纹（SHA3-512）进行校验。
type AttachmentIndexResponse struct {
	// Fingerprint 是规范附件指纹（SHA3-512，64 字节）。
	Fingerprint types.AttachmentHash
	// IsLargeAttachment 指示该附件是否达到或超过 DataBoundaryBytes。
	IsLargeAttachment bool
	// Data 包含小附件（< DataBoundaryBytes）的原始字节。
	// 当 IsLargeAttachment 为 true 时为空。
	Data []byte
	// FragmentIndex 包含大附件的序列化分片索引。
	// 当 IsLargeAttachment 为 false 时为空。
	FragmentIndex []byte
	// FragmentCount 是大附件分片数量；小附件为 0。
	FragmentCount uint32
}

// Blockqs 是 Blockqs（区块查询）公共服务的接口边界
// (第 15 章 §3，DEC-0603).
//
// Blockqs 提供区块交易数据、Merkle 证明、状态证明、
// 近期区块证明包和附件索引的快速查询。
// 实现位于外部仓库（github.com/cxio/blockqs）；该接口定义边界契约。
//
// Blockqs 不是信任根：所有返回数据都必须基于区块头链、CheckRoot、
// TxID 或附件指纹进行独立验证。
// 对关键数据，客户端必须向多个 Blockqs 节点交叉查询。
//
// 服务节点会提供其链上账户地址用于奖励分配；
// 该地址声明不构成判断响应真实性的依据。
// 服务不可用不会影响区块合法性。
type Blockqs interface {
	// LookupTx 查询指定年份与 TxID 对应的完整交易数据。
	LookupTx(year uint64, txID types.TxID) (TxLookupResponse, error)

	// TxProof 查询从交易到其所在区块树根的 Merkle 证明路径。
	TxProof(blockHeight uint32, txID types.TxID) (TxProofResponse, error)

	// BlockTxList 查询区块完整 TxID 序列或网络概要。
	// summaryMode 为 true 请求紧凑概要；为 false 返回完整 TxIDs。
	BlockTxList(blockID types.BlockID, summaryMode bool) (BlockTxListResponse, error)

	// StateProof 查询给定交易输出引用对应的 UTXO/UTCO 状态位证明。
	StateProof(txIDs []types.TxID) (StateProofResponse, error)

	// RecentBlockProofs 查询最近区块证明包。
	// count 指定期望数量；响应必须至少包含
	// MinRecentBlockProofs（31）个证明包。
	RecentBlockProofs(count int) (RecentBlockProofsResponse, error)

	// AttachmentIndex 查询给定附件指纹对应的数据或分片索引。
	AttachmentIndex(fingerprint types.AttachmentHash) (AttachmentIndexResponse, error)
}
