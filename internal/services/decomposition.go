package services

// DataBoundaryBytes 是 Blockqs 与 Depots 职责划分的规范数据量边界（10 MB）。
//
// 低于此阈值的附件与分片索引文件归 Blockqs 服务；
// 达到或超过此阈值的附件、完整区块文件、分片数据归 Depots 服务。
// 该边界为建议性划分：两类服务对同一数据的提供可能重叠，但验证口径必须相同
// （第 15 章 §1，DEC-0603）。
const DataBoundaryBytes = 10 * 1024 * 1024 // 10 MB

// MinRecentBlockProofs 是 RecentBlockProofs 响应所需的最小区块证明包数量。
// 至少 31 个以覆盖分叉安全窗口（第 15 章 §6，第 13 章，DEC-0601）。
const MinRecentBlockProofs = 31

// ServiceKind 标识公共服务的类别。
type ServiceKind uint8

const (
	// ServiceKindBase 表示基网（节点发现与广播基础层）。
	// 外部项目：github.com/cxio/p2p。
	// 基网为所有服务/应用子网提供精简 P2P 节点发现与通用广播支持（第 15 章 §1）。
	// 不参与区块/交易/PoH/脚本验证；不可达不影响区块合法性。
	ServiceKindBase ServiceKind = iota + 1

	// ServiceKindSTUN 表示 STUN/stun2p 服务（NAT 探测与打洞协助）。
	// 外部项目：github.com/cxio/stun2p。
	// 提供 UDP P2P 连接支持；不参与共识；不可达不影响区块合法性（第 15 章 §1）。
	ServiceKindSTUN

	// ServiceKindDepots 表示 Depots（数据驿站）服务（大尺寸数据开放存储与分享）。
	// 外部项目：github.com/cxio/depots。
	// 数据量范围：>= 10 MB 附件、完整区块文件、分片数据（第 15 章 §1·§2，DEC-0603）。
	ServiceKindDepots

	// ServiceKindBlockqs 表示 Blockqs（区块查询）服务（交易数据快速查询）。
	// 外部项目：github.com/cxio/blockqs。
	// 数据量范围：< 10 MB 附件、大附件分片索引文件（第 15 章 §1·§3，DEC-0603）。
	ServiceKindBlockqs
)
