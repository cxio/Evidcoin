# Phase 4：UTXO/UTCO 集与指纹 实现计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现 UTXO/UTCO 双集状态管理——包括条目结构、集合增删查改、4 级层次哈希指纹（SHA3-512）、增量更新、UTCO 凭信过期索引、以及缓存层。

**Architecture:** `internal/utxo` 和 `internal/utco` 两个包，仅依赖 `pkg/types` 和 `pkg/crypto`。UTXO 管理币金输出，UTCO 管理凭信输出，二者共享相似的 4 级指纹结构但有不同的业务规则。

**Tech Stack:** Go 1.25+, pkg/types (Hash512, PubKeyHash, OutputConfig, constants), pkg/crypto (SHA3_512Sum)

---

## 前置依赖

本 Phase 假设 Phase 1 和 Phase 3 已完成，以下类型和函数已可用：

```go
// pkg/types
type Hash512 [64]byte              // SHA-512（64 字节）
type PubKeyHash [48]byte           // 公钥哈希（SHA3-384）
type OutputConfig byte             // 输出配置字节
const HashLen = 64
const PubKeyHashLen = 48
const BlocksPerYear = 87661        // 每年区块数
const OutCustomClass OutputConfig = 1 << 7   // bit7: 自定义类别
const OutDestroy OutputConfig = 1 << 5       // bit5: 销毁标记
const OutTypeCoin OutputConfig = 1           // 币金类型
const OutTypeCredit OutputConfig = 2         // 凭信类型
const OutTypeProof OutputConfig = 3          // 存证类型
const OutTypeMediator OutputConfig = 4       // 中介类型
func (c OutputConfig) Type() OutputConfig    // 提取低 4 位类型
func (c OutputConfig) IsCustom() bool        // 检查自定义标记
func (c OutputConfig) IsDestroy() bool       // 检查销毁标记
func PutVarint(buf []byte, v uint64) int     // 编码 varint
func Varint(buf []byte) (uint64, int)        // 解码 varint
func VarintSize(v uint64) int                // varint 编码长度

// pkg/crypto
func SHA3_512Sum(data []byte) types.Hash512  // SHA3-512 哈希
```

> **注意：** 如果 Phase 1/3 的具体 API 与以上描述有差异，请在实现时以 `pkg/types` 和 `pkg/crypto` 的实际导出符号为准，适当调整本计划中的调用方式。

---

## Task 1: UTXOEntry 与 OutPoint (internal/utxo/entry.go)

**Files:**
- Create: `internal/utxo/entry.go`
- Test: `internal/utxo/entry_test.go`

本 Task 实现 UTXO 条目结构 `UTXOEntry`、输出点标识 `OutPoint`、年份计算辅助函数 `YearFromHeight`、以及叶子节点哈希计算函数 `CalcInfoHash` 和 `FlagOutputsFromIndices`。

### Step 1: 写失败测试

创建 `internal/utxo/entry_test.go`：

```go
package utxo

import (
	"fmt"
	"strings"
	"testing"

	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// --- OutPoint 测试 ---

// 测试 OutPoint.String() 格式为 "hex(txid):index"
func TestOutPoint_String(t *testing.T) {
	var txID types.Hash512
	txID[0] = 0xab
	txID[1] = 0xcd

	op := OutPoint{TxID: txID, Index: 3}
	s := op.String()

	// 应该包含 "abcd" 前缀和 ":3" 后缀
	if !strings.HasPrefix(s, "abcd") {
		t.Errorf("String() should start with hex of TxID, got %q", s)
	}
	if !strings.HasSuffix(s, ":3") {
		t.Errorf("String() should end with ':3', got %q", s)
	}
}

// 测试 OutPoint.Key() 与 String() 返回相同结果
func TestOutPoint_Key(t *testing.T) {
	var txID types.Hash512
	txID[0] = 0xff
	op := OutPoint{TxID: txID, Index: 42}

	if op.Key() != op.String() {
		t.Errorf("Key() = %q, want %q (same as String())", op.Key(), op.String())
	}
}

// 测试不同 OutPoint 产生不同 String
func TestOutPoint_String_Different(t *testing.T) {
	var txID1, txID2 types.Hash512
	txID1[0] = 0x01
	txID2[0] = 0x02

	op1 := OutPoint{TxID: txID1, Index: 0}
	op2 := OutPoint{TxID: txID2, Index: 0}
	op3 := OutPoint{TxID: txID1, Index: 1}

	if op1.String() == op2.String() {
		t.Error("different TxID should produce different String()")
	}
	if op1.String() == op3.String() {
		t.Error("different Index should produce different String()")
	}
}

// --- YearFromHeight 测试 ---

func TestYearFromHeight(t *testing.T) {
	tests := []struct {
		name   string
		height uint64
		want   int
	}{
		{"genesis block", 0, GenesisYear},
		{"first block", 1, GenesisYear},
		{"last block of year 0", uint64(types.BlocksPerYear - 1), GenesisYear},
		{"first block of year 1", uint64(types.BlocksPerYear), GenesisYear + 1},
		{"middle of year 2", uint64(types.BlocksPerYear*2 + 1000), GenesisYear + 2},
		{"ten years later", uint64(types.BlocksPerYear * 10), GenesisYear + 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := YearFromHeight(tt.height)
			if got != tt.want {
				t.Errorf("YearFromHeight(%d) = %d, want %d", tt.height, got, tt.want)
			}
		})
	}
}

// --- UTXOEntry.Validate 测试 ---

func TestUTXOEntry_Validate(t *testing.T) {
	validTxID := types.Hash512{0x01}
	validAddr := types.PubKeyHash{0x02}
	validScript := []byte{0x76, 0xa9} // 模拟锁定脚本

	tests := []struct {
		name    string
		entry   *UTXOEntry
		wantErr bool
	}{
		{
			name: "valid entry",
			entry: &UTXOEntry{
				OutPoint:   OutPoint{TxID: validTxID, Index: 0},
				Amount:     1000,
				Address:    validAddr,
				Height:     100,
				IsCoinbase: false,
				LockScript: validScript,
			},
			wantErr: false,
		},
		{
			name: "zero amount",
			entry: &UTXOEntry{
				OutPoint:   OutPoint{TxID: validTxID, Index: 0},
				Amount:     0,
				Address:    validAddr,
				Height:     100,
				LockScript: validScript,
			},
			wantErr: true,
		},
		{
			name: "negative amount",
			entry: &UTXOEntry{
				OutPoint:   OutPoint{TxID: validTxID, Index: 0},
				Amount:     -100,
				Address:    validAddr,
				Height:     100,
				LockScript: validScript,
			},
			wantErr: true,
		},
		{
			name: "zero txid",
			entry: &UTXOEntry{
				OutPoint:   OutPoint{TxID: types.Hash512{}, Index: 0},
				Amount:     1000,
				Address:    validAddr,
				Height:     100,
				LockScript: validScript,
			},
			wantErr: true,
		},
		{
			name: "zero address",
			entry: &UTXOEntry{
				OutPoint:   OutPoint{TxID: validTxID, Index: 0},
				Amount:     1000,
				Address:    types.PubKeyHash{},
				Height:     100,
				LockScript: validScript,
			},
			wantErr: true,
		},
		{
			name: "nil lock script",
			entry: &UTXOEntry{
				OutPoint:   OutPoint{TxID: validTxID, Index: 0},
				Amount:     1000,
				Address:    validAddr,
				Height:     100,
				LockScript: nil,
			},
			wantErr: true,
		},
		{
			name: "empty lock script",
			entry: &UTXOEntry{
				OutPoint:   OutPoint{TxID: validTxID, Index: 0},
				Amount:     1000,
				Address:    validAddr,
				Height:     100,
				LockScript: []byte{},
			},
			wantErr: true,
		},
		{
			name: "coinbase entry valid",
			entry: &UTXOEntry{
				OutPoint:   OutPoint{TxID: validTxID, Index: 0},
				Amount:     5000000000,
				Address:    validAddr,
				Height:     0,
				IsCoinbase: true,
				LockScript: validScript,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.entry.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// --- FlagOutputsFromIndices 测试 ---

func TestFlagOutputsFromIndices(t *testing.T) {
	tests := []struct {
		name    string
		indices []uint16
		want    []byte
	}{
		{
			name:    "empty indices",
			indices: nil,
			want:    []byte{},
		},
		{
			name:    "single index 0",
			indices: []uint16{0},
			want:    []byte{0x01}, // bit 0 设置
		},
		{
			name:    "single index 1",
			indices: []uint16{1},
			want:    []byte{0x02}, // bit 1 设置
		},
		{
			name:    "single index 7",
			indices: []uint16{7},
			want:    []byte{0x80}, // bit 7 设置
		},
		{
			name:    "index 8 needs second byte",
			indices: []uint16{8},
			want:    []byte{0x00, 0x01}, // bit 0 of byte 1
		},
		{
			name:    "indices 0 and 7",
			indices: []uint16{0, 7},
			want:    []byte{0x81}, // bit 0 和 bit 7
		},
		{
			name:    "indices 0, 1, 8",
			indices: []uint16{0, 1, 8},
			want:    []byte{0x03, 0x01}, // byte0: bit0+bit1, byte1: bit0
		},
		{
			name:    "index 15",
			indices: []uint16{15},
			want:    []byte{0x00, 0x80}, // bit 7 of byte 1
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FlagOutputsFromIndices(tt.indices)
			if len(got) != len(tt.want) {
				t.Errorf("FlagOutputsFromIndices() length = %d, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("FlagOutputsFromIndices()[%d] = 0x%02x, want 0x%02x", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// --- CalcInfoHash 测试 ---

// 测试 CalcInfoHash 确定性
func TestCalcInfoHash_Deterministic(t *testing.T) {
	txID := types.Hash512{0xaa, 0xbb}
	flags := []byte{0x03}

	h1 := CalcInfoHash(txID, flags)
	h2 := CalcInfoHash(txID, flags)

	if h1 != h2 {
		t.Error("CalcInfoHash() should be deterministic")
	}
	if h1.IsZero() {
		t.Error("CalcInfoHash() should not return zero hash")
	}
}

// 测试不同输入产生不同哈希
func TestCalcInfoHash_DifferentInputs(t *testing.T) {
	txID1 := types.Hash512{0x01}
	txID2 := types.Hash512{0x02}
	flags1 := []byte{0x01}
	flags2 := []byte{0x03}

	h1 := CalcInfoHash(txID1, flags1)
	h2 := CalcInfoHash(txID2, flags1)
	h3 := CalcInfoHash(txID1, flags2)

	if h1 == h2 {
		t.Error("different TxID should produce different InfoHash")
	}
	if h1 == h3 {
		t.Error("different flagOutputs should produce different InfoHash")
	}
}

// 测试 CalcInfoHash 手动验证
func TestCalcInfoHash_Manual(t *testing.T) {
	var txID types.Hash512
	txID[0] = 0x42

	flags := []byte{0x07} // 输出 0, 1, 2

	// 手动构建: TxID(64) || varint(len(flags))(1) || flags(1)
	buf := make([]byte, types.HashLen+1+len(flags))
	copy(buf[:types.HashLen], txID[:])
	// varint(1) = 0x01
	buf[types.HashLen] = 0x01
	copy(buf[types.HashLen+1:], flags)

	expected := crypto.SHA3_512Sum(buf)
	got := CalcInfoHash(txID, flags)

	if got != expected {
		t.Errorf("CalcInfoHash() = %x, want %x", got[:8], expected[:8])
	}
}

// 测试空 flagOutputs
func TestCalcInfoHash_EmptyFlags(t *testing.T) {
	txID := types.Hash512{0x01}

	h := CalcInfoHash(txID, []byte{})
	if h.IsZero() {
		t.Error("CalcInfoHash() with empty flags should still produce non-zero hash")
	}
}

// --- 综合 OutPoint 构造测试 ---

func TestOutPoint_ZeroValue(t *testing.T) {
	var op OutPoint
	s := op.String()
	// 零值的 OutPoint 应该产生全零 TxID 的字符串
	expected := fmt.Sprintf("%s:0", types.Hash512{}.String())
	if s != expected {
		t.Errorf("zero OutPoint.String() = %q, want %q", s, expected)
	}
}
```

### Step 2: 运行测试验证失败

```bash
go test -v ./internal/utxo/ -run "TestOutPoint|TestYearFromHeight|TestUTXOEntry|TestFlagOutputsFromIndices|TestCalcInfoHash"
```

预期输出：编译失败，`OutPoint`、`UTXOEntry`、`YearFromHeight`、`CalcInfoHash`、`FlagOutputsFromIndices`、`GenesisYear` 等未定义。

### Step 3: 写最小实现

创建 `internal/utxo/entry.go`：

```go
package utxo

import (
	"errors"
	"fmt"

	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// GenesisYear 创世年份，即区块高度 0 对应的年份。
const GenesisYear = 2026

// OutPoint 唯一标识一个交易输出。
// TxID 为产出该输出的交易哈希，Index 为该输出在交易输出列表中的序号。
type OutPoint struct {
	TxID  types.Hash512 // 交易 ID
	Index uint16        // 输出序号
}

// String 返回 OutPoint 的字符串表示，格式为 "hex(txid):index"。
func (o OutPoint) String() string {
	return fmt.Sprintf("%s:%d", o.TxID.String(), o.Index)
}

// Key 返回用作 map 键的字符串，与 String() 相同。
func (o OutPoint) Key() string {
	return o.String()
}

// UTXOEntry 表示 UTXO 集中的一个未花费输出条目。
type UTXOEntry struct {
	OutPoint                     // 嵌入输出点
	Amount     int64             // 金额（单位：最小币金）
	Address    types.PubKeyHash  // 接收者公钥哈希
	Height     uint64            // 创建时区块高度
	IsCoinbase bool              // 是否 Coinbase 产出
	LockScript []byte            // 锁定脚本
}

// UTXO 条目验证错误
var (
	ErrZeroTxID      = errors.New("utxo entry txid is zero")
	ErrInvalidAmount = errors.New("utxo entry amount must be positive")
	ErrZeroAddress   = errors.New("utxo entry address is zero")
	ErrEmptyScript   = errors.New("utxo entry lock script is empty")
)

// Validate 对 UTXO 条目执行基本字段验证。
func (e *UTXOEntry) Validate() error {
	if e.TxID.IsZero() {
		return ErrZeroTxID
	}
	if e.Amount <= 0 {
		return ErrInvalidAmount
	}
	if e.Address.IsZero() {
		return ErrZeroAddress
	}
	if len(e.LockScript) == 0 {
		return ErrEmptyScript
	}
	return nil
}

// YearFromHeight 从区块高度计算对应的年份。
// 年份 = GenesisYear + height / BlocksPerYear
func YearFromHeight(height uint64) int {
	return GenesisYear + int(height/types.BlocksPerYear)
}

// FlagOutputsFromIndices 根据输出序号列表构造标记位字节序列。
// 每个 bit 表示对应序号的输出是否在 UTXO 集中：
// indices[i] 对应第 indices[i]/8 个字节的第 indices[i]%8 位。
func FlagOutputsFromIndices(indices []uint16) []byte {
	if len(indices) == 0 {
		return []byte{}
	}

	// 找出最大序号，确定所需字节数
	var maxIdx uint16
	for _, idx := range indices {
		if idx > maxIdx {
			maxIdx = idx
		}
	}

	// 分配所需字节数
	size := int(maxIdx/8) + 1
	flags := make([]byte, size)

	// 设置对应位
	for _, idx := range indices {
		bytePos := idx / 8
		bitPos := idx % 8
		flags[bytePos] |= 1 << bitPos
	}

	return flags
}

// CalcInfoHash 根据 TxID 和输出标记位序列计算叶子节点哈希。
// infoHash = SHA3-512( TxID || varint(len(flagOutputs)) || flagOutputs )
func CalcInfoHash(txID types.Hash512, flagOutputs []byte) types.Hash512 {
	flagLen := uint64(len(flagOutputs))
	varintSize := types.VarintSize(flagLen)

	// 构建缓冲区: TxID(64) + varint(len) + flagOutputs
	buf := make([]byte, types.HashLen+varintSize+len(flagOutputs))
	offset := 0

	// 写入 TxID
	copy(buf[offset:offset+types.HashLen], txID[:])
	offset += types.HashLen

	// 写入 varint 编码的 flagOutputs 长度
	types.PutVarint(buf[offset:], flagLen)
	offset += varintSize

	// 写入 flagOutputs
	copy(buf[offset:], flagOutputs)

	return crypto.SHA3_512Sum(buf)
}
```

### Step 4: 运行测试验证通过

```bash
go test -v ./internal/utxo/ -run "TestOutPoint|TestYearFromHeight|TestUTXOEntry|TestFlagOutputsFromIndices|TestCalcInfoHash"
```

预期输出：全部 PASS。

### Step 5: 提交

```bash
git add internal/utxo/entry.go internal/utxo/entry_test.go
git commit -m "feat(utxo): add UTXOEntry, OutPoint, YearFromHeight, CalcInfoHash and FlagOutputsFromIndices"
```

---

## Task 2: UTXO 集管理 (internal/utxo/set.go)

**Files:**
- Create: `internal/utxo/set.go`
- Test: `internal/utxo/set_test.go`

本 Task 实现 `UTXOSet` 结构体，提供 UTXO 集合的增删查改操作，以及输出类型过滤逻辑 `ShouldInclude`。

### Step 1: 写失败测试

创建 `internal/utxo/set_test.go`：

```go
package utxo

import (
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// 辅助函数：创建测试用 UTXOEntry
func makeTestEntry(txByte byte, index uint16, amount int64, height uint64) *UTXOEntry {
	var txID types.Hash512
	txID[0] = txByte
	var addr types.PubKeyHash
	addr[0] = txByte + 0x10

	return &UTXOEntry{
		OutPoint:   OutPoint{TxID: txID, Index: index},
		Amount:     amount,
		Address:    addr,
		Height:     height,
		IsCoinbase: false,
		LockScript: []byte{0x76, 0xa9},
	}
}

// --- NewUTXOSet 测试 ---

func TestNewUTXOSet(t *testing.T) {
	set := NewUTXOSet()
	if set == nil {
		t.Fatal("NewUTXOSet() returned nil")
	}
	if set.Count() != 0 {
		t.Errorf("NewUTXOSet().Count() = %d, want 0", set.Count())
	}
}

// --- Add 测试 ---

func TestUTXOSet_Add(t *testing.T) {
	set := NewUTXOSet()
	entry := makeTestEntry(0x01, 0, 1000, 100)

	err := set.Add(entry)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if set.Count() != 1 {
		t.Errorf("Count() = %d, want 1", set.Count())
	}
}

// 测试重复添加应返回错误
func TestUTXOSet_Add_Duplicate(t *testing.T) {
	set := NewUTXOSet()
	entry := makeTestEntry(0x01, 0, 1000, 100)

	_ = set.Add(entry)
	err := set.Add(entry)
	if err == nil {
		t.Error("Add() duplicate should return error")
	}
}

// 测试添加同一交易的不同输出序号
func TestUTXOSet_Add_SameTxDifferentIndex(t *testing.T) {
	set := NewUTXOSet()
	entry1 := makeTestEntry(0x01, 0, 1000, 100)
	entry2 := makeTestEntry(0x01, 1, 2000, 100)

	if err := set.Add(entry1); err != nil {
		t.Fatalf("Add() entry1 error = %v", err)
	}
	if err := set.Add(entry2); err != nil {
		t.Fatalf("Add() entry2 error = %v", err)
	}
	if set.Count() != 2 {
		t.Errorf("Count() = %d, want 2", set.Count())
	}
}

// --- Get 测试 ---

func TestUTXOSet_Get(t *testing.T) {
	set := NewUTXOSet()
	entry := makeTestEntry(0x01, 0, 1000, 100)
	_ = set.Add(entry)

	got, ok := set.Get(entry.OutPoint)
	if !ok {
		t.Fatal("Get() should find the entry")
	}
	if got.Amount != 1000 {
		t.Errorf("Get().Amount = %d, want 1000", got.Amount)
	}
}

// 测试查询不存在的条目
func TestUTXOSet_Get_NotFound(t *testing.T) {
	set := NewUTXOSet()
	op := OutPoint{TxID: types.Hash512{0xff}, Index: 0}

	_, ok := set.Get(op)
	if ok {
		t.Error("Get() should return false for non-existent entry")
	}
}

// --- Has 测试 ---

func TestUTXOSet_Has(t *testing.T) {
	set := NewUTXOSet()
	entry := makeTestEntry(0x01, 0, 1000, 100)
	_ = set.Add(entry)

	if !set.Has(entry.OutPoint) {
		t.Error("Has() should return true for existing entry")
	}

	op := OutPoint{TxID: types.Hash512{0xff}, Index: 0}
	if set.Has(op) {
		t.Error("Has() should return false for non-existent entry")
	}
}

// --- Remove 测试 ---

func TestUTXOSet_Remove(t *testing.T) {
	set := NewUTXOSet()
	entry := makeTestEntry(0x01, 0, 1000, 100)
	_ = set.Add(entry)

	removed, err := set.Remove(entry.OutPoint)
	if err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if removed.Amount != 1000 {
		t.Errorf("Remove() returned entry with Amount = %d, want 1000", removed.Amount)
	}
	if set.Count() != 0 {
		t.Errorf("Count() after Remove = %d, want 0", set.Count())
	}
}

// 测试移除不存在的条目
func TestUTXOSet_Remove_NotFound(t *testing.T) {
	set := NewUTXOSet()
	op := OutPoint{TxID: types.Hash512{0xff}, Index: 0}

	_, err := set.Remove(op)
	if err == nil {
		t.Error("Remove() should return error for non-existent entry")
	}
}

// 测试移除同一交易的某个输出后，该交易的其他输出仍在
func TestUTXOSet_Remove_PartialTx(t *testing.T) {
	set := NewUTXOSet()
	entry1 := makeTestEntry(0x01, 0, 1000, 100)
	entry2 := makeTestEntry(0x01, 1, 2000, 100)
	_ = set.Add(entry1)
	_ = set.Add(entry2)

	_, err := set.Remove(entry1.OutPoint)
	if err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	// entry2 应仍在集合中
	if !set.Has(entry2.OutPoint) {
		t.Error("Remove() should only remove the specified entry")
	}
	if set.Count() != 1 {
		t.Errorf("Count() after partial Remove = %d, want 1", set.Count())
	}
}

// --- GetByTxID 测试 ---

func TestUTXOSet_GetByTxID(t *testing.T) {
	set := NewUTXOSet()
	entry1 := makeTestEntry(0x01, 0, 1000, 100)
	entry2 := makeTestEntry(0x01, 1, 2000, 100)
	entry3 := makeTestEntry(0x02, 0, 3000, 200)
	_ = set.Add(entry1)
	_ = set.Add(entry2)
	_ = set.Add(entry3)

	entries := set.GetByTxID(entry1.TxID)
	if len(entries) != 2 {
		t.Errorf("GetByTxID() returned %d entries, want 2", len(entries))
	}

	// 查询不存在的 TxID
	entries = set.GetByTxID(types.Hash512{0xff})
	if len(entries) != 0 {
		t.Errorf("GetByTxID() for non-existent TxID returned %d entries, want 0", len(entries))
	}
}

// 测试移除所有输出后 GetByTxID 返回空
func TestUTXOSet_GetByTxID_AfterRemoveAll(t *testing.T) {
	set := NewUTXOSet()
	entry1 := makeTestEntry(0x01, 0, 1000, 100)
	entry2 := makeTestEntry(0x01, 1, 2000, 100)
	_ = set.Add(entry1)
	_ = set.Add(entry2)

	_, _ = set.Remove(entry1.OutPoint)
	_, _ = set.Remove(entry2.OutPoint)

	entries := set.GetByTxID(entry1.TxID)
	if len(entries) != 0 {
		t.Errorf("GetByTxID() after removing all returned %d entries, want 0", len(entries))
	}
}

// --- ShouldInclude 测试 ---

func TestShouldInclude(t *testing.T) {
	tests := []struct {
		name   string
		config types.OutputConfig
		want   bool
	}{
		{
			name:   "Coin type",
			config: types.OutTypeCoin,
			want:   true,
		},
		{
			name:   "Credit type",
			config: types.OutTypeCredit,
			want:   false, // 凭信由 UTCO 管理
		},
		{
			name:   "Proof type",
			config: types.OutTypeProof,
			want:   false,
		},
		{
			name:   "Mediator type",
			config: types.OutTypeMediator,
			want:   false,
		},
		{
			name:   "Custom class",
			config: types.OutCustomClass | types.OutTypeCoin,
			want:   false,
		},
		{
			name:   "Destroy flag",
			config: types.OutDestroy | types.OutTypeCoin,
			want:   false,
		},
		{
			name:   "Destroy + Coin",
			config: types.OutDestroy | types.OutTypeCoin,
			want:   false,
		},
		{
			name:   "zero config",
			config: 0,
			want:   false, // 类型 0 无效
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldInclude(tt.config)
			if got != tt.want {
				t.Errorf("ShouldInclude(%08b) = %v, want %v", tt.config, got, tt.want)
			}
		})
	}
}

// --- 综合测试：大量操作 ---

func TestUTXOSet_BulkOperations(t *testing.T) {
	set := NewUTXOSet()

	// 添加 100 个条目
	for i := 0; i < 100; i++ {
		entry := makeTestEntry(byte(i), 0, int64(i+1)*100, uint64(i))
		if err := set.Add(entry); err != nil {
			t.Fatalf("Add() entry %d error = %v", i, err)
		}
	}

	if set.Count() != 100 {
		t.Errorf("Count() = %d, want 100", set.Count())
	}

	// 移除前 50 个
	for i := 0; i < 50; i++ {
		var txID types.Hash512
		txID[0] = byte(i)
		op := OutPoint{TxID: txID, Index: 0}
		if _, err := set.Remove(op); err != nil {
			t.Fatalf("Remove() entry %d error = %v", i, err)
		}
	}

	if set.Count() != 50 {
		t.Errorf("Count() after bulk remove = %d, want 50", set.Count())
	}

	// 验证后 50 个仍在
	for i := 50; i < 100; i++ {
		var txID types.Hash512
		txID[0] = byte(i)
		op := OutPoint{TxID: txID, Index: 0}
		if !set.Has(op) {
			t.Errorf("entry %d should still exist", i)
		}
	}
}
```

### Step 2: 运行测试验证失败

```bash
go test -v ./internal/utxo/ -run "TestNewUTXOSet|TestUTXOSet|TestShouldInclude"
```

预期输出：编译失败，`UTXOSet`、`NewUTXOSet`、`ShouldInclude` 等未定义。

### Step 3: 写最小实现

创建 `internal/utxo/set.go`：

```go
package utxo

import (
	"errors"
	"fmt"

	"github.com/cxio/evidcoin/pkg/types"
)

// UTXOSet UTXO 集合，存储所有未花费的交易输出。
// entries 为按 OutPoint 索引的主映射，txOutputs 为按 TxID 分组的辅助索引。
type UTXOSet struct {
	entries   map[OutPoint]*UTXOEntry                   // 主索引：OutPoint -> Entry
	txOutputs map[types.Hash512]map[uint16]*UTXOEntry   // 辅助索引：TxID -> index -> Entry
}

// UTXO 集合操作错误
var (
	ErrDuplicateEntry = errors.New("utxo entry already exists")
	ErrEntryNotFound  = errors.New("utxo entry not found")
)

// NewUTXOSet 创建一个空的 UTXO 集合。
func NewUTXOSet() *UTXOSet {
	return &UTXOSet{
		entries:   make(map[OutPoint]*UTXOEntry),
		txOutputs: make(map[types.Hash512]map[uint16]*UTXOEntry),
	}
}

// Add 向集合中添加一个 UTXO 条目。
// 如果该 OutPoint 已存在，返回错误。
func (s *UTXOSet) Add(entry *UTXOEntry) error {
	if _, exists := s.entries[entry.OutPoint]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicateEntry, entry.OutPoint.String())
	}

	s.entries[entry.OutPoint] = entry

	// 更新 TxID 辅助索引
	txMap, ok := s.txOutputs[entry.TxID]
	if !ok {
		txMap = make(map[uint16]*UTXOEntry)
		s.txOutputs[entry.TxID] = txMap
	}
	txMap[entry.Index] = entry

	return nil
}

// Remove 从集合中移除指定 OutPoint 的条目并返回。
// 如果不存在，返回错误。
func (s *UTXOSet) Remove(outpoint OutPoint) (*UTXOEntry, error) {
	entry, exists := s.entries[outpoint]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrEntryNotFound, outpoint.String())
	}

	delete(s.entries, outpoint)

	// 清理 TxID 辅助索引
	if txMap, ok := s.txOutputs[outpoint.TxID]; ok {
		delete(txMap, outpoint.Index)
		if len(txMap) == 0 {
			delete(s.txOutputs, outpoint.TxID)
		}
	}

	return entry, nil
}

// Get 查询指定 OutPoint 的 UTXO 条目。
// 返回条目和是否存在。
func (s *UTXOSet) Get(outpoint OutPoint) (*UTXOEntry, bool) {
	entry, ok := s.entries[outpoint]
	return entry, ok
}

// Has 判断指定 OutPoint 是否存在于集合中。
func (s *UTXOSet) Has(outpoint OutPoint) bool {
	_, ok := s.entries[outpoint]
	return ok
}

// Count 返回集合中的条目总数。
func (s *UTXOSet) Count() int {
	return len(s.entries)
}

// GetByTxID 获取某个交易的所有未花费输出。
// 返回该交易剩余的 UTXO 条目切片，如果没有则返回空切片。
func (s *UTXOSet) GetByTxID(txID types.Hash512) []*UTXOEntry {
	txMap, ok := s.txOutputs[txID]
	if !ok {
		return nil
	}

	entries := make([]*UTXOEntry, 0, len(txMap))
	for _, entry := range txMap {
		entries = append(entries, entry)
	}
	return entries
}

// ShouldInclude 判断给定的输出配置是否应纳入 UTXO 集。
// UTXO 集仅管理币金（Coin）类型的输出。
// 排除以下类型：
//   - Destroy 标记的输出（已销毁）
//   - Proof 类型（存证，静态证明不可花费）
//   - Mediator 类型（中介节点输出）
//   - Custom 类别（自定义输出）
//   - Credit 类型（凭信，由 UTCO 管理）
//   - 类型为 0 的无效输出
func ShouldInclude(config types.OutputConfig) bool {
	// 排除销毁标记
	if config.IsDestroy() {
		return false
	}
	// 排除自定义类别
	if config.IsCustom() {
		return false
	}
	// 仅接纳 Coin 类型
	return config.Type() == types.OutTypeCoin
}
```

### Step 4: 运行测试验证通过

```bash
go test -v ./internal/utxo/ -run "TestNewUTXOSet|TestUTXOSet|TestShouldInclude"
```

预期输出：全部 PASS。

### Step 5: 提交

```bash
git add internal/utxo/set.go internal/utxo/set_test.go
git commit -m "feat(utxo): add UTXOSet with Add/Remove/Get/Has/GetByTxID and ShouldInclude filter"
```

---

## Task 3: 4 级层次哈希指纹 (internal/utxo/fingerprint.go)

**Files:**
- Create: `internal/utxo/fingerprint.go`
- Test: `internal/utxo/fingerprint_test.go`

本 Task 实现 UTXO 集的 4 级层次哈希指纹树。这是整个 UTXO/UTCO 系统的核心，用于快速验证两个节点的 UTXO 集是否一致。全部使用 **SHA3-512** 算法。

**4 级层次结构**：

```
层级        键值              计算方式
─────────────────────────────────────────────────────────────────
Root        (全局)            SHA3-512( YearHash_1 || YearHash_2 || ... )    按年份升序
Year        year              SHA3-512( Tx8Hash_0 || Tx8Hash_1 || ... )      按 TxID[7] 值排序（0-255）
Tx8         TxID[7]           SHA3-512( Tx13Hash_0 || ... )                   按 TxID[12] 值排序（0-255）
Tx13        TxID[12]          SHA3-512( Tx18Hash_0 || ... )                   按 TxID[17] 值排序（0-255）
Tx18        TxID[17]          SHA3-512( InfoHash_1 || InfoHash_2 || ... )     按 TxID 字典序排列
InfoHash    (叶子)            SHA3-512( TxID || varint(len(flags)) || flags )
```

> **分层键值说明**：使用 TxID 的第 7、12、17 字节作为分层键，将哈希空间均匀切分为 256 个桶。这些字节位置的选择使得不同层级的键值相对独立。

### Step 1: 写失败测试

创建 `internal/utxo/fingerprint_test.go`：

```go
package utxo

import (
	"bytes"
	"sort"
	"testing"

	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// 辅助函数：创建指定特征字节的 TxID
func makeTxID(b0 byte, b7 byte, b12 byte, b17 byte) types.Hash512 {
	var txID types.Hash512
	txID[0] = b0
	txID[7] = b7
	txID[12] = b12
	txID[17] = b17
	return txID
}

// --- 空指纹测试 ---

func TestFingerprint_Empty(t *testing.T) {
	fp := NewFingerprint()
	root := fp.RootHash()

	if !root.IsZero() {
		t.Error("empty Fingerprint RootHash() should be zero")
	}
}

// --- 单个叶子测试 ---

func TestFingerprint_SingleLeaf(t *testing.T) {
	fp := NewFingerprint()
	txID := makeTxID(0x01, 0xAA, 0xBB, 0xCC)
	year := 2026

	fp.AddLeaf(txID, year, 0)

	root := fp.RootHash()
	if root.IsZero() {
		t.Error("Fingerprint with one leaf should have non-zero RootHash()")
	}
}

// --- 添加后移除应恢复零值 ---

func TestFingerprint_AddThenRemove_ReturnsZero(t *testing.T) {
	fp := NewFingerprint()
	txID := makeTxID(0x01, 0xAA, 0xBB, 0xCC)
	year := 2026

	fp.AddLeaf(txID, year, 0)
	fp.RemoveLeaf(txID, year, 0)

	root := fp.RootHash()
	if !root.IsZero() {
		t.Errorf("Fingerprint after add+remove should have zero RootHash(), got %x", root[:8])
	}
}

// --- 同一交易多个输出合并 ---

func TestFingerprint_SameTxMultipleOutputs(t *testing.T) {
	fp := NewFingerprint()
	txID := makeTxID(0x01, 0xAA, 0xBB, 0xCC)
	year := 2026

	fp.AddLeaf(txID, year, 0)
	fp.AddLeaf(txID, year, 1)
	fp.AddLeaf(txID, year, 3)

	root := fp.RootHash()
	if root.IsZero() {
		t.Error("Fingerprint with multiple outputs should have non-zero RootHash()")
	}

	// 手动验证：标记位应该是 0b00001011 = 0x0B（bit 0, 1, 3）
	yearNode, ok := fp.YearNodes[year]
	if !ok {
		t.Fatal("year node not found")
	}
	tx8 := yearNode.Tx8Nodes[0xAA]
	if tx8 == nil {
		t.Fatal("tx8 node not found")
	}
	tx13 := tx8.Tx13Nodes[0xBB]
	if tx13 == nil {
		t.Fatal("tx13 node not found")
	}
	tx18 := tx13.Tx18Nodes[0xCC]
	if tx18 == nil {
		t.Fatal("tx18 node not found")
	}
	leaf, ok := tx18.Leaves[txID]
	if !ok {
		t.Fatal("leaf not found")
	}

	expectedFlags := []byte{0x0B} // bit 0, 1, 3
	if !bytes.Equal(leaf.FlagOutputs, expectedFlags) {
		t.Errorf("FlagOutputs = %x, want %x", leaf.FlagOutputs, expectedFlags)
	}
}

// --- 移除同一交易的部分输出 ---

func TestFingerprint_RemovePartialOutputs(t *testing.T) {
	fp := NewFingerprint()
	txID := makeTxID(0x01, 0xAA, 0xBB, 0xCC)
	year := 2026

	fp.AddLeaf(txID, year, 0)
	fp.AddLeaf(txID, year, 1)
	rootBefore := fp.RootHash()

	fp.RemoveLeaf(txID, year, 0)
	rootAfter := fp.RootHash()

	// 移除部分输出后根哈希应改变但不为零
	if rootAfter.IsZero() {
		t.Error("partial remove should not make root zero")
	}
	if rootBefore == rootAfter {
		t.Error("partial remove should change root hash")
	}
}

// --- 两个不同交易的排序测试 ---

func TestFingerprint_TwoLeaves_DeterministicOrder(t *testing.T) {
	// 两个不同 TxID 落在同一个 Tx18 桶中（相同的 byte 7, 12, 17）
	txID1 := makeTxID(0x01, 0xAA, 0xBB, 0xCC)
	txID2 := makeTxID(0x02, 0xAA, 0xBB, 0xCC)
	year := 2026

	// 顺序 1: 先加 txID1 再加 txID2
	fp1 := NewFingerprint()
	fp1.AddLeaf(txID1, year, 0)
	fp1.AddLeaf(txID2, year, 0)

	// 顺序 2: 先加 txID2 再加 txID1
	fp2 := NewFingerprint()
	fp2.AddLeaf(txID2, year, 0)
	fp2.AddLeaf(txID1, year, 0)

	root1 := fp1.RootHash()
	root2 := fp2.RootHash()

	if root1 != root2 {
		t.Errorf("insertion order should not affect root hash\nroot1 = %x\nroot2 = %x", root1[:8], root2[:8])
	}
}

// --- 不同年份的叶子 ---

func TestFingerprint_DifferentYears(t *testing.T) {
	fp := NewFingerprint()
	txID1 := makeTxID(0x01, 0xAA, 0xBB, 0xCC)
	txID2 := makeTxID(0x02, 0xDD, 0xEE, 0xFF)

	fp.AddLeaf(txID1, 2026, 0)
	fp.AddLeaf(txID2, 2027, 0)

	root := fp.RootHash()
	if root.IsZero() {
		t.Error("multi-year fingerprint should have non-zero root")
	}

	// 移除 2026 年的叶子
	fp.RemoveLeaf(txID1, 2026, 0)
	root2 := fp.RootHash()
	if root2.IsZero() {
		t.Error("removing one year's leaf should leave root non-zero")
	}
	if root == root2 {
		t.Error("root should change after removing a leaf")
	}
}

// --- 增量更新 vs 全量重算结果一致 ---

func TestFingerprint_IncrementalVsFullRecalc(t *testing.T) {
	fp := NewFingerprint()

	txIDs := []types.Hash512{
		makeTxID(0x01, 0x10, 0x20, 0x30),
		makeTxID(0x02, 0x10, 0x20, 0x31), // 同 tx8, tx13, 不同 tx18
		makeTxID(0x03, 0x10, 0x21, 0x30), // 同 tx8, 不同 tx13
		makeTxID(0x04, 0x11, 0x20, 0x30), // 不同 tx8
		makeTxID(0x05, 0x10, 0x20, 0x30), // 与 txIDs[0] 同桶
	}

	for _, txID := range txIDs {
		fp.AddLeaf(txID, 2026, 0)
	}

	// 获取增量计算的根哈希
	incrementalRoot := fp.RootHash()

	// 强制全量重算
	fp.Recalculate()
	fullRoot := fp.RootHash()

	if incrementalRoot != fullRoot {
		t.Errorf("incremental root != full recalc root\nincremental = %x\nfull        = %x",
			incrementalRoot[:8], fullRoot[:8])
	}
}

// --- UpdateLeaf 直接更新叶子 ---

func TestFingerprint_UpdateLeaf(t *testing.T) {
	fp := NewFingerprint()
	txID := makeTxID(0x01, 0xAA, 0xBB, 0xCC)
	year := 2026

	fp.AddLeaf(txID, year, 0)
	fp.AddLeaf(txID, year, 1)
	root1 := fp.RootHash()

	// 直接更新标记位为仅包含 output 2
	fp.UpdateLeaf(txID, year, FlagOutputsFromIndices([]uint16{2}))
	root2 := fp.RootHash()

	if root1 == root2 {
		t.Error("UpdateLeaf should change root hash")
	}
	if root2.IsZero() {
		t.Error("UpdateLeaf should produce non-zero root")
	}
}

// --- 手动验证单叶子根哈希计算 ---

func TestFingerprint_SingleLeaf_ManualVerify(t *testing.T) {
	fp := NewFingerprint()
	txID := makeTxID(0x42, 0x10, 0x20, 0x30)
	year := 2026

	fp.AddLeaf(txID, year, 0)

	// 手动计算预期的根哈希：
	// 1. infoHash = CalcInfoHash(txID, FlagOutputsFromIndices([]uint16{0}))
	flags := FlagOutputsFromIndices([]uint16{0})
	infoHash := CalcInfoHash(txID, flags)

	// 2. tx18Hash = SHA3-512(infoHash)（只有一个叶子）
	tx18Hash := crypto.SHA3_512Sum(infoHash[:])

	// 3. tx13Hash = SHA3-512(tx18Hash)（只有 tx18[0x30] 非空）
	tx13Hash := crypto.SHA3_512Sum(tx18Hash[:])

	// 4. tx8Hash = SHA3-512(tx13Hash)（只有 tx13[0x20] 非空）
	tx8Hash := crypto.SHA3_512Sum(tx13Hash[:])

	// 5. yearHash = SHA3-512(tx8Hash)（只有 tx8[0x10] 非空）
	yearHash := crypto.SHA3_512Sum(tx8Hash[:])

	// 6. rootHash = SHA3-512(yearHash)（只有 year 2026）
	expectedRoot := crypto.SHA3_512Sum(yearHash[:])

	got := fp.RootHash()
	if got != expectedRoot {
		t.Errorf("root hash mismatch\ngot      = %x\nexpected = %x", got[:8], expectedRoot[:8])
	}
}

// --- Recalculate 后脏标记应被清除 ---

func TestFingerprint_Recalculate_ClearsDirty(t *testing.T) {
	fp := NewFingerprint()
	txID := makeTxID(0x01, 0xAA, 0xBB, 0xCC)
	fp.AddLeaf(txID, 2026, 0)

	// 全量重算
	fp.Recalculate()

	// 再次获取根哈希不应触发重新计算（结果应一致）
	root1 := fp.RootHash()
	root2 := fp.RootHash()
	if root1 != root2 {
		t.Error("consecutive RootHash() calls should return same result")
	}
}

// --- 多年份排序正确性 ---

func TestFingerprint_YearOrdering(t *testing.T) {
	fp := NewFingerprint()

	// 逆序添加年份
	txID1 := makeTxID(0x01, 0x10, 0x20, 0x30)
	txID2 := makeTxID(0x02, 0x10, 0x20, 0x30)
	txID3 := makeTxID(0x03, 0x10, 0x20, 0x30)

	fp.AddLeaf(txID3, 2028, 0) // 后年
	fp.AddLeaf(txID1, 2026, 0) // 创世年
	fp.AddLeaf(txID2, 2027, 0) // 明年

	root1 := fp.RootHash()

	// 正序添加年份应产生相同根哈希
	fp2 := NewFingerprint()
	fp2.AddLeaf(txID1, 2026, 0)
	fp2.AddLeaf(txID2, 2027, 0)
	fp2.AddLeaf(txID3, 2028, 0)

	root2 := fp2.RootHash()

	if root1 != root2 {
		t.Errorf("year ordering should not depend on insertion order\nroot1 = %x\nroot2 = %x",
			root1[:8], root2[:8])
	}
}

// --- 大量叶子压力测试 ---

func TestFingerprint_ManyLeaves(t *testing.T) {
	fp := NewFingerprint()

	// 添加 256 个不同的交易
	for i := 0; i < 256; i++ {
		var txID types.Hash512
		txID[0] = byte(i)
		txID[7] = byte(i)     // 分散到不同的 Tx8 桶
		txID[12] = byte(i)    // 分散到不同的 Tx13 桶
		txID[17] = byte(i)    // 分散到不同的 Tx18 桶
		fp.AddLeaf(txID, 2026, 0)
	}

	root := fp.RootHash()
	if root.IsZero() {
		t.Error("fingerprint with 256 leaves should have non-zero root")
	}

	// 全量重算应一致
	fp.Recalculate()
	root2 := fp.RootHash()
	if root != root2 {
		t.Error("incremental vs full recalc mismatch for 256 leaves")
	}
}

// --- Tx18 内叶子按 TxID 字典序排列 ---

func TestFingerprint_Tx18LeavesOrder(t *testing.T) {
	// 创建多个 TxID 落在同一 Tx18 桶中的叶子
	txIDs := make([]types.Hash512, 5)
	for i := range txIDs {
		txIDs[i] = makeTxID(byte(i*37+13), 0xAA, 0xBB, 0xCC) // 相同 byte7/12/17
	}

	// 正序添加
	fp1 := NewFingerprint()
	for _, txID := range txIDs {
		fp1.AddLeaf(txID, 2026, 0)
	}

	// 逆序添加
	fp2 := NewFingerprint()
	for i := len(txIDs) - 1; i >= 0; i-- {
		fp2.AddLeaf(txIDs[i], 2026, 0)
	}

	root1 := fp1.RootHash()
	root2 := fp2.RootHash()
	if root1 != root2 {
		t.Error("Tx18 leaf ordering should not depend on insertion order")
	}

	// 手动验证排序方式：收集 InfoHash 并按 TxID 字典序排列后拼接
	var leaves []types.Hash512
	for _, txID := range txIDs {
		leaves = append(leaves, txID)
	}
	sort.Slice(leaves, func(i, j int) bool {
		return bytes.Compare(leaves[i][:], leaves[j][:]) < 0
	})

	var infoConcat []byte
	for _, txID := range leaves {
		flags := FlagOutputsFromIndices([]uint16{0})
		infoHash := CalcInfoHash(txID, flags)
		infoConcat = append(infoConcat, infoHash[:]...)
	}
	expectedTx18Hash := crypto.SHA3_512Sum(infoConcat)

	// 获取实际的 Tx18 节点哈希
	yearNode := fp1.YearNodes[2026]
	tx8Node := yearNode.Tx8Nodes[0xAA]
	tx13Node := tx8Node.Tx13Nodes[0xBB]
	tx18Node := tx13Node.Tx18Nodes[0xCC]

	// 触发计算
	_ = fp1.RootHash()

	if tx18Node.Hash != expectedTx18Hash {
		t.Errorf("Tx18 hash mismatch\ngot      = %x\nexpected = %x",
			tx18Node.Hash[:8], expectedTx18Hash[:8])
	}
}
```

### Step 2: 运行测试验证失败

```bash
go test -v ./internal/utxo/ -run "TestFingerprint"
```

预期输出：编译失败，`Fingerprint`、`YearNode`、`Tx8Node` 等未定义。

### Step 3: 写最小实现

创建 `internal/utxo/fingerprint.go`：

```go
package utxo

import (
	"bytes"
	"sort"

	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// 指纹树的分层键值字节偏移位置。
// 使用 TxID 的第 7、12、17 字节分别作为 Tx8、Tx13、Tx18 层的键值。
const (
	tx8KeyOffset  = 7  // TxID[7] -> Tx8 层
	tx13KeyOffset = 12 // TxID[12] -> Tx13 层
	tx18KeyOffset = 17 // TxID[17] -> Tx18 层
)

// LeafInfo 叶子节点信息，对应一个交易的未花费输出集合。
type LeafInfo struct {
	TxID        types.Hash512 // 交易 ID
	FlagOutputs []byte        // 输出标记位序列
	InfoHash    types.Hash512 // 叶子哈希（缓存值）
}

// Tx18Node Tx18 层节点，包含按 TxID 字典序排列的叶子节点。
type Tx18Node struct {
	Hash   types.Hash512                  // 本节点哈希（缓存值）
	Leaves map[types.Hash512]*LeafInfo    // TxID -> LeafInfo
	dirty  bool                           // 是否需要重算
}

// Tx13Node Tx13 层节点，包含 256 个 Tx18 子节点。
type Tx13Node struct {
	Hash      types.Hash512   // 本节点哈希（缓存值）
	Tx18Nodes [256]*Tx18Node  // 按 TxID[17] 索引
	dirty     bool            // 是否需要重算
}

// Tx8Node Tx8 层节点，包含 256 个 Tx13 子节点。
type Tx8Node struct {
	Hash      types.Hash512    // 本节点哈希（缓存值）
	Tx13Nodes [256]*Tx13Node   // 按 TxID[12] 索引
	dirty     bool             // 是否需要重算
}

// YearNode 年份节点，包含 256 个 Tx8 子节点。
type YearNode struct {
	Year     int              // 年份
	Hash     types.Hash512    // 本节点哈希（缓存值）
	Tx8Nodes [256]*Tx8Node    // 按 TxID[7] 索引
	dirty    bool             // 是否需要重算
}

// Fingerprint 4 级层次哈希指纹树。
// 用于快速比较两个节点的 UTXO 集合是否一致。
type Fingerprint struct {
	Root      types.Hash512     // 根哈希（缓存值）
	YearNodes map[int]*YearNode // 年份 -> 年份节点
	dirty     bool              // 是否需要重算
}

// NewFingerprint 创建一个空的指纹树。
func NewFingerprint() *Fingerprint {
	return &Fingerprint{
		YearNodes: make(map[int]*YearNode),
	}
}

// getOrCreatePath 获取或创建从 Year 到 Tx18 的完整路径。
// 返回路径上的四个节点，如果不存在则自动创建。
func (f *Fingerprint) getOrCreatePath(txID types.Hash512, year int) (*YearNode, *Tx8Node, *Tx13Node, *Tx18Node) {
	// Year 层
	yearNode, ok := f.YearNodes[year]
	if !ok {
		yearNode = &YearNode{Year: year}
		f.YearNodes[year] = yearNode
	}

	// Tx8 层
	tx8Key := txID[tx8KeyOffset]
	tx8Node := yearNode.Tx8Nodes[tx8Key]
	if tx8Node == nil {
		tx8Node = &Tx8Node{}
		yearNode.Tx8Nodes[tx8Key] = tx8Node
	}

	// Tx13 层
	tx13Key := txID[tx13KeyOffset]
	tx13Node := tx8Node.Tx13Nodes[tx13Key]
	if tx13Node == nil {
		tx13Node = &Tx13Node{}
		tx8Node.Tx13Nodes[tx13Key] = tx13Node
	}

	// Tx18 层
	tx18Key := txID[tx18KeyOffset]
	tx18Node := tx13Node.Tx18Nodes[tx18Key]
	if tx18Node == nil {
		tx18Node = &Tx18Node{
			Leaves: make(map[types.Hash512]*LeafInfo),
		}
		tx13Node.Tx18Nodes[tx18Key] = tx18Node
	}

	return yearNode, tx8Node, tx13Node, tx18Node
}

// markDirty 标记从叶子到根的路径为脏（需要重算）。
func (f *Fingerprint) markDirty(yearNode *YearNode, tx8Node *Tx8Node, tx13Node *Tx13Node, tx18Node *Tx18Node) {
	tx18Node.dirty = true
	tx13Node.dirty = true
	tx8Node.dirty = true
	yearNode.dirty = true
	f.dirty = true
}

// AddLeaf 向指纹树中添加一个输出。
// 如果同一交易已有叶子节点，则合并输出标记位。
func (f *Fingerprint) AddLeaf(txID types.Hash512, year int, outputIndex uint16) {
	yearNode, tx8Node, tx13Node, tx18Node := f.getOrCreatePath(txID, year)

	leaf, ok := tx18Node.Leaves[txID]
	if !ok {
		// 新建叶子节点
		flags := FlagOutputsFromIndices([]uint16{outputIndex})
		leaf = &LeafInfo{
			TxID:        txID,
			FlagOutputs: flags,
			InfoHash:    CalcInfoHash(txID, flags),
		}
		tx18Node.Leaves[txID] = leaf
	} else {
		// 合并输出标记位
		bytePos := int(outputIndex / 8)
		bitPos := outputIndex % 8

		// 如有需要，扩展标记位
		for len(leaf.FlagOutputs) <= bytePos {
			leaf.FlagOutputs = append(leaf.FlagOutputs, 0)
		}
		leaf.FlagOutputs[bytePos] |= 1 << bitPos
		leaf.InfoHash = CalcInfoHash(txID, leaf.FlagOutputs)
	}

	f.markDirty(yearNode, tx8Node, tx13Node, tx18Node)
}

// RemoveLeaf 从指纹树中移除一个输出。
// 如果移除后该交易无剩余输出，则删除叶子节点。
func (f *Fingerprint) RemoveLeaf(txID types.Hash512, year int, outputIndex uint16) {
	yearNode, ok := f.YearNodes[year]
	if !ok {
		return
	}

	tx8Key := txID[tx8KeyOffset]
	tx8Node := yearNode.Tx8Nodes[tx8Key]
	if tx8Node == nil {
		return
	}

	tx13Key := txID[tx13KeyOffset]
	tx13Node := tx8Node.Tx13Nodes[tx13Key]
	if tx13Node == nil {
		return
	}

	tx18Key := txID[tx18KeyOffset]
	tx18Node := tx13Node.Tx18Nodes[tx18Key]
	if tx18Node == nil {
		return
	}

	leaf, ok := tx18Node.Leaves[txID]
	if !ok {
		return
	}

	// 清除对应位
	bytePos := int(outputIndex / 8)
	bitPos := outputIndex % 8
	if bytePos < len(leaf.FlagOutputs) {
		leaf.FlagOutputs[bytePos] &^= 1 << bitPos
	}

	// 检查标记位是否已全部清零
	allZero := true
	for _, b := range leaf.FlagOutputs {
		if b != 0 {
			allZero = false
			break
		}
	}

	if allZero {
		// 删除叶子
		delete(tx18Node.Leaves, txID)
		// 逐级清理空节点
		f.cleanupPath(yearNode, tx8Key, tx13Key, tx18Key)
	} else {
		// 重算叶子哈希
		leaf.InfoHash = CalcInfoHash(txID, leaf.FlagOutputs)
	}

	f.markDirty(yearNode, tx8Node, tx13Node, tx18Node)
}

// cleanupPath 清理空的中间节点。
// 从 Tx18 层向上逐级检查，如果子节点为空则删除。
func (f *Fingerprint) cleanupPath(yearNode *YearNode, tx8Key, tx13Key, tx18Key byte) {
	tx8Node := yearNode.Tx8Nodes[tx8Key]
	if tx8Node == nil {
		return
	}
	tx13Node := tx8Node.Tx13Nodes[tx13Key]
	if tx13Node == nil {
		return
	}
	tx18Node := tx13Node.Tx18Nodes[tx18Key]

	// 清理 Tx18
	if tx18Node != nil && len(tx18Node.Leaves) == 0 {
		tx13Node.Tx18Nodes[tx18Key] = nil
	}

	// 清理 Tx13（检查所有 Tx18 子节点是否为空）
	tx13Empty := true
	for _, n := range tx13Node.Tx18Nodes {
		if n != nil {
			tx13Empty = false
			break
		}
	}
	if tx13Empty {
		tx8Node.Tx13Nodes[tx13Key] = nil
	}

	// 清理 Tx8（检查所有 Tx13 子节点是否为空）
	tx8Empty := true
	for _, n := range tx8Node.Tx13Nodes {
		if n != nil {
			tx8Empty = false
			break
		}
	}
	if tx8Empty {
		yearNode.Tx8Nodes[tx8Key] = nil
	}

	// 清理 Year（检查所有 Tx8 子节点是否为空）
	yearEmpty := true
	for _, n := range yearNode.Tx8Nodes {
		if n != nil {
			yearEmpty = false
			break
		}
	}
	if yearEmpty {
		delete(f.YearNodes, yearNode.Year)
	}
}

// UpdateLeaf 直接更新某个交易的叶子标记位。
// 用于批量操作场景，可避免多次 AddLeaf/RemoveLeaf 的重复计算。
func (f *Fingerprint) UpdateLeaf(txID types.Hash512, year int, flagOutputs []byte) {
	yearNode, tx8Node, tx13Node, tx18Node := f.getOrCreatePath(txID, year)

	// 检查是否全零
	allZero := true
	for _, b := range flagOutputs {
		if b != 0 {
			allZero = false
			break
		}
	}

	if allZero {
		// 删除叶子
		delete(tx18Node.Leaves, txID)
		tx8Key := txID[tx8KeyOffset]
		tx13Key := txID[tx13KeyOffset]
		tx18Key := txID[tx18KeyOffset]
		f.cleanupPath(yearNode, tx8Key, tx13Key, tx18Key)
	} else {
		leaf, ok := tx18Node.Leaves[txID]
		if !ok {
			leaf = &LeafInfo{TxID: txID}
			tx18Node.Leaves[txID] = leaf
		}
		// 复制标记位（防止外部修改）
		leaf.FlagOutputs = make([]byte, len(flagOutputs))
		copy(leaf.FlagOutputs, flagOutputs)
		leaf.InfoHash = CalcInfoHash(txID, leaf.FlagOutputs)
	}

	f.markDirty(yearNode, tx8Node, tx13Node, tx18Node)
}

// RootHash 返回指纹树的根哈希。
// 如果树被标记为脏，则触发惰性重算。
func (f *Fingerprint) RootHash() types.Hash512 {
	if f.dirty {
		f.recalcRoot()
	}
	return f.Root
}

// Recalculate 强制对整棵指纹树执行全量重算。
// 忽略脏标记，重新计算所有中间节点和根节点。
func (f *Fingerprint) Recalculate() {
	// 标记所有节点为脏
	for _, yearNode := range f.YearNodes {
		yearNode.dirty = true
		for _, tx8Node := range yearNode.Tx8Nodes {
			if tx8Node == nil {
				continue
			}
			tx8Node.dirty = true
			for _, tx13Node := range tx8Node.Tx13Nodes {
				if tx13Node == nil {
					continue
				}
				tx13Node.dirty = true
				for _, tx18Node := range tx13Node.Tx18Nodes {
					if tx18Node == nil {
						continue
					}
					tx18Node.dirty = true
				}
			}
		}
	}
	f.dirty = true

	// 触发完全重算
	f.recalcRoot()
}

// recalcRoot 重算根哈希。
// 按年份升序拼接所有年份节点的哈希值，然后计算 SHA3-512。
func (f *Fingerprint) recalcRoot() {
	if len(f.YearNodes) == 0 {
		f.Root = types.Hash512{}
		f.dirty = false
		return
	}

	// 收集年份并排序
	years := make([]int, 0, len(f.YearNodes))
	for year := range f.YearNodes {
		years = append(years, year)
	}
	sort.Ints(years)

	// 先重算每个年份节点
	for _, year := range years {
		yearNode := f.YearNodes[year]
		if yearNode.dirty {
			f.recalcYear(yearNode)
		}
	}

	// 拼接年份哈希
	buf := make([]byte, 0, len(years)*types.HashLen)
	for _, year := range years {
		yearNode := f.YearNodes[year]
		buf = append(buf, yearNode.Hash[:]...)
	}

	f.Root = crypto.SHA3_512Sum(buf)
	f.dirty = false
}

// recalcYear 重算年份节点的哈希。
// 按 Tx8 桶下标（0-255）升序拼接非空子节点的哈希值。
func (f *Fingerprint) recalcYear(yearNode *YearNode) {
	var buf []byte

	for i := 0; i < 256; i++ {
		tx8Node := yearNode.Tx8Nodes[i]
		if tx8Node == nil {
			continue
		}
		if tx8Node.dirty {
			f.recalcTx8(tx8Node)
		}
		buf = append(buf, tx8Node.Hash[:]...)
	}

	if len(buf) == 0 {
		yearNode.Hash = types.Hash512{}
	} else {
		yearNode.Hash = crypto.SHA3_512Sum(buf)
	}
	yearNode.dirty = false
}

// recalcTx8 重算 Tx8 节点的哈希。
// 按 Tx13 桶下标（0-255）升序拼接非空子节点的哈希值。
func (f *Fingerprint) recalcTx8(tx8Node *Tx8Node) {
	var buf []byte

	for i := 0; i < 256; i++ {
		tx13Node := tx8Node.Tx13Nodes[i]
		if tx13Node == nil {
			continue
		}
		if tx13Node.dirty {
			f.recalcTx13(tx13Node)
		}
		buf = append(buf, tx13Node.Hash[:]...)
	}

	if len(buf) == 0 {
		tx8Node.Hash = types.Hash512{}
	} else {
		tx8Node.Hash = crypto.SHA3_512Sum(buf)
	}
	tx8Node.dirty = false
}

// recalcTx13 重算 Tx13 节点的哈希。
// 按 Tx18 桶下标（0-255）升序拼接非空子节点的哈希值。
func (f *Fingerprint) recalcTx13(tx13Node *Tx13Node) {
	var buf []byte

	for i := 0; i < 256; i++ {
		tx18Node := tx13Node.Tx18Nodes[i]
		if tx18Node == nil {
			continue
		}
		if tx18Node.dirty {
			f.recalcTx18(tx18Node)
		}
		buf = append(buf, tx18Node.Hash[:]...)
	}

	if len(buf) == 0 {
		tx13Node.Hash = types.Hash512{}
	} else {
		tx13Node.Hash = crypto.SHA3_512Sum(buf)
	}
	tx13Node.dirty = false
}

// recalcTx18 重算 Tx18 节点的哈希。
// 按 TxID 字典序拼接所有叶子的 InfoHash。
func (f *Fingerprint) recalcTx18(tx18Node *Tx18Node) {
	if len(tx18Node.Leaves) == 0 {
		tx18Node.Hash = types.Hash512{}
		tx18Node.dirty = false
		return
	}

	// 收集 TxID 并按字典序排序
	txIDs := make([]types.Hash512, 0, len(tx18Node.Leaves))
	for txID := range tx18Node.Leaves {
		txIDs = append(txIDs, txID)
	}
	sort.Slice(txIDs, func(i, j int) bool {
		return bytes.Compare(txIDs[i][:], txIDs[j][:]) < 0
	})

	// 拼接 InfoHash
	buf := make([]byte, 0, len(txIDs)*types.HashLen)
	for _, txID := range txIDs {
		leaf := tx18Node.Leaves[txID]
		buf = append(buf, leaf.InfoHash[:]...)
	}

	tx18Node.Hash = crypto.SHA3_512Sum(buf)
	tx18Node.dirty = false
}

// recalcBranch 增量重算指定路径上的分支。
// 仅重新计算从 Tx18 节点到根节点的路径。
func (f *Fingerprint) recalcBranch(year int, tx8 byte, tx13 byte, tx18 byte) {
	yearNode, ok := f.YearNodes[year]
	if !ok {
		return
	}
	tx8Node := yearNode.Tx8Nodes[tx8]
	if tx8Node == nil {
		return
	}
	tx13Node := tx8Node.Tx13Nodes[tx13]
	if tx13Node == nil {
		return
	}
	tx18Node := tx13Node.Tx18Nodes[tx18]
	if tx18Node != nil {
		f.recalcTx18(tx18Node)
	}
	f.recalcTx13(tx13Node)
	f.recalcTx8(tx8Node)
	f.recalcYear(yearNode)
	f.recalcRoot()
}
```

### Step 4: 运行测试验证通过

```bash
go test -v ./internal/utxo/ -run "TestFingerprint"
```

预期输出：全部 PASS。

### Step 5: 提交

```bash
git add internal/utxo/fingerprint.go internal/utxo/fingerprint_test.go
git commit -m "feat(utxo): add 4-level hierarchical SHA3-512 fingerprint tree with incremental updates"
```

---

## Task 4: UTXO 缓存 (internal/utxo/cache.go)

**Files:**
- Create: `internal/utxo/cache.go`
- Test: `internal/utxo/cache_test.go`

本 Task 实现 `UTXOCache` 暂存层，在 `UTXOSet` 之上提供临时变更暂存，用于处理区块时先暂存变更、最后统一提交或回滚。

### Step 1: 写失败测试

创建 `internal/utxo/cache_test.go`：

```go
package utxo

import (
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// 辅助函数：向 UTXOSet 中添加测试条目
func addToSet(set *UTXOSet, txByte byte, index uint16, amount int64, height uint64) *UTXOEntry {
	entry := makeTestEntry(txByte, index, amount, height)
	_ = set.Add(entry)
	return entry
}

// --- NewUTXOCache 测试 ---

func TestNewUTXOCache(t *testing.T) {
	base := NewUTXOSet()
	cache := NewUTXOCache(base)
	if cache == nil {
		t.Fatal("NewUTXOCache() returned nil")
	}
}

// --- Add 测试 ---

func TestUTXOCache_Add(t *testing.T) {
	base := NewUTXOSet()
	cache := NewUTXOCache(base)

	entry := makeTestEntry(0x01, 0, 1000, 100)
	err := cache.Add(entry)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	// 通过 cache 应能查到
	got, ok := cache.Get(entry.OutPoint)
	if !ok {
		t.Fatal("Get() should find the added entry")
	}
	if got.Amount != 1000 {
		t.Errorf("Get().Amount = %d, want 1000", got.Amount)
	}

	// base 中不应有
	if base.Has(entry.OutPoint) {
		t.Error("base should not contain the entry before Commit()")
	}
}

// 测试向缓存中添加 base 已有的条目
func TestUTXOCache_Add_DuplicateInBase(t *testing.T) {
	base := NewUTXOSet()
	addToSet(base, 0x01, 0, 1000, 100)

	cache := NewUTXOCache(base)
	entry := makeTestEntry(0x01, 0, 2000, 200)

	err := cache.Add(entry)
	if err == nil {
		t.Error("Add() should return error for entry already in base")
	}
}

// 测试向缓存中重复添加
func TestUTXOCache_Add_DuplicateInCache(t *testing.T) {
	base := NewUTXOSet()
	cache := NewUTXOCache(base)

	entry := makeTestEntry(0x01, 0, 1000, 100)
	_ = cache.Add(entry)

	err := cache.Add(entry)
	if err == nil {
		t.Error("Add() should return error for duplicate entry in cache")
	}
}

// --- Spend 测试 ---

// 测试花费 base 中的条目
func TestUTXOCache_Spend_FromBase(t *testing.T) {
	base := NewUTXOSet()
	entry := addToSet(base, 0x01, 0, 1000, 100)

	cache := NewUTXOCache(base)
	spent, err := cache.Spend(entry.OutPoint)
	if err != nil {
		t.Fatalf("Spend() error = %v", err)
	}
	if spent.Amount != 1000 {
		t.Errorf("Spend().Amount = %d, want 1000", spent.Amount)
	}

	// 花费后通过 cache 应查不到
	if cache.Has(entry.OutPoint) {
		t.Error("Has() should return false for spent entry")
	}

	// base 中仍应存在（未提交）
	if !base.Has(entry.OutPoint) {
		t.Error("base should still contain the entry before Commit()")
	}
}

// 测试花费缓存中新添加的条目
func TestUTXOCache_Spend_FromAdded(t *testing.T) {
	base := NewUTXOSet()
	cache := NewUTXOCache(base)

	entry := makeTestEntry(0x01, 0, 1000, 100)
	_ = cache.Add(entry)

	spent, err := cache.Spend(entry.OutPoint)
	if err != nil {
		t.Fatalf("Spend() error = %v", err)
	}
	if spent.Amount != 1000 {
		t.Errorf("Spend().Amount = %d, want 1000", spent.Amount)
	}

	// 花费后应查不到
	if cache.Has(entry.OutPoint) {
		t.Error("Has() should return false for spent entry from added")
	}
}

// 测试花费不存在的条目
func TestUTXOCache_Spend_NotFound(t *testing.T) {
	base := NewUTXOSet()
	cache := NewUTXOCache(base)

	op := OutPoint{TxID: types.Hash512{0xff}, Index: 0}
	_, err := cache.Spend(op)
	if err == nil {
		t.Error("Spend() should return error for non-existent entry")
	}
}

// 测试重复花费
func TestUTXOCache_Spend_AlreadySpent(t *testing.T) {
	base := NewUTXOSet()
	entry := addToSet(base, 0x01, 0, 1000, 100)

	cache := NewUTXOCache(base)
	_, _ = cache.Spend(entry.OutPoint)

	_, err := cache.Spend(entry.OutPoint)
	if err == nil {
		t.Error("Spend() should return error for already spent entry")
	}
}

// --- Get/Has 测试 ---

// 测试 Get 合并 base + added - removed
func TestUTXOCache_Get_MergedView(t *testing.T) {
	base := NewUTXOSet()
	baseEntry := addToSet(base, 0x01, 0, 1000, 100)
	spendEntry := addToSet(base, 0x02, 0, 2000, 200)

	cache := NewUTXOCache(base)
	newEntry := makeTestEntry(0x03, 0, 3000, 300)
	_ = cache.Add(newEntry)
	_, _ = cache.Spend(spendEntry.OutPoint)

	// baseEntry: 在 base 中，未被花费 -> 可见
	if _, ok := cache.Get(baseEntry.OutPoint); !ok {
		t.Error("base entry should be visible through cache")
	}

	// spendEntry: 在 base 中，已被花费 -> 不可见
	if _, ok := cache.Get(spendEntry.OutPoint); ok {
		t.Error("spent entry should not be visible through cache")
	}

	// newEntry: 在 added 中 -> 可见
	if _, ok := cache.Get(newEntry.OutPoint); !ok {
		t.Error("added entry should be visible through cache")
	}

	// 不存在的 -> 不可见
	op := OutPoint{TxID: types.Hash512{0xff}, Index: 0}
	if _, ok := cache.Get(op); ok {
		t.Error("non-existent entry should not be visible")
	}
}

// --- Commit 测试 ---

func TestUTXOCache_Commit(t *testing.T) {
	base := NewUTXOSet()
	spendEntry := addToSet(base, 0x01, 0, 1000, 100)

	cache := NewUTXOCache(base)
	newEntry := makeTestEntry(0x02, 0, 2000, 200)
	_ = cache.Add(newEntry)
	_, _ = cache.Spend(spendEntry.OutPoint)

	err := cache.Commit()
	if err != nil {
		t.Fatalf("Commit() error = %v", err)
	}

	// 提交后：spendEntry 应从 base 中移除
	if base.Has(spendEntry.OutPoint) {
		t.Error("spent entry should be removed from base after Commit()")
	}

	// 提交后：newEntry 应在 base 中
	if !base.Has(newEntry.OutPoint) {
		t.Error("added entry should be in base after Commit()")
	}

	// 提交后：缓存应为空状态
	if len(cache.AddedEntries()) != 0 {
		t.Error("AddedEntries() should be empty after Commit()")
	}
	if len(cache.SpentEntries()) != 0 {
		t.Error("SpentEntries() should be empty after Commit()")
	}
}

// --- Rollback 测试 ---

func TestUTXOCache_Rollback(t *testing.T) {
	base := NewUTXOSet()
	spendEntry := addToSet(base, 0x01, 0, 1000, 100)

	cache := NewUTXOCache(base)
	newEntry := makeTestEntry(0x02, 0, 2000, 200)
	_ = cache.Add(newEntry)
	_, _ = cache.Spend(spendEntry.OutPoint)

	cache.Rollback()

	// 回滚后：spendEntry 应仍在 base 中
	if !base.Has(spendEntry.OutPoint) {
		t.Error("spent entry should still be in base after Rollback()")
	}

	// 回滚后：newEntry 不应在 base 中
	if base.Has(newEntry.OutPoint) {
		t.Error("added entry should not be in base after Rollback()")
	}

	// 回滚后：缓存应为空状态
	if len(cache.AddedEntries()) != 0 {
		t.Error("AddedEntries() should be empty after Rollback()")
	}
	if len(cache.SpentEntries()) != 0 {
		t.Error("SpentEntries() should be empty after Rollback()")
	}

	// 回滚后通过 cache 应能查到 base 中的条目
	if !cache.Has(spendEntry.OutPoint) {
		t.Error("base entry should be visible through cache after Rollback()")
	}
}

// --- AddedEntries / SpentEntries 测试 ---

func TestUTXOCache_AddedEntries(t *testing.T) {
	base := NewUTXOSet()
	cache := NewUTXOCache(base)

	entry1 := makeTestEntry(0x01, 0, 1000, 100)
	entry2 := makeTestEntry(0x02, 0, 2000, 200)
	_ = cache.Add(entry1)
	_ = cache.Add(entry2)

	added := cache.AddedEntries()
	if len(added) != 2 {
		t.Errorf("AddedEntries() length = %d, want 2", len(added))
	}
}

func TestUTXOCache_SpentEntries(t *testing.T) {
	base := NewUTXOSet()
	entry1 := addToSet(base, 0x01, 0, 1000, 100)
	entry2 := addToSet(base, 0x02, 0, 2000, 200)

	cache := NewUTXOCache(base)
	_, _ = cache.Spend(entry1.OutPoint)
	_, _ = cache.Spend(entry2.OutPoint)

	spent := cache.SpentEntries()
	if len(spent) != 2 {
		t.Errorf("SpentEntries() length = %d, want 2", len(spent))
	}
}

// --- 综合场景：区块处理模拟 ---

func TestUTXOCache_BlockProcessing(t *testing.T) {
	base := NewUTXOSet()

	// 准备 base：3 个 UTXO
	e1 := addToSet(base, 0x10, 0, 50000, 1)
	e2 := addToSet(base, 0x20, 0, 30000, 2)
	_ = addToSet(base, 0x30, 0, 20000, 3)

	if base.Count() != 3 {
		t.Fatalf("base.Count() = %d, want 3", base.Count())
	}

	// 模拟处理一个区块
	cache := NewUTXOCache(base)

	// 花费 e1 和 e2
	_, err := cache.Spend(e1.OutPoint)
	if err != nil {
		t.Fatalf("Spend e1 error = %v", err)
	}
	_, err = cache.Spend(e2.OutPoint)
	if err != nil {
		t.Fatalf("Spend e2 error = %v", err)
	}

	// 产生 2 个新输出
	newEntry1 := makeTestEntry(0x40, 0, 45000, 10)
	newEntry2 := makeTestEntry(0x40, 1, 34000, 10)
	_ = cache.Add(newEntry1)
	_ = cache.Add(newEntry2)

	// 提交到 base
	if err := cache.Commit(); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}

	// 验证最终状态
	if base.Count() != 3 { // 3 - 2 + 2 = 3
		t.Errorf("base.Count() = %d, want 3", base.Count())
	}
	if base.Has(e1.OutPoint) {
		t.Error("e1 should be spent")
	}
	if base.Has(e2.OutPoint) {
		t.Error("e2 should be spent")
	}
	if !base.Has(newEntry1.OutPoint) {
		t.Error("newEntry1 should be in base")
	}
	if !base.Has(newEntry2.OutPoint) {
		t.Error("newEntry2 should be in base")
	}
}

// --- 花费后又添加同一 OutPoint（在同区块内花费后重新产出）---

func TestUTXOCache_SpendThenAdd_SameOutPoint(t *testing.T) {
	base := NewUTXOSet()
	oldEntry := addToSet(base, 0x01, 0, 1000, 100)

	cache := NewUTXOCache(base)

	// 先花费旧条目
	_, err := cache.Spend(oldEntry.OutPoint)
	if err != nil {
		t.Fatalf("Spend() error = %v", err)
	}

	// 再添加同一 OutPoint 的新条目（虽然实际中不太可能，但需要处理）
	newEntry := makeTestEntry(0x01, 0, 5000, 200)
	err = cache.Add(newEntry)
	if err != nil {
		t.Fatalf("Add() after Spend() same OutPoint error = %v", err)
	}

	// 通过 cache 查询应得到新条目
	got, ok := cache.Get(oldEntry.OutPoint)
	if !ok {
		t.Fatal("Get() should find the re-added entry")
	}
	if got.Amount != 5000 {
		t.Errorf("Get().Amount = %d, want 5000", got.Amount)
	}
}
```

### Step 2: 运行测试验证失败

```bash
go test -v ./internal/utxo/ -run "TestNewUTXOCache|TestUTXOCache"
```

预期输出：编译失败，`UTXOCache`、`NewUTXOCache` 等未定义。

### Step 3: 写最小实现

创建 `internal/utxo/cache.go`：

```go
package utxo

import (
	"fmt"
)

// UTXOCache 在 UTXOSet 之上提供暂存层。
// 用于处理区块时的临时变更：先将花费和新产出记录在缓存中，
// 全部处理完毕后统一 Commit 写入 base，或 Rollback 丢弃变更。
type UTXOCache struct {
	base    *UTXOSet                 // 底层 UTXO 集合
	added   map[OutPoint]*UTXOEntry  // 新增的条目
	removed map[OutPoint]*UTXOEntry  // 已花费的条目
}

// NewUTXOCache 创建一个基于指定 UTXOSet 的缓存层。
func NewUTXOCache(base *UTXOSet) *UTXOCache {
	return &UTXOCache{
		base:    base,
		added:   make(map[OutPoint]*UTXOEntry),
		removed: make(map[OutPoint]*UTXOEntry),
	}
}

// Add 向缓存中添加一个新的 UTXO 条目。
// 如果该 OutPoint 在 base 或 added 中已存在（且未被花费），返回错误。
func (c *UTXOCache) Add(entry *UTXOEntry) error {
	op := entry.OutPoint

	// 如果之前已被花费，则允许重新添加（从 removed 中移除标记）
	if _, wasRemoved := c.removed[op]; wasRemoved {
		delete(c.removed, op)
		c.added[op] = entry
		return nil
	}

	// 检查 added 中是否已有
	if _, exists := c.added[op]; exists {
		return fmt.Errorf("%w: %s (in cache)", ErrDuplicateEntry, op.String())
	}

	// 检查 base 中是否已有
	if c.base.Has(op) {
		return fmt.Errorf("%w: %s (in base)", ErrDuplicateEntry, op.String())
	}

	c.added[op] = entry
	return nil
}

// Spend 标记花费一个 UTXO。
// 优先从 added 中查找，再从 base 中查找。
// 返回被花费的条目。
func (c *UTXOCache) Spend(outpoint OutPoint) (*UTXOEntry, error) {
	// 检查是否已被花费
	if _, wasRemoved := c.removed[outpoint]; wasRemoved {
		return nil, fmt.Errorf("%w: %s (already spent)", ErrEntryNotFound, outpoint.String())
	}

	// 优先从 added 中查找
	if entry, ok := c.added[outpoint]; ok {
		delete(c.added, outpoint)
		// 如果 base 中也有，需要记录到 removed
		// 如果 base 中没有，直接删除即可（在同一区块内添加又花费）
		if c.base.Has(outpoint) {
			c.removed[outpoint] = entry
		}
		return entry, nil
	}

	// 从 base 中查找
	if entry, ok := c.base.Get(outpoint); ok {
		c.removed[outpoint] = entry
		return entry, nil
	}

	return nil, fmt.Errorf("%w: %s", ErrEntryNotFound, outpoint.String())
}

// Get 查询指定 OutPoint 的 UTXO 条目。
// 合并视图：base + added - removed。
func (c *UTXOCache) Get(outpoint OutPoint) (*UTXOEntry, bool) {
	// 如果在 removed 中，不可见
	if _, wasRemoved := c.removed[outpoint]; wasRemoved {
		// 但可能又被重新添加到 added 中
		if entry, ok := c.added[outpoint]; ok {
			return entry, true
		}
		return nil, false
	}

	// 优先查 added
	if entry, ok := c.added[outpoint]; ok {
		return entry, true
	}

	// 再查 base
	return c.base.Get(outpoint)
}

// Has 判断指定 OutPoint 在合并视图中是否存在。
func (c *UTXOCache) Has(outpoint OutPoint) bool {
	_, ok := c.Get(outpoint)
	return ok
}

// Commit 将缓存中的变更写入底层 UTXOSet。
// 先移除已花费的条目，再添加新增的条目。
// 提交后缓存恢复为空状态。
func (c *UTXOCache) Commit() error {
	// 先处理移除
	for op := range c.removed {
		if _, err := c.base.Remove(op); err != nil {
			return fmt.Errorf("commit remove %s: %w", op.String(), err)
		}
	}

	// 再处理添加
	for _, entry := range c.added {
		if err := c.base.Add(entry); err != nil {
			return fmt.Errorf("commit add %s: %w", entry.OutPoint.String(), err)
		}
	}

	// 清空缓存
	c.added = make(map[OutPoint]*UTXOEntry)
	c.removed = make(map[OutPoint]*UTXOEntry)

	return nil
}

// Rollback 丢弃缓存中的所有变更，恢复为空状态。
// 底层 UTXOSet 不受影响。
func (c *UTXOCache) Rollback() {
	c.added = make(map[OutPoint]*UTXOEntry)
	c.removed = make(map[OutPoint]*UTXOEntry)
}

// AddedEntries 返回缓存中所有新增的条目。
func (c *UTXOCache) AddedEntries() []*UTXOEntry {
	entries := make([]*UTXOEntry, 0, len(c.added))
	for _, entry := range c.added {
		entries = append(entries, entry)
	}
	return entries
}

// SpentEntries 返回缓存中所有已花费的条目。
func (c *UTXOCache) SpentEntries() []*UTXOEntry {
	entries := make([]*UTXOEntry, 0, len(c.removed))
	for _, entry := range c.removed {
		entries = append(entries, entry)
	}
	return entries
}
```

### Step 4: 运行测试验证通过

```bash
go test -v ./internal/utxo/ -run "TestNewUTXOCache|TestUTXOCache"
```

预期输出：全部 PASS。

### Step 5: 提交

```bash
git add internal/utxo/cache.go internal/utxo/cache_test.go
git commit -m "feat(utxo): add UTXOCache with Add/Spend/Commit/Rollback for block processing"
```

---

<!-- UTCO 部分（Task 5-9）开始 -->

## Task 5: UTCOEntry 结构

**Files:**
- Create: `internal/utco/entry.go`
- Test: `internal/utco/entry_test.go`

本 Task 实现 `OutPoint` 与 `UTCOEntry` 结构体，包括年份计算、基本验证、过期/活动检查、InfoHash 计算、以及综合移除判定。

### Step 1: 写失败测试

创建 `internal/utco/entry_test.go`：

```go
package utco

import (
	"fmt"
	"testing"

	"github.com/cxio/evidcoin/internal/tx"
	"github.com/cxio/evidcoin/pkg/types"
)

// --- OutPoint 测试 ---

// 测试 OutPoint.String() 返回格式化字符串
func TestOutPoint_String(t *testing.T) {
	op := OutPoint{
		TxID:  types.Hash512{0xAA, 0xBB, 0xCC},
		Index: 42,
	}

	s := op.String()
	if s == "" {
		t.Error("String() should not be empty")
	}

	// 应包含索引值
	expected := fmt.Sprintf("%x:%d", op.TxID[:8], op.Index)
	if s != expected {
		t.Errorf("String() = %q, want %q", s, expected)
	}
}

// 测试两个相同的 OutPoint 相等
func TestOutPoint_Equality(t *testing.T) {
	op1 := OutPoint{TxID: types.Hash512{0x01}, Index: 1}
	op2 := OutPoint{TxID: types.Hash512{0x01}, Index: 1}

	if op1 != op2 {
		t.Error("identical OutPoints should be equal")
	}
}

// 测试不同 OutPoint 不相等
func TestOutPoint_Inequality(t *testing.T) {
	base := OutPoint{TxID: types.Hash512{0x01}, Index: 1}

	tests := []struct {
		name  string
		other OutPoint
	}{
		{"different_txid", OutPoint{TxID: types.Hash512{0x02}, Index: 1}},
		{"different_index", OutPoint{TxID: types.Hash512{0x01}, Index: 2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if base == tt.other {
				t.Error("different OutPoints should not be equal")
			}
		})
	}
}

// --- YearFromHeight 测试 ---

// 表驱动测试年份计算
func TestYearFromHeight(t *testing.T) {
	tests := []struct {
		name   string
		height uint64
		want   int
	}{
		{"genesis_block", 0, GenesisYear},
		{"first_block", 1, GenesisYear},
		{"last_block_year_0", types.BlocksPerYear - 1, GenesisYear},
		{"first_block_year_1", types.BlocksPerYear, GenesisYear + 1},
		{"mid_year_2", types.BlocksPerYear*2 + 1000, GenesisYear + 2},
		{"year_100", types.BlocksPerYear * 100, GenesisYear + 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := YearFromHeight(tt.height)
			if got != tt.want {
				t.Errorf("YearFromHeight(%d) = %d, want %d", tt.height, got, tt.want)
			}
		})
	}
}

// --- UTCOEntry.Validate 测试 ---

// 辅助函数：创建有效的 UTCOEntry
func validUTCOEntry() *UTCOEntry {
	return &UTCOEntry{
		OutPoint: OutPoint{
			TxID:  types.Hash512{0x01, 0x02, 0x03},
			Index: 0,
		},
		Address:          types.PubKeyHash{0x10, 0x20},
		Height:           100,
		LastActiveHeight: 100,
		CredConfig:       tx.CredNew,
		LockScript:       []byte{0x01, 0x02, 0x03},
	}
}

// 表驱动测试 Validate 各种情况
func TestUTCOEntry_Validate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*UTCOEntry)
		wantErr bool
	}{
		{
			name:    "valid_basic",
			modify:  func(e *UTCOEntry) {},
			wantErr: false,
		},
		{
			name: "valid_with_xfer_count",
			modify: func(e *UTCOEntry) {
				e.CredConfig = tx.CredNew | tx.CredHasXferCount
				e.XferCount = 100
			},
			wantErr: false,
		},
		{
			name: "valid_with_expiry",
			modify: func(e *UTCOEntry) {
				e.CredConfig = tx.CredNew | tx.CredHasExpiry
				e.ExpiryHeight = 100 + 1000
			},
			wantErr: false,
		},
		{
			name: "valid_mutable",
			modify: func(e *UTCOEntry) {
				e.CredConfig = tx.CredNew | tx.CredMutable
				e.IsMutable = true
			},
			wantErr: false,
		},
		{
			name: "zero_txid",
			modify: func(e *UTCOEntry) {
				e.TxID = types.Hash512{}
			},
			wantErr: true,
		},
		{
			name: "zero_address",
			modify: func(e *UTCOEntry) {
				e.Address = types.PubKeyHash{}
			},
			wantErr: true,
		},
		{
			name: "empty_lock_script",
			modify: func(e *UTCOEntry) {
				e.LockScript = nil
			},
			wantErr: true,
		},
		{
			name: "lock_script_too_long",
			modify: func(e *UTCOEntry) {
				e.LockScript = make([]byte, types.MaxLockScript+1)
			},
			wantErr: true,
		},
		{
			name: "last_active_before_creation",
			modify: func(e *UTCOEntry) {
				e.Height = 200
				e.LastActiveHeight = 100
			},
			wantErr: true,
		},
		{
			name: "expiry_height_zero_when_has_expiry",
			modify: func(e *UTCOEntry) {
				e.CredConfig = tx.CredNew | tx.CredHasExpiry
				e.ExpiryHeight = 0
			},
			wantErr: true,
		},
		{
			name: "expiry_height_before_creation",
			modify: func(e *UTCOEntry) {
				e.CredConfig = tx.CredNew | tx.CredHasExpiry
				e.ExpiryHeight = 50
			},
			wantErr: true,
		},
		{
			name: "expiry_delta_exceeds_max",
			modify: func(e *UTCOEntry) {
				e.CredConfig = tx.CredNew | tx.CredHasExpiry
				e.ExpiryHeight = e.Height + MaxExpiryDelta + 1
			},
			wantErr: true,
		},
		{
			name: "xfer_count_zero_when_has_xfer",
			modify: func(e *UTCOEntry) {
				e.CredConfig = tx.CredNew | tx.CredHasXferCount
				e.XferCount = 0
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := validUTCOEntry()
			tt.modify(e)
			err := e.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// --- IsExpired 测试 ---

// 表驱动测试高度截止过期检查
func TestUTCOEntry_IsExpired(t *testing.T) {
	tests := []struct {
		name          string
		hasExpiry     bool
		expiryHeight  uint64
		currentHeight uint64
		want          bool
	}{
		{
			name:          "no_expiry_never_expired",
			hasExpiry:     false,
			expiryHeight:  0,
			currentHeight: 999999999,
			want:          false,
		},
		{
			name:          "before_expiry",
			hasExpiry:     true,
			expiryHeight:  1000,
			currentHeight: 999,
			want:          false,
		},
		{
			name:          "at_expiry",
			hasExpiry:     true,
			expiryHeight:  1000,
			currentHeight: 1000,
			want:          true,
		},
		{
			name:          "after_expiry",
			hasExpiry:     true,
			expiryHeight:  1000,
			currentHeight: 1001,
			want:          true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &UTCOEntry{
				Height: 100,
			}
			if tt.hasExpiry {
				e.CredConfig = tx.CredNew | tx.CredHasExpiry
				e.ExpiryHeight = tt.expiryHeight
			} else {
				e.CredConfig = tx.CredNew
			}

			got := e.IsExpired(tt.currentHeight)
			if got != tt.want {
				t.Errorf("IsExpired(%d) = %v, want %v", tt.currentHeight, got, tt.want)
			}
		})
	}
}

// --- IsInactive 测试 ---

// 表驱动测试 11 年活动规则
func TestUTCOEntry_IsInactive(t *testing.T) {
	tests := []struct {
		name             string
		hasExpiry        bool
		lastActiveHeight uint64
		currentHeight    uint64
		want             bool
	}{
		{
			name:             "recently_active",
			hasExpiry:        false,
			lastActiveHeight: 1000,
			currentHeight:    1000 + ActivityBlocks - 1,
			want:             false,
		},
		{
			name:             "exactly_at_deadline",
			hasExpiry:        false,
			lastActiveHeight: 1000,
			currentHeight:    1000 + ActivityBlocks,
			want:             true,
		},
		{
			name:             "past_deadline",
			hasExpiry:        false,
			lastActiveHeight: 1000,
			currentHeight:    1000 + ActivityBlocks + 100,
			want:             true,
		},
		{
			name:             "has_expiry_exempt",
			hasExpiry:        true,
			lastActiveHeight: 1000,
			currentHeight:    1000 + ActivityBlocks + 100000,
			want:             false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &UTCOEntry{
				Height:           100,
				LastActiveHeight: tt.lastActiveHeight,
			}
			if tt.hasExpiry {
				e.CredConfig = tx.CredNew | tx.CredHasExpiry
				e.ExpiryHeight = tt.lastActiveHeight + MaxExpiryDelta
			} else {
				e.CredConfig = tx.CredNew
			}

			got := e.IsInactive(tt.currentHeight)
			if got != tt.want {
				t.Errorf("IsInactive(%d) = %v, want %v", tt.currentHeight, got, tt.want)
			}
		})
	}
}

// --- ShouldRemove 测试 ---

// 综合移除判定测试
func TestUTCOEntry_ShouldRemove(t *testing.T) {
	tests := []struct {
		name          string
		setup         func() *UTCOEntry
		currentHeight uint64
		want          bool
	}{
		{
			name: "active_no_expiry",
			setup: func() *UTCOEntry {
				e := validUTCOEntry()
				e.LastActiveHeight = 1000
				return e
			},
			currentHeight: 1000 + ActivityBlocks - 1,
			want:          false,
		},
		{
			name: "expired_by_height",
			setup: func() *UTCOEntry {
				e := validUTCOEntry()
				e.CredConfig = tx.CredNew | tx.CredHasExpiry
				e.ExpiryHeight = 500
				return e
			},
			currentHeight: 500,
			want:          true,
		},
		{
			name: "inactive_no_expiry",
			setup: func() *UTCOEntry {
				e := validUTCOEntry()
				e.LastActiveHeight = 1000
				return e
			},
			currentHeight: 1000 + ActivityBlocks,
			want:          true,
		},
		{
			name: "has_expiry_not_yet_expired",
			setup: func() *UTCOEntry {
				e := validUTCOEntry()
				e.CredConfig = tx.CredNew | tx.CredHasExpiry
				e.ExpiryHeight = 50000
				e.LastActiveHeight = 100
				return e
			},
			currentHeight: 49999,
			want:          false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := tt.setup()
			got := e.ShouldRemove(tt.currentHeight)
			if got != tt.want {
				t.Errorf("ShouldRemove(%d) = %v, want %v", tt.currentHeight, got, tt.want)
			}
		})
	}
}

// --- CalcInfoHash 测试 ---

// 测试 CalcInfoHash 确定性
func TestCalcInfoHash_Deterministic(t *testing.T) {
	txID := types.Hash512{0x01, 0x02, 0x03}
	flagOutputs := []byte{0xFF, 0x0F}

	h1 := CalcInfoHash(txID, flagOutputs)
	h2 := CalcInfoHash(txID, flagOutputs)

	if h1 != h2 {
		t.Error("CalcInfoHash should be deterministic")
	}
}

// 测试不同输入产生不同 InfoHash
func TestCalcInfoHash_DifferentInputs(t *testing.T) {
	txID1 := types.Hash512{0x01}
	txID2 := types.Hash512{0x02}
	flags := []byte{0xFF}

	h1 := CalcInfoHash(txID1, flags)
	h2 := CalcInfoHash(txID2, flags)

	if h1 == h2 {
		t.Error("different TxIDs should produce different InfoHash")
	}

	// 不同标记也应不同
	h3 := CalcInfoHash(txID1, []byte{0x0F})
	if h1 == h3 {
		t.Error("different flagOutputs should produce different InfoHash")
	}
}

// 测试空标记输出
func TestCalcInfoHash_EmptyFlags(t *testing.T) {
	txID := types.Hash512{0x01}

	h1 := CalcInfoHash(txID, nil)
	h2 := CalcInfoHash(txID, []byte{})

	// nil 和空切片的行为应一致
	if h1 != h2 {
		t.Error("nil and empty flagOutputs should produce same InfoHash")
	}
}

// --- FlagOutputsFromIndices 测试 ---

// 表驱动测试标记位生成
func TestFlagOutputsFromIndices(t *testing.T) {
	tests := []struct {
		name    string
		indices []uint16
		want    []byte
	}{
		{
			name:    "empty",
			indices: nil,
			want:    nil,
		},
		{
			name:    "single_index_0",
			indices: []uint16{0},
			want:    []byte{0x80}, // bit 0 of byte 0 (MSB)
		},
		{
			name:    "single_index_7",
			indices: []uint16{7},
			want:    []byte{0x01}, // bit 7 of byte 0 (LSB)
		},
		{
			name:    "single_index_8",
			indices: []uint16{8},
			want:    []byte{0x00, 0x80}, // bit 0 of byte 1
		},
		{
			name:    "multiple_indices",
			indices: []uint16{0, 1, 7},
			want:    []byte{0xC1}, // 1100_0001
		},
		{
			name:    "unsorted_indices",
			indices: []uint16{7, 0, 1},
			want:    []byte{0xC1}, // 排序无关，结果应相同
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FlagOutputsFromIndices(tt.indices)
			if len(got) != len(tt.want) {
				t.Errorf("FlagOutputsFromIndices() len = %d, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("FlagOutputsFromIndices()[%d] = 0x%02X, want 0x%02X", i, got[i], tt.want[i])
				}
			}
		})
	}
}
```

### Step 2: 运行测试验证失败

```bash
go test -v ./internal/utco/ -run "TestOutPoint|TestYearFromHeight|TestUTCOEntry|TestCalcInfoHash|TestFlagOutputsFromIndices"
```

预期输出：编译失败，`OutPoint`、`UTCOEntry`、`YearFromHeight`、`CalcInfoHash` 等未定义。

### Step 3: 写最小实现

创建 `internal/utco/entry.go`：

```go
package utco

import (
	"errors"
	"fmt"

	"github.com/cxio/evidcoin/internal/tx"
	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// --- 常量 ---

const (
	// GenesisYear 创世年份（UTC）。
	GenesisYear = 2026

	// MaxExpiryDelta 最大截止高度差（约 100 年区块数）。
	// 100 × 87661 = 8,766,100
	MaxExpiryDelta = 8_766_100

	// ActivityBlocks 活动要求区块数（约 11 年）。
	// 11 × 87661 = 964,271
	ActivityBlocks = 964_271
)

// --- OutPoint 输出点 ---

// OutPoint 标识一个交易输出的位置。
// 与 UTXO 包中的 OutPoint 独立定义，避免跨包依赖。
type OutPoint struct {
	TxID  types.Hash512 // 交易 ID
	Index uint16        // 输出索引
}

// String 返回 OutPoint 的可读字符串表示。
// 格式：TxID前8字节十六进制:索引
func (op OutPoint) String() string {
	return fmt.Sprintf("%x:%d", op.TxID[:8], op.Index)
}

// --- UTCOEntry 未转移凭信输出条目 ---

// UTCOEntry 表示 UTCO 集中的一个未转移凭信输出条目。
// 记录凭信的位置、地址、创建高度、最近活动高度、凭信配置等元数据。
type UTCOEntry struct {
	OutPoint                          // 交易输出位置（嵌入）
	Address          types.PubKeyHash // 持有者地址
	Height           uint64           // 创建区块高度
	LastActiveHeight uint64           // 最近活动（转移）的区块高度
	CredConfig       tx.CredConf      // 凭信配置位域
	XferCount        uint16           // 剩余转移次数（仅 HasXferCount 时有效）
	ExpiryHeight     uint64           // 截止高度（仅 HasExpiry 时有效）
	IsMutable        bool             // 是否可变
	LockScript       []byte           // 锁定脚本
}

// YearFromHeight 根据区块高度计算所属 UTC 年份。
// 创世区块高度 0 对应 GenesisYear（2026）。
func YearFromHeight(height uint64) int {
	return GenesisYear + int(height/types.BlocksPerYear)
}

// 验证错误
var (
	ErrZeroTxID               = errors.New("utco entry txid is zero")
	ErrZeroAddress            = errors.New("utco entry address is zero")
	ErrEmptyLockScript        = errors.New("utco entry lock script is empty")
	ErrLockScriptTooLong      = errors.New("utco entry lock script exceeds max length")
	ErrLastActiveBeforeCreate = errors.New("utco entry last active height before creation height")
	ErrExpiryHeightZero       = errors.New("utco entry expiry height is zero when has expiry")
	ErrExpiryBeforeCreate     = errors.New("utco entry expiry height before creation height")
	ErrExpiryDeltaExceeded    = errors.New("utco entry expiry delta exceeds max")
	ErrXferCountZero          = errors.New("utco entry xfer count is zero when has xfer count")
)

// Validate 执行 UTCOEntry 的基本字段验证。
func (e *UTCOEntry) Validate() error {
	// 检查 TxID 非零
	if e.TxID.IsZero() {
		return ErrZeroTxID
	}

	// 检查地址非零
	if e.Address.IsZero() {
		return ErrZeroAddress
	}

	// 检查锁定脚本
	if len(e.LockScript) == 0 {
		return ErrEmptyLockScript
	}
	if len(e.LockScript) > types.MaxLockScript {
		return ErrLockScriptTooLong
	}

	// 最近活动高度不能早于创建高度
	if e.LastActiveHeight < e.Height {
		return ErrLastActiveBeforeCreate
	}

	// 如果有截止高度，进行相关验证
	if e.CredConfig.HasExpiry() {
		if e.ExpiryHeight == 0 {
			return ErrExpiryHeightZero
		}
		if e.ExpiryHeight <= e.Height {
			return ErrExpiryBeforeCreate
		}
		if e.ExpiryHeight-e.Height > MaxExpiryDelta {
			return ErrExpiryDeltaExceeded
		}
	}

	// 如果有转移次数，不能为零（为零应不入 UTCO 集）
	if e.CredConfig.HasXferCount() && e.XferCount == 0 {
		return ErrXferCountZero
	}

	return nil
}

// IsExpired 检查凭信是否因高度截止而过期。
// 仅对设置了截止高度的 Credit 有效。
func (e *UTCOEntry) IsExpired(currentHeight uint64) bool {
	if !e.CredConfig.HasExpiry() {
		return false
	}
	return currentHeight >= e.ExpiryHeight
}

// IsInactive 检查凭信是否因 11 年不活动而过期。
// 仅对无截止高度的 Credit 适用。
// 有截止高度的 Credit 不受此约束。
func (e *UTCOEntry) IsInactive(currentHeight uint64) bool {
	// 有截止高度的 Credit 免除活动要求
	if e.CredConfig.HasExpiry() {
		return false
	}
	return currentHeight >= e.LastActiveHeight+ActivityBlocks
}

// ShouldRemove 综合检查凭信是否应从 UTCO 集中移除。
// 满足以下任一条件则应移除：
//   - 高度截止过期
//   - 11 年不活动（无截止高度的 Credit）
func (e *UTCOEntry) ShouldRemove(currentHeight uint64) bool {
	return e.IsExpired(currentHeight) || e.IsInactive(currentHeight)
}

// --- InfoHash 计算 ---

// CalcInfoHash 计算指纹叶子节点的 InfoHash。
// InfoHash = SHA3-512( TxID || varint(len(flagOutputs)) || flagOutputs )
func CalcInfoHash(txID types.Hash512, flagOutputs []byte) types.Hash512 {
	// 预分配缓冲区：TxID(64) + varint(最大10) + flagOutputs
	buf := make([]byte, 0, types.HashLen+10+len(flagOutputs))
	buf = append(buf, txID[:]...)

	// 写入 flagOutputs 长度的 varint 编码
	var varintBuf [10]byte
	n := types.PutVarint(varintBuf[:], uint64(len(flagOutputs)))
	buf = append(buf, varintBuf[:n]...)

	// 写入 flagOutputs
	buf = append(buf, flagOutputs...)

	return crypto.SHA3_512Sum(buf)
}

// FlagOutputsFromIndices 从输出索引列表生成标记位字节序列。
// 每个位对应一个输出索引，MSB 优先。
// 索引 0 对应第 0 字节的最高位（bit 7），索引 7 对应第 0 字节的最低位（bit 0），
// 索引 8 对应第 1 字节的最高位，以此类推。
func FlagOutputsFromIndices(indices []uint16) []byte {
	if len(indices) == 0 {
		return nil
	}

	// 找到最大索引以确定需要的字节数
	var maxIdx uint16
	for _, idx := range indices {
		if idx > maxIdx {
			maxIdx = idx
		}
	}

	// 分配字节数组
	byteLen := int(maxIdx/8) + 1
	flags := make([]byte, byteLen)

	// 设置对应位（MSB 优先）
	for _, idx := range indices {
		bytePos := idx / 8
		bitPos := 7 - idx%8 // MSB 优先
		flags[bytePos] |= 1 << bitPos
	}

	return flags
}
```

### Step 4: 运行测试验证通过

```bash
go test -v ./internal/utco/ -run "TestOutPoint|TestYearFromHeight|TestUTCOEntry|TestCalcInfoHash|TestFlagOutputsFromIndices"
```

预期输出：全部 PASS。

### Step 5: 提交

```bash
git add internal/utco/entry.go internal/utco/entry_test.go
git commit -m "feat(utco): add UTCOEntry struct with validation, expiry check, and InfoHash computation"
```

---

## Task 6: UTCO 集管理

**Files:**
- Create: `internal/utco/set.go`
- Test: `internal/utco/set_test.go`

本 Task 实现 `UTCOSet` 结构体，提供 UTCO 集的增删查改和准入判定逻辑。双索引结构（OutPoint 索引 + TxID 索引）支持按交易 ID 批量查询。

### Step 1: 写失败测试

创建 `internal/utco/set_test.go`：

```go
package utco

import (
	"testing"

	"github.com/cxio/evidcoin/internal/tx"
	"github.com/cxio/evidcoin/pkg/types"
)

// 辅助函数：创建用于集合测试的 UTCOEntry
func makeTestEntry(txID byte, index uint16) *UTCOEntry {
	var tid types.Hash512
	tid[0] = txID
	return &UTCOEntry{
		OutPoint: OutPoint{
			TxID:  tid,
			Index: index,
		},
		Address:          types.PubKeyHash{0x10},
		Height:           100,
		LastActiveHeight: 100,
		CredConfig:       tx.CredNew,
		LockScript:       []byte{0x01},
	}
}

// --- NewUTCOSet 测试 ---

// 测试创建空集
func TestNewUTCOSet(t *testing.T) {
	s := NewUTCOSet()
	if s == nil {
		t.Fatal("NewUTCOSet() returned nil")
	}
	if s.Count() != 0 {
		t.Errorf("Count() = %d, want 0", s.Count())
	}
}

// --- Add 测试 ---

// 测试添加单个条目
func TestUTCOSet_Add_Single(t *testing.T) {
	s := NewUTCOSet()
	e := makeTestEntry(0x01, 0)

	err := s.Add(e)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if s.Count() != 1 {
		t.Errorf("Count() = %d, want 1", s.Count())
	}
}

// 测试添加多个条目
func TestUTCOSet_Add_Multiple(t *testing.T) {
	s := NewUTCOSet()
	e1 := makeTestEntry(0x01, 0)
	e2 := makeTestEntry(0x01, 1) // 同一交易不同索引
	e3 := makeTestEntry(0x02, 0) // 不同交易

	for _, e := range []*UTCOEntry{e1, e2, e3} {
		if err := s.Add(e); err != nil {
			t.Fatalf("Add() error = %v", err)
		}
	}
	if s.Count() != 3 {
		t.Errorf("Count() = %d, want 3", s.Count())
	}
}

// 测试重复添加返回错误
func TestUTCOSet_Add_Duplicate(t *testing.T) {
	s := NewUTCOSet()
	e := makeTestEntry(0x01, 0)

	if err := s.Add(e); err != nil {
		t.Fatalf("first Add() error = %v", err)
	}

	err := s.Add(e)
	if err == nil {
		t.Error("duplicate Add() should return error")
	}
}

// 测试添加 nil 返回错误
func TestUTCOSet_Add_Nil(t *testing.T) {
	s := NewUTCOSet()
	err := s.Add(nil)
	if err == nil {
		t.Error("Add(nil) should return error")
	}
}

// --- Remove 测试 ---

// 测试移除已存在的条目
func TestUTCOSet_Remove_Existing(t *testing.T) {
	s := NewUTCOSet()
	e := makeTestEntry(0x01, 0)
	_ = s.Add(e)

	removed, err := s.Remove(e.OutPoint)
	if err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if removed.TxID != e.TxID || removed.Index != e.Index {
		t.Error("Remove() returned wrong entry")
	}
	if s.Count() != 0 {
		t.Errorf("Count() after remove = %d, want 0", s.Count())
	}
}

// 测试移除不存在的条目返回错误
func TestUTCOSet_Remove_NotFound(t *testing.T) {
	s := NewUTCOSet()
	op := OutPoint{TxID: types.Hash512{0xFF}, Index: 0}

	_, err := s.Remove(op)
	if err == nil {
		t.Error("Remove() on missing entry should return error")
	}
}

// 测试移除同一交易的部分输出后 TxID 索引正确
func TestUTCOSet_Remove_PartialTx(t *testing.T) {
	s := NewUTCOSet()
	e1 := makeTestEntry(0x01, 0)
	e2 := makeTestEntry(0x01, 1)
	_ = s.Add(e1)
	_ = s.Add(e2)

	_, _ = s.Remove(e1.OutPoint)

	// TxID 索引应仍有一条
	entries := s.GetByTxID(e1.TxID)
	if len(entries) != 1 {
		t.Errorf("GetByTxID() after partial remove = %d entries, want 1", len(entries))
	}
}

// 测试移除同一交易的全部输出后 TxID 索引清理
func TestUTCOSet_Remove_AllFromTx(t *testing.T) {
	s := NewUTCOSet()
	e1 := makeTestEntry(0x01, 0)
	e2 := makeTestEntry(0x01, 1)
	_ = s.Add(e1)
	_ = s.Add(e2)

	_, _ = s.Remove(e1.OutPoint)
	_, _ = s.Remove(e2.OutPoint)

	entries := s.GetByTxID(e1.TxID)
	if len(entries) != 0 {
		t.Errorf("GetByTxID() after full remove = %d entries, want 0", len(entries))
	}
}

// --- Get / Has 测试 ---

// 测试获取已存在的条目
func TestUTCOSet_Get_Existing(t *testing.T) {
	s := NewUTCOSet()
	e := makeTestEntry(0x01, 0)
	_ = s.Add(e)

	got, ok := s.Get(e.OutPoint)
	if !ok {
		t.Fatal("Get() should return true for existing entry")
	}
	if got.TxID != e.TxID {
		t.Error("Get() returned wrong entry")
	}
}

// 测试获取不存在的条目
func TestUTCOSet_Get_NotFound(t *testing.T) {
	s := NewUTCOSet()
	op := OutPoint{TxID: types.Hash512{0xFF}, Index: 0}

	_, ok := s.Get(op)
	if ok {
		t.Error("Get() should return false for missing entry")
	}
}

// 测试 Has 判定
func TestUTCOSet_Has(t *testing.T) {
	s := NewUTCOSet()
	e := makeTestEntry(0x01, 0)
	_ = s.Add(e)

	if !s.Has(e.OutPoint) {
		t.Error("Has() should return true for existing entry")
	}

	missing := OutPoint{TxID: types.Hash512{0xFF}, Index: 0}
	if s.Has(missing) {
		t.Error("Has() should return false for missing entry")
	}
}

// --- GetByTxID 测试 ---

// 测试按 TxID 查询
func TestUTCOSet_GetByTxID(t *testing.T) {
	s := NewUTCOSet()
	e1 := makeTestEntry(0x01, 0)
	e2 := makeTestEntry(0x01, 1)
	e3 := makeTestEntry(0x02, 0)
	_ = s.Add(e1)
	_ = s.Add(e2)
	_ = s.Add(e3)

	entries := s.GetByTxID(e1.TxID)
	if len(entries) != 2 {
		t.Errorf("GetByTxID(0x01) = %d entries, want 2", len(entries))
	}

	entries = s.GetByTxID(e3.TxID)
	if len(entries) != 1 {
		t.Errorf("GetByTxID(0x02) = %d entries, want 1", len(entries))
	}

	var unknown types.Hash512
	unknown[0] = 0xFF
	entries = s.GetByTxID(unknown)
	if len(entries) != 0 {
		t.Errorf("GetByTxID(unknown) = %d entries, want 0", len(entries))
	}
}

// --- ShouldInclude 测试 ---

// 表驱动测试 UTCO 集准入判定
func TestShouldInclude(t *testing.T) {
	tests := []struct {
		name         string
		config       types.OutputConfig
		xferCount    uint16
		hasXferCount bool
		want         bool
	}{
		{
			name:   "credit_basic",
			config: types.OutTypeCredit,
			want:   true,
		},
		{
			name:         "credit_with_xfer_count",
			config:       types.OutTypeCredit,
			xferCount:    10,
			hasXferCount: true,
			want:         true,
		},
		{
			name:         "credit_xfer_count_zero",
			config:       types.OutTypeCredit,
			xferCount:    0,
			hasXferCount: true,
			want:         false,
		},
		{
			name:   "coin_type_excluded",
			config: types.OutTypeCoin,
			want:   false,
		},
		{
			name:   "proof_type_excluded",
			config: types.OutTypeProof,
			want:   false,
		},
		{
			name:   "mediator_type_excluded",
			config: types.OutTypeMediator,
			want:   false,
		},
		{
			name:   "credit_destroy_excluded",
			config: types.OutTypeCredit | types.OutDestroy,
			want:   false,
		},
		{
			name:   "credit_custom_excluded",
			config: types.OutTypeCredit | types.OutCustomClass,
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldInclude(tt.config, tt.xferCount, tt.hasXferCount)
			if got != tt.want {
				t.Errorf("ShouldInclude() = %v, want %v", got, tt.want)
			}
		})
	}
}
```

### Step 2: 运行测试验证失败

```bash
go test -v ./internal/utco/ -run "TestNewUTCOSet|TestUTCOSet|TestShouldInclude"
```

预期输出：编译失败，`UTCOSet`、`NewUTCOSet`、`ShouldInclude` 等未定义。

### Step 3: 写最小实现

创建 `internal/utco/set.go`：

```go
package utco

import (
	"errors"

	"github.com/cxio/evidcoin/pkg/types"
)

// 集合操作错误
var (
	ErrNilEntry       = errors.New("utco set: nil entry")
	ErrDuplicateEntry = errors.New("utco set: duplicate outpoint")
	ErrEntryNotFound  = errors.New("utco set: outpoint not found")
)

// UTCOSet 管理未转移凭信输出的集合。
// 维护两个索引：
//   - entries: OutPoint -> UTCOEntry，主索引
//   - txOutputs: TxID -> (Index -> UTCOEntry)，按交易 ID 的辅助索引
type UTCOSet struct {
	entries   map[OutPoint]*UTCOEntry
	txOutputs map[types.Hash512]map[uint16]*UTCOEntry
}

// NewUTCOSet 创建一个空的 UTCO 集。
func NewUTCOSet() *UTCOSet {
	return &UTCOSet{
		entries:   make(map[OutPoint]*UTCOEntry),
		txOutputs: make(map[types.Hash512]map[uint16]*UTCOEntry),
	}
}

// Add 向 UTCO 集中添加一个条目。
// 如果 OutPoint 已存在，返回错误。
func (s *UTCOSet) Add(entry *UTCOEntry) error {
	if entry == nil {
		return ErrNilEntry
	}

	if _, exists := s.entries[entry.OutPoint]; exists {
		return ErrDuplicateEntry
	}

	// 主索引
	s.entries[entry.OutPoint] = entry

	// TxID 辅助索引
	txMap, ok := s.txOutputs[entry.TxID]
	if !ok {
		txMap = make(map[uint16]*UTCOEntry)
		s.txOutputs[entry.TxID] = txMap
	}
	txMap[entry.Index] = entry

	return nil
}

// Remove 从 UTCO 集中移除指定 OutPoint 的条目。
// 返回被移除的条目，如果不存在则返回错误。
func (s *UTCOSet) Remove(outpoint OutPoint) (*UTCOEntry, error) {
	entry, ok := s.entries[outpoint]
	if !ok {
		return nil, ErrEntryNotFound
	}

	// 从主索引移除
	delete(s.entries, outpoint)

	// 从 TxID 辅助索引移除
	if txMap, ok := s.txOutputs[entry.TxID]; ok {
		delete(txMap, entry.Index)
		// 如果该交易的所有输出都已移除，清理 TxID 索引
		if len(txMap) == 0 {
			delete(s.txOutputs, entry.TxID)
		}
	}

	return entry, nil
}

// Get 获取指定 OutPoint 的条目。
// 返回条目和是否存在的标志。
func (s *UTCOSet) Get(outpoint OutPoint) (*UTCOEntry, bool) {
	entry, ok := s.entries[outpoint]
	return entry, ok
}

// Has 检查指定 OutPoint 是否存在于集合中。
func (s *UTCOSet) Has(outpoint OutPoint) bool {
	_, ok := s.entries[outpoint]
	return ok
}

// Count 返回集合中的条目数量。
func (s *UTCOSet) Count() int {
	return len(s.entries)
}

// GetByTxID 返回指定交易 ID 的所有未转移凭信输出条目。
func (s *UTCOSet) GetByTxID(txID types.Hash512) []*UTCOEntry {
	txMap, ok := s.txOutputs[txID]
	if !ok {
		return nil
	}

	entries := make([]*UTCOEntry, 0, len(txMap))
	for _, e := range txMap {
		entries = append(entries, e)
	}
	return entries
}

// ShouldInclude 判断给定的输出配置是否应纳入 UTCO 集。
// 准入条件：
//   - 必须是 Credit 类型（OutTypeCredit）
//   - 不能是销毁输出（OutDestroy）
//   - 不能是自定义类型（OutCustomClass）
//   - 如果有转移次数且为 0，则不入集
func ShouldInclude(config types.OutputConfig, xferCount uint16, hasXferCount bool) bool {
	// 必须是 Credit 类型
	if config.Type() != types.OutTypeCredit {
		return false
	}

	// 不能是销毁输出
	if config.IsDestroy() {
		return false
	}

	// 不能是自定义类型
	if config.IsCustom() {
		return false
	}

	// 转移次数为 0 不入集
	if hasXferCount && xferCount == 0 {
		return false
	}

	return true
}
```

### Step 4: 运行测试验证通过

```bash
go test -v ./internal/utco/ -run "TestNewUTCOSet|TestUTCOSet|TestShouldInclude"
```

预期输出：全部 PASS。

### Step 5: 提交

```bash
git add internal/utco/set.go internal/utco/set_test.go
git commit -m "feat(utco): add UTCOSet with dual-index management and ShouldInclude admission check"
```

---

## Task 7: 4 级层次哈希指纹

**Files:**
- Create: `internal/utco/fingerprint.go`
- Test: `internal/utco/fingerprint_test.go`

本 Task 实现 UTCO 集的 4 级层次哈希指纹结构，代码独立于 UTXO 包。指纹结构按 年度→TxID[8]→TxID[13]→TxID[18] 四级分组，支持增量更新。

### Step 1: 写失败测试

创建 `internal/utco/fingerprint_test.go`：

```go
package utco

import (
	"testing"

	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// --- Fingerprint 基本操作测试 ---

// 测试创建空指纹
func TestNewFingerprint(t *testing.T) {
	fp := NewFingerprint()
	if fp == nil {
		t.Fatal("NewFingerprint() returned nil")
	}

	root := fp.RootHash()
	if root != (types.Hash512{}) {
		t.Error("empty fingerprint should have zero root hash")
	}
}

// --- LeafInfo 辅助结构测试 ---

// 测试 LeafInfo 的年份字段
func TestLeafInfo_Year(t *testing.T) {
	leaf := LeafInfo{
		TxID:        types.Hash512{0x01},
		Year:        2026,
		InfoHash:    types.Hash512{0xAA},
		FlagOutputs: []byte{0xFF},
	}

	if leaf.Year != 2026 {
		t.Errorf("Year = %d, want 2026", leaf.Year)
	}
}

// --- AddLeaf 测试 ---

// 测试添加单个叶子
func TestFingerprint_AddLeaf(t *testing.T) {
	fp := NewFingerprint()
	txID := types.Hash512{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09}
	flagOutputs := []byte{0xFF}
	infoHash := CalcInfoHash(txID, flagOutputs)

	leaf := LeafInfo{
		TxID:        txID,
		Year:        2026,
		InfoHash:    infoHash,
		FlagOutputs: flagOutputs,
	}

	fp.AddLeaf(leaf)
	fp.Recalculate()

	root := fp.RootHash()
	if root == (types.Hash512{}) {
		t.Error("root hash should not be zero after adding leaf")
	}
}

// 测试添加多个叶子后根哈希变化
func TestFingerprint_AddLeaf_Multiple(t *testing.T) {
	fp := NewFingerprint()

	txID1 := types.Hash512{0x01}
	txID2 := types.Hash512{0x02}
	flags := []byte{0xFF}

	fp.AddLeaf(LeafInfo{TxID: txID1, Year: 2026, InfoHash: CalcInfoHash(txID1, flags), FlagOutputs: flags})
	fp.Recalculate()
	root1 := fp.RootHash()

	fp.AddLeaf(LeafInfo{TxID: txID2, Year: 2026, InfoHash: CalcInfoHash(txID2, flags), FlagOutputs: flags})
	fp.Recalculate()
	root2 := fp.RootHash()

	if root1 == root2 {
		t.Error("root hash should change after adding another leaf")
	}
}

// 测试不同年份的叶子
func TestFingerprint_AddLeaf_DifferentYears(t *testing.T) {
	fp := NewFingerprint()

	txID1 := types.Hash512{0x01}
	txID2 := types.Hash512{0x02}
	flags := []byte{0xFF}

	fp.AddLeaf(LeafInfo{TxID: txID1, Year: 2026, InfoHash: CalcInfoHash(txID1, flags), FlagOutputs: flags})
	fp.AddLeaf(LeafInfo{TxID: txID2, Year: 2027, InfoHash: CalcInfoHash(txID2, flags), FlagOutputs: flags})
	fp.Recalculate()

	root := fp.RootHash()
	if root == (types.Hash512{}) {
		t.Error("root hash should not be zero after adding leaves from different years")
	}

	// 验证年度节点存在
	if fp.YearNodes[2026] == nil {
		t.Error("year 2026 node should exist")
	}
	if fp.YearNodes[2027] == nil {
		t.Error("year 2027 node should exist")
	}
}

// --- RemoveLeaf 测试 ---

// 测试移除叶子后根哈希恢复为零
func TestFingerprint_RemoveLeaf_BackToEmpty(t *testing.T) {
	fp := NewFingerprint()

	txID := types.Hash512{0x01}
	flags := []byte{0xFF}
	leaf := LeafInfo{TxID: txID, Year: 2026, InfoHash: CalcInfoHash(txID, flags), FlagOutputs: flags}

	fp.AddLeaf(leaf)
	fp.Recalculate()

	fp.RemoveLeaf(leaf)
	fp.Recalculate()

	root := fp.RootHash()
	if root != (types.Hash512{}) {
		t.Error("root hash should be zero after removing all leaves")
	}
}

// 测试移除一个叶子后另一个叶子的根哈希
func TestFingerprint_RemoveLeaf_Partial(t *testing.T) {
	fp := NewFingerprint()

	txID1 := types.Hash512{0x01}
	txID2 := types.Hash512{0x02}
	flags := []byte{0xFF}
	leaf1 := LeafInfo{TxID: txID1, Year: 2026, InfoHash: CalcInfoHash(txID1, flags), FlagOutputs: flags}
	leaf2 := LeafInfo{TxID: txID2, Year: 2026, InfoHash: CalcInfoHash(txID2, flags), FlagOutputs: flags}

	// 先只添加 leaf1，记录其根哈希
	fpSingle := NewFingerprint()
	fpSingle.AddLeaf(leaf1)
	fpSingle.Recalculate()
	singleRoot := fpSingle.RootHash()

	// 添加两个再移除第二个
	fp.AddLeaf(leaf1)
	fp.AddLeaf(leaf2)
	fp.RemoveLeaf(leaf2)
	fp.Recalculate()

	if fp.RootHash() != singleRoot {
		t.Error("root hash after remove should match single-leaf fingerprint")
	}
}

// --- UpdateLeaf 测试 ---

// 测试更新叶子后根哈希变化
func TestFingerprint_UpdateLeaf(t *testing.T) {
	fp := NewFingerprint()

	txID := types.Hash512{0x01}
	flags1 := []byte{0xFF}
	flags2 := []byte{0x0F}
	leaf := LeafInfo{TxID: txID, Year: 2026, InfoHash: CalcInfoHash(txID, flags1), FlagOutputs: flags1}

	fp.AddLeaf(leaf)
	fp.Recalculate()
	root1 := fp.RootHash()

	// 更新标记位
	updatedLeaf := LeafInfo{TxID: txID, Year: 2026, InfoHash: CalcInfoHash(txID, flags2), FlagOutputs: flags2}
	fp.UpdateLeaf(updatedLeaf)
	fp.Recalculate()
	root2 := fp.RootHash()

	if root1 == root2 {
		t.Error("root hash should change after updating leaf")
	}
}

// --- Recalculate 确定性测试 ---

// 测试相同叶子集产生相同根哈希
func TestFingerprint_Recalculate_Deterministic(t *testing.T) {
	flags := []byte{0xFF}

	// 按 A、B 顺序添加
	fp1 := NewFingerprint()
	txID1 := types.Hash512{0x01}
	txID2 := types.Hash512{0x02}
	fp1.AddLeaf(LeafInfo{TxID: txID1, Year: 2026, InfoHash: CalcInfoHash(txID1, flags), FlagOutputs: flags})
	fp1.AddLeaf(LeafInfo{TxID: txID2, Year: 2026, InfoHash: CalcInfoHash(txID2, flags), FlagOutputs: flags})
	fp1.Recalculate()

	// 按 B、A 顺序添加
	fp2 := NewFingerprint()
	fp2.AddLeaf(LeafInfo{TxID: txID2, Year: 2026, InfoHash: CalcInfoHash(txID2, flags), FlagOutputs: flags})
	fp2.AddLeaf(LeafInfo{TxID: txID1, Year: 2026, InfoHash: CalcInfoHash(txID1, flags), FlagOutputs: flags})
	fp2.Recalculate()

	if fp1.RootHash() != fp2.RootHash() {
		t.Error("insertion order should not affect root hash")
	}
}

// --- 分支路径测试 ---

// 测试 TxID 的字节位置正确路由
func TestFingerprint_BranchRouting(t *testing.T) {
	fp := NewFingerprint()

	// 构造 TxID 使 byte[8]、byte[13]、byte[18] 各有特定值
	var txID types.Hash512
	txID[0] = 0xAA
	txID[8] = 0x11  // Tx8 分组键
	txID[13] = 0x22 // Tx13 分组键
	txID[18] = 0x33 // Tx18 分组键

	flags := []byte{0xFF}
	leaf := LeafInfo{TxID: txID, Year: 2026, InfoHash: CalcInfoHash(txID, flags), FlagOutputs: flags}

	fp.AddLeaf(leaf)
	fp.Recalculate()

	// 验证年度节点
	yearNode := fp.YearNodes[2026]
	if yearNode == nil {
		t.Fatal("year 2026 node should exist")
	}

	// 验证 Tx8 节点
	tx8Node := yearNode.Tx8Nodes[0x11]
	if tx8Node == nil {
		t.Fatal("Tx8Node at index 0x11 should exist")
	}

	// 验证 Tx13 节点
	tx13Node := tx8Node.Tx13Nodes[0x22]
	if tx13Node == nil {
		t.Fatal("Tx13Node at index 0x22 should exist")
	}

	// 验证 Tx18 节点
	tx18Node := tx13Node.Tx18Nodes[0x33]
	if tx18Node == nil {
		t.Fatal("Tx18Node at index 0x33 should exist")
	}

	// 验证叶子
	if len(tx18Node.InfoHashes) != 1 {
		t.Errorf("Tx18Node should have 1 info hash, got %d", len(tx18Node.InfoHashes))
	}
}

// --- RootHash 计算验证 ---

// 手动验证单个叶子的根哈希计算
func TestFingerprint_RootHash_SingleLeaf(t *testing.T) {
	fp := NewFingerprint()

	var txID types.Hash512
	txID[0] = 0x01
	txID[8] = 0x10
	txID[13] = 0x20
	txID[18] = 0x30

	flags := []byte{0xFF}
	infoHash := CalcInfoHash(txID, flags)
	leaf := LeafInfo{TxID: txID, Year: 2026, InfoHash: infoHash, FlagOutputs: flags}

	fp.AddLeaf(leaf)
	fp.Recalculate()

	// 手动计算预期根哈希：
	// Tx18Hash = SHA3-512(infoHash)
	tx18Hash := crypto.SHA3_512Sum(infoHash[:])
	// Tx13Hash = SHA3-512(tx18Hash)
	tx13Hash := crypto.SHA3_512Sum(tx18Hash[:])
	// Tx8Hash = SHA3-512(tx13Hash)
	tx8Hash := crypto.SHA3_512Sum(tx13Hash[:])
	// YearHash = SHA3-512(tx8Hash)
	yearHash := crypto.SHA3_512Sum(tx8Hash[:])
	// RootHash = SHA3-512(yearHash)
	expectedRoot := crypto.SHA3_512Sum(yearHash[:])

	got := fp.RootHash()
	if got != expectedRoot {
		t.Errorf("RootHash mismatch\ngot:  %x\nwant: %x", got[:8], expectedRoot[:8])
	}
}
```

### Step 2: 运行测试验证失败

```bash
go test -v ./internal/utco/ -run "TestNewFingerprint|TestLeafInfo|TestFingerprint"
```

预期输出：编译失败，`Fingerprint`、`NewFingerprint`、`LeafInfo` 等未定义。

### Step 3: 写最小实现

创建 `internal/utco/fingerprint.go`：

```go
package utco

import (
	"sort"

	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// --- 指纹节点类型定义 ---

// LeafInfo 叶子节点信息，用于指纹树的增删改操作。
type LeafInfo struct {
	TxID        types.Hash512 // 交易 ID
	Year        int           // 所属年份（UTC）
	InfoHash    types.Hash512 // InfoHash = SHA3-512(TxID || varint(len(flagOutputs)) || flagOutputs)
	FlagOutputs []byte        // 标记位字节序列
}

// Tx18Node 第 4 级节点，按 TxID[18] 分组。
// 包含该分组下的所有 InfoHash 叶子节点。
type Tx18Node struct {
	Index      byte                      // TxID[18] 的值
	Hash       types.Hash512             // 分组哈希
	InfoHashes []types.Hash512           // 叶子节点列表
	txIDMap    map[types.Hash512]int     // TxID -> InfoHashes 中的位置（内部索引）
}

// Tx13Node 第 3 级节点，按 TxID[13] 分组。
type Tx13Node struct {
	Index     byte               // TxID[13] 的值
	Hash      types.Hash512      // 分组哈希
	Tx18Nodes map[byte]*Tx18Node // TxID[18] -> Tx18Node
}

// Tx8Node 第 2 级节点，按 TxID[8] 分组。
type Tx8Node struct {
	Index     byte               // TxID[8] 的值
	Hash      types.Hash512      // 分组哈希
	Tx13Nodes map[byte]*Tx13Node // TxID[13] -> Tx13Node
}

// YearNode 第 1 级节点，按年度分组。
type YearNode struct {
	Year     int               // UTC 年份
	Hash     types.Hash512     // 年度哈希
	Tx8Nodes map[byte]*Tx8Node // TxID[8] -> Tx8Node
}

// Fingerprint UTCO 指纹的 4 级层次哈希结构。
//
// 层次结构：
//
//	Root:  SHA3-512( YearHash_1 || YearHash_2 || ... )  （年度按升序排列）
//	Year:  SHA3-512( Tx8Hash_0 || Tx8Hash_1 || ... )    （按 TxID[8] 值升序）
//	Tx8:   SHA3-512( Tx13Hash_0 || ... )                 （按 TxID[13] 值升序）
//	Tx13:  SHA3-512( Tx18Hash_0 || ... )                 （按 TxID[18] 值升序）
//	Tx18:  SHA3-512( InfoHash_1 || InfoHash_2 || ... )   （按 InfoHash 字节序升序）
type Fingerprint struct {
	Root      types.Hash512         // 指纹根哈希
	YearNodes map[int]*YearNode     // 年度 -> 年度节点
}

// NewFingerprint 创建一个空的 UTCO 指纹。
func NewFingerprint() *Fingerprint {
	return &Fingerprint{
		YearNodes: make(map[int]*YearNode),
	}
}

// RootHash 返回当前指纹根哈希。
func (fp *Fingerprint) RootHash() types.Hash512 {
	return fp.Root
}

// AddLeaf 添加一个叶子节点到指纹树中。
// 添加后需调用 Recalculate 更新根哈希。
func (fp *Fingerprint) AddLeaf(leaf LeafInfo) {
	// 获取或创建年度节点
	yearNode := fp.YearNodes[leaf.Year]
	if yearNode == nil {
		yearNode = &YearNode{
			Year:     leaf.Year,
			Tx8Nodes: make(map[byte]*Tx8Node),
		}
		fp.YearNodes[leaf.Year] = yearNode
	}

	// 获取或创建 Tx8 节点
	tx8Key := leaf.TxID[8]
	tx8Node := yearNode.Tx8Nodes[tx8Key]
	if tx8Node == nil {
		tx8Node = &Tx8Node{
			Index:     tx8Key,
			Tx13Nodes: make(map[byte]*Tx13Node),
		}
		yearNode.Tx8Nodes[tx8Key] = tx8Node
	}

	// 获取或创建 Tx13 节点
	tx13Key := leaf.TxID[13]
	tx13Node := tx8Node.Tx13Nodes[tx13Key]
	if tx13Node == nil {
		tx13Node = &Tx13Node{
			Index:     tx13Key,
			Tx18Nodes: make(map[byte]*Tx18Node),
		}
		tx8Node.Tx13Nodes[tx13Key] = tx13Node
	}

	// 获取或创建 Tx18 节点
	tx18Key := leaf.TxID[18]
	tx18Node := tx13Node.Tx18Nodes[tx18Key]
	if tx18Node == nil {
		tx18Node = &Tx18Node{
			Index:   tx18Key,
			txIDMap: make(map[types.Hash512]int),
		}
		tx13Node.Tx18Nodes[tx18Key] = tx18Node
	}

	// 添加 InfoHash 到叶子列表
	tx18Node.InfoHashes = append(tx18Node.InfoHashes, leaf.InfoHash)
	tx18Node.txIDMap[leaf.TxID] = len(tx18Node.InfoHashes) - 1
}

// RemoveLeaf 从指纹树中移除一个叶子节点。
// 移除后需调用 Recalculate 更新根哈希。
func (fp *Fingerprint) RemoveLeaf(leaf LeafInfo) {
	yearNode := fp.YearNodes[leaf.Year]
	if yearNode == nil {
		return
	}

	tx8Node := yearNode.Tx8Nodes[leaf.TxID[8]]
	if tx8Node == nil {
		return
	}

	tx13Node := tx8Node.Tx13Nodes[leaf.TxID[13]]
	if tx13Node == nil {
		return
	}

	tx18Node := tx13Node.Tx18Nodes[leaf.TxID[18]]
	if tx18Node == nil {
		return
	}

	// 从叶子列表中移除
	idx, ok := tx18Node.txIDMap[leaf.TxID]
	if !ok {
		return
	}

	// 用最后一个元素覆盖被删除的位置
	last := len(tx18Node.InfoHashes) - 1
	if idx != last {
		tx18Node.InfoHashes[idx] = tx18Node.InfoHashes[last]
		// 更新被移动元素在 txIDMap 中的位置
		for txID, i := range tx18Node.txIDMap {
			if i == last {
				tx18Node.txIDMap[txID] = idx
				break
			}
		}
	}
	tx18Node.InfoHashes = tx18Node.InfoHashes[:last]
	delete(tx18Node.txIDMap, leaf.TxID)

	// 清理空节点
	if len(tx18Node.InfoHashes) == 0 {
		delete(tx13Node.Tx18Nodes, leaf.TxID[18])
	}
	if len(tx13Node.Tx18Nodes) == 0 {
		delete(tx8Node.Tx13Nodes, leaf.TxID[13])
	}
	if len(tx8Node.Tx13Nodes) == 0 {
		delete(yearNode.Tx8Nodes, leaf.TxID[8])
	}
	if len(yearNode.Tx8Nodes) == 0 {
		delete(fp.YearNodes, leaf.Year)
	}
}

// UpdateLeaf 更新一个已存在的叶子节点。
// 如果叶子不存在则当作新增。
// 更新后需调用 Recalculate 更新根哈希。
func (fp *Fingerprint) UpdateLeaf(leaf LeafInfo) {
	// 先尝试定位现有叶子
	yearNode := fp.YearNodes[leaf.Year]
	if yearNode == nil {
		fp.AddLeaf(leaf)
		return
	}

	tx8Node := yearNode.Tx8Nodes[leaf.TxID[8]]
	if tx8Node == nil {
		fp.AddLeaf(leaf)
		return
	}

	tx13Node := tx8Node.Tx13Nodes[leaf.TxID[13]]
	if tx13Node == nil {
		fp.AddLeaf(leaf)
		return
	}

	tx18Node := tx13Node.Tx18Nodes[leaf.TxID[18]]
	if tx18Node == nil {
		fp.AddLeaf(leaf)
		return
	}

	idx, ok := tx18Node.txIDMap[leaf.TxID]
	if !ok {
		fp.AddLeaf(leaf)
		return
	}

	// 原地更新 InfoHash
	tx18Node.InfoHashes[idx] = leaf.InfoHash
}

// Recalculate 重新计算整个指纹树的所有哈希值。
func (fp *Fingerprint) Recalculate() {
	if len(fp.YearNodes) == 0 {
		fp.Root = types.Hash512{}
		return
	}

	// 收集年份并排序
	years := make([]int, 0, len(fp.YearNodes))
	for y := range fp.YearNodes {
		years = append(years, y)
	}
	sort.Ints(years)

	// 计算各年度哈希
	var rootBuf []byte
	for _, y := range years {
		yearNode := fp.YearNodes[y]
		fp.recalcYearNode(yearNode)
		rootBuf = append(rootBuf, yearNode.Hash[:]...)
	}

	fp.Root = crypto.SHA3_512Sum(rootBuf)
}

// recalcYearNode 重新计算年度节点的哈希。
func (fp *Fingerprint) recalcYearNode(node *YearNode) {
	// 收集 Tx8 键并排序
	keys := make([]byte, 0, len(node.Tx8Nodes))
	for k := range node.Tx8Nodes {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	var buf []byte
	for _, k := range keys {
		tx8Node := node.Tx8Nodes[k]
		fp.recalcTx8Node(tx8Node)
		buf = append(buf, tx8Node.Hash[:]...)
	}

	node.Hash = crypto.SHA3_512Sum(buf)
}

// recalcTx8Node 重新计算 Tx8 节点的哈希。
func (fp *Fingerprint) recalcTx8Node(node *Tx8Node) {
	keys := make([]byte, 0, len(node.Tx13Nodes))
	for k := range node.Tx13Nodes {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	var buf []byte
	for _, k := range keys {
		tx13Node := node.Tx13Nodes[k]
		fp.recalcTx13Node(tx13Node)
		buf = append(buf, tx13Node.Hash[:]...)
	}

	node.Hash = crypto.SHA3_512Sum(buf)
}

// recalcTx13Node 重新计算 Tx13 节点的哈希。
func (fp *Fingerprint) recalcTx13Node(node *Tx13Node) {
	keys := make([]byte, 0, len(node.Tx18Nodes))
	for k := range node.Tx18Nodes {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	var buf []byte
	for _, k := range keys {
		tx18Node := node.Tx18Nodes[k]
		fp.recalcTx18Node(tx18Node)
		buf = append(buf, tx18Node.Hash[:]...)
	}

	node.Hash = crypto.SHA3_512Sum(buf)
}

// recalcTx18Node 重新计算 Tx18 节点的哈希。
// 叶子按 InfoHash 字节序升序排序后拼接计算。
func (fp *Fingerprint) recalcTx18Node(node *Tx18Node) {
	if len(node.InfoHashes) == 0 {
		node.Hash = types.Hash512{}
		return
	}

	// 复制后排序，避免影响 txIDMap 的索引
	sorted := make([]types.Hash512, len(node.InfoHashes))
	copy(sorted, node.InfoHashes)

	sort.Slice(sorted, func(i, j int) bool {
		for k := 0; k < types.HashLen; k++ {
			if sorted[i][k] != sorted[j][k] {
				return sorted[i][k] < sorted[j][k]
			}
		}
		return false
	})

	var buf []byte
	for _, h := range sorted {
		buf = append(buf, h[:]...)
	}

	node.Hash = crypto.SHA3_512Sum(buf)
}

// recalcBranch 增量重算从指定叶子到根的分支路径。
// 适用于单个叶子变更后的局部更新。
func (fp *Fingerprint) recalcBranch(leaf LeafInfo) {
	yearNode := fp.YearNodes[leaf.Year]
	if yearNode == nil {
		// 年度节点不存在，需要全量重算根
		fp.recalcRoot()
		return
	}

	tx8Node := yearNode.Tx8Nodes[leaf.TxID[8]]
	if tx8Node != nil {
		tx13Node := tx8Node.Tx13Nodes[leaf.TxID[13]]
		if tx13Node != nil {
			tx18Node := tx13Node.Tx18Nodes[leaf.TxID[18]]
			if tx18Node != nil {
				fp.recalcTx18Node(tx18Node)
			}
			fp.recalcTx13Node(tx13Node)
		}
		fp.recalcTx8Node(tx8Node)
	}
	fp.recalcYearNode(yearNode)
	fp.recalcRoot()
}

// recalcRoot 重新计算根哈希。
func (fp *Fingerprint) recalcRoot() {
	if len(fp.YearNodes) == 0 {
		fp.Root = types.Hash512{}
		return
	}

	years := make([]int, 0, len(fp.YearNodes))
	for y := range fp.YearNodes {
		years = append(years, y)
	}
	sort.Ints(years)

	var buf []byte
	for _, y := range years {
		buf = append(buf, fp.YearNodes[y].Hash[:]...)
	}

	fp.Root = crypto.SHA3_512Sum(buf)
}
```

### Step 4: 运行测试验证通过

```bash
go test -v ./internal/utco/ -run "TestNewFingerprint|TestLeafInfo|TestFingerprint"
```

预期输出：全部 PASS。

### Step 5: 提交

```bash
git add internal/utco/fingerprint.go internal/utco/fingerprint_test.go
git commit -m "feat(utco): add 4-level hierarchical hash fingerprint for UTCO set"
```

---

## Task 8: 凭信过期索引

**Files:**
- Create: `internal/utco/expiry.go`
- Test: `internal/utco/expiry_test.go`

本 Task 实现 UTCO 独有的过期索引（`ExpiryIndex`），支持硬过期（高度截止）和活动过期（11 年规则）两种类型。提供注册、取消注册、按高度查询过期记录、批量清理等功能。

### Step 1: 写失败测试

创建 `internal/utco/expiry_test.go`：

```go
package utco

import (
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// --- ExpiryType 常量测试 ---

func TestExpiryType_Constants(t *testing.T) {
	if ExpiryHard != 1 {
		t.Errorf("ExpiryHard = %d, want 1", ExpiryHard)
	}
	if ExpiryActivity != 2 {
		t.Errorf("ExpiryActivity = %d, want 2", ExpiryActivity)
	}
}

// --- NewExpiryIndex 测试 ---

func TestNewExpiryIndex(t *testing.T) {
	idx := NewExpiryIndex()
	if idx == nil {
		t.Fatal("NewExpiryIndex() returned nil")
	}
	if idx.Count() != 0 {
		t.Errorf("Count() = %d, want 0", idx.Count())
	}
}

// --- Register 测试 ---

// 测试注册单个过期记录
func TestExpiryIndex_Register_Single(t *testing.T) {
	idx := NewExpiryIndex()
	op := OutPoint{TxID: types.Hash512{0x01}, Index: 0}

	idx.Register(op, 1000, ExpiryHard)

	if idx.Count() != 1 {
		t.Errorf("Count() = %d, want 1", idx.Count())
	}
	if !idx.HasHeight(1000) {
		t.Error("HasHeight(1000) should be true")
	}
}

// 测试注册多个过期记录（同一高度）
func TestExpiryIndex_Register_SameHeight(t *testing.T) {
	idx := NewExpiryIndex()
	op1 := OutPoint{TxID: types.Hash512{0x01}, Index: 0}
	op2 := OutPoint{TxID: types.Hash512{0x02}, Index: 0}

	idx.Register(op1, 1000, ExpiryHard)
	idx.Register(op2, 1000, ExpiryActivity)

	if idx.Count() != 2 {
		t.Errorf("Count() = %d, want 2", idx.Count())
	}

	records := idx.GetExpiredAt(1000)
	if len(records) != 2 {
		t.Errorf("GetExpiredAt(1000) = %d records, want 2", len(records))
	}
}

// 测试注册不同高度的过期记录
func TestExpiryIndex_Register_DifferentHeights(t *testing.T) {
	idx := NewExpiryIndex()
	op1 := OutPoint{TxID: types.Hash512{0x01}, Index: 0}
	op2 := OutPoint{TxID: types.Hash512{0x02}, Index: 0}

	idx.Register(op1, 1000, ExpiryHard)
	idx.Register(op2, 2000, ExpiryActivity)

	if idx.Count() != 2 {
		t.Errorf("Count() = %d, want 2", idx.Count())
	}
	if !idx.HasHeight(1000) {
		t.Error("HasHeight(1000) should be true")
	}
	if !idx.HasHeight(2000) {
		t.Error("HasHeight(2000) should be true")
	}
}

// --- Unregister 测试 ---

// 测试取消注册
func TestExpiryIndex_Unregister(t *testing.T) {
	idx := NewExpiryIndex()
	op := OutPoint{TxID: types.Hash512{0x01}, Index: 0}

	idx.Register(op, 1000, ExpiryHard)
	idx.Unregister(op, 1000)

	if idx.Count() != 0 {
		t.Errorf("Count() after unregister = %d, want 0", idx.Count())
	}
	if idx.HasHeight(1000) {
		t.Error("HasHeight(1000) should be false after removing all records")
	}
}

// 测试取消一个，保留同高度的另一个
func TestExpiryIndex_Unregister_Partial(t *testing.T) {
	idx := NewExpiryIndex()
	op1 := OutPoint{TxID: types.Hash512{0x01}, Index: 0}
	op2 := OutPoint{TxID: types.Hash512{0x02}, Index: 0}

	idx.Register(op1, 1000, ExpiryHard)
	idx.Register(op2, 1000, ExpiryActivity)

	idx.Unregister(op1, 1000)

	if idx.Count() != 1 {
		t.Errorf("Count() = %d, want 1", idx.Count())
	}

	records := idx.GetExpiredAt(1000)
	if len(records) != 1 {
		t.Errorf("GetExpiredAt(1000) = %d records, want 1", len(records))
	}
	if records[0].OutPoint != op2 {
		t.Error("remaining record should be op2")
	}
}

// 测试取消不存在的记录（静默忽略）
func TestExpiryIndex_Unregister_NotFound(t *testing.T) {
	idx := NewExpiryIndex()
	op := OutPoint{TxID: types.Hash512{0x01}, Index: 0}

	// 不应 panic
	idx.Unregister(op, 1000)

	if idx.Count() != 0 {
		t.Errorf("Count() = %d, want 0", idx.Count())
	}
}

// --- GetExpiredAt 测试 ---

// 测试查询空高度
func TestExpiryIndex_GetExpiredAt_Empty(t *testing.T) {
	idx := NewExpiryIndex()

	records := idx.GetExpiredAt(1000)
	if len(records) != 0 {
		t.Errorf("GetExpiredAt(1000) = %d records, want 0", len(records))
	}
}

// 测试查询指定高度
func TestExpiryIndex_GetExpiredAt(t *testing.T) {
	idx := NewExpiryIndex()
	op1 := OutPoint{TxID: types.Hash512{0x01}, Index: 0}
	op2 := OutPoint{TxID: types.Hash512{0x02}, Index: 0}
	op3 := OutPoint{TxID: types.Hash512{0x03}, Index: 0}

	idx.Register(op1, 100, ExpiryHard)
	idx.Register(op2, 200, ExpiryActivity)
	idx.Register(op3, 100, ExpiryActivity)

	records := idx.GetExpiredAt(100)
	if len(records) != 2 {
		t.Errorf("GetExpiredAt(100) = %d records, want 2", len(records))
	}

	records = idx.GetExpiredAt(200)
	if len(records) != 1 {
		t.Errorf("GetExpiredAt(200) = %d records, want 1", len(records))
	}

	records = idx.GetExpiredAt(300)
	if len(records) != 0 {
		t.Errorf("GetExpiredAt(300) = %d records, want 0", len(records))
	}
}

// --- CleanupTo 测试 ---

// 测试批量清理
func TestExpiryIndex_CleanupTo(t *testing.T) {
	idx := NewExpiryIndex()
	op1 := OutPoint{TxID: types.Hash512{0x01}, Index: 0}
	op2 := OutPoint{TxID: types.Hash512{0x02}, Index: 0}
	op3 := OutPoint{TxID: types.Hash512{0x03}, Index: 0}
	op4 := OutPoint{TxID: types.Hash512{0x04}, Index: 0}

	idx.Register(op1, 100, ExpiryHard)
	idx.Register(op2, 200, ExpiryActivity)
	idx.Register(op3, 300, ExpiryHard)
	idx.Register(op4, 400, ExpiryActivity)

	// 清理 <= 200 的所有记录
	cleaned := idx.CleanupTo(200)
	if len(cleaned) != 2 {
		t.Errorf("CleanupTo(200) = %d records, want 2", len(cleaned))
	}
	if idx.Count() != 2 {
		t.Errorf("Count() after cleanup = %d, want 2", idx.Count())
	}
	if idx.HasHeight(100) {
		t.Error("HasHeight(100) should be false after cleanup")
	}
	if idx.HasHeight(200) {
		t.Error("HasHeight(200) should be false after cleanup")
	}
	if !idx.HasHeight(300) {
		t.Error("HasHeight(300) should be true after cleanup")
	}
}

// 测试清理高度大于所有记录
func TestExpiryIndex_CleanupTo_All(t *testing.T) {
	idx := NewExpiryIndex()
	op1 := OutPoint{TxID: types.Hash512{0x01}, Index: 0}
	op2 := OutPoint{TxID: types.Hash512{0x02}, Index: 0}

	idx.Register(op1, 100, ExpiryHard)
	idx.Register(op2, 200, ExpiryActivity)

	cleaned := idx.CleanupTo(999)
	if len(cleaned) != 2 {
		t.Errorf("CleanupTo(999) = %d records, want 2", len(cleaned))
	}
	if idx.Count() != 0 {
		t.Errorf("Count() after full cleanup = %d, want 0", idx.Count())
	}
}

// 测试清理空索引
func TestExpiryIndex_CleanupTo_Empty(t *testing.T) {
	idx := NewExpiryIndex()

	cleaned := idx.CleanupTo(999)
	if len(cleaned) != 0 {
		t.Errorf("CleanupTo(999) on empty = %d records, want 0", len(cleaned))
	}
}

// --- HasHeight 测试 ---

func TestExpiryIndex_HasHeight(t *testing.T) {
	idx := NewExpiryIndex()

	if idx.HasHeight(100) {
		t.Error("HasHeight(100) should be false on empty index")
	}

	op := OutPoint{TxID: types.Hash512{0x01}, Index: 0}
	idx.Register(op, 100, ExpiryHard)

	if !idx.HasHeight(100) {
		t.Error("HasHeight(100) should be true after register")
	}
}

// --- 11 年活动规则集成测试 ---

// 测试完整的活动过期流程：注册 -> 转移 -> 重新注册
func TestExpiryIndex_ActivityRule_Lifecycle(t *testing.T) {
	idx := NewExpiryIndex()
	op := OutPoint{TxID: types.Hash512{0x01}, Index: 0}

	// 凭信在高度 1000 创建，无截止高度
	// 计算首次活动截止高度
	initialDeadline := uint64(1000) + ActivityBlocks
	idx.Register(op, initialDeadline, ExpiryActivity)

	if idx.Count() != 1 {
		t.Fatalf("Count() = %d, want 1", idx.Count())
	}

	// 凭信在高度 500000 被转移（早于截止）
	// 取消旧注册
	idx.Unregister(op, initialDeadline)
	if idx.Count() != 0 {
		t.Fatalf("Count() after unregister = %d, want 0", idx.Count())
	}

	// 以新的 lastActiveHeight 重新注册
	newDeadline := uint64(500000) + ActivityBlocks
	idx.Register(op, newDeadline, ExpiryActivity)

	if idx.Count() != 1 {
		t.Errorf("Count() after re-register = %d, want 1", idx.Count())
	}
	if !idx.HasHeight(newDeadline) {
		t.Errorf("HasHeight(%d) should be true", newDeadline)
	}
	if idx.HasHeight(initialDeadline) {
		t.Errorf("HasHeight(%d) should be false (old deadline)", initialDeadline)
	}
}

// 测试 ExpiryRecord 类型字段
func TestExpiryRecord_Type(t *testing.T) {
	idx := NewExpiryIndex()
	op1 := OutPoint{TxID: types.Hash512{0x01}, Index: 0}
	op2 := OutPoint{TxID: types.Hash512{0x02}, Index: 0}

	idx.Register(op1, 100, ExpiryHard)
	idx.Register(op2, 100, ExpiryActivity)

	records := idx.GetExpiredAt(100)
	if len(records) != 2 {
		t.Fatalf("GetExpiredAt(100) = %d records, want 2", len(records))
	}

	// 验证类型字段被正确保存
	foundHard := false
	foundActivity := false
	for _, r := range records {
		if r.Type == ExpiryHard {
			foundHard = true
		}
		if r.Type == ExpiryActivity {
			foundActivity = true
		}
	}
	if !foundHard {
		t.Error("should have ExpiryHard record")
	}
	if !foundActivity {
		t.Error("should have ExpiryActivity record")
	}
}
```

### Step 2: 运行测试验证失败

```bash
go test -v ./internal/utco/ -run "TestExpiryType|TestNewExpiryIndex|TestExpiryIndex|TestExpiryRecord"
```

预期输出：编译失败，`ExpiryType`、`ExpiryIndex`、`ExpiryRecord` 等未定义。

### Step 3: 写最小实现

创建 `internal/utco/expiry.go`：

```go
package utco

import (
	"sort"
)

// --- 过期类型 ---

// ExpiryType 过期类型。
type ExpiryType byte

const (
	// ExpiryHard 硬过期：高度截止到达。
	ExpiryHard ExpiryType = 1

	// ExpiryActivity 活动过期：11 年不活动。
	ExpiryActivity ExpiryType = 2
)

// --- 过期记录 ---

// ExpiryRecord 一条过期记录，关联一个 OutPoint 与其过期类型。
type ExpiryRecord struct {
	OutPoint OutPoint   // 关联的输出点
	Type     ExpiryType // 过期类型
}

// --- 过期索引 ---

// ExpiryIndex 凭信过期索引。
// 按区块高度索引过期记录，支持：
//   - 硬过期（高度截止）：Credit 设置了 ExpiryHeight
//   - 活动过期（11 年规则）：无截止高度的 Credit，deadline = lastActiveHeight + ActivityBlocks
//
// 当 Credit 被转移时，需取消旧的活动过期注册，以新的 lastActiveHeight 重新注册。
type ExpiryIndex struct {
	index map[uint64][]ExpiryRecord // height -> 该高度的过期记录列表
	count int                       // 记录总数
}

// NewExpiryIndex 创建一个空的过期索引。
func NewExpiryIndex() *ExpiryIndex {
	return &ExpiryIndex{
		index: make(map[uint64][]ExpiryRecord),
	}
}

// Register 注册一条过期记录。
// 将指定的 OutPoint 在给定高度以给定类型注册。
//
// 对于硬过期：expiryHeight 为 Credit 的 ExpiryHeight。
// 对于活动过期：expiryHeight = lastActiveHeight + ActivityBlocks。
func (ei *ExpiryIndex) Register(outpoint OutPoint, expiryHeight uint64, typ ExpiryType) {
	record := ExpiryRecord{
		OutPoint: outpoint,
		Type:     typ,
	}
	ei.index[expiryHeight] = append(ei.index[expiryHeight], record)
	ei.count++
}

// Unregister 取消注册指定 OutPoint 在指定高度的过期记录。
// 如果记录不存在，静默忽略。
// 当 Credit 被转移时，用于取消旧的活动过期注册。
func (ei *ExpiryIndex) Unregister(outpoint OutPoint, expiryHeight uint64) {
	records, ok := ei.index[expiryHeight]
	if !ok {
		return
	}

	// 查找并移除匹配的记录
	for i, r := range records {
		if r.OutPoint == outpoint {
			// 用最后一个元素覆盖被删除的位置
			records[i] = records[len(records)-1]
			records = records[:len(records)-1]
			ei.count--

			if len(records) == 0 {
				delete(ei.index, expiryHeight)
			} else {
				ei.index[expiryHeight] = records
			}
			return
		}
	}
}

// GetExpiredAt 返回指定高度的所有过期记录。
// 返回的切片是内部数据的副本，可安全修改。
func (ei *ExpiryIndex) GetExpiredAt(height uint64) []ExpiryRecord {
	records, ok := ei.index[height]
	if !ok {
		return nil
	}

	// 返回副本
	result := make([]ExpiryRecord, len(records))
	copy(result, records)
	return result
}

// CleanupTo 返回并清理所有 <= height 的过期记录。
// 按高度升序返回所有被清理的记录。
func (ei *ExpiryIndex) CleanupTo(height uint64) []ExpiryRecord {
	if len(ei.index) == 0 {
		return nil
	}

	// 收集需要清理的高度
	heights := make([]uint64, 0)
	for h := range ei.index {
		if h <= height {
			heights = append(heights, h)
		}
	}

	if len(heights) == 0 {
		return nil
	}

	// 按高度排序
	sort.Slice(heights, func(i, j int) bool { return heights[i] < heights[j] })

	// 收集并删除记录
	var result []ExpiryRecord
	for _, h := range heights {
		records := ei.index[h]
		result = append(result, records...)
		ei.count -= len(records)
		delete(ei.index, h)
	}

	return result
}

// Count 返回索引中的记录总数。
func (ei *ExpiryIndex) Count() int {
	return ei.count
}

// HasHeight 检查指定高度是否有过期记录。
func (ei *ExpiryIndex) HasHeight(height uint64) bool {
	records, ok := ei.index[height]
	return ok && len(records) > 0
}
```

### Step 4: 运行测试验证通过

```bash
go test -v ./internal/utco/ -run "TestExpiryType|TestNewExpiryIndex|TestExpiryIndex|TestExpiryRecord"
```

预期输出：全部 PASS。

### Step 5: 提交

```bash
git add internal/utco/expiry.go internal/utco/expiry_test.go
git commit -m "feat(utco): add ExpiryIndex for hard expiry and 11-year activity rule tracking"
```

---

## Task 9: UTCO 缓存

**Files:**
- Create: `internal/utco/cache.go`
- Test: `internal/utco/cache_test.go`

本 Task 实现 `UTCOCache` 结构体，作为 UTCO 集的写入缓冲层。在区块处理过程中记录新增和花费的条目，支持原子性提交和回滚操作。

### Step 1: 写失败测试

创建 `internal/utco/cache_test.go`：

```go
package utco

import (
	"testing"

	"github.com/cxio/evidcoin/internal/tx"
	"github.com/cxio/evidcoin/pkg/types"
)

// 辅助函数：创建用于缓存测试的基础集合
func makeCacheTestBase() *UTCOSet {
	s := NewUTCOSet()

	// 添加两个基础条目
	e1 := &UTCOEntry{
		OutPoint: OutPoint{
			TxID:  types.Hash512{0x01},
			Index: 0,
		},
		Address:          types.PubKeyHash{0x10},
		Height:           100,
		LastActiveHeight: 100,
		CredConfig:       tx.CredNew,
		LockScript:       []byte{0x01},
	}
	e2 := &UTCOEntry{
		OutPoint: OutPoint{
			TxID:  types.Hash512{0x02},
			Index: 0,
		},
		Address:          types.PubKeyHash{0x20},
		Height:           200,
		LastActiveHeight: 200,
		CredConfig:       tx.CredNew | tx.CredHasXferCount,
		XferCount:        10,
		LockScript:       []byte{0x02},
	}
	_ = s.Add(e1)
	_ = s.Add(e2)
	return s
}

// --- NewUTCOCache 测试 ---

func TestNewUTCOCache(t *testing.T) {
	base := makeCacheTestBase()
	cache := NewUTCOCache(base)

	if cache == nil {
		t.Fatal("NewUTCOCache() returned nil")
	}
}

// --- Cache.Get 测试 ---

// 测试从基础集合获取
func TestUTCOCache_Get_FromBase(t *testing.T) {
	base := makeCacheTestBase()
	cache := NewUTCOCache(base)

	op := OutPoint{TxID: types.Hash512{0x01}, Index: 0}
	entry, ok := cache.Get(op)
	if !ok {
		t.Fatal("Get() should find entry from base set")
	}
	if entry.TxID != op.TxID {
		t.Error("Get() returned wrong entry")
	}
}

// 测试获取新增的条目
func TestUTCOCache_Get_FromAdded(t *testing.T) {
	base := makeCacheTestBase()
	cache := NewUTCOCache(base)

	newEntry := &UTCOEntry{
		OutPoint: OutPoint{
			TxID:  types.Hash512{0x03},
			Index: 0,
		},
		Address:          types.PubKeyHash{0x30},
		Height:           300,
		LastActiveHeight: 300,
		CredConfig:       tx.CredNew,
		LockScript:       []byte{0x03},
	}
	err := cache.Add(newEntry)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	entry, ok := cache.Get(newEntry.OutPoint)
	if !ok {
		t.Fatal("Get() should find added entry")
	}
	if entry.Height != 300 {
		t.Errorf("Height = %d, want 300", entry.Height)
	}
}

// 测试获取已花费的条目（应不存在）
func TestUTCOCache_Get_Spent(t *testing.T) {
	base := makeCacheTestBase()
	cache := NewUTCOCache(base)

	op := OutPoint{TxID: types.Hash512{0x01}, Index: 0}
	_, err := cache.Spend(op)
	if err != nil {
		t.Fatalf("Spend() error = %v", err)
	}

	_, ok := cache.Get(op)
	if ok {
		t.Error("Get() should not find spent entry")
	}
}

// 测试获取不存在的条目
func TestUTCOCache_Get_NotFound(t *testing.T) {
	base := makeCacheTestBase()
	cache := NewUTCOCache(base)

	op := OutPoint{TxID: types.Hash512{0xFF}, Index: 0}
	_, ok := cache.Get(op)
	if ok {
		t.Error("Get() should return false for non-existent entry")
	}
}

// --- Cache.Has 测试 ---

func TestUTCOCache_Has(t *testing.T) {
	base := makeCacheTestBase()
	cache := NewUTCOCache(base)

	// 基础集合中的条目
	if !cache.Has(OutPoint{TxID: types.Hash512{0x01}, Index: 0}) {
		t.Error("Has() should be true for base entry")
	}

	// 不存在的条目
	if cache.Has(OutPoint{TxID: types.Hash512{0xFF}, Index: 0}) {
		t.Error("Has() should be false for non-existent entry")
	}
}

// --- Cache.Add 测试 ---

// 测试添加新条目
func TestUTCOCache_Add(t *testing.T) {
	base := makeCacheTestBase()
	cache := NewUTCOCache(base)

	newEntry := &UTCOEntry{
		OutPoint: OutPoint{
			TxID:  types.Hash512{0x03},
			Index: 0,
		},
		Address:          types.PubKeyHash{0x30},
		Height:           300,
		LastActiveHeight: 300,
		CredConfig:       tx.CredNew,
		LockScript:       []byte{0x03},
	}

	err := cache.Add(newEntry)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	if !cache.Has(newEntry.OutPoint) {
		t.Error("Has() should be true after Add()")
	}
}

// 测试重复添加返回错误
func TestUTCOCache_Add_DuplicateInBase(t *testing.T) {
	base := makeCacheTestBase()
	cache := NewUTCOCache(base)

	duplicate := &UTCOEntry{
		OutPoint: OutPoint{
			TxID:  types.Hash512{0x01}, // 与基础集合冲突
			Index: 0,
		},
		Address:          types.PubKeyHash{0x10},
		Height:           100,
		LastActiveHeight: 100,
		CredConfig:       tx.CredNew,
		LockScript:       []byte{0x01},
	}

	err := cache.Add(duplicate)
	if err == nil {
		t.Error("Add() should return error for duplicate in base set")
	}
}

// 测试添加 nil 返回错误
func TestUTCOCache_Add_Nil(t *testing.T) {
	base := makeCacheTestBase()
	cache := NewUTCOCache(base)

	err := cache.Add(nil)
	if err == nil {
		t.Error("Add(nil) should return error")
	}
}

// 测试添加后花费再添加（同一 OutPoint）
func TestUTCOCache_Add_AfterSpend(t *testing.T) {
	base := makeCacheTestBase()
	cache := NewUTCOCache(base)

	op := OutPoint{TxID: types.Hash512{0x01}, Index: 0}

	// 先花费
	_, err := cache.Spend(op)
	if err != nil {
		t.Fatalf("Spend() error = %v", err)
	}

	// 再添加回来（不同条目但相同 OutPoint——模拟重建场景）
	reEntry := &UTCOEntry{
		OutPoint:         op,
		Address:          types.PubKeyHash{0x99},
		Height:           500,
		LastActiveHeight: 500,
		CredConfig:       tx.CredNew,
		LockScript:       []byte{0x99},
	}
	err = cache.Add(reEntry)
	if err != nil {
		t.Fatalf("Add() after Spend() error = %v", err)
	}

	entry, ok := cache.Get(op)
	if !ok {
		t.Fatal("Get() should find re-added entry")
	}
	if entry.Height != 500 {
		t.Errorf("Height = %d, want 500", entry.Height)
	}
}

// --- Cache.Spend 测试 ---

// 测试花费基础集合中的条目
func TestUTCOCache_Spend_FromBase(t *testing.T) {
	base := makeCacheTestBase()
	cache := NewUTCOCache(base)

	op := OutPoint{TxID: types.Hash512{0x01}, Index: 0}
	entry, err := cache.Spend(op)
	if err != nil {
		t.Fatalf("Spend() error = %v", err)
	}
	if entry.TxID != op.TxID {
		t.Error("Spend() returned wrong entry")
	}

	// 花费后不可见
	if cache.Has(op) {
		t.Error("Has() should be false after Spend()")
	}
}

// 测试花费新增的条目
func TestUTCOCache_Spend_FromAdded(t *testing.T) {
	base := makeCacheTestBase()
	cache := NewUTCOCache(base)

	newEntry := &UTCOEntry{
		OutPoint: OutPoint{
			TxID:  types.Hash512{0x03},
			Index: 0,
		},
		Address:          types.PubKeyHash{0x30},
		Height:           300,
		LastActiveHeight: 300,
		CredConfig:       tx.CredNew,
		LockScript:       []byte{0x03},
	}
	_ = cache.Add(newEntry)

	entry, err := cache.Spend(newEntry.OutPoint)
	if err != nil {
		t.Fatalf("Spend() error = %v", err)
	}
	if entry.Height != 300 {
		t.Errorf("Spend() returned entry with Height = %d, want 300", entry.Height)
	}
}

// 测试花费不存在的条目返回错误
func TestUTCOCache_Spend_NotFound(t *testing.T) {
	base := makeCacheTestBase()
	cache := NewUTCOCache(base)

	op := OutPoint{TxID: types.Hash512{0xFF}, Index: 0}
	_, err := cache.Spend(op)
	if err == nil {
		t.Error("Spend() should return error for non-existent entry")
	}
}

// 测试重复花费返回错误
func TestUTCOCache_Spend_Double(t *testing.T) {
	base := makeCacheTestBase()
	cache := NewUTCOCache(base)

	op := OutPoint{TxID: types.Hash512{0x01}, Index: 0}
	_, _ = cache.Spend(op)

	_, err := cache.Spend(op)
	if err == nil {
		t.Error("double Spend() should return error")
	}
}

// --- Commit 测试 ---

// 测试提交后基础集合反映变更
func TestUTCOCache_Commit(t *testing.T) {
	base := makeCacheTestBase()
	cache := NewUTCOCache(base)

	// 花费 0x01
	spentOp := OutPoint{TxID: types.Hash512{0x01}, Index: 0}
	_, _ = cache.Spend(spentOp)

	// 添加 0x03
	newEntry := &UTCOEntry{
		OutPoint: OutPoint{
			TxID:  types.Hash512{0x03},
			Index: 0,
		},
		Address:          types.PubKeyHash{0x30},
		Height:           300,
		LastActiveHeight: 300,
		CredConfig:       tx.CredNew,
		LockScript:       []byte{0x03},
	}
	_ = cache.Add(newEntry)

	// 提交
	err := cache.Commit()
	if err != nil {
		t.Fatalf("Commit() error = %v", err)
	}

	// 基础集合应反映变更
	if base.Has(spentOp) {
		t.Error("base should not have spent entry after commit")
	}
	if !base.Has(newEntry.OutPoint) {
		t.Error("base should have added entry after commit")
	}

	// 缓存应被清空
	added := cache.AddedEntries()
	if len(added) != 0 {
		t.Errorf("AddedEntries() after commit = %d, want 0", len(added))
	}
	spent := cache.SpentEntries()
	if len(spent) != 0 {
		t.Errorf("SpentEntries() after commit = %d, want 0", len(spent))
	}
}

// --- Rollback 测试 ---

// 测试回滚后缓存被清空，基础集合不变
func TestUTCOCache_Rollback(t *testing.T) {
	base := makeCacheTestBase()
	baseCount := base.Count()
	cache := NewUTCOCache(base)

	// 做一些变更
	spentOp := OutPoint{TxID: types.Hash512{0x01}, Index: 0}
	_, _ = cache.Spend(spentOp)

	newEntry := &UTCOEntry{
		OutPoint: OutPoint{
			TxID:  types.Hash512{0x03},
			Index: 0,
		},
		Address:          types.PubKeyHash{0x30},
		Height:           300,
		LastActiveHeight: 300,
		CredConfig:       tx.CredNew,
		LockScript:       []byte{0x03},
	}
	_ = cache.Add(newEntry)

	// 回滚
	cache.Rollback()

	// 基础集合不变
	if base.Count() != baseCount {
		t.Errorf("base Count() = %d, want %d after rollback", base.Count(), baseCount)
	}
	if !base.Has(spentOp) {
		t.Error("base should still have entry after rollback")
	}
	if base.Has(newEntry.OutPoint) {
		t.Error("base should not have added entry after rollback")
	}

	// 缓存被清空
	if len(cache.AddedEntries()) != 0 {
		t.Error("AddedEntries() should be empty after rollback")
	}
	if len(cache.SpentEntries()) != 0 {
		t.Error("SpentEntries() should be empty after rollback")
	}

	// 缓存可以继续使用
	if !cache.Has(spentOp) {
		t.Error("cache should see base entry after rollback")
	}
}

// --- AddedEntries / SpentEntries 测试 ---

func TestUTCOCache_AddedEntries(t *testing.T) {
	base := makeCacheTestBase()
	cache := NewUTCOCache(base)

	// 初始为空
	if len(cache.AddedEntries()) != 0 {
		t.Error("AddedEntries() should be empty initially")
	}

	// 添加后有内容
	newEntry := &UTCOEntry{
		OutPoint: OutPoint{
			TxID:  types.Hash512{0x03},
			Index: 0,
		},
		Address:          types.PubKeyHash{0x30},
		Height:           300,
		LastActiveHeight: 300,
		CredConfig:       tx.CredNew,
		LockScript:       []byte{0x03},
	}
	_ = cache.Add(newEntry)

	added := cache.AddedEntries()
	if len(added) != 1 {
		t.Errorf("AddedEntries() = %d, want 1", len(added))
	}
}

func TestUTCOCache_SpentEntries(t *testing.T) {
	base := makeCacheTestBase()
	cache := NewUTCOCache(base)

	// 初始为空
	if len(cache.SpentEntries()) != 0 {
		t.Error("SpentEntries() should be empty initially")
	}

	// 花费后有内容
	op := OutPoint{TxID: types.Hash512{0x01}, Index: 0}
	_, _ = cache.Spend(op)

	spent := cache.SpentEntries()
	if len(spent) != 1 {
		t.Errorf("SpentEntries() = %d, want 1", len(spent))
	}
}

// --- 边界场景测试 ---

// 测试空基础集合的缓存
func TestUTCOCache_EmptyBase(t *testing.T) {
	base := NewUTCOSet()
	cache := NewUTCOCache(base)

	// 添加到空基础
	newEntry := &UTCOEntry{
		OutPoint: OutPoint{
			TxID:  types.Hash512{0x01},
			Index: 0,
		},
		Address:          types.PubKeyHash{0x10},
		Height:           100,
		LastActiveHeight: 100,
		CredConfig:       tx.CredNew,
		LockScript:       []byte{0x01},
	}
	err := cache.Add(newEntry)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	// 提交
	err = cache.Commit()
	if err != nil {
		t.Fatalf("Commit() error = %v", err)
	}

	if base.Count() != 1 {
		t.Errorf("base Count() = %d, want 1", base.Count())
	}
}

// 测试多次提交
func TestUTCOCache_MultipleCommits(t *testing.T) {
	base := NewUTCOSet()
	cache := NewUTCOCache(base)

	// 第一轮：添加
	e1 := &UTCOEntry{
		OutPoint: OutPoint{
			TxID:  types.Hash512{0x01},
			Index: 0,
		},
		Address:          types.PubKeyHash{0x10},
		Height:           100,
		LastActiveHeight: 100,
		CredConfig:       tx.CredNew,
		LockScript:       []byte{0x01},
	}
	_ = cache.Add(e1)
	_ = cache.Commit()

	if base.Count() != 1 {
		t.Fatalf("base Count() after first commit = %d, want 1", base.Count())
	}

	// 第二轮：花费并添加新的
	_, _ = cache.Spend(e1.OutPoint)
	e2 := &UTCOEntry{
		OutPoint: OutPoint{
			TxID:  types.Hash512{0x02},
			Index: 0,
		},
		Address:          types.PubKeyHash{0x20},
		Height:           200,
		LastActiveHeight: 200,
		CredConfig:       tx.CredNew,
		LockScript:       []byte{0x02},
	}
	_ = cache.Add(e2)
	_ = cache.Commit()

	if base.Count() != 1 {
		t.Errorf("base Count() after second commit = %d, want 1", base.Count())
	}
	if base.Has(e1.OutPoint) {
		t.Error("base should not have e1 after second commit")
	}
	if !base.Has(e2.OutPoint) {
		t.Error("base should have e2 after second commit")
	}
}
```

### Step 2: 运行测试验证失败

```bash
go test -v ./internal/utco/ -run "TestNewUTCOCache|TestUTCOCache"
```

预期输出：编译失败，`UTCOCache`、`NewUTCOCache` 等未定义。

### Step 3: 写最小实现

创建 `internal/utco/cache.go`：

```go
package utco

import (
	"errors"
)

// 缓存操作错误
var (
	ErrCacheNilEntry     = errors.New("utco cache: nil entry")
	ErrCacheDuplicate    = errors.New("utco cache: outpoint already exists")
	ErrCacheNotFound     = errors.New("utco cache: outpoint not found")
	ErrCacheAlreadySpent = errors.New("utco cache: outpoint already spent")
)

// UTCOCache 是 UTCO 集的写入缓冲层。
// 在区块处理过程中记录新增和花费的条目，支持原子性提交（Commit）和回滚（Rollback）。
//
// 查询优先级：
//  1. 如果在 removed 中 -> 不存在（已花费）
//  2. 如果在 added 中 -> 返回新增条目
//  3. 查询 base 集合
type UTCOCache struct {
	base    *UTCOSet                // 底层 UTCO 集（只读引用）
	added   map[OutPoint]*UTCOEntry // 本轮新增的条目
	removed map[OutPoint]*UTCOEntry // 本轮花费的条目
}

// NewUTCOCache 创建一个新的 UTCO 缓存层。
// base 是底层 UTCO 集，缓存层在提交前不会修改 base。
func NewUTCOCache(base *UTCOSet) *UTCOCache {
	return &UTCOCache{
		base:    base,
		added:   make(map[OutPoint]*UTCOEntry),
		removed: make(map[OutPoint]*UTCOEntry),
	}
}

// Add 向缓存添加一个新条目。
// 如果条目已存在于基础集合或缓存中（且未被花费），返回错误。
func (c *UTCOCache) Add(entry *UTCOEntry) error {
	if entry == nil {
		return ErrCacheNilEntry
	}

	op := entry.OutPoint

	// 如果之前被花费过，允许重新添加（清除 removed 记录）
	if _, wasRemoved := c.removed[op]; wasRemoved {
		delete(c.removed, op)
		c.added[op] = entry
		return nil
	}

	// 检查是否已在 added 中
	if _, exists := c.added[op]; exists {
		return ErrCacheDuplicate
	}

	// 检查是否已在基础集合中
	if c.base.Has(op) {
		return ErrCacheDuplicate
	}

	c.added[op] = entry
	return nil
}

// Spend 花费一个条目。
// 返回被花费的条目数据。如果条目不存在或已被花费，返回错误。
func (c *UTCOCache) Spend(outpoint OutPoint) (*UTCOEntry, error) {
	// 检查是否已被花费
	if _, alreadySpent := c.removed[outpoint]; alreadySpent {
		return nil, ErrCacheAlreadySpent
	}

	// 优先从 added 中查找
	if entry, ok := c.added[outpoint]; ok {
		delete(c.added, outpoint)
		c.removed[outpoint] = entry
		return entry, nil
	}

	// 从 base 中查找
	entry, ok := c.base.Get(outpoint)
	if !ok {
		return nil, ErrCacheNotFound
	}

	c.removed[outpoint] = entry
	return entry, nil
}

// Get 获取条目。
// 按优先级查找：removed（不存在）-> added -> base。
func (c *UTCOCache) Get(outpoint OutPoint) (*UTCOEntry, bool) {
	// 已被花费
	if _, spent := c.removed[outpoint]; spent {
		return nil, false
	}

	// 在新增缓存中
	if entry, ok := c.added[outpoint]; ok {
		return entry, true
	}

	// 在基础集合中
	return c.base.Get(outpoint)
}

// Has 检查条目是否存在（考虑缓存层的变更）。
func (c *UTCOCache) Has(outpoint OutPoint) bool {
	_, ok := c.Get(outpoint)
	return ok
}

// Commit 将缓存的变更提交到基础集合。
// 先移除花费的条目，再添加新增的条目。
// 提交后缓存被清空，可继续使用。
func (c *UTCOCache) Commit() error {
	// 先移除花费的条目
	for op := range c.removed {
		// 仅移除基础集合中存在的条目
		// （added 中的条目已在 Spend 时从 added 删除）
		if c.base.Has(op) {
			if _, err := c.base.Remove(op); err != nil {
				return err
			}
		}
	}

	// 再添加新增的条目
	for _, entry := range c.added {
		if err := c.base.Add(entry); err != nil {
			return err
		}
	}

	// 清空缓存
	c.added = make(map[OutPoint]*UTCOEntry)
	c.removed = make(map[OutPoint]*UTCOEntry)

	return nil
}

// Rollback 回滚缓存的所有变更。
// 清空新增和花费记录，基础集合不受影响。
func (c *UTCOCache) Rollback() {
	c.added = make(map[OutPoint]*UTCOEntry)
	c.removed = make(map[OutPoint]*UTCOEntry)
}

// AddedEntries 返回本轮新增的所有条目。
// 返回的 map 是内部数据的副本，可安全修改。
func (c *UTCOCache) AddedEntries() map[OutPoint]*UTCOEntry {
	result := make(map[OutPoint]*UTCOEntry, len(c.added))
	for op, entry := range c.added {
		result[op] = entry
	}
	return result
}

// SpentEntries 返回本轮花费的所有条目。
// 返回的 map 是内部数据的副本，可安全修改。
func (c *UTCOCache) SpentEntries() map[OutPoint]*UTCOEntry {
	result := make(map[OutPoint]*UTCOEntry, len(c.removed))
	for op, entry := range c.removed {
		result[op] = entry
	}
	return result
}
```

### Step 4: 运行测试验证通过

```bash
go test -v ./internal/utco/ -run "TestNewUTCOCache|TestUTCOCache"
```

预期输出：全部 PASS。

### Step 5: 提交

```bash
git add internal/utco/cache.go internal/utco/cache_test.go
git commit -m "feat(utco): add UTCOCache write-buffer layer with commit and rollback support"
```

---

## 汇总

### 文件清单

| Task | 实现文件 | 测试文件 | 描述 |
|------|----------|----------|------|
| 5 | `internal/utco/entry.go` | `internal/utco/entry_test.go` | OutPoint、UTCOEntry 结构、年份计算、验证、过期/活动检查、InfoHash |
| 6 | `internal/utco/set.go` | `internal/utco/set_test.go` | UTCOSet 双索引集合管理、ShouldInclude 准入判定 |
| 7 | `internal/utco/fingerprint.go` | `internal/utco/fingerprint_test.go` | 4 级层次哈希指纹（Year→Tx8→Tx13→Tx18→InfoHash） |
| 8 | `internal/utco/expiry.go` | `internal/utco/expiry_test.go` | ExpiryIndex 过期索引（硬过期 + 11 年活动规则） |
| 9 | `internal/utco/cache.go` | `internal/utco/cache_test.go` | UTCOCache 写入缓冲层（Commit/Rollback） |

### 依赖关系

```
Task 5: entry.go          ← 依赖 pkg/types, pkg/crypto, internal/tx
Task 6: set.go            ← 依赖 entry.go, pkg/types
Task 7: fingerprint.go    ← 依赖 entry.go (CalcInfoHash), pkg/crypto, pkg/types
Task 8: expiry.go         ← 依赖 entry.go (OutPoint, ActivityBlocks)
Task 9: cache.go          ← 依赖 entry.go, set.go
```

### 验证命令

```bash
# 编译
go build ./internal/utco/...

# 运行所有 utco 包测试
go test -v ./internal/utco/...

# 测试覆盖率
go test -cover ./internal/utco/...

# 格式化
go fmt ./internal/utco/...
```

