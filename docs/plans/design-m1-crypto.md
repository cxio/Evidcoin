# M1: crypto 模块设计

> **模块路径:** `pkg/crypto`
> **依赖:** 无（基础模块）
> **预估工时:** 3-4 天

## 概述

密码学基础库，提供哈希计算、数字签名和地址编码功能。作为公共包，可被外部项目引用。

## 功能清单

| 功能 | 文件 | 说明 |
|------|------|------|
| SHA-512 哈希 | `hash.go` | 64字节哈希，用于交易/区块哈希 |
| BLAKE3 哈希 | `hash.go` | 用于附件哈希，可变长度输出 |
| Ed25519 签名 | `signature.go` | 过渡期签名算法 |
| 签名抽象层 | `signature.go` | 预留后量子算法扩展 |
| Base58Check | `address.go` | 地址编码/解码 |
| 公钥哈希 | `address.go` | 公钥到地址的转换 |

---

## 详细设计

### 1. hash.go - 哈希函数

```go
package crypto

import (
    "crypto/sha512"
    
    "github.com/zeebo/blake3"
)

// Hash512 计算 SHA-512 哈希
// 用于交易ID、区块ID等核心数据的哈希
func Hash512(data []byte) [64]byte {
    return sha512.Sum512(data)
}

// Hash512Multi 计算多个数据片段的组合哈希
func Hash512Multi(parts ...[]byte) [64]byte {
    h := sha512.New()
    for _, p := range parts {
        h.Write(p)
    }
    var result [64]byte
    copy(result[:], h.Sum(nil))
    return result
}

// HashBlake3 计算 BLAKE3 哈希
// 用于附件数据的哈希，支持可变长度输出
// size: 输出长度，范围 [16, 64]，步长 4
func HashBlake3(data []byte, size int) []byte {
    if size < 16 || size > 64 {
        size = 32 // 默认 256 bit
    }
    h := blake3.New()
    h.Write(data)
    result := make([]byte, size)
    h.Digest().Read(result)
    return result
}

// DoubleHash512 双重哈希
// 用于某些需要增强安全性的场景
func DoubleHash512(data []byte) [64]byte {
    first := sha512.Sum512(data)
    return sha512.Sum512(first[:])
}
```

### 2. signature.go - 数字签名

```go
package crypto

import (
    "crypto/ed25519"
    "crypto/rand"
    "errors"
)

// SignatureType 签名算法类型
type SignatureType byte

const (
    SigTypeEd25519 SignatureType = 1 // Ed25519（过渡期）
    SigTypeML_DSA  SignatureType = 2 // ML-DSA（后量子，预留）
)

// PrivateKey 私钥接口
type PrivateKey interface {
    // Sign 对消息签名
    Sign(message []byte) ([]byte, error)
    // Public 获取对应的公钥
    Public() PublicKey
    // Type 返回签名算法类型
    Type() SignatureType
    // Bytes 返回私钥字节序列
    Bytes() []byte
}

// PublicKey 公钥接口
type PublicKey interface {
    // Verify 验证签名
    Verify(message, signature []byte) bool
    // Type 返回签名算法类型
    Type() SignatureType
    // Bytes 返回公钥字节序列
    Bytes() []byte
    // Hash 计算公钥哈希（用于地址生成）
    Hash() []byte
}

// Ed25519PrivateKey Ed25519 私钥实现
type Ed25519PrivateKey struct {
    key ed25519.PrivateKey
}

// Ed25519PublicKey Ed25519 公钥实现
type Ed25519PublicKey struct {
    key ed25519.PublicKey
}

// GenerateEd25519Key 生成 Ed25519 密钥对
func GenerateEd25519Key() (*Ed25519PrivateKey, error) {
    _, priv, err := ed25519.GenerateKey(rand.Reader)
    if err != nil {
        return nil, err
    }
    return &Ed25519PrivateKey{key: priv}, nil
}

// NewEd25519PrivateKey 从字节序列创建私钥
func NewEd25519PrivateKey(data []byte) (*Ed25519PrivateKey, error) {
    if len(data) != ed25519.PrivateKeySize {
        return nil, errors.New("invalid private key size")
    }
    return &Ed25519PrivateKey{key: ed25519.PrivateKey(data)}, nil
}

// NewEd25519PublicKey 从字节序列创建公钥
func NewEd25519PublicKey(data []byte) (*Ed25519PublicKey, error) {
    if len(data) != ed25519.PublicKeySize {
        return nil, errors.New("invalid public key size")
    }
    return &Ed25519PublicKey{key: ed25519.PublicKey(data)}, nil
}

func (k *Ed25519PrivateKey) Sign(message []byte) ([]byte, error) {
    return ed25519.Sign(k.key, message), nil
}

func (k *Ed25519PrivateKey) Public() PublicKey {
    pub := k.key.Public().(ed25519.PublicKey)
    return &Ed25519PublicKey{key: pub}
}

func (k *Ed25519PrivateKey) Type() SignatureType {
    return SigTypeEd25519
}

func (k *Ed25519PrivateKey) Bytes() []byte {
    return k.key
}

func (k *Ed25519PublicKey) Verify(message, signature []byte) bool {
    return ed25519.Verify(k.key, message, signature)
}

func (k *Ed25519PublicKey) Type() SignatureType {
    return SigTypeEd25519
}

func (k *Ed25519PublicKey) Bytes() []byte {
    return k.key
}

func (k *Ed25519PublicKey) Hash() []byte {
    // 使用 SHA-512 的前 48 字节作为公钥哈希
    hash := Hash512(k.key)
    return hash[:48]
}
```

### 3. address.go - 地址编码

```go
package crypto

import (
    "errors"
    
    "github.com/btcsuite/btcutil/base58"
)

const (
    // AddressVersion 地址版本前缀
    AddressVersion byte = 0x00
    
    // ChecksumLength 校验码长度
    ChecksumLength = 4
    
    // PubKeyHashLength 公钥哈希长度
    PubKeyHashLength = 48
)

var (
    ErrInvalidAddress  = errors.New("invalid address format")
    ErrChecksumFailed  = errors.New("address checksum failed")
    ErrInvalidPubKeyHash = errors.New("invalid public key hash length")
)

// EncodeAddress 将公钥哈希编码为地址字符串
// 格式: Base58Check(version + pubKeyHash + checksum)
func EncodeAddress(pubKeyHash []byte) (string, error) {
    if len(pubKeyHash) != PubKeyHashLength {
        return "", ErrInvalidPubKeyHash
    }
    
    // 添加版本前缀
    versionedPayload := make([]byte, 1+PubKeyHashLength)
    versionedPayload[0] = AddressVersion
    copy(versionedPayload[1:], pubKeyHash)
    
    // 计算校验码：对 (version + pubKeyHash) 双重哈希，取前4字节
    checksum := computeChecksum(versionedPayload)
    
    // 拼接：pubKeyHash + checksum（不含 version，version 仅用于校验计算）
    fullPayload := make([]byte, PubKeyHashLength+ChecksumLength)
    copy(fullPayload, pubKeyHash)
    copy(fullPayload[PubKeyHashLength:], checksum)
    
    // Base58 编码
    return base58.Encode(fullPayload), nil
}

// DecodeAddress 将地址字符串解码为公钥哈希
func DecodeAddress(address string) ([]byte, error) {
    decoded := base58.Decode(address)
    if len(decoded) != PubKeyHashLength+ChecksumLength {
        return nil, ErrInvalidAddress
    }
    
    pubKeyHash := decoded[:PubKeyHashLength]
    providedChecksum := decoded[PubKeyHashLength:]
    
    // 重建版本前缀并验证校验码
    versionedPayload := make([]byte, 1+PubKeyHashLength)
    versionedPayload[0] = AddressVersion
    copy(versionedPayload[1:], pubKeyHash)
    
    expectedChecksum := computeChecksum(versionedPayload)
    
    for i := 0; i < ChecksumLength; i++ {
        if providedChecksum[i] != expectedChecksum[i] {
            return nil, ErrChecksumFailed
        }
    }
    
    return pubKeyHash, nil
}

// computeChecksum 计算校验码
func computeChecksum(data []byte) []byte {
    hash := DoubleHash512(data)
    return hash[len(hash)-ChecksumLength:]
}

// PubKeyToAddress 从公钥直接生成地址
func PubKeyToAddress(pubKey PublicKey) (string, error) {
    return EncodeAddress(pubKey.Hash())
}

// IsValidAddress 检查地址格式是否有效
func IsValidAddress(address string) bool {
    _, err := DecodeAddress(address)
    return err == nil
}

// NullAddress 空地址（用于销毁）
var NullAddress = make([]byte, PubKeyHashLength)

// IsNullAddress 检查是否为空地址
func IsNullAddress(pubKeyHash []byte) bool {
    if len(pubKeyHash) != PubKeyHashLength {
        return false
    }
    for _, b := range pubKeyHash {
        if b != 0 {
            return false
        }
    }
    return true
}
```

### 4. multisig.go - 多重签名支持

```go
package crypto

import (
    "errors"
    "sort"
)

var (
    ErrInvalidThreshold = errors.New("invalid threshold: m must be <= n")
    ErrTooFewSignatures = errors.New("not enough signatures")
    ErrDuplicateKey     = errors.New("duplicate public key")
)

// MultiSigConfig 多重签名配置
type MultiSigConfig struct {
    M int // 需要的最少签名数
    N int // 总公钥数
}

// MultiSigAddress 计算多重签名地址
// 公钥哈希按字典序排序后串接，前置 m/n 配置
func MultiSigAddress(m, n int, pubKeyHashes [][]byte) ([]byte, error) {
    if m > n || m <= 0 || n <= 0 || n > 255 {
        return nil, ErrInvalidThreshold
    }
    if len(pubKeyHashes) != n {
        return nil, errors.New("pubKeyHashes count mismatch")
    }
    
    // 排序公钥哈希
    sorted := make([][]byte, n)
    copy(sorted, pubKeyHashes)
    sort.Slice(sorted, func(i, j int) bool {
        return compareBytes(sorted[i], sorted[j]) < 0
    })
    
    // 检查重复
    for i := 1; i < n; i++ {
        if compareBytes(sorted[i-1], sorted[i]) == 0 {
            return nil, ErrDuplicateKey
        }
    }
    
    // 构造数据: m(1) + n(1) + PKH1 + PKH2 + ...
    data := make([]byte, 2+n*PubKeyHashLength)
    data[0] = byte(m)
    data[1] = byte(n)
    offset := 2
    for _, pkh := range sorted {
        copy(data[offset:], pkh)
        offset += PubKeyHashLength
    }
    
    // 计算复合哈希
    hash := Hash512(data)
    return hash[:PubKeyHashLength], nil
}

// VerifyMultiSig 验证多重签名
func VerifyMultiSig(
    message []byte,
    signatures [][]byte,
    pubKeys []PublicKey,
    fullPubKeyHashes [][]byte, // 全部 n 个公钥哈希（含补全集）
    expectedAddress []byte,
) error {
    m := len(signatures)
    n := len(fullPubKeyHashes)
    
    if m > n {
        return ErrInvalidThreshold
    }
    if len(pubKeys) != m {
        return errors.New("signature and pubKey count mismatch")
    }
    
    // 验证地址
    computedAddr, err := MultiSigAddress(m, n, fullPubKeyHashes)
    if err != nil {
        return err
    }
    if compareBytes(computedAddr, expectedAddress) != 0 {
        return errors.New("address mismatch")
    }
    
    // 验证每个签名
    for i, sig := range signatures {
        if !pubKeys[i].Verify(message, sig) {
            return errors.New("signature verification failed")
        }
    }
    
    return nil
}

func compareBytes(a, b []byte) int {
    minLen := len(a)
    if len(b) < minLen {
        minLen = len(b)
    }
    for i := 0; i < minLen; i++ {
        if a[i] < b[i] {
            return -1
        }
        if a[i] > b[i] {
            return 1
        }
    }
    if len(a) < len(b) {
        return -1
    }
    if len(a) > len(b) {
        return 1
    }
    return 0
}
```

---

## 测试用例

### crypto_test.go

```go
package crypto

import (
    "bytes"
    "testing"
)

func TestHash512(t *testing.T) {
    data := []byte("hello world")
    hash := Hash512(data)
    
    if len(hash) != 64 {
        t.Errorf("expected hash length 64, got %d", len(hash))
    }
    
    // 相同输入应产生相同输出
    hash2 := Hash512(data)
    if hash != hash2 {
        t.Error("same input should produce same hash")
    }
    
    // 不同输入应产生不同输出
    hash3 := Hash512([]byte("hello world!"))
    if hash == hash3 {
        t.Error("different input should produce different hash")
    }
}

func TestEd25519KeyGeneration(t *testing.T) {
    priv, err := GenerateEd25519Key()
    if err != nil {
        t.Fatalf("failed to generate key: %v", err)
    }
    
    pub := priv.Public()
    
    // 测试签名和验证
    message := []byte("test message")
    sig, err := priv.Sign(message)
    if err != nil {
        t.Fatalf("failed to sign: %v", err)
    }
    
    if !pub.Verify(message, sig) {
        t.Error("signature verification failed")
    }
    
    // 错误的消息不应通过验证
    if pub.Verify([]byte("wrong message"), sig) {
        t.Error("wrong message should not verify")
    }
}

func TestAddressEncoding(t *testing.T) {
    // 生成密钥
    priv, _ := GenerateEd25519Key()
    pub := priv.Public()
    
    // 生成地址
    addr, err := PubKeyToAddress(pub)
    if err != nil {
        t.Fatalf("failed to generate address: %v", err)
    }
    
    // 解码地址
    decoded, err := DecodeAddress(addr)
    if err != nil {
        t.Fatalf("failed to decode address: %v", err)
    }
    
    // 验证解码结果
    expected := pub.Hash()
    if !bytes.Equal(decoded, expected) {
        t.Error("decoded address does not match original pubKeyHash")
    }
}

func TestMultiSigAddress(t *testing.T) {
    // 生成 3 个密钥
    var pubKeyHashes [][]byte
    for i := 0; i < 3; i++ {
        priv, _ := GenerateEd25519Key()
        pubKeyHashes = append(pubKeyHashes, priv.Public().Hash())
    }
    
    // 2-of-3 多签地址
    addr, err := MultiSigAddress(2, 3, pubKeyHashes)
    if err != nil {
        t.Fatalf("failed to create multisig address: %v", err)
    }
    
    if len(addr) != PubKeyHashLength {
        t.Errorf("expected address length %d, got %d", PubKeyHashLength, len(addr))
    }
    
    // 相同配置应产生相同地址
    addr2, _ := MultiSigAddress(2, 3, pubKeyHashes)
    if !bytes.Equal(addr, addr2) {
        t.Error("same config should produce same address")
    }
}

func TestNullAddress(t *testing.T) {
    if !IsNullAddress(NullAddress) {
        t.Error("NullAddress should be null")
    }
    
    nonNull := make([]byte, PubKeyHashLength)
    nonNull[0] = 1
    if IsNullAddress(nonNull) {
        t.Error("non-zero address should not be null")
    }
}
```

---

## 实现步骤

### Step 1: 创建包结构

```bash
mkdir -p pkg/crypto
touch pkg/crypto/hash.go
touch pkg/crypto/signature.go
touch pkg/crypto/address.go
touch pkg/crypto/multisig.go
touch pkg/crypto/crypto_test.go
```

### Step 2: 添加依赖

```bash
go get github.com/zeebo/blake3
go get github.com/btcsuite/btcutil/base58
```

### Step 3: 按顺序实现

1. `hash.go` - 哈希函数（无依赖）
2. `signature.go` - 签名接口和 Ed25519 实现
3. `address.go` - 地址编码（依赖 hash.go）
4. `multisig.go` - 多重签名（依赖以上全部）

### Step 4: 测试验证

```bash
go test -v ./pkg/crypto/...
```

---

## 注意事项

1. **安全性**: 私钥处理需谨慎，避免在日志或错误消息中泄露
2. **可扩展性**: 签名接口设计需考虑后量子算法的集成
3. **性能**: BLAKE3 比 SHA-512 快，适合大文件附件
4. **兼容性**: Base58Check 与 Bitcoin 兼容，便于用户理解
