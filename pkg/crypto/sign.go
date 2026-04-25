package crypto

import (
	"github.com/cloudflare/circl/sign/mldsa/mldsa65"
)

// PrivateKey ML-DSA-65 私钥。
type PrivateKey = mldsa65.PrivateKey

// PublicKey ML-DSA-65 公钥。
type PublicKey = mldsa65.PublicKey

// GenerateKey 生成 ML-DSA-65 密钥对。
func GenerateKey() (*PublicKey, *PrivateKey, error) {
	pub, priv, err := mldsa65.GenerateKey(nil)
	return pub, priv, err
}

// Sign 使用私钥对数据签名，返回签名字节。
// 签名过程在算法正确实现的前提下不会失败，panic 仅用于捕获不可恢复的编程错误。
func Sign(priv *PrivateKey, data []byte) []byte {
	sig := make([]byte, mldsa65.SignatureSize)
	if err := mldsa65.SignTo(priv, data, nil, false, sig); err != nil {
		panic("crypto.Sign: unexpected signing failure: " + err.Error())
	}
	return sig
}

// Verify 使用公钥验证数据签名。返回 true 表示验证通过。
func Verify(pub *PublicKey, data, sig []byte) bool {
	return mldsa65.Verify(pub, data, nil, sig)
}

// PublicKeyBytes 返回公钥的字节序列。
func PublicKeyBytes(pub *PublicKey) []byte {
	b, _ := pub.MarshalBinary()
	return b
}

// PrivateKeyBytes 返回私钥的字节序列。
func PrivateKeyBytes(priv *PrivateKey) []byte {
	b, _ := priv.MarshalBinary()
	return b
}

// PublicKeyFromBytes 从字节序列还原公钥。
func PublicKeyFromBytes(b []byte) (*PublicKey, error) {
	var pub mldsa65.PublicKey
	err := pub.UnmarshalBinary(b)
	return &pub, err
}

// PrivateKeyFromBytes 从字节序列还原私钥。
func PrivateKeyFromBytes(b []byte) (*PrivateKey, error) {
	var priv mldsa65.PrivateKey
	err := priv.UnmarshalBinary(b)
	return &priv, err
}
