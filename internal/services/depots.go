package services

import "github.com/cxio/evidcoin/pkg/types"

// Depots 是 Depots（数据仓）公共服务的接口边界
// (第 15 章 §2，DEC-0603).
//
// Depots 负责大体量区块/交易附件数据的开放格式存储与共享。
// 数据查询通过基础网络的通用广播机制完成；
// 稀缺度通过广播回复中的跳数评估。
//
// 实现位于外部仓库（github.com/cxio/depots）；该接口定义
// 本地区块链节点与 Depots 节点交互的边界契约。
//
// Depots 不参与区块验证、交易执行、PoH 或脚本验证。
// 服务不可用不会影响区块合法性。
//
// 服务节点在连接应用节点时会提供其链上账户地址，
// 以便接收可能的奖励分配（见 proposal §14）。
type Depots interface {
	// FetchAttachment 按规范指纹请求大附件（>= DataBoundaryBytes）。
	// 返回原始附件字节；不可用时返回错误。
	FetchAttachment(fingerprint types.AttachmentHash) ([]byte, error)

	// FetchBlock 按 BlockID 请求完整序列化区块。
	// 返回原始区块字节；不可用时返回错误。
	FetchBlock(blockID types.BlockID) ([]byte, error)

	// FetchFragment 按附件指纹和分片索引请求单个数据分片。
	// 返回原始分片字节；不可用时返回错误。
	FetchFragment(fingerprint types.AttachmentHash, fragIndex uint32) ([]byte, error)

	// UploadAttachment 将本地产生的附件数据上传到 Depots 网络。
	// 节点作为数据源；Depots 节点收到查询广播后执行存储。
	// 上传请求失败时返回错误。
	UploadAttachment(fingerprint types.AttachmentHash, data []byte) error
}
