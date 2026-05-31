// Package tx 实现 Layer 1 交易模型：普通/Coinbase 交易头、输入项与短引用、
// 输出 envelope、Coin/Credit/Proof/Mediator/Custom 信元载荷、交易输入/输出哈希
// 与本地可判定的结构规则。本包只表达交易数据、规范化编码、Hash 与本地结构验证，
// 不检查状态可用性、不执行脚本、不验证 PoH 资格、不做完整 Coinbase 奖励结算。
// 仅依赖 pkg/types、pkg/crypto、pkg/hashtree 与 internal/blockchain 的链身份类型，
// 不依赖状态、脚本或共识具体实现（第 06、07 章）。
package tx

import (
	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// TxHeader 是普通交易头结构（第 06 章 §1、DEC-0003）。
// Coinbase 头字段集不同，由 coinbase.go 单独建模，两者解析 profile 不共用。
type TxHeader struct {
	// Version 是交易版本号。
	Version uint16
	// HashInputs 是输入项根哈希（BLAKE3-256，见 input_hash.go）。
	HashInputs types.Hash32
	// HashOutputs 是输出项根哈希（BLAKE3-256，见 output_hash.go）。
	HashOutputs types.Hash32
	// Timestamp 是交易时间戳（Unix 毫秒），不限制负值。
	Timestamp int64
	// MintPKHash 是可选铸凭公钥哈希；长度必须为 0（未设置）或 32。
	// 普通交易头采用变长封装 varint(len)||bytes，与 Coinbase 定长 32B 不共用编码器。
	MintPKHash []byte
}

// CanonicalBytes 返回普通交易头的规范编码（DEC-0003）：
//
//	Version(uint16 BE) || HashInputs[32] || HashOutputs[32]
//	  || Timestamp(int64 BE) || MintPKHash(varint(len)||bytes)
//
// 当 MintPKHash 长度不属于 {0, 32} 时返回 ErrMintPKHashLength。
func (h *TxHeader) CanonicalBytes() ([]byte, error) {
	if len(h.MintPKHash) != 0 && len(h.MintPKHash) != 32 {
		return nil, ErrMintPKHashLength
	}
	dst := make([]byte, 0, 107)
	dst = types.AppendUint16BE(dst, h.Version)
	dst = append(dst, h.HashInputs.Bytes()...)
	dst = append(dst, h.HashOutputs.Bytes()...)
	dst = types.AppendInt64BE(dst, h.Timestamp)
	dst = types.AppendBytes(dst, h.MintPKHash)
	return dst, nil
}

// TxID 计算普通交易头的交易标识（第 06 章 §1.1）：
//
//	TxID = SHA3-384( DomainTag("tx.header") || CanonicalBytes() )
//
// 普通头与 Coinbase 头使用不同前像，但共用 tx.header 域标签。
func (h *TxHeader) TxID() (types.TxID, error) {
	pre, err := h.CanonicalBytes()
	if err != nil {
		return types.TxID{}, err
	}
	return crypto.HashTxHeader(pre), nil
}
