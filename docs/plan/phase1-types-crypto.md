# Phase 1: Basic Types & Cryptography（基础类型与密码学）Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现 `pkg/types` 和 `pkg/crypto` 两个零依赖基础包，为整个项目提供核心类型定义与密码学原语。

**Architecture:** 分层架构的最底层（Layer 0），无任何内部依赖。`pkg/types` 定义系统全局类型（哈希别名、基础常量、地址编码等），`pkg/crypto` 封装密码学操作（ML-DSA-65 签名、多种哈希算法）。

**Tech Stack:** Go 1.25+, `golang.org/x/crypto`（SHA3）, `lukechampine.com/blake3`（BLAKE3）, `github.com/mr-tron/base58`（地址编码），ML-DSA-65 优先使用 Go 1.25 标准库（`crypto/mlkem` 或 `crypto/mldsa`），如不可用则退回 `github.com/cloudflare/circl`。

---

## 目录结构（预期）

```
pkg/
  types/
    hash.go          # 哈希类型别名与辅助函数
    address.go       # 地址编码/解码
    constants.go     # 系统全局常量
    types_test.go    # 单元测试
  crypto/
    hash.go          # 哈希函数封装（SHA3-384/512, SHA3-256, BLAKE3-256/512）
    sign.go          # ML-DSA-65 签名/验证
    pubkey.go        # 公钥哈希计算（SHA3-256(BLAKE2b-512)）
    crypto_test.go   # 单元测试
```

---

## Task 1: 初始化模块与依赖

**Files:**
- Modify: `go.mod`

**Step 1: 添加所需外部依赖**

```bash
go get golang.org/x/crypto
go get lukechampine.com/blake3
go get github.com/mr-tron/base58
```

**Step 2: 验证依赖安装成功**

```bash
go mod tidy && go mod verify
```
预期：无错误输出。

**Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add cryptographic dependencies"
```

---

## Task 2: 基础常量（pkg/types/constants.go）

**Files:**
- Create: `pkg/types/constants.go`

**Step 1: 编写常量文件**

```go
// Package types 定义 Evidcoin 系统的核心类型与全局常量。
package types

import "time"

// 哈希长度常量
const (
	// Hash384Len SHA3-384 / SHA3-384 哈希字节长度（区块 ID、交易 ID、CheckRoot 等）
	Hash384Len = 48
	// Hash512Len SHA3-512 / BLAKE3-512 哈希字节长度（铸凭哈希）
	Hash512Len = 64
	// Hash256Len BLAKE3-256 / SHA3-256 哈希字节长度（UTXO/UTCO 树内部节点）
	Hash256Len = 32
	// PubKeyHashLen 公钥哈希字节长度：SHA3-256(BLAKE2b-512)
	PubKeyHashLen = 32
)

// 区块链参数
const (
	// BlockInterval 固定出块间隔（6 分钟）
	BlockInterval = 6 * time.Minute
	// BlocksPerYear 每年区块数（基于恒星年 365.25636 天）
	BlocksPerYear = 87661
	// GenesisHeight 创世区块高度
	GenesisHeight = 0
)

// 交易尺寸限制
const (
	// MaxTxSize 单笔交易最大字节数
	MaxTxSize = 65535
	// MaxLockScript 锁定脚本最大字节数
	MaxLockScript = 1024
	// MaxUnlockScript 解锁脚本最大字节数
	MaxUnlockScript = 4096
	// MaxMemo 币金附言最大字节数
	MaxMemo = 255
	// MaxTitle 凭信/存证标题最大字节数
	MaxTitle = 255
	// MaxCredDesc 凭信描述最大字节数（低 10 位编码长度）
	MaxCredDesc = 1023
	// MaxAttestContent 存证内容最大字节数（低 12 位编码长度）
	MaxAttestContent = 4095
	// MaxSelfData Coinbase 自由数据最大字节数
	MaxSelfData = 255
)

// 脚本执行限制
const (
	// MaxStackHeight 脚本执行栈最大深度
	MaxStackHeight = 256
	// MaxStackItem 栈项最大字节数
	MaxStackItem = 1024
)

// 择优池参数
const (
	// BestPoolCapacity 择优池容量（最多 20 名候选者）
	BestPoolCapacity = 20
	// ForkWindowSize 分叉竞争窗口（29 个区块）
	ForkWindowSize = 29
)

// 交易过期
const (
	// TxExpiryBlocks 未确认交易过期区块数（240 个区块，约 24 小时）
	TxExpiryBlocks = 240
	// FeeRecalcPeriod 最低手续费重算周期（6000 个区块，约 25 天）
	FeeRecalcPeriod = 6000
)

// 凭信生命周期约束
const (
	// MaxCreditExpiryBlocks 凭信最大有效期（100 年对应区块数）
	MaxCreditExpiryBlocks = 100 * BlocksPerYear
	// CreditActivityBlocks 无期限凭信最长不活跃区块数（11 年）
	CreditActivityBlocks = 11 * BlocksPerYear
)

// 信元类型
const (
	// OutTypeCoin 输出类型：币金
	OutTypeCoin = byte(1)
	// OutTypeCredit 输出类型：凭信
	OutTypeCredit = byte(2)
	// OutTypeProof 输出类型：存证
	OutTypeProof = byte(3)
	// OutTypeMediator 输出类型：介管脚本
	OutTypeMediator = byte(4)
)

// 铸币哈希混合常数
const MintHashMix = uint64(0x517cc1b727220a95)
```

**Step 2: 运行构建确认无误**

```bash
go build ./pkg/types/...
```
预期：无错误输出。

---

## Task 3: 哈希类型别名（pkg/types/hash.go）

**Files:**
- Create: `pkg/types/hash.go`

**Step 1: 编写哈希类型定义**

```go
package types

import "encoding/hex"

// Hash384 48 字节哈希，用于区块 ID、交易 ID、CheckRoot、公钥哈希路径等。
// 算法：SHA3-384
type Hash384 [Hash384Len]byte

// Hash512 64 字节哈希，用于铸凭哈希（MintHash）。
// 算法：BLAKE3-512
type Hash512 [Hash512Len]byte

// Hash256 32 字节哈希，用于 UTXO/UTCO 树内部节点、输出哈希树根等。
// 算法：BLAKE3-256
type Hash256 [Hash256Len]byte

// PubKeyHash 32 字节公钥哈希。
// 算法：SHA3-256( BLAKE2b-512( publicKey ) )
type PubKeyHash [PubKeyHashLen]byte

// IsZero 判断哈希是否为全零。
func (h Hash384) IsZero() bool {
	for _, b := range h {
		if b != 0 {
			return false
		}
	}
	return true
}

// String 返回哈希的十六进制字符串表示。
func (h Hash384) String() string {
	return hex.EncodeToString(h[:])
}

// IsZero 判断哈希是否为全零。
func (h Hash512) IsZero() bool {
	for _, b := range h {
		if b != 0 {
			return false
		}
	}
	return true
}

// String 返回哈希的十六进制字符串表示。
func (h Hash512) String() string {
	return hex.EncodeToString(h[:])
}

// IsZero 判断哈希是否为全零。
func (h Hash256) IsZero() bool {
	for _, b := range h {
		if b != 0 {
			return false
		}
	}
	return true
}

// String 返回哈希的十六进制字符串表示。
func (h Hash256) String() string {
	return hex.EncodeToString(h[:])
}

// IsZero 判断哈希是否为全零。
func (h PubKeyHash) IsZero() bool {
	for _, b := range h {
		if b != 0 {
			return false
		}
	}
	return true
}
```

**Step 2: 运行构建**

```bash
go build ./pkg/types/...
```

---

## Task 4: 地址编码（pkg/types/address.go）

**Files:**
- Create: `pkg/types/address.go`

**Step 1: 编写地址结构与编解码**

地址格式：`Prefix + Base58( PubKeyHash[32] || Checksum[4] )`

校验码计算：取 `hash( prefix || PKH )` 的最后 4 字节。

```go
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
```

**Step 2: 运行构建**

```bash
go build ./pkg/types/...
```

---

## Task 5: 类型包单元测试（pkg/types/types_test.go）

**Files:**
- Create: `pkg/types/types_test.go`

**Step 1: 编写表驱动测试**

```go
package types_test

import (
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// TestHash384IsZero 测试零哈希检测。
func TestHash384IsZero(t *testing.T) {
	cases := []struct {
		name string
		h    types.Hash384
		want bool
	}{
		{"zero hash", types.Hash384{}, true},
		{"non-zero hash", types.Hash384{1}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.h.IsZero(); got != tc.want {
				t.Errorf("IsZero() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestAddressEncodeDecodeRoundtrip 测试地址编解码往返一致性。
func TestAddressEncodeDecodeRoundtrip(t *testing.T) {
	pkh := types.PubKeyHash{1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
		11, 12, 13, 14, 15, 16, 17, 18, 19, 20,
		21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	addr := types.NewAddress(pkh, types.MainnetPrefix)
	encoded := addr.Encode()

	decoded, err := types.DecodeAddress(encoded, types.MainnetPrefix)
	if err != nil {
		t.Fatalf("DecodeAddress() error = %v", err)
	}
	if decoded.PKHash() != pkh {
		t.Errorf("PKHash mismatch: got %v, want %v", decoded.PKHash(), pkh)
	}
}

// TestAddressChecksumMismatch 测试错误校验码被正确拒绝。
func TestAddressChecksumMismatch(t *testing.T) {
	pkh := types.PubKeyHash{0xff}
	addr := types.NewAddress(pkh, types.MainnetPrefix)
	encoded := addr.Encode()
	// 篡改最后一个字节
	tampered := encoded[:len(encoded)-1] + "X"
	_, err := types.DecodeAddress(tampered, types.MainnetPrefix)
	if err == nil {
		t.Error("expected error for tampered address, got nil")
	}
}

// TestAddressWrongPrefix 测试前缀不匹配被拒绝。
func TestAddressWrongPrefix(t *testing.T) {
	pkh := types.PubKeyHash{0x01}
	addr := types.NewAddress(pkh, types.MainnetPrefix)
	encoded := addr.Encode()
	_, err := types.DecodeAddress(encoded, types.TestnetPrefix)
	if err == nil {
		t.Error("expected error for wrong prefix, got nil")
	}
}
```

**Step 2: 运行测试，确认通过**

```bash
go test ./pkg/types/... -v
```
预期：所有测试 PASS。

**Step 3: Commit**

```bash
git add pkg/types/
git commit -m "feat: add pkg/types with hash types, address encoding, and constants"
```

---

## Task 6: 哈希函数封装（pkg/crypto/hash.go）

**Files:**
- Create: `pkg/crypto/hash.go`

**Step 1: 编写哈希函数**

```go
// Package crypto 封装 Evidcoin 使用的密码学原语。
package crypto

import (
	"golang.org/x/crypto/sha3"
	"golang.org/x/crypto/blake2b"
	"lukechampine.com/blake3"

	"github.com/cxio/evidcoin/pkg/types"
)

// SHA3_384 计算数据的 SHA3-384 哈希，返回 Hash384。
// 用途：区块 ID、交易 ID（TxID）、CheckRoot、公钥哈希路径等。
func SHA3_384(data []byte) types.Hash384 {
	return sha3.Sum384(data)
}

// SHA3_512 计算数据的 SHA3-512 哈希，返回 Hash512（用 [64]byte 表示）。
// 用途：暂保留备用（铸凭哈希使用 BLAKE3-512）。
func SHA3_512(data []byte) [64]byte {
	return sha3.Sum512(data)
}

// SHA3_256 计算数据的 SHA3-256 哈希，返回 Hash256。
// 用途：公钥哈希双重哈希的外层。
func SHA3_256(data []byte) types.Hash256 {
	var h types.Hash256
	d := sha3.Sum256(data)
	copy(h[:], d[:])
	return h
}

// BLAKE3_256 计算数据的 BLAKE3-256 哈希，返回 Hash256。
// 用途：UTXO/UTCO 树内部节点、输出哈希树内部节点。
func BLAKE3_256(data []byte) types.Hash256 {
	var h types.Hash256
	sum := blake3.Sum256(data)
	copy(h[:], sum[:])
	return h
}

// BLAKE3_512 计算数据的 BLAKE3-512 哈希，返回 Hash512。
// 用途：铸凭哈希（MintHash）。
func BLAKE3_512(data []byte) types.Hash512 {
	var h types.Hash512
	// blake3 的 XOF 输出 64 字节
	out := make([]byte, types.Hash512Len)
	blake3.DeriveKey("evidcoin-mint", data, out)  // 带域分离
	copy(h[:], out)
	return h
}

// BLAKE2b_512 计算数据的 BLAKE2b-512 哈希，返回 64 字节。
// 用途：公钥哈希双重哈希的内层。
func BLAKE2b_512(data []byte) []byte {
	h, _ := blake2b.New512(nil) // 无 key，不会失败
	h.Write(data)
	return h.Sum(nil)
}
```

> **注：** `BLAKE3_512` 使用带域标签的 `DeriveKey` 以增强抗碰撞性。若需纯哈希模式，可改为 `blake3.New(64).Write(data).Sum(nil)`。

**Step 2: 运行构建**

```bash
go build ./pkg/crypto/...
```

---

## Task 7: 公钥哈希（pkg/crypto/pubkey.go）

**Files:**
- Create: `pkg/crypto/pubkey.go`

**Step 1: 编写公钥哈希计算**

```go
package crypto

import "github.com/cxio/evidcoin/pkg/types"

// PubKeyHash 计算公钥哈希。
// 算法：SHA3-256( BLAKE2b-512( publicKey ) )
// 输出 32 字节，用作地址的内核。
func PubKeyHash(pubKey []byte) types.PubKeyHash {
	inner := BLAKE2b_512(pubKey)
	outer := SHA3_256(inner)
	var pkh types.PubKeyHash
	copy(pkh[:], outer[:])
	return pkh
}

// MultiSigPubKeyHash 计算多重签名的复合公钥哈希。
// 算法：SHA3-256( BLAKE2b-512( [m, N] || PKH_1 || PKH_2 || ... || PKH_N ) )
// @m: 所需最小签名数
// @pkhs: 所有参与者的公钥哈希（按排序顺序）
func MultiSigPubKeyHash(m int, pkhs []types.PubKeyHash) types.PubKeyHash {
	// 构造 [m, N] 前缀 + 有序公钥哈希串联
	N := len(pkhs)
	payload := make([]byte, 2+N*types.PubKeyHashLen)
	payload[0] = byte(m)
	payload[1] = byte(N)
	for i, pkh := range pkhs {
		copy(payload[2+i*types.PubKeyHashLen:], pkh[:])
	}
	inner := BLAKE2b_512(payload)
	outer := SHA3_256(inner)
	var result types.PubKeyHash
	copy(result[:], outer[:])
	return result
}
```

**Step 2: 运行构建**

```bash
go build ./pkg/crypto/...
```

---

## Task 8: ML-DSA-65 签名封装（pkg/crypto/sign.go）

**Files:**
- Create: `pkg/crypto/sign.go`

**Step 1: 检查 Go 1.25 标准库是否内置 ML-DSA**

```bash
go doc crypto/mldsa 2>/dev/null || echo "not in stdlib, use circl"
```

**Step 2: 编写签名封装**

若标准库已内置，使用 `crypto/mldsa`；否则使用 `github.com/cloudflare/circl/sign/mldsa/mldsa65`。

```go
package crypto

import (
	// 优先使用标准库；若不可用，替换为 circl
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
func Sign(priv *PrivateKey, data []byte) []byte {
	sig := make([]byte, mldsa65.SignatureSize)
	mldsa65.SignTo(priv, data, nil, false, sig)
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
```

> **注：** 如果 Go 1.25 标准库已内置 `crypto/mldsa`，实现时替换 import 路径，API 保持一致。在 go.mod 中按需添加 `github.com/cloudflare/circl`。

**Step 3: 若使用 circl，添加依赖**

```bash
go get github.com/cloudflare/circl
go mod tidy
```

**Step 4: 运行构建**

```bash
go build ./pkg/crypto/...
```

---

## Task 9: 密码学包单元测试（pkg/crypto/crypto_test.go）

**Files:**
- Create: `pkg/crypto/crypto_test.go`

**Step 1: 编写表驱动测试**

```go
package crypto_test

import (
	"bytes"
	"testing"

	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// TestSHA3_384Deterministic 测试 SHA3-384 输出确定性。
func TestSHA3_384Deterministic(t *testing.T) {
	data := []byte("hello evidcoin")
	h1 := crypto.SHA3_384(data)
	h2 := crypto.SHA3_384(data)
	if h1 != h2 {
		t.Error("SHA3_384 is not deterministic")
	}
}

// TestSHA3_384Different 测试不同输入产生不同哈希。
func TestSHA3_384Different(t *testing.T) {
	h1 := crypto.SHA3_384([]byte("a"))
	h2 := crypto.SHA3_384([]byte("b"))
	if h1 == h2 {
		t.Error("SHA3_384 collision on different inputs")
	}
}

// TestBLAKE3_256Deterministic 测试 BLAKE3-256 确定性。
func TestBLAKE3_256Deterministic(t *testing.T) {
	data := []byte("test data")
	h1 := crypto.BLAKE3_256(data)
	h2 := crypto.BLAKE3_256(data)
	if h1 != h2 {
		t.Error("BLAKE3_256 is not deterministic")
	}
}

// TestPubKeyHashDeterministic 测试公钥哈希确定性。
func TestPubKeyHashDeterministic(t *testing.T) {
	pub := []byte("fake-public-key-bytes")
	h1 := crypto.PubKeyHash(pub)
	h2 := crypto.PubKeyHash(pub)
	if h1 != h2 {
		t.Error("PubKeyHash is not deterministic")
	}
}

// TestPubKeyHashLen 测试公钥哈希长度为 32 字节。
func TestPubKeyHashLen(t *testing.T) {
	pub := []byte("some-public-key")
	h := crypto.PubKeyHash(pub)
	if len(h) != types.PubKeyHashLen {
		t.Errorf("PubKeyHash length = %d, want %d", len(h), types.PubKeyHashLen)
	}
}

// TestSignVerifyRoundtrip 测试签名与验证往返一致性。
func TestSignVerifyRoundtrip(t *testing.T) {
	pub, priv, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	data := []byte("sign this message")
	sig := crypto.Sign(priv, data)
	if !crypto.Verify(pub, data, sig) {
		t.Error("Verify() returned false for valid signature")
	}
}

// TestSignVerifyTampered 测试篡改数据后验证失败。
func TestSignVerifyTampered(t *testing.T) {
	pub, priv, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	data := []byte("original message")
	sig := crypto.Sign(priv, data)
	tampered := []byte("tampered message")
	if crypto.Verify(pub, tampered, sig) {
		t.Error("Verify() returned true for tampered data")
	}
}

// TestPublicKeyBytesRoundtrip 测试公钥序列化往返一致性。
func TestPublicKeyBytesRoundtrip(t *testing.T) {
	pub, _, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	b := crypto.PublicKeyBytes(pub)
	pub2, err := crypto.PublicKeyFromBytes(b)
	if err != nil {
		t.Fatalf("PublicKeyFromBytes() error = %v", err)
	}
	b2 := crypto.PublicKeyBytes(pub2)
	if !bytes.Equal(b, b2) {
		t.Error("public key bytes roundtrip mismatch")
	}
}

// TestMultiSigPubKeyHashDifferentM 测试不同 m 值产生不同哈希。
func TestMultiSigPubKeyHashDifferentM(t *testing.T) {
	pkhs := []types.PubKeyHash{{1}, {2}, {3}}
	h1 := crypto.MultiSigPubKeyHash(2, pkhs)
	h2 := crypto.MultiSigPubKeyHash(3, pkhs)
	if h1 == h2 {
		t.Error("MultiSigPubKeyHash should differ for different m values")
	}
}
```

**Step 2: 运行测试**

```bash
go test ./pkg/crypto/... -v
```
预期：所有测试 PASS。

**Step 3: 检查覆盖率**

```bash
go test ./pkg/... -cover
```
预期：核心逻辑覆盖率 ≥ 80%。

**Step 4: Commit**

```bash
git add pkg/crypto/
git commit -m "feat: add pkg/crypto with hash functions, pubkey hash, and ML-DSA-65 signing"
```

---

## Task 10: 整体验收

**Step 1: 完整构建**

```bash
go build ./...
```
预期：无错误。

**Step 2: 全量测试**

```bash
go test ./pkg/... -v -cover
```
预期：所有测试通过，覆盖率 ≥ 80%。

**Step 3: 代码格式**

```bash
go fmt ./pkg/... && gofmt -s -l ./pkg/
```
预期：无文件列出（无需格式化）。

**Step 4: 静态分析**

```bash
golangci-lint run ./pkg/...
```
预期：无警告。

---

## 注意事项

1. **BLAKE3_512 实现**：`lukechampine.com/blake3` 的 `Sum512` 不直接存在，需使用 `blake3.New(64)` 写入数据后调用 `Sum(nil)`，或使用 XOF 方式输出 64 字节。编写时以实际 API 为准。

2. **ML-DSA-65 依赖**：若 Go 1.25 标准库已内置，优先使用，无需 circl 依赖。编码前先检查：`go doc crypto/mldsa`。

3. **BLAKE2b import 路径**：`golang.org/x/crypto/blake2b` 是正确路径，不是标准库。

4. **地址校验码算法**：提案使用"对带前缀数据哈希"，当前实现使用 SHA256；若后续提案明确指定 SHA3-256，需调整 `addressChecksum` 函数。
