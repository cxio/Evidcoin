package tx

import (
	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// Coinbase 与 CheckRoot 签名消息（DEC-0102 §5）。Coinbase 不使用授权种类，
// 直接对整笔交易签名；铸造者对区块 CheckRoot 的签名独立于 Coinbase 交易签名。

// CoinbaseSignatureMessage 构造 Coinbase 交易签名消息字节序列（DEC-0102 §5）：
//
//	DomainTag("signature.message") || ChainScope || 0x00 || CoinbaseTxID(48)
//
// chk_type=0 标记 Coinbase 域，不走 auth_flag 覆盖范围路径；CoinbaseTxID 为完整
// Coinbase 交易 TxID（48 字节）。该消息即铸造者对整笔 Coinbase 交易签名的输入。
func CoinbaseSignatureMessage(chain ChainScope, coinbaseTxID types.TxID) []byte {
	dst := crypto.SignatureMessageTag()
	dst = chain.appendCanonical(dst)
	dst = append(dst, byte(ChkCoinbase))
	dst = append(dst, coinbaseTxID.Bytes()...)
	return dst
}

// CheckRootSignatureMessage 构造铸造者对区块 CheckRoot 的签名消息（DEC-0102 §5）。
//
// 该签名独立于 Coinbase 交易签名：CheckRoot 自身已是按 checkroot 域标签隔离的哈希
// （见 internal/blockchain.ComputeCheckRoot 与 DEC-0002），因此签名消息直接取
// CheckRoot 的 48 字节，不再叠加 signature.message 域标签，与 Coinbase 签名消息天然区分。
// 构造确定可复现，满足创世 CheckRoot 签名必须保留的链根锚定要求（第 05 章 / DEC-0103）。
//
// 说明：DEC-0102 §5 仅锚定「CheckRoot 签名独立于 Coinbase、使用各自域标签/消息构造」，
// 未冻结额外封装；此处采用「直接对已域隔离的 CheckRoot 签名」这一最小且可复现的构造，
// 不引入 14 项域标签全集之外的新标签（DEC-0002）。
func CheckRootSignatureMessage(checkRoot types.CheckRoot) []byte {
	return checkRoot.Bytes()
}
