# Phase 3: Transaction Model（交易模型）Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现 `internal/tx` 包，定义完整的交易数据结构、哈希计算、签名模型、Coinbase 交易和附件 ID 系统。

**Architecture:** Layer 1 核心层。依赖 `pkg/types` 和 `pkg/crypto`（Layer 0）。不含 UTXO/UTCO 状态（阶段 4），不含脚本执行（阶段 5）。本包专注于交易数据的结构定义、序列化与哈希验证。

**Tech Stack:** Go 1.25+，`encoding/binary`（定长字段），自定义 varint 编码（可变长度整数）。

---

## 目录结构（预期）

```
internal/tx/
  types.go          # 核心类型：Output, Input, TxHeader 等
  hash.go           # 哈希计算：TxID, InputHash, OutputHash, CheckRoot 的辅助函数
  transaction.go    # Transaction / CoinbaseTx 结构
  sig.go            # 签名授权标志与解锁数据结构
  attachment.go     # AttachmentID 结构
  varint.go         # 可变长整数编解码
  tx_test.go        # 单元测试
```

---

## Task 1: 可变长整数编解码（internal/tx/varint.go）

**Files:**
- Create: `internal/tx/varint.go`

**Step 1: 编写 varint 工具**

```go
// Package tx 定义 Evidcoin 的交易数据结构与哈希计算。
package tx

import (
	"encoding/binary"
	"errors"
)

// AppendVarint 将 v 编码为 varint 并追加到 buf，返回新的 buf。
// 使用 protobuf 风格的 varint 编码（每字节 7 位有效，最高位为延续标志）。
func AppendVarint(buf []byte, v uint64) []byte {
	for v >= 0x80 {
		buf = append(buf, byte(v)|0x80)
		v >>= 7
	}
	return append(buf, byte(v))
}

// ReadVarint 从 data 读取一个 varint，返回值、消耗字节数与错误。
func ReadVarint(data []byte) (uint64, int, error) {
	var v uint64
	var s uint
	for i, b := range data {
		if i == 10 {
			return 0, 0, errors.New("varint overflow")
		}
		if b < 0x80 {
			v |= uint64(b) << s
			return v, i + 1, nil
		}
		v |= uint64(b&0x7f) << s
		s += 7
	}
	return 0, 0, errors.New("varint truncated")
}

// EncodeInt64 将 int64 编码为大端 8 字节序列（用于铸凭哈希计算）。
func EncodeInt64(v int64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(v))
	return buf
}
```

**Step 2: 运行构建**

```bash
go build ./internal/tx/...
```

---

## Task 2: 核心类型定义（internal/tx/types.go）

**Files:**
- Create: `internal/tx/types.go`

**Step 1: 编写类型结构**

```go
package tx

import "github.com/cxio/evidcoin/pkg/types"

// ---- 输出配置字节标志 ----

const (
	// OutFlagCustomClass 自定义类标志（Bit 7），余下 7 位为类 ID 长度
	OutFlagCustomClass = byte(1 << 7)
	// OutFlagHasAttach 包含附件标志（Bit 6）
	OutFlagHasAttach = byte(1 << 6)
	// OutFlagDestroy 销毁标志（Bit 5）
	OutFlagDestroy = byte(1 << 5)
	// OutTypeMask 低 4 位类型掩码
	OutTypeMask = byte(0x0f)
)

// CoinbaseOutputTarget Coinbase 输出目标类型（低 4 位）
type CoinbaseOutputTarget byte

const (
	// CoinbaseMinter 铸凭者（10%）
	CoinbaseMinter CoinbaseOutputTarget = 1
	// CoinbaseCheckTeam 校验组（40%）
	CoinbaseCheckTeam CoinbaseOutputTarget = 2
	// CoinbaseBlockqs 区块查询服务（20%）
	CoinbaseBlockqs CoinbaseOutputTarget = 3
	// CoinbaseDepots 数据驿站服务（20%）
	CoinbaseDepots CoinbaseOutputTarget = 4
	// CoinbaseSTUN NAT 穿透服务（10%）
	CoinbaseSTUN CoinbaseOutputTarget = 5
)

// ---- 输出结构 ----

// Output 交易输出项，承载 Coin / Credit / Proof / Mediator 之一。
type Output struct {
	// Serial 输出序位（从 0 开始）
	Serial int
	// Config 配置字节（类型 + 标志位）
	Config byte
	// Amount 币金数量（最小单位 chx/聪，仅 Coin 有效）
	Amount int64
	// Address 接收地址（公钥哈希，32 字节；Destroy 时可为零值）
	Address types.PubKeyHash
	// LockScript 锁定脚本（最大 1024 字节；Proof 无此字段，改用 IdentScript）
	LockScript []byte
	// ---- 以下字段按信元类型选择性存在 ----
	// Memo 币金附言（最大 255 字节）
	Memo []byte
	// Creator 创建者标识（凭信/存证，< 256 字节）
	Creator []byte
	// Title 标题（凭信/存证，最大 255 字节）
	Title []byte
	// Desc 描述（凭信，最大 1023 字节）
	Desc []byte
	// Content 内容（存证，最大 4095 字节）
	Content []byte
	// AttachID 附件 ID（可选）
	AttachID []byte
	// CredConfig 凭信配置（2 字节）
	CredConfig uint16
	// ProofContentLen 存证内容长度字段（2 字节，低 12 位为内容长度）
	ProofContentLen uint16
	// IdentScript 识别脚本（仅存证使用）
	IdentScript []byte
}

// Type 返回输出类型（低 4 位）。
func (o *Output) Type() byte {
	return o.Config & OutTypeMask
}

// IsDestroy 返回销毁标志是否置位。
func (o *Output) IsDestroy() bool {
	return o.Config&OutFlagDestroy != 0
}

// HasAttachment 返回是否包含附件。
func (o *Output) HasAttachment() bool {
	return o.Config&OutFlagHasAttach != 0
}

// IsCustomClass 返回是否为自定义类输出。
func (o *Output) IsCustomClass() bool {
	return o.Config&OutFlagCustomClass != 0
}

// CanBeUTXO 返回该输出是否可以进入 UTXO 集。
func (o *Output) CanBeUTXO() bool {
	if o.IsDestroy() || o.IsCustomClass() {
		return false
	}
	return o.Type() == types.OutTypeCoin
}

// CanBeUTCO 返回该输出是否可以进入 UTCO 集。
func (o *Output) CanBeUTCO() bool {
	if o.IsDestroy() || o.IsCustomClass() {
		return false
	}
	return o.Type() == types.OutTypeCredit
}

// ---- 输入结构 ----

// LeadInput 首领输入（使用完整 48 字节 TxID）。
type LeadInput struct {
	// Year 被引用交易所在年度
	Year int
	// TxID 完整交易 ID（48 字节）
	TxID types.Hash384
	// OutIndex 被引用输出序位
	OutIndex int
}

// RestInput 非首领输入（使用截断的 20 字节 TxID 引用）。
type RestInput struct {
	// Year 被引用交易所在年度
	Year int
	// TxIDPart 交易 ID 前 20 字节
	TxIDPart [20]byte
	// OutIndex 被引用输出序位
	OutIndex int
	// TransferIndex 凭信转出序位（可选，-1 表示不存在）
	TransferIndex int
}

// ---- 交易头 ----

// TxHeader 交易头结构。
// TxID = SHA3-384( TxHeader )，签名不参与计算。
type TxHeader struct {
	// Version 版本号
	Version int
	// Timestamp 交易时间戳（Unix 毫秒）
	Timestamp int64
	// HashInputs 输入项根哈希（BLAKE3-256，32 字节）
	HashInputs types.Hash256
	// HashOutputs 输出项根哈希（BLAKE3-256，32 字节）
	HashOutputs types.Hash256
}

// SigFlag 签名授权标志（1 字节）。
type SigFlag byte

const (
	// SIGIN_ALL 全部输入项（独项，Bit 7）
	SIGIN_ALL SigFlag = 1 << 7
	// SIGIN_SELF 仅当前输入项（独项，Bit 6）
	SIGIN_SELF SigFlag = 1 << 6
	// SIGOUT_ALL 全部输出项（主项，Bit 5）
	SIGOUT_ALL SigFlag = 1 << 5
	// SIGOUT_SELF 与当前输入同序位的输出项（主项，Bit 4）
	SIGOUT_SELF SigFlag = 1 << 4
	// SIGOUTPUT 完整输出条目（辅项，Bit 3）
	SIGOUTPUT SigFlag = 1 << 3
	// SIGSCRIPT 输出的锁定脚本（辅项，Bit 2）
	SIGSCRIPT SigFlag = 1 << 2
	// SIGCONTENT 输出内容（辅项，Bit 1）
	SIGCONTENT SigFlag = 1 << 1
	// SIGRECEIVER 输出的接收者（辅项，Bit 0）
	SIGRECEIVER SigFlag = 1 << 0
)

// Outpoint 引用交易输出的唯一标识。
type Outpoint struct {
	// TxID 所在交易 ID
	TxID types.Hash384
	// Index 输出序位
	Index int
}
```

**Step 2: 运行构建**

```bash
go build ./internal/tx/...
```

---

## Task 3: 哈希计算（internal/tx/hash.go）

**Files:**
- Create: `internal/tx/hash.go`

**Step 1: 编写哈希辅助函数**

```go
package tx

import (
	"encoding/binary"

	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// ComputeTxID 计算交易 ID：SHA3-384( TxHeader.Bytes() )。
func ComputeTxID(header *TxHeader) types.Hash384 {
	return crypto.SHA3_384(txHeaderBytes(header))
}

// txHeaderBytes 将交易头序列化为字节（用于 TxID 计算）。
func txHeaderBytes(h *TxHeader) []byte {
	buf := make([]byte, 0, 8+8+types.Hash256Len*2)
	buf = AppendVarint(buf, uint64(h.Version))
	ts := make([]byte, 8)
	binary.BigEndian.PutUint64(ts, uint64(h.Timestamp))
	buf = append(buf, ts...)
	buf = append(buf, h.HashInputs[:]...)
	buf = append(buf, h.HashOutputs[:]...)
	return buf
}

// ComputeLeadHash 计算首领输入哈希：BLAKE3-256( Year || TxID[48] || OutIndex )。
func ComputeLeadHash(lead *LeadInput) types.Hash256 {
	buf := make([]byte, 0, 8+types.Hash384Len+8)
	buf = AppendVarint(buf, uint64(lead.Year))
	buf = append(buf, lead.TxID[:]...)
	buf = AppendVarint(buf, uint64(lead.OutIndex))
	return crypto.BLAKE3_256(buf)
}

// ComputeRestHash 计算非首领输入集合哈希：BLAKE3-256( rest1 || rest2 || ... )。
func ComputeRestHash(rests []RestInput) types.Hash256 {
	var buf []byte
	for _, r := range rests {
		buf = AppendVarint(buf, uint64(r.Year))
		buf = append(buf, r.TxIDPart[:]...)
		buf = AppendVarint(buf, uint64(r.OutIndex))
		if r.TransferIndex >= 0 {
			buf = AppendVarint(buf, uint64(r.TransferIndex))
		}
	}
	return crypto.BLAKE3_256(buf)
}

// ComputeInputHash 计算输入集根哈希：BLAKE3-256( LeadHash || RestHash )。
func ComputeInputHash(lead *LeadInput, rests []RestInput) types.Hash256 {
	leadHash := ComputeLeadHash(lead)
	restHash := ComputeRestHash(rests)
	combined := append(leadHash[:], restHash[:]...)
	return crypto.BLAKE3_256(combined)
}

// ComputeOutputHash 计算输出集根哈希（二元哈希树，BLAKE3-256 内部节点，SHA3-384 叶子）。
// OutputHash = BLAKE3-256( BinaryTree( SHA3-384(output_0), SHA3-384(output_1), ... ) )
func ComputeOutputHash(outputs []Output) types.Hash256 {
	if len(outputs) == 0 {
		return types.Hash256{}
	}
	// 计算各叶子哈希
	leaves := make([][]byte, len(outputs))
	for i, o := range outputs {
		h := outputLeafHash(&o)
		leaves[i] = h[:]
	}
	return binaryTreeHash(leaves)
}

// outputLeafHash 计算单个输出的叶子哈希（SHA3-384）。
func outputLeafHash(o *Output) types.Hash384 {
	// 序列化输出的关键字段用于哈希
	buf := make([]byte, 0, 64)
	buf = AppendVarint(buf, uint64(o.Serial))
	buf = append(buf, o.Config)
	buf = append(buf, EncodeInt64(o.Amount)...)
	buf = append(buf, o.Address[:]...)
	buf = append(buf, o.LockScript...)
	buf = append(buf, o.Memo...)
	buf = append(buf, o.Creator...)
	buf = append(buf, o.Title...)
	buf = append(buf, o.Desc...)
	buf = append(buf, o.Content...)
	buf = append(buf, o.AttachID...)
	return crypto.SHA3_384(buf)
}

// binaryTreeHash 对叶子列表计算类 Merkle 二元哈希树根（BLAKE3-256 内部节点）。
func binaryTreeHash(leaves [][]byte) types.Hash256 {
	if len(leaves) == 1 {
		return crypto.BLAKE3_256(leaves[0])
	}
	mid := len(leaves) / 2
	left := binaryTreeHash(leaves[:mid])
	right := binaryTreeHash(leaves[mid:])
	combined := append(left[:], right[:]...)
	return crypto.BLAKE3_256(combined)
}

// ComputeCheckRoot 计算 CheckRoot。
// CheckRoot = SHA3-384( TxTreeRoot || UTXOFingerprint || UTCOFingerprint )
func ComputeCheckRoot(txTreeRoot types.Hash256, utxoFP, utcoFP types.Hash256) types.Hash384 {
	buf := make([]byte, 0, types.Hash256Len+types.Hash256Len*2)
	buf = append(buf, txTreeRoot[:]...)
	buf = append(buf, utxoFP[:]...)
	buf = append(buf, utcoFP[:]...)
	return crypto.SHA3_384(buf)
}

// ComputeTxTreeRoot 计算区块内所有交易 ID 的哈希树根。
// 每个叶子为 3 字节序号前缀 + TxID（48 字节）。
func ComputeTxTreeRoot(txIDs []types.Hash384) types.Hash256 {
	if len(txIDs) == 0 {
		return types.Hash256{}
	}
	leaves := make([][]byte, len(txIDs))
	for i, txID := range txIDs {
		leaf := make([]byte, 3+types.Hash384Len)
		// 3 字节大端序号
		leaf[0] = byte(i >> 16)
		leaf[1] = byte(i >> 8)
		leaf[2] = byte(i)
		copy(leaf[3:], txID[:])
		leaves[i] = leaf
	}
	return binaryTreeHash(leaves)
}
```

**Step 2: 运行构建**

```bash
go build ./internal/tx/...
```

---

## Task 4: 交易与 Coinbase 结构（internal/tx/transaction.go）

**Files:**
- Create: `internal/tx/transaction.go`

**Step 1: 编写交易结构**

```go
package tx

import (
	"errors"

	"github.com/cxio/evidcoin/pkg/types"
)

// Transaction 标准交易结构。
type Transaction struct {
	// Header 交易头（TxID 由此计算）
	Header TxHeader
	// Lead 首领输入
	Lead LeadInput
	// Rests 非首领输入列表
	Rests []RestInput
	// Outputs 输出项列表
	Outputs []Output
	// Unlocks 各输入对应的解锁数据（按输入顺序，不参与 TxID 计算）
	Unlocks []UnlockData
}

// ID 返回该交易的 TxID。
func (tx *Transaction) ID() types.Hash384 {
	return ComputeTxID(&tx.Header)
}

// InputCount 返回输入项总数（首领 + 非首领）。
func (tx *Transaction) InputCount() int {
	return 1 + len(tx.Rests)
}

// CoinInputAmount 返回所有 Coin 输入来源的总金额（从外部提供，不存储于交易）。
// 手续费 = CoinInputAmount - SumOutputAmounts
// 注意：该函数需要外部传入查询结果，tx 包本身不查询 UTXO。
func CoinFee(inputTotal int64, outputs []Output) (int64, error) {
	var outputTotal int64
	for _, o := range outputs {
		if o.Type() == types.OutTypeCoin && !o.IsDestroy() {
			outputTotal += o.Amount
		}
	}
	fee := inputTotal - outputTotal
	if fee < 0 {
		return 0, errors.New("negative fee: outputs exceed inputs")
	}
	return fee, nil
}

// CoinbaseTx 铸币交易（每个区块的第一笔交易，索引 [0]，无输入）。
type CoinbaseTx struct {
	// Header 标准交易头
	Header TxHeader
	// BlockHeight 区块高度
	BlockHeight int32
	// MeritProof 择优凭证字节（铸造者证明）
	MeritProof []byte
	// TotalReward 收益总额（聪）
	TotalReward int64
	// SelfData 自由数据（最大 255 字节）
	SelfData []byte
	// Outputs 收益分配输出列表
	Outputs []CoinbaseOutput
}

// CoinbaseOutput Coinbase 专用输出（配置字节低 4 位为目标类型）。
type CoinbaseOutput struct {
	// Target 奖励目标类型
	Target CoinbaseOutputTarget
	// Amount 金额
	Amount int64
	// Address 接收地址
	Address types.PubKeyHash
	// LockScript 锁定脚本（包含 SYS_AWARD 指令）
	LockScript []byte
	// IsBurn 是否为销毁输出（手续费中 50% 销毁）
	IsBurn bool
}

// ID 返回 Coinbase 交易的 TxID。
func (cb *CoinbaseTx) ID() types.Hash384 {
	return ComputeTxID(&cb.Header)
}

// ValidateSize 验证交易大小是否超限（不含解锁数据）。
func ValidateSize(serializedSize int) error {
	if serializedSize > types.MaxTxSize {
		return errors.New("transaction size exceeds limit")
	}
	return nil
}
```

**Step 2: 运行构建**

```bash
go build ./internal/tx/...
```

---

## Task 5: 签名解锁数据（internal/tx/sig.go）

**Files:**
- Create: `internal/tx/sig.go`

**Step 1: 编写解锁数据结构**

```go
package tx

// UnlockData 输入项解锁数据（不参与 TxID 计算）。
type UnlockData struct {
	// InputIndex 对应的输入序位
	InputIndex int
	// Flag 签名授权标志
	Flag SigFlag
	// Kind 解锁类型（单签/多签）
	Kind UnlockKind
	// Single 单签名解锁数据（Kind == UnlockSingle 时有效）
	Single *SingleSigUnlock
	// Multi 多签名解锁数据（Kind == UnlockMulti 时有效）
	Multi *MultiSigUnlock
}

// UnlockKind 解锁类型。
type UnlockKind byte

const (
	// UnlockSingle 单签名解锁
	UnlockSingle UnlockKind = 1
	// UnlockMulti 多重签名解锁
	UnlockMulti UnlockKind = 2
)

// SingleSigUnlock 单签名解锁数据。
type SingleSigUnlock struct {
	// Signature ML-DSA-65 签名字节
	Signature []byte
	// PublicKey 对应的公钥字节
	PublicKey []byte
}

// MultiSigUnlock 多重签名解锁数据。
type MultiSigUnlock struct {
	// Signatures m 个签名字节列表
	Signatures [][]byte
	// PublicKeys m 个对应公钥字节列表
	PublicKeys [][]byte
	// Complement N-m 个未参与签名的公钥哈希列表
	Complement [][32]byte
}

// M 返回多签所需最小签名数（由 Signatures 数量推导）。
func (m *MultiSigUnlock) M() int {
	return len(m.Signatures)
}

// N 返回多签总参与者数（M + len(Complement)）。
func (m *MultiSigUnlock) N() int {
	return m.M() + len(m.Complement)
}
```

**Step 2: 运行构建**

```bash
go build ./internal/tx/...
```

---

## Task 6: 附件 ID（internal/tx/attachment.go）

**Files:**
- Create: `internal/tx/attachment.go`

**Step 1: 编写附件 ID 结构**

```go
package tx

import (
	"errors"

	"github.com/cxio/evidcoin/pkg/types"
)

// AttachmentID 附件标识结构（总长 < 256 字节）。
type AttachmentID struct {
	// TotalLen ID 总字节长
	TotalLen byte
	// MajorType 附件大类（参考 MIME）
	MajorType byte
	// MinorType 附件小类
	MinorType byte
	// Fingerprint SHA3-512 哈希指纹（固定 64 字节，对完整附件数据）
	Fingerprint types.Hash512
	// ShardCount 分片数量（0 = 无分片字段，1 = 单体有哈希，>1 = 分片树）
	ShardCount uint16
	// ShardTreeRoot 片组哈希树根（BLAKE3-256，32 字节）——条件存在
	ShardTreeRoot *types.Hash256
	// DataSize 附件大小（字节）
	DataSize int64
}

// Validate 验证 AttachmentID 的结构合法性。
func (a *AttachmentID) Validate() error {
	if len(a.Fingerprint) != types.Hash512Len {
		return errors.New("fingerprint must be 64 bytes (SHA3-512)")
	}
	if a.ShardCount == 0 && a.ShardTreeRoot != nil {
		return errors.New("shard tree root must be absent when shard count is 0")
	}
	if a.ShardCount > 0 && a.ShardTreeRoot == nil {
		return errors.New("shard tree root required when shard count > 0")
	}
	return nil
}

// Encode 将 AttachmentID 序列化为字节序列。
func (a *AttachmentID) Encode() []byte {
	buf := make([]byte, 0, 16)
	buf = append(buf, a.TotalLen)
	buf = append(buf, a.MajorType)
	buf = append(buf, a.MinorType)
	buf = append(buf, a.Fingerprint[:]...)
	sc := make([]byte, 2)
	sc[0] = byte(a.ShardCount >> 8)
	sc[1] = byte(a.ShardCount)
	buf = append(buf, sc...)
	if a.ShardTreeRoot != nil {
		buf = append(buf, a.ShardTreeRoot[:]...)
	}
	buf = append(buf, AppendVarint(nil, uint64(a.DataSize))...)
	return buf
}
```

**Step 2: 运行构建**

```bash
go build ./internal/tx/...
```

---

## Task 7: 单元测试（internal/tx/tx_test.go）

**Files:**
- Create: `internal/tx/tx_test.go`

**Step 1: 编写表驱动测试**

```go
package tx_test

import (
	"testing"

	"github.com/cxio/evidcoin/internal/tx"
	"github.com/cxio/evidcoin/pkg/types"
)

// ---- varint 测试 ----

// TestVarintRoundtrip 测试可变长整数编解码往返一致性。
func TestVarintRoundtrip(t *testing.T) {
	cases := []uint64{0, 1, 127, 128, 16383, 16384, 1<<32 - 1, 1<<63 - 1}
	for _, v := range cases {
		buf := tx.AppendVarint(nil, v)
		got, n, err := tx.ReadVarint(buf)
		if err != nil {
			t.Errorf("ReadVarint(%d) error = %v", v, err)
			continue
		}
		if got != v {
			t.Errorf("ReadVarint(%d) = %d, want %d", v, got, v)
		}
		if n != len(buf) {
			t.Errorf("ReadVarint(%d) consumed %d bytes, expected %d", v, n, len(buf))
		}
	}
}

// ---- 哈希计算测试 ----

// TestComputeTxIDDeterministic 测试 TxID 计算确定性。
func TestComputeTxIDDeterministic(t *testing.T) {
	header := &tx.TxHeader{
		Version:     1,
		Timestamp:   1700000000000,
		HashInputs:  types.Hash384{1},
		HashOutputs: types.Hash384{2},
	}
	id1 := tx.ComputeTxID(header)
	id2 := tx.ComputeTxID(header)
	if id1 != id2 {
		t.Error("ComputeTxID is not deterministic")
	}
}

// TestComputeTxIDDifferentTimestamp 测试不同时间戳产生不同 TxID。
func TestComputeTxIDDifferentTimestamp(t *testing.T) {
	h1 := &tx.TxHeader{Version: 1, Timestamp: 100}
	h2 := &tx.TxHeader{Version: 1, Timestamp: 200}
	if tx.ComputeTxID(h1) == tx.ComputeTxID(h2) {
		t.Error("different timestamps should produce different TxIDs")
	}
}

// TestComputeLeadHashDeterministic 测试首领输入哈希确定性。
func TestComputeLeadHashDeterministic(t *testing.T) {
	lead := &tx.LeadInput{Year: 2024, TxID: types.Hash384{0xaa}, OutIndex: 0}
	h1 := tx.ComputeLeadHash(lead)
	h2 := tx.ComputeLeadHash(lead)
	if h1 != h2 {
		t.Error("ComputeLeadHash is not deterministic")
	}
}

// TestComputeOutputHashEmpty 测试空输出集哈希为零。
func TestComputeOutputHashEmpty(t *testing.T) {
	h := tx.ComputeOutputHash(nil)
	if !h.IsZero() {
		t.Error("empty output hash should be zero")
	}
}

// TestComputeOutputHashDeterministic 测试输出哈希确定性。
func TestComputeOutputHashDeterministic(t *testing.T) {
	outputs := []tx.Output{
		{Serial: 0, Config: types.OutTypeCoin, Amount: 1000, Address: types.PubKeyHash{1}},
		{Serial: 1, Config: types.OutTypeCoin, Amount: 500, Address: types.PubKeyHash{2}},
	}
	h1 := tx.ComputeOutputHash(outputs)
	h2 := tx.ComputeOutputHash(outputs)
	if h1 != h2 {
		t.Error("ComputeOutputHash is not deterministic")
	}
}

// TestComputeCheckRoot 测试 CheckRoot 合并哈希。
func TestComputeCheckRoot(t *testing.T) {
	txRoot := types.Hash384{0x01}
	utxoFP := types.Hash256{0x02}
	utcoFP := types.Hash256{0x03}
	r1 := tx.ComputeCheckRoot(txRoot, utxoFP, utcoFP)
	r2 := tx.ComputeCheckRoot(txRoot, utxoFP, utcoFP)
	if r1 != r2 {
		t.Error("ComputeCheckRoot is not deterministic")
	}
	// 改变任一输入应产生不同结果
	r3 := tx.ComputeCheckRoot(types.Hash384{0x99}, utxoFP, utcoFP)
	if r1 == r3 {
		t.Error("ComputeCheckRoot collision on different inputs")
	}
}

// ---- 输出类型测试 ----

// TestOutputType 测试输出类型字段提取。
func TestOutputType(t *testing.T) {
	cases := []struct {
		config byte
		want   byte
	}{
		{types.OutTypeCoin, types.OutTypeCoin},
		{types.OutTypeCredit, types.OutTypeCredit},
		{types.OutTypeProof, types.OutTypeProof},
		{tx.OutFlagDestroy | types.OutTypeCoin, types.OutTypeCoin},
	}
	for _, tc := range cases {
		o := &tx.Output{Config: tc.config}
		if got := o.Type(); got != tc.want {
			t.Errorf("Config 0x%02x Type() = %d, want %d", tc.config, got, tc.want)
		}
	}
}

// TestOutputCanBeUTXO 测试 UTXO 资格判断。
func TestOutputCanBeUTXO(t *testing.T) {
	cases := []struct {
		name   string
		config byte
		want   bool
	}{
		{"coin normal", types.OutTypeCoin, true},
		{"coin destroy", tx.OutFlagDestroy | types.OutTypeCoin, false},
		{"credit normal", types.OutTypeCredit, false},
		{"proof normal", types.OutTypeProof, false},
		{"custom class", tx.OutFlagCustomClass | types.OutTypeCoin, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			o := &tx.Output{Config: tc.config}
			if got := o.CanBeUTXO(); got != tc.want {
				t.Errorf("CanBeUTXO() = %v, want %v", got, tc.want)
			}
		})
	}
}

// ---- 附件 ID 测试 ----

// TestAttachmentIDValidate 测试附件 ID 验证逻辑。
func TestAttachmentIDValidate(t *testing.T) {
	root := types.Hash384{0xcc}
	valid := &tx.AttachmentID{
		TotalLen:       20,
		FingerprintLen: 0, // 16 字节
		Fingerprint:    make([]byte, 16),
		ShardCount:     1,
		ShardTreeRoot:  &root,
		DataSize:       1024,
	}
	if err := valid.Validate(); err != nil {
		t.Errorf("Validate() unexpected error = %v", err)
	}

	// ShardCount=0 但有 ShardTreeRoot：应报错
	invalid := &tx.AttachmentID{
		FingerprintLen: 0,
		Fingerprint:    make([]byte, 16),
		ShardCount:     0,
		ShardTreeRoot:  &root,
	}
	if err := invalid.Validate(); err == nil {
		t.Error("expected error when ShardCount=0 but ShardTreeRoot set")
	}
}

// TestCoinFee 测试手续费计算。
func TestCoinFee(t *testing.T) {
	outputs := []tx.Output{
		{Config: types.OutTypeCoin, Amount: 300},
		{Config: types.OutTypeCoin, Amount: 200},
	}
	fee, err := tx.CoinFee(600, outputs)
	if err != nil {
		t.Fatalf("CoinFee() error = %v", err)
	}
	if fee != 100 {
		t.Errorf("CoinFee() = %d, want 100", fee)
	}
}

// TestCoinFeeNegative 测试输出超过输入时返回错误。
func TestCoinFeeNegative(t *testing.T) {
	outputs := []tx.Output{
		{Config: types.OutTypeCoin, Amount: 1000},
	}
	_, err := tx.CoinFee(500, outputs)
	if err == nil {
		t.Error("expected error for negative fee, got nil")
	}
}
```

**Step 2: 运行所有测试**

```bash
go test ./internal/tx/... -v
```
预期：所有测试 PASS。

**Step 3: 检查覆盖率**

```bash
go test ./internal/tx/... -cover
```
预期：覆盖率 ≥ 80%。

**Step 4: Commit**

```bash
git add internal/tx/
git commit -m "feat: implement transaction model with hash computation and signature structures"
```

---

## Task 8: 整体验收

**Step 1: 完整构建**

```bash
go build ./...
```

**Step 2: 全量测试**

```bash
go test ./... -cover
```

**Step 3: 代码格式与静态分析**

```bash
go fmt ./... && golangci-lint run ./internal/tx/...
```

---

## 注意事项

1. **HashOutputs 字段类型**：TxHeader 中 `HashOutputs` 字段类型为 `Hash384`（48 字节），但 `ComputeOutputHash` 返回 `Hash384`（前 32 字节存放 BLAKE3-256 结果，后 16 字节为零）。这是适配提案中"外层根使用 BLAKE3-256（32B）"的兼容方案。

2. **输出哈希树**：提案指明输出哈希树"每个输出作为叶子节点采用 SHA3-384 单独哈希（48 字节），树内部节点采用 BLAKE3-256 计算（32 字节）"。`binaryTreeHash` 实现遵循此规则。

3. **CoinbaseTx 与普通 Transaction 的区别**：Coinbase 无 Lead/Rests 输入，其哈希计算与普通交易共用 `ComputeTxID`。Coinbase 的 `Header.HashInputs` 按约定可为零值（无输入）。

4. **解锁数据不参与 TxID**：`UnlockData` 不序列化进 `TxHeader`，保证 TxID 的不可塑性（malleability resistance）。
