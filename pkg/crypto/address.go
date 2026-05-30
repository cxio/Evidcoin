package crypto

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"sort"

	"github.com/cxio/evidcoin/pkg/types"
	"github.com/mr-tron/base58"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/sha3"
)

// Network 标识地址所属网络，并决定文本前缀（DEC-0104）。
type Network string

const (
	// Mainnet 使用 "Cx" 地址前缀。
	Mainnet Network = "Cx"
	// Testnet 使用 "Tx" 地址前缀。
	Testnet Network = "Tx"
	// Devnet 使用 "Dx" 地址前缀。
	Devnet Network = "Dx"
)

// 地址相关错误。
var (
	// ErrMultisigRatio 表示 m/n 配比非法（m 或 n 为 0，或 m > n）。
	ErrMultisigRatio = errors.New("crypto: invalid multisig m/n ratio")
	// ErrMultisigDuplicate 表示多签公钥中存在重复项。
	ErrMultisigDuplicate = errors.New("crypto: duplicate multisig public key")
	// ErrUnknownNetwork 表示网络前缀无法识别。
	ErrUnknownNetwork = errors.New("crypto: unknown network prefix")
	// ErrBadChecksum 表示地址校验和验证失败。
	ErrBadChecksum = errors.New("crypto: address checksum mismatch")
	// ErrBadAddress 表示地址格式非法（base58 非法或长度不对）。
	ErrBadAddress = errors.New("crypto: malformed address")
)

// AddressHashSingle 计算单签公钥哈希：
// SHA3-256( DomainTag("address.single") || BLAKE2b-512(pubKey) )（DEC-0002/DEC-0104）。
func AddressHashSingle(pubKey []byte) types.AddressHash {
	inner := blake2b.Sum512(pubKey)
	pre := make([]byte, 0, len(tagAddressSingle)+len(inner))
	pre = append(pre, tagAddressSingle...)
	pre = append(pre, inner[:]...)
	out := sha3.Sum256(pre)
	h, _ := types.NewAddressHash(out[:])
	return h
}

// AddressHashMulti 为 m-of-n 多签方案计算组合公钥哈希（DEC-0104）：
//  1. BaseH_i = BLAKE3-256(pubKey_i)
//  2. 按字典序排序 BaseH 并拼接
//  3. PKHmix = SHA3-256( DomainTag("address.multi") || m || n || BaseHAll )
//
// 要求 m、n 均非 0 且 m <= n，len(pubKeys) == n，并且公钥不能重复。
func AddressHashMulti(m, n uint8, pubKeys [][]byte) (types.AddressHash, error) {
	var zero types.AddressHash
	if m == 0 || n == 0 || m > n {
		return zero, ErrMultisigRatio
	}
	if len(pubKeys) != int(n) {
		return zero, ErrMultisigRatio
	}

	baseHashes := make([][]byte, len(pubKeys))
	for i, pk := range pubKeys {
		bh := blake3_256(pk)
		baseHashes[i] = bh[:]
	}
	sort.Slice(baseHashes, func(i, j int) bool {
		return bytes.Compare(baseHashes[i], baseHashes[j]) < 0
	})
	// 拒绝重复公钥（排序后其 base hash 会相邻）。
	for i := 1; i < len(baseHashes); i++ {
		if bytes.Equal(baseHashes[i-1], baseHashes[i]) {
			return zero, ErrMultisigDuplicate
		}
	}

	pre := make([]byte, 0, len(tagAddressMulti)+2+32*len(baseHashes))
	pre = append(pre, tagAddressMulti...)
	pre = append(pre, m, n)
	for _, bh := range baseHashes {
		pre = append(pre, bh...)
	}
	inner := blake2b.Sum512(pre)
	out := sha3.Sum256(inner[:])
	return types.NewAddressHash(out[:])
}

// EncodeAddress 将 32 字节公钥哈希编码为地址文本：
// prefix || Base58(pubKeyHash || checksum)，其中
// checksum = last4(SHA2-256(SHA2-256(prefix || pubKeyHash)))（DEC-0104）。
// prefix 参与校验和计算，但不进入 base58 载荷。
func EncodeAddress(net Network, h types.AddressHash) (string, error) {
	prefix := string(net)
	if !validNetwork(net) {
		return "", ErrUnknownNetwork
	}
	hash := h.Bytes()
	cksum := addressChecksum(prefix, hash)
	payload := make([]byte, 0, len(hash)+4)
	payload = append(payload, hash...)
	payload = append(payload, cksum...)
	return prefix + base58.Encode(payload), nil
}

// DecodeAddress 从地址文本恢复网络与 32 字节公钥哈希，并校验校验和。
// 对未知前缀、非法 base58、长度错误与校验失败均会拒绝。
func DecodeAddress(addr string) (Network, types.AddressHash, error) {
	var zero types.AddressHash
	if len(addr) < 2 {
		return "", zero, ErrBadAddress
	}
	net := Network(addr[:2])
	if !validNetwork(net) {
		return "", zero, ErrUnknownNetwork
	}
	payload, err := base58.Decode(addr[2:])
	if err != nil {
		return "", zero, ErrBadAddress
	}
	if len(payload) != 32+4 {
		return "", zero, ErrBadAddress
	}
	hash := payload[:32]
	gotSum := payload[32:]
	wantSum := addressChecksum(string(net), hash)
	if !bytes.Equal(gotSum, wantSum) {
		return "", zero, ErrBadChecksum
	}
	h, err := types.NewAddressHash(hash)
	if err != nil {
		return "", zero, err
	}
	return net, h, nil
}

func validNetwork(net Network) bool {
	switch net {
	case Mainnet, Testnet, Devnet:
		return true
	default:
		return false
	}
}

// addressChecksum 返回 last4(SHA2-256(SHA2-256(prefix || pubKeyHash))).
func addressChecksum(prefix string, pubKeyHash []byte) []byte {
	buf := make([]byte, 0, len(prefix)+len(pubKeyHash))
	buf = append(buf, prefix...)
	buf = append(buf, pubKeyHash...)
	first := sha256.Sum256(buf)
	second := sha256.Sum256(first[:])
	out := make([]byte, 4)
	copy(out, second[28:])
	return out
}
