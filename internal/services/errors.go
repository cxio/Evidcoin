// Package services 定义公共服务组件（Blockqs、Depots、STUN、基网）的接口边界（Layer 5）。
//
// 本包属 Layer 5，可依赖 Layer 0-4 接口与稳定类型，不被 Layer 0-4 反向 import。
// 不实现 Depots/Blockqs/STUN/基网内部逻辑，不规格化 P2P 传输线格式（C-10 边界）。
// 接口返回 Data + Proof，验证函数本地执行；不把 HTTP/RPC/P2P 客户端写进核心接口。
//
// 核心原则：公共服务与共识无关，不是信任根；验证节点对所有数据均须独立验证
// （第 15 章，DEC-0602，DEC-0603）。
package services

import "errors"

// 区块概要与 TxID 前缀相关错误（第 15 章 §4，DEC-0602）。
var (
	// ErrCollisionDetected 表示区块概要中的 TxID 前缀在本地候选交易中有多个匹配。
	// 接收方应按交易序位请求碰撞回退（CollisionFallback），获取完整 48 字节 TxID。
	ErrCollisionDetected = errors.New("services: block summary prefix collision detected")

	// ErrInvalidSummary 表示区块概要编码格式非法或截断。
	ErrInvalidSummary = errors.New("services: malformed block summary encoding")

	// ErrInvalidTxIDPrefixLen 表示提供的 TxID 前缀长度不是规定的 16 字节。
	ErrInvalidTxIDPrefixLen = errors.New("services: tx id prefix must be exactly 16 bytes")
)

// Blockqs 响应验证相关错误（第 15 章 §5·§6，DEC-0603）。
var (
	// ErrRecentBlockProofsInsufficient 表示 RecentBlockProofs 响应包含的区块证明包
	// 数量不足最低要求（至少 31 个，以覆盖分叉安全窗口）。
	ErrRecentBlockProofsInsufficient = errors.New("services: recent block proofs count below minimum (31 required)")

	// ErrResponseNotVerifiable 表示响应数据无法通过本地区块头链、CheckRoot、
	// TxID 或附件指纹进行验证。
	ErrResponseNotVerifiable = errors.New("services: response data cannot be verified against chain anchors")
)
