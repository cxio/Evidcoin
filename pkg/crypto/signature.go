package crypto

import "errors"

// 签名抽象（DEC-0104 固定 ML-DSA-65 / cloudflare-circl profile）。上层只依赖这些
// 接口，不直接绑定具体后量子签名库，便于未来单独评估迁移（A-2 观察项）。

// AlgorithmID 标识签名算法配置。
type AlgorithmID uint8

const (
	// AlgUnknown 是零值，表示无效算法。
	AlgUnknown AlgorithmID = 0
	// AlgMLDSA65 是 ML-DSA-65 配置（cloudflare/circl，DEC-0104）。
	AlgMLDSA65 AlgorithmID = 1
)

// ErrAlgorithmMismatch 表示验签器与签名在算法配置上不一致。
var ErrAlgorithmMismatch = errors.New("crypto: signature algorithm mismatch")

// PublicKey 是带算法标记的公钥，其规范字节编码用于公钥哈希与地址派生。
type PublicKey interface {
	// Algorithm 返回签名算法配置。
	Algorithm() AlgorithmID
	// Bytes 返回规范公钥编码（新副本）。
	Bytes() []byte
}

// Signature 是带算法标记的签名值。
type Signature interface {
	// Algorithm 返回签名算法配置。
	Algorithm() AlgorithmID
	// Bytes 返回规范签名编码（新副本）。
	Bytes() []byte
}

// Signer 负责对签名消息字节序列产生签名
// （消息配置定义见第 08 章 / DEC-0102）。
type Signer interface {
	// PublicKey 返回签名者公钥。
	PublicKey() PublicKey
	// Sign 对给定消息字节进行签名。
	Sign(message []byte) (Signature, error)
}

// Verifier 使用公钥验证签名。
type Verifier interface {
	// Verify 返回 sig 是否为 pub 对 message 的合法签名。
	// 当算法配置不一致时返回 ErrAlgorithmMismatch。
	Verify(pub PublicKey, message []byte, sig Signature) (bool, error)
}
