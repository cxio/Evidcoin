package types

import (
	"bytes"
	"crypto/sha256"
	"errors"

	"github.com/mr-tron/base58"
)

// AddressPrefix 地址网络前缀。
type AddressPrefix string

const (
	// MainnetPrefix 主网地址前缀
	MainnetPrefix AddressPrefix = "EC"
	// TestnetPrefix 测试网地址前缀
	TestnetPrefix AddressPrefix = "ET"
)

// Address 表示一个 Evidcoin 地址，由公钥哈希与网络前缀派生。
type Address struct {
	prefix AddressPrefix
	pkHash PubKeyHash
}

// NewAddress 从公钥哈希和前缀构造地址。
func NewAddress(pkh PubKeyHash, prefix AddressPrefix) Address {
	return Address{prefix: prefix, pkHash: pkh}
}

// PKHash 返回公钥哈希。
func (a Address) PKHash() PubKeyHash {
	return a.pkHash
}

// Encode 将地址编码为可读字符串。
// 格式：Prefix + Base58( PKHash[32] || Checksum[4] )
func (a Address) Encode() string {
	checksum := addressChecksum(a.prefix, a.pkHash)
	payload := make([]byte, PubKeyHashLen+4)
	copy(payload[:PubKeyHashLen], a.pkHash[:])
	copy(payload[PubKeyHashLen:], checksum)
	return string(a.prefix) + base58.Encode(payload)
}

// DecodeAddress 从可读字符串解码地址。
// 返回 error 如果格式无效或校验码不匹配。
func DecodeAddress(s string, prefix AddressPrefix) (Address, error) {
	pfx := string(prefix)
	if len(s) <= len(pfx) || s[:len(pfx)] != pfx {
		return Address{}, errors.New("invalid address prefix")
	}
	decoded, err := base58.Decode(s[len(pfx):])
	if err != nil {
		return Address{}, errors.New("invalid base58 encoding")
	}
	if len(decoded) != PubKeyHashLen+4 {
		return Address{}, errors.New("invalid address length")
	}
	var pkh PubKeyHash
	copy(pkh[:], decoded[:PubKeyHashLen])
	expected := addressChecksum(prefix, pkh)
	if !bytes.Equal(decoded[PubKeyHashLen:], expected) {
		return Address{}, errors.New("address checksum mismatch")
	}
	return Address{prefix: prefix, pkHash: pkh}, nil
}

// addressChecksum 计算地址校验码：取 SHA256( prefix || PKHash ) 的最后 4 字节。
func addressChecksum(prefix AddressPrefix, pkh PubKeyHash) []byte {
	h := sha256.New()
	h.Write([]byte(prefix))
	h.Write(pkh[:])
	sum := h.Sum(nil)
	return sum[len(sum)-4:]
}
