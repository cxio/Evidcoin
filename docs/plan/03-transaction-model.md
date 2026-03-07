# Phase 3：交易模型 实现计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现完整的交易模型——包括 TxHeader 与 TxID 计算、LeadInput/RestInput 输入项、四类 Output 输出项（Coin/Credit/Proof/Mediator）、输入/输出哈希树、Tx 结构体、CoinbaseTx 铸币交易、AttachmentID 附件标识、SigFlag 签名数据构造、以及费用计算与交易优先级。

**Architecture:** `internal/tx` 包，仅依赖 `pkg/types` 和 `pkg/crypto`。交易结构不包含校验逻辑或脚本执行，仅负责数据的序列化、反序列化与基本字段验证。

**Tech Stack:** Go 1.25+, pkg/types (Hash512, PubKeyHash, OutputConfig, varint, constants), pkg/crypto (SHA512Sum)

---

## 前置依赖

本 Phase 假设 Phase 1 已完成，以下类型和函数已可用：

```go
// pkg/types
type Hash512 [64]byte              // SHA-512
type Hash384 [48]byte              // SHA3-384
type PubKeyHash [48]byte           // 公钥哈希
type OutputConfig byte             // 输出配置
type SigFlag byte                  // 签名授权标志
func PutVarint(buf []byte, v uint64) int
func Varint(buf []byte) (uint64, int)
func VarintSize(v uint64) int
const HashLen = 64
const Hash384Len = 48
const PubKeyHashLen = 48
const MaxTxSize = 65535
const MaxLockScript = 1024
const MaxMemo = 255
const MaxTitle = 255
const MaxCredDesc = 1023
const MaxProofContent = 4095
const OutCustomClass OutputConfig = 1 << 7
const OutHasAttach OutputConfig = 1 << 6
const OutDestroy OutputConfig = 1 << 5
const OutTypeCoin OutputConfig = 1
const OutTypeCredit OutputConfig = 2
const OutTypeProof OutputConfig = 3
const OutTypeMediator OutputConfig = 4

// pkg/crypto
func SHA512Sum(data []byte) types.Hash512
func SHA3_384Sum(data []byte) types.Hash384
```

> **注意：** 如果 Phase 1 的具体 API 与以上描述有差异，请在实现时以 `pkg/types` 和 `pkg/crypto` 的实际导出符号为准，适当调整本计划中的调用方式。

---

## Task 1: TxHeader 与 TxID 计算

**Files:**
- Create: `internal/tx/header.go`
- Test: `internal/tx/header_test.go`

本 Task 实现 `TxHeader` 结构体、固定 138 字节的二进制序列化、`TxID` 计算、反序列化、以及基本字段验证。

### Step 1: 写失败测试

创建 `internal/tx/header_test.go`：

```go
package tx

import (
	"testing"

	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// 测试序列化产出固定 138 字节
func TestTxHeader_Serialize_Size(t *testing.T) {
	h := &TxHeader{
		Version:     1,
		Timestamp:   1700000000000,
		HashInputs:  types.Hash512{0x01, 0x02},
		HashOutputs: types.Hash512{0x03, 0x04},
	}

	data := h.Serialize()
	if len(data) != TxHeaderSize {
		t.Errorf("Serialize() length = %d, want %d", len(data), TxHeaderSize)
	}
}

// 测试序列化→反序列化往返一致
func TestTxHeader_Serialize_Roundtrip(t *testing.T) {
	original := &TxHeader{
		Version:   2,
		Timestamp: 1700000000000,
	}
	// 填充非零哈希
	for i := range original.HashInputs {
		original.HashInputs[i] = byte(i)
	}
	for i := range original.HashOutputs {
		original.HashOutputs[i] = byte(i + 100)
	}

	data := original.Serialize()

	restored, err := DeserializeTxHeader(data)
	if err != nil {
		t.Fatalf("DeserializeTxHeader() error = %v", err)
	}

	if restored.Version != original.Version {
		t.Errorf("Version = %d, want %d", restored.Version, original.Version)
	}
	if restored.Timestamp != original.Timestamp {
		t.Errorf("Timestamp = %d, want %d", restored.Timestamp, original.Timestamp)
	}
	if restored.HashInputs != original.HashInputs {
		t.Error("HashInputs mismatch after roundtrip")
	}
	if restored.HashOutputs != original.HashOutputs {
		t.Error("HashOutputs mismatch after roundtrip")
	}
}

// 测试同一 header 两次计算 TxID 结果一致
func TestTxHeader_TxID_Deterministic(t *testing.T) {
	h := &TxHeader{
		Version:     1,
		Timestamp:   1700000000000,
		HashInputs:  types.Hash512{0xAA},
		HashOutputs: types.Hash512{0xBB},
	}

	id1 := h.TxID()
	id2 := h.TxID()

	if id1 != id2 {
		t.Error("TxID() should be deterministic")
	}

	// TxID 应为 SHA-512(Serialize())
	expected := crypto.SHA512Sum(h.Serialize())
	if id1 != expected {
		t.Error("TxID() != SHA512Sum(Serialize())")
	}
}

// 测试不同 header 产生不同 TxID
func TestTxHeader_TxID_DifferentHeaders(t *testing.T) {
	h1 := &TxHeader{
		Version:     1,
		Timestamp:   1700000000000,
		HashInputs:  types.Hash512{0x01},
		HashOutputs: types.Hash512{0x02},
	}
	h2 := &TxHeader{
		Version:     1,
		Timestamp:   1700000000001, // 时间戳不同
		HashInputs:  types.Hash512{0x01},
		HashOutputs: types.Hash512{0x02},
	}

	if h1.TxID() == h2.TxID() {
		t.Error("different headers should produce different TxIDs")
	}
}

// 测试数据不足时反序列化返回错误
func TestDeserializeTxHeader_TooShort(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"nil", nil},
		{"empty", []byte{}},
		{"one byte", []byte{0x01}},
		{"137 bytes", make([]byte, TxHeaderSize-1)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DeserializeTxHeader(tt.data)
			if err == nil {
				t.Error("DeserializeTxHeader() should return error for short data")
			}
		})
	}
}

// 表驱动测试覆盖各种无效情况
func TestTxHeader_Validate(t *testing.T) {
	validInputs := types.Hash512{0x01}
	validOutputs := types.Hash512{0x02}

	tests := []struct {
		name    string
		header  *TxHeader
		wantErr bool
	}{
		{
			name: "valid",
			header: &TxHeader{
				Version:     1,
				Timestamp:   1700000000000,
				HashInputs:  validInputs,
				HashOutputs: validOutputs,
			},
			wantErr: false,
		},
		{
			name: "zero_version",
			header: &TxHeader{
				Version:     0,
				Timestamp:   1700000000000,
				HashInputs:  validInputs,
				HashOutputs: validOutputs,
			},
			wantErr: true,
		},
		{
			name: "zero_timestamp",
			header: &TxHeader{
				Version:     1,
				Timestamp:   0,
				HashInputs:  validInputs,
				HashOutputs: validOutputs,
			},
			wantErr: true,
		},
		{
			name: "negative_timestamp",
			header: &TxHeader{
				Version:     1,
				Timestamp:   -1,
				HashInputs:  validInputs,
				HashOutputs: validOutputs,
			},
			wantErr: true,
		},
		{
			name: "zero_hash_inputs",
			header: &TxHeader{
				Version:     1,
				Timestamp:   1700000000000,
				HashInputs:  types.Hash512{}, // 全零
				HashOutputs: validOutputs,
			},
			wantErr: true,
		},
		{
			name: "zero_hash_outputs",
			header: &TxHeader{
				Version:     1,
				Timestamp:   1700000000000,
				HashInputs:  validInputs,
				HashOutputs: types.Hash512{}, // 全零
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.header.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
```

### Step 2: 运行测试验证失败

```bash
go test -v ./internal/tx/ -run "TestTxHeader|TestDeserializeTxHeader"
```

预期输出：编译失败，`TxHeader`、`TxHeaderSize`、`DeserializeTxHeader` 等未定义。

### Step 3: 写最小实现

创建 `internal/tx/header.go`：

```go
package tx

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// TxHeaderSize 交易头固定序列化大小（字节）。
// Version(2) + Timestamp(8) + HashInputs(64) + HashOutputs(64) = 138
const TxHeaderSize = 2 + 8 + types.HashLen + types.HashLen

// TxHeader 交易头结构。
// 包含交易版本号、时间戳、输入项根哈希和输出项根哈希。
// TxID = SHA-512(Serialize())。
type TxHeader struct {
	Version     uint16       // 交易版本号
	Timestamp   int64        // 交易时间戳（Unix 毫秒）
	HashInputs  types.Hash512 // 输入项根哈希
	HashOutputs types.Hash512 // 输出项根哈希
}

// Serialize 将交易头序列化为固定 138 字节的二进制格式。
// 格式：Version(2 LE) || Timestamp(8 LE) || HashInputs(64) || HashOutputs(64)
func (h *TxHeader) Serialize() []byte {
	buf := make([]byte, TxHeaderSize)
	offset := 0

	// Version: 2 字节小端序
	binary.LittleEndian.PutUint16(buf[offset:offset+2], h.Version)
	offset += 2

	// Timestamp: 8 字节小端序
	binary.LittleEndian.PutUint64(buf[offset:offset+8], uint64(h.Timestamp))
	offset += 8

	// HashInputs: 64 字节
	copy(buf[offset:offset+types.HashLen], h.HashInputs[:])
	offset += types.HashLen

	// HashOutputs: 64 字节
	copy(buf[offset:offset+types.HashLen], h.HashOutputs[:])

	return buf
}

// TxID 计算交易 ID。
// TxID = SHA-512(Serialize())
func (h *TxHeader) TxID() types.Hash512 {
	return crypto.SHA512Sum(h.Serialize())
}

// DeserializeTxHeader 从字节流反序列化交易头。
func DeserializeTxHeader(data []byte) (*TxHeader, error) {
	if len(data) < TxHeaderSize {
		return nil, fmt.Errorf("data too short for tx header: got %d, want %d", len(data), TxHeaderSize)
	}

	h := &TxHeader{}
	offset := 0

	h.Version = binary.LittleEndian.Uint16(data[offset : offset+2])
	offset += 2

	h.Timestamp = int64(binary.LittleEndian.Uint64(data[offset : offset+8]))
	offset += 8

	copy(h.HashInputs[:], data[offset:offset+types.HashLen])
	offset += types.HashLen

	copy(h.HashOutputs[:], data[offset:offset+types.HashLen])

	return h, nil
}

// 交易头验证错误
var (
	ErrZeroVersion     = errors.New("tx version is zero")
	ErrInvalidTimestamp = errors.New("tx timestamp is not positive")
	ErrZeroHashInputs  = errors.New("tx hash inputs is zero")
	ErrZeroHashOutputs = errors.New("tx hash outputs is zero")
)

// Validate 对交易头执行基本字段验证。
func (h *TxHeader) Validate() error {
	if h.Version == 0 {
		return ErrZeroVersion
	}
	if h.Timestamp <= 0 {
		return ErrInvalidTimestamp
	}
	if h.HashInputs.IsZero() {
		return ErrZeroHashInputs
	}
	if h.HashOutputs.IsZero() {
		return ErrZeroHashOutputs
	}
	return nil
}
```

### Step 4: 运行测试验证通过

```bash
go test -v ./internal/tx/ -run "TestTxHeader|TestDeserializeTxHeader"
```

预期输出：全部 PASS。

### Step 5: 提交

```bash
git add internal/tx/header.go internal/tx/header_test.go
git commit -m "feat(tx): add TxHeader struct, TxID computation, serialization and validation"
```

---

## Task 2: LeadInput 与 RestInput

**Files:**
- Create: `internal/tx/input.go`
- Test: `internal/tx/input_test.go`

本 Task 实现交易的两种输入项结构：LeadInput（首项输入）和 RestInput（余项输入），包括序列化、反序列化和基本字段验证。

### Step 1: 写失败测试

创建 `internal/tx/input_test.go`：

```go
package tx

import (
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// --- LeadInput 测试 ---

// 测试 LeadInput 序列化为固定 70 字节
func TestLeadInput_Serialize_Size(t *testing.T) {
	li := &LeadInput{
		Year:     2026,
		TxID:     types.Hash512{0x01},
		OutIndex: 0,
	}

	data := li.Serialize()
	if len(data) != LeadInputSize {
		t.Errorf("Serialize() length = %d, want %d", len(data), LeadInputSize)
	}
}

// 测试 LeadInput 序列化→反序列化往返一致
func TestLeadInput_Serialize_Roundtrip(t *testing.T) {
	original := &LeadInput{
		Year:     2026,
		OutIndex: 42,
	}
	for i := range original.TxID {
		original.TxID[i] = byte(i * 3)
	}

	data := original.Serialize()

	restored, err := DeserializeLeadInput(data)
	if err != nil {
		t.Fatalf("DeserializeLeadInput() error = %v", err)
	}

	if restored.Year != original.Year {
		t.Errorf("Year = %d, want %d", restored.Year, original.Year)
	}
	if restored.TxID != original.TxID {
		t.Error("TxID mismatch after roundtrip")
	}
	if restored.OutIndex != original.OutIndex {
		t.Errorf("OutIndex = %d, want %d", restored.OutIndex, original.OutIndex)
	}
}

// 表驱动测试 LeadInput 验证
func TestLeadInput_Validate(t *testing.T) {
	validTxID := types.Hash512{0x01}

	tests := []struct {
		name    string
		input   *LeadInput
		wantErr bool
	}{
		{
			name:    "valid",
			input:   &LeadInput{Year: 2026, TxID: validTxID, OutIndex: 0},
			wantErr: false,
		},
		{
			name:    "zero_year",
			input:   &LeadInput{Year: 0, TxID: validTxID, OutIndex: 0},
			wantErr: true,
		},
		{
			name:    "zero_txid",
			input:   &LeadInput{Year: 2026, TxID: types.Hash512{}, OutIndex: 0},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// 测试 LeadInput 反序列化数据不足时报错
func TestDeserializeLeadInput_TooShort(t *testing.T) {
	_, err := DeserializeLeadInput(make([]byte, LeadInputSize-1))
	if err == nil {
		t.Error("DeserializeLeadInput() should return error for short data")
	}
}

// --- RestInput 测试 ---

// 测试 RestInput 序列化为固定 28 字节
func TestRestInput_Serialize_Size(t *testing.T) {
	ri := &RestInput{
		Year:          2026,
		TxIDPart:      [TxIDPartLen]byte{0x01},
		OutIndex:      1,
		TransferIndex: -1,
	}

	data := ri.Serialize()
	if len(data) != RestInputSize {
		t.Errorf("Serialize() length = %d, want %d", len(data), RestInputSize)
	}
}

// 测试 RestInput 序列化→反序列化往返一致
func TestRestInput_Serialize_Roundtrip(t *testing.T) {
	original := &RestInput{
		Year:          2026,
		OutIndex:      7,
		TransferIndex: 3,
	}
	for i := range original.TxIDPart {
		original.TxIDPart[i] = byte(i * 5)
	}

	data := original.Serialize()

	restored, err := DeserializeRestInput(data)
	if err != nil {
		t.Fatalf("DeserializeRestInput() error = %v", err)
	}

	if restored.Year != original.Year {
		t.Errorf("Year = %d, want %d", restored.Year, original.Year)
	}
	if restored.TxIDPart != original.TxIDPart {
		t.Error("TxIDPart mismatch after roundtrip")
	}
	if restored.OutIndex != original.OutIndex {
		t.Errorf("OutIndex = %d, want %d", restored.OutIndex, original.OutIndex)
	}
	if restored.TransferIndex != original.TransferIndex {
		t.Errorf("TransferIndex = %d, want %d", restored.TransferIndex, original.TransferIndex)
	}
}

// 测试 TransferIndex 为 -1 时的往返
func TestRestInput_Serialize_Roundtrip_NoTransfer(t *testing.T) {
	original := &RestInput{
		Year:          2026,
		TxIDPart:      [TxIDPartLen]byte{0xAA},
		OutIndex:      0,
		TransferIndex: -1, // 不使用
	}

	data := original.Serialize()
	restored, err := DeserializeRestInput(data)
	if err != nil {
		t.Fatalf("DeserializeRestInput() error = %v", err)
	}

	if restored.TransferIndex != -1 {
		t.Errorf("TransferIndex = %d, want -1", restored.TransferIndex)
	}
}

// 表驱动测试 RestInput 验证
func TestRestInput_Validate(t *testing.T) {
	validPart := [TxIDPartLen]byte{0x01}

	tests := []struct {
		name    string
		input   *RestInput
		wantErr bool
	}{
		{
			name:    "valid",
			input:   &RestInput{Year: 2026, TxIDPart: validPart, OutIndex: 0, TransferIndex: -1},
			wantErr: false,
		},
		{
			name:    "valid_with_transfer",
			input:   &RestInput{Year: 2026, TxIDPart: validPart, OutIndex: 1, TransferIndex: 5},
			wantErr: false,
		},
		{
			name:    "zero_year",
			input:   &RestInput{Year: 0, TxIDPart: validPart, OutIndex: 0, TransferIndex: -1},
			wantErr: true,
		},
		{
			name:    "zero_txid_part",
			input:   &RestInput{Year: 2026, TxIDPart: [TxIDPartLen]byte{}, OutIndex: 0, TransferIndex: -1},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// 测试 RestInput 反序列化数据不足时报错
func TestDeserializeRestInput_TooShort(t *testing.T) {
	_, err := DeserializeRestInput(make([]byte, RestInputSize-1))
	if err == nil {
		t.Error("DeserializeRestInput() should return error for short data")
	}
}
```

### Step 2: 运行测试验证失败

```bash
go test -v ./internal/tx/ -run "TestLeadInput|TestRestInput|TestDeserializeLeadInput|TestDeserializeRestInput"
```

预期输出：编译失败，`LeadInput`、`RestInput`、`LeadInputSize`、`RestInputSize` 等未定义。

### Step 3: 写最小实现

创建 `internal/tx/input.go`：

```go
package tx

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/cxio/evidcoin/pkg/types"
)

// TxIDPartLen 余项输入中 TxID 缩写部分的长度（字节）。
const TxIDPartLen = 20

// 输入项固定序列化大小
const (
	// LeadInputSize 首项输入序列化大小：Year(4) + TxID(64) + OutIndex(2) = 70
	LeadInputSize = 4 + types.HashLen + 2

	// RestInputSize 余项输入序列化大小：Year(4) + TxIDPart(20) + OutIndex(2) + TransferIndex(2) = 28
	RestInputSize = 4 + TxIDPartLen + 2 + 2
)

// LeadInput 首项输入（交易的第一个输入项）。
// 包含完整的 TxID 引用，作为整笔交易输入列表的首项。
type LeadInput struct {
	Year     uint32       // 引用交易所在年份
	TxID     types.Hash512 // 引用交易的完整 ID
	OutIndex uint16       // 引用交易中的输出索引
}

// Serialize 将首项输入序列化为固定 70 字节的二进制格式。
// 格式：Year(4 LE) || TxID(64) || OutIndex(2 LE)
func (li *LeadInput) Serialize() []byte {
	buf := make([]byte, LeadInputSize)
	offset := 0

	// Year: 4 字节小端序
	binary.LittleEndian.PutUint32(buf[offset:offset+4], li.Year)
	offset += 4

	// TxID: 64 字节
	copy(buf[offset:offset+types.HashLen], li.TxID[:])
	offset += types.HashLen

	// OutIndex: 2 字节小端序
	binary.LittleEndian.PutUint16(buf[offset:offset+2], li.OutIndex)

	return buf
}

// DeserializeLeadInput 从字节流反序列化首项输入。
func DeserializeLeadInput(data []byte) (*LeadInput, error) {
	if len(data) < LeadInputSize {
		return nil, fmt.Errorf("data too short for lead input: got %d, want %d", len(data), LeadInputSize)
	}

	li := &LeadInput{}
	offset := 0

	li.Year = binary.LittleEndian.Uint32(data[offset : offset+4])
	offset += 4

	copy(li.TxID[:], data[offset:offset+types.HashLen])
	offset += types.HashLen

	li.OutIndex = binary.LittleEndian.Uint16(data[offset : offset+2])

	return li, nil
}

// 首项输入验证错误
var (
	ErrLeadInputZeroYear = errors.New("lead input year is zero")
	ErrLeadInputZeroTxID = errors.New("lead input txid is zero")
)

// Validate 对首项输入执行基本字段验证。
func (li *LeadInput) Validate() error {
	if li.Year == 0 {
		return ErrLeadInputZeroYear
	}
	if li.TxID.IsZero() {
		return ErrLeadInputZeroTxID
	}
	return nil
}

// RestInput 余项输入（交易的第二个及后续输入项）。
// 使用 TxID 的前 20 字节缩写以节省空间。
type RestInput struct {
	Year          uint32             // 引用交易所在年份
	TxIDPart      [TxIDPartLen]byte  // 引用交易 TxID 的前 20 字节
	OutIndex      uint16             // 引用交易中的输出索引
	TransferIndex int16              // 转移索引，-1 表示不使用
}

// Serialize 将余项输入序列化为固定 28 字节的二进制格式。
// 格式：Year(4 LE) || TxIDPart(20) || OutIndex(2 LE) || TransferIndex(2 LE)
func (ri *RestInput) Serialize() []byte {
	buf := make([]byte, RestInputSize)
	offset := 0

	// Year: 4 字节小端序
	binary.LittleEndian.PutUint32(buf[offset:offset+4], ri.Year)
	offset += 4

	// TxIDPart: 20 字节
	copy(buf[offset:offset+TxIDPartLen], ri.TxIDPart[:])
	offset += TxIDPartLen

	// OutIndex: 2 字节小端序
	binary.LittleEndian.PutUint16(buf[offset:offset+2], ri.OutIndex)
	offset += 2

	// TransferIndex: 2 字节小端序（有符号，直接转为 uint16 存储）
	binary.LittleEndian.PutUint16(buf[offset:offset+2], uint16(ri.TransferIndex))

	return buf
}

// DeserializeRestInput 从字节流反序列化余项输入。
func DeserializeRestInput(data []byte) (*RestInput, error) {
	if len(data) < RestInputSize {
		return nil, fmt.Errorf("data too short for rest input: got %d, want %d", len(data), RestInputSize)
	}

	ri := &RestInput{}
	offset := 0

	ri.Year = binary.LittleEndian.Uint32(data[offset : offset+4])
	offset += 4

	copy(ri.TxIDPart[:], data[offset:offset+TxIDPartLen])
	offset += TxIDPartLen

	ri.OutIndex = binary.LittleEndian.Uint16(data[offset : offset+2])
	offset += 2

	ri.TransferIndex = int16(binary.LittleEndian.Uint16(data[offset : offset+2]))

	return ri, nil
}

// 余项输入验证错误
var (
	ErrRestInputZeroYear    = errors.New("rest input year is zero")
	ErrRestInputZeroTxIDPart = errors.New("rest input txid part is all zeros")
)

// Validate 对余项输入执行基本字段验证。
func (ri *RestInput) Validate() error {
	if ri.Year == 0 {
		return ErrRestInputZeroYear
	}
	// 检查 TxIDPart 是否全零
	allZero := true
	for _, b := range ri.TxIDPart {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		return ErrRestInputZeroTxIDPart
	}
	return nil
}
```

### Step 4: 运行测试验证通过

```bash
go test -v ./internal/tx/ -run "TestLeadInput|TestRestInput|TestDeserializeLeadInput|TestDeserializeRestInput"
```

预期输出：全部 PASS。

### Step 5: 提交

```bash
git add internal/tx/input.go internal/tx/input_test.go
git commit -m "feat(tx): add LeadInput and RestInput with serialization and validation"
```

---

## Task 3: Output 结构

**Files:**
- Create: `internal/tx/output.go`
- Create: `internal/tx/util.go`
- Test: `internal/tx/output_test.go`

本 Task 实现四种输出类型（Coin/Credit/Proof/Mediator）的统一 Output 结构体，包括 CredConf/ProofConf 位域类型、Serialize/SerializeContent 方法、以及基本验证逻辑。

### Step 1: 写失败测试

创建 `internal/tx/output_test.go`：

```go
package tx

import (
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// --- CredConf 位域测试 ---

func TestCredConf(t *testing.T) {
	tests := []struct {
		name          string
		conf          CredConf
		isNew         bool
		isMutable     bool
		isModified    bool
		hasXferCount  bool
		hasExpiry     bool
		descLen       int
	}{
		{
			name:          "new_mutable_with_desc",
			conf:          CredNew | CredMutable | CredConf(100), // 描述长度 100
			isNew:         true,
			isMutable:     true,
			isModified:    false,
			hasXferCount:  false,
			hasExpiry:     false,
			descLen:       100,
		},
		{
			name:          "modified_with_xfer_and_expiry",
			conf:          CredModified | CredHasXferCount | CredHasExpiry | CredConf(512),
			isNew:         false,
			isMutable:     false,
			isModified:    true,
			hasXferCount:  true,
			hasExpiry:     true,
			descLen:       512,
		},
		{
			name:          "max_desc_len",
			conf:          CredConf(CredDescLenMask), // 1023
			isNew:         false,
			isMutable:     false,
			isModified:    false,
			hasXferCount:  false,
			hasExpiry:     false,
			descLen:       1023,
		},
		{
			name:          "all_flags",
			conf:          CredNew | CredMutable | CredModified | CredHasXferCount | CredHasExpiry | CredConf(0),
			isNew:         true,
			isMutable:     true,
			isModified:    true,
			hasXferCount:  true,
			hasExpiry:     true,
			descLen:       0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.conf.IsNew(); got != tt.isNew {
				t.Errorf("IsNew() = %v, want %v", got, tt.isNew)
			}
			if got := tt.conf.IsMutable(); got != tt.isMutable {
				t.Errorf("IsMutable() = %v, want %v", got, tt.isMutable)
			}
			if got := tt.conf.IsModified(); got != tt.isModified {
				t.Errorf("IsModified() = %v, want %v", got, tt.isModified)
			}
			if got := tt.conf.HasXferCount(); got != tt.hasXferCount {
				t.Errorf("HasXferCount() = %v, want %v", got, tt.hasXferCount)
			}
			if got := tt.conf.HasExpiry(); got != tt.hasExpiry {
				t.Errorf("HasExpiry() = %v, want %v", got, tt.hasExpiry)
			}
			if got := tt.conf.DescLen(); got != tt.descLen {
				t.Errorf("DescLen() = %d, want %d", got, tt.descLen)
			}
		})
	}
}

// --- ProofConf 位域测试 ---

func TestProofConf(t *testing.T) {
	tests := []struct {
		name       string
		conf       ProofConf
		contentLen int
	}{
		{"zero", ProofConf(0), 0},
		{"small", ProofConf(100), 100},
		{"max", ProofConf(ProofContentLenMask), 4095},
		{"with_upper_bits", ProofConf(0xF000 | 256), 256}, // 高位不影响内容长度
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.conf.ContentLen(); got != tt.contentLen {
				t.Errorf("ContentLen() = %d, want %d", got, tt.contentLen)
			}
		})
	}
}

// --- Output 序列化测试 ---

// 测试 Coin 输出序列化
func TestOutput_Serialize_Coin(t *testing.T) {
	var addr types.PubKeyHash
	addr[0] = 0xAA
	lockScript := []byte{0x01, 0x02, 0x03}
	memo := []byte("test memo")

	out := &Output{
		Serial:     1,
		Config:     types.OutTypeCoin,
		Address:    addr,
		Amount:     50000,
		Memo:       memo,
		LockScript: lockScript,
	}

	data := out.Serialize()
	if len(data) == 0 {
		t.Fatal("Serialize() returned empty data")
	}

	// 验证结构：Serial(2) + Config(1) + Address(48) + Amount(varint) + MemoLen(1) + Memo + LockScriptLen(varint) + LockScript
	// 最小预期长度：2 + 1 + 48 + 1 + 1 + len(memo) + 1 + len(lockScript)
	minLen := 2 + 1 + types.PubKeyHashLen + 1 + 1 + len(memo) + 1 + len(lockScript)
	if len(data) < minLen {
		t.Errorf("Serialize() length = %d, less than minimum expected %d", len(data), minLen)
	}

	// 验证前 3 字节
	if data[0] != 0x01 || data[1] != 0x00 { // Serial = 1, LE
		t.Errorf("Serial bytes = [%02x %02x], want [01 00]", data[0], data[1])
	}
	if data[2] != byte(types.OutTypeCoin) {
		t.Errorf("Config byte = %02x, want %02x", data[2], byte(types.OutTypeCoin))
	}
}

// 测试 Credit 输出序列化（含 XferCount 和 Expiry）
func TestOutput_Serialize_Credit(t *testing.T) {
	var addr types.PubKeyHash
	addr[0] = 0xBB
	lockScript := []byte{0x10, 0x20}
	creator := []byte("alice")
	title := []byte("test credit")
	desc := []byte("credit description")

	credConf := CredNew | CredMutable | CredHasXferCount | CredHasExpiry | CredConf(len(desc))

	out := &Output{
		Serial:       2,
		Config:       types.OutTypeCredit,
		Address:      addr,
		CredConfig:   credConf,
		XferCount:    10,
		ExpiryHeight: 87661,
		Creator:      creator,
		Title:        title,
		Desc:         desc,
		LockScript:   lockScript,
	}

	data := out.Serialize()
	if len(data) == 0 {
		t.Fatal("Serialize() returned empty data")
	}

	// 验证前 3 字节
	if data[2] != byte(types.OutTypeCredit) {
		t.Errorf("Config byte = %02x, want %02x", data[2], byte(types.OutTypeCredit))
	}
}

// 测试 Proof 输出序列化
func TestOutput_Serialize_Proof(t *testing.T) {
	creator := []byte("bob")
	title := []byte("test proof")
	content := []byte("proof content data")

	proofConf := ProofConf(len(content))

	out := &Output{
		Serial:      3,
		Config:      types.OutTypeProof,
		ProofConfig: proofConf,
		Creator:     creator,
		Title:       title,
		Content:     content,
		IdentScript: []byte{0xAA, 0xBB},
	}

	data := out.Serialize()
	if len(data) == 0 {
		t.Fatal("Serialize() returned empty data")
	}

	// 验证 Config 字节
	if data[2] != byte(types.OutTypeProof) {
		t.Errorf("Config byte = %02x, want %02x", data[2], byte(types.OutTypeProof))
	}
}

// 测试 Mediator 输出序列化
func TestOutput_Serialize_Mediator(t *testing.T) {
	var addr types.PubKeyHash
	addr[47] = 0xFF
	lockScript := []byte{0x50, 0x60, 0x70, 0x80}

	out := &Output{
		Serial:     4,
		Config:     types.OutTypeMediator,
		Address:    addr,
		LockScript: lockScript,
	}

	data := out.Serialize()
	if len(data) == 0 {
		t.Fatal("Serialize() returned empty data")
	}

	// Mediator: Serial(2) + Config(1) + Address(48) + LockScriptLen(varint) + LockScript
	expectedLen := 2 + 1 + types.PubKeyHashLen + 1 + len(lockScript) // varint(4) = 1 字节
	if len(data) != expectedLen {
		t.Errorf("Serialize() length = %d, want %d", len(data), expectedLen)
	}
}

// 测试 Credit 输出带附件的序列化
func TestOutput_Serialize_Credit_WithAttach(t *testing.T) {
	var addr types.PubKeyHash
	addr[0] = 0xCC
	attachID := []byte("attach-id-data-here-32bytes-long!")
	creator := []byte("carol")
	title := []byte("credit with attach")
	desc := []byte("desc")

	credConf := CredNew | CredConf(len(desc))

	out := &Output{
		Serial:     5,
		Config:     types.OutTypeCredit | types.OutHasAttach,
		Address:    addr,
		CredConfig: credConf,
		Creator:    creator,
		Title:      title,
		Desc:       desc,
		AttachID:   attachID,
		LockScript: []byte{0x01},
	}

	data := out.Serialize()
	if len(data) == 0 {
		t.Fatal("Serialize() returned empty data")
	}
}

// --- Output 验证测试 ---

func TestOutput_Validate(t *testing.T) {
	var validAddr types.PubKeyHash
	validAddr[0] = 0x01

	tests := []struct {
		name    string
		output  *Output
		wantErr bool
	}{
		{
			name: "valid_coin",
			output: &Output{
				Serial:     0,
				Config:     types.OutTypeCoin,
				Address:    validAddr,
				Amount:     1000,
				LockScript: []byte{0x01},
			},
			wantErr: false,
		},
		{
			name: "coin_zero_amount",
			output: &Output{
				Serial:     0,
				Config:     types.OutTypeCoin,
				Address:    validAddr,
				Amount:     0,
				LockScript: []byte{0x01},
			},
			wantErr: true,
		},
		{
			name: "coin_negative_amount",
			output: &Output{
				Serial:     0,
				Config:     types.OutTypeCoin,
				Address:    validAddr,
				Amount:     -100,
				LockScript: []byte{0x01},
			},
			wantErr: true,
		},
		{
			name: "coin_zero_address",
			output: &Output{
				Serial:     0,
				Config:     types.OutTypeCoin,
				Address:    types.PubKeyHash{}, // 全零
				Amount:     1000,
				LockScript: []byte{0x01},
			},
			wantErr: true,
		},
		{
			name: "coin_empty_lockscript",
			output: &Output{
				Serial:  0,
				Config:  types.OutTypeCoin,
				Address: validAddr,
				Amount:  1000,
			},
			wantErr: true,
		},
		{
			name: "coin_lockscript_too_long",
			output: &Output{
				Serial:     0,
				Config:     types.OutTypeCoin,
				Address:    validAddr,
				Amount:     1000,
				LockScript: make([]byte, types.MaxLockScript+1),
			},
			wantErr: true,
		},
		{
			name: "coin_memo_too_long",
			output: &Output{
				Serial:     0,
				Config:     types.OutTypeCoin,
				Address:    validAddr,
				Amount:     1000,
				Memo:       make([]byte, types.MaxMemo+1),
				LockScript: []byte{0x01},
			},
			wantErr: true,
		},
		{
			name: "valid_credit",
			output: &Output{
				Serial:     1,
				Config:     types.OutTypeCredit,
				Address:    validAddr,
				CredConfig: CredNew | CredConf(4),
				Creator:    []byte("test"),
				Title:      []byte("title"),
				Desc:       []byte("desc"),
				LockScript: []byte{0x01},
			},
			wantErr: false,
		},
		{
			name: "credit_title_too_long",
			output: &Output{
				Serial:     1,
				Config:     types.OutTypeCredit,
				Address:    validAddr,
				CredConfig: CredNew,
				Creator:    []byte("test"),
				Title:      make([]byte, types.MaxTitle+1),
				LockScript: []byte{0x01},
			},
			wantErr: true,
		},
		{
			name: "valid_proof",
			output: &Output{
				Serial:      2,
				Config:      types.OutTypeProof,
				ProofConfig: ProofConf(7),
				Creator:     []byte("test"),
				Title:       []byte("title"),
				Content:     []byte("content"),
				IdentScript: []byte{0x01},
			},
			wantErr: false,
		},
		{
			name: "proof_content_too_long",
			output: &Output{
				Serial:      2,
				Config:      types.OutTypeProof,
				ProofConfig: ProofConf(types.MaxProofContent + 1),
				Creator:     []byte("test"),
				Title:       []byte("title"),
				Content:     make([]byte, types.MaxProofContent+1),
				IdentScript: []byte{0x01},
			},
			wantErr: true,
		},
		{
			name: "valid_mediator",
			output: &Output{
				Serial:     3,
				Config:     types.OutTypeMediator,
				Address:    validAddr,
				LockScript: []byte{0x01},
			},
			wantErr: false,
		},
		{
			name: "mediator_zero_address",
			output: &Output{
				Serial:     3,
				Config:     types.OutTypeMediator,
				Address:    types.PubKeyHash{},
				LockScript: []byte{0x01},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.output.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// --- SerializeContent 测试 ---

func TestOutput_SerializeContent_Coin(t *testing.T) {
	out := &Output{
		Config: types.OutTypeCoin,
		Amount: 1000,
		Memo:   []byte("hello"),
	}
	data := out.SerializeContent()
	if len(data) == 0 {
		t.Fatal("SerializeContent() returned empty for Coin")
	}
}

func TestOutput_SerializeContent_Credit(t *testing.T) {
	out := &Output{
		Config:     types.OutTypeCredit,
		CredConfig: CredNew | CredConf(4),
		Creator:    []byte("test"),
		Title:      []byte("credit"),
		Desc:       []byte("desc"),
		AttachID:   []byte("attach-id"),
	}
	data := out.SerializeContent()
	if len(data) == 0 {
		t.Fatal("SerializeContent() returned empty for Credit")
	}
}

func TestOutput_SerializeContent_Proof(t *testing.T) {
	out := &Output{
		Config:      types.OutTypeProof,
		ProofConfig: ProofConf(7),
		Creator:     []byte("test"),
		Title:       []byte("proof"),
		Content:     []byte("content"),
	}
	data := out.SerializeContent()
	if len(data) == 0 {
		t.Fatal("SerializeContent() returned empty for Proof")
	}
}
```

### Step 2: 运行测试验证失败

```bash
go test -v ./internal/tx/ -run "TestCredConf|TestProofConf|TestOutput"
```

预期输出：编译失败，`CredConf`、`ProofConf`、`Output` 等未定义。

### Step 3: 写最小实现

创建 `internal/tx/util.go`：

```go
package tx

import (
	"bytes"

	"github.com/cxio/evidcoin/pkg/types"
)

// writeVarint 将 varint 编码写入 buffer。
func writeVarint(buf *bytes.Buffer, v uint64) {
	var tmp [10]byte
	n := types.PutVarint(tmp[:], v)
	buf.Write(tmp[:n])
}
```

创建 `internal/tx/output.go`：

```go
package tx

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/cxio/evidcoin/pkg/types"
)

// --- CredConf 凭信配置位域 ---

// CredConf 凭信配置（uint16）。
// 位域布局：
//
//	Bit 15:    CredNew         - 新建
//	Bit 14:    CredMutable     - 可变
//	Bit 13:    CredModified    - 已修改
//	Bit 11:    CredHasXferCount - 有转移次数
//	Bit 10:    CredHasExpiry   - 有截止高度
//	Bits [0:9]: CredDescLenMask - 描述长度（最大 1023）
type CredConf uint16

const (
	CredNew          CredConf = 1 << 15 // 新建凭信
	CredMutable      CredConf = 1 << 14 // 可变凭信
	CredModified     CredConf = 1 << 13 // 已修改
	CredHasXferCount CredConf = 1 << 11 // 含转移次数
	CredHasExpiry    CredConf = 1 << 10 // 含截止高度
	CredDescLenMask  CredConf = 0x03FF  // 低 10 位：描述长度掩码
)

// IsNew 检查是否为新建凭信。
func (c CredConf) IsNew() bool { return c&CredNew != 0 }

// IsMutable 检查是否为可变凭信。
func (c CredConf) IsMutable() bool { return c&CredMutable != 0 }

// IsModified 检查是否已修改。
func (c CredConf) IsModified() bool { return c&CredModified != 0 }

// HasXferCount 检查是否含转移次数。
func (c CredConf) HasXferCount() bool { return c&CredHasXferCount != 0 }

// HasExpiry 检查是否含截止高度。
func (c CredConf) HasExpiry() bool { return c&CredHasExpiry != 0 }

// DescLen 返回描述长度。
func (c CredConf) DescLen() int { return int(c & CredDescLenMask) }

// --- ProofConf 存证配置位域 ---

// ProofConf 存证配置（uint16）。
// 位域布局：
//
//	Bits [0:11]: ProofContentLenMask - 内容长度（最大 4095）
type ProofConf uint16

const (
	ProofContentLenMask ProofConf = 0x0FFF // 低 12 位：内容长度掩码
)

// ContentLen 返回内容长度。
func (p ProofConf) ContentLen() int { return int(p & ProofContentLenMask) }

// --- Output 输出项结构 ---

// Output 交易输出项。
// 根据 Config.Type() 区分四种类型：Coin、Credit、Proof、Mediator。
// 不同类型使用不同的字段子集。
type Output struct {
	Serial       uint16             // 输出序号
	Config       types.OutputConfig // 输出配置（类型 + 标志位）
	Address      types.PubKeyHash   // 接收者地址（Coin/Credit/Mediator）
	Amount       int64              // 金额，仅 Coin
	Memo         []byte             // 附言，仅 Coin（≤255）
	CredConfig   CredConf           // 凭信配置，仅 Credit
	XferCount    uint16             // 转移次数，Credit 且 HasXferCount 时有效
	ExpiryHeight uint64             // 截止高度，Credit 且 HasExpiry 时有效
	ProofConfig  ProofConf          // 存证配置，仅 Proof
	Creator      []byte             // 创建者，Credit/Proof（≤255）
	Title        []byte             // 标题，Credit/Proof（≤255）
	Desc         []byte             // 描述，仅 Credit（长度由 CredConfig 编码）
	Content      []byte             // 内容，仅 Proof（长度由 ProofConfig 编码）
	AttachID     []byte             // 附件 ID，Credit/Proof 可选
	LockScript   []byte             // 锁定脚本，Coin/Credit/Mediator（≤1024）
	IdentScript  []byte             // 身份脚本，仅 Proof
}

// Serialize 将输出项序列化为二进制格式。
// 根据 Config.Type() 分支选择不同的序列化格式。
func (o *Output) Serialize() []byte {
	var buf bytes.Buffer

	// 公共前缀：Serial(2) + Config(1)
	var tmp [2]byte
	binary.LittleEndian.PutUint16(tmp[:], o.Serial)
	buf.Write(tmp[:])
	buf.WriteByte(byte(o.Config))

	switch o.Config.Type() {
	case types.OutTypeCoin:
		o.serializeCoin(&buf)
	case types.OutTypeCredit:
		o.serializeCredit(&buf)
	case types.OutTypeProof:
		o.serializeProof(&buf)
	case types.OutTypeMediator:
		o.serializeMediator(&buf)
	}

	return buf.Bytes()
}

// serializeCoin 序列化 Coin 类型输出。
// 格式：Address(48) || Amount(varint) || MemoLen(1) || Memo || LockScriptLen(varint) || LockScript
func (o *Output) serializeCoin(buf *bytes.Buffer) {
	buf.Write(o.Address[:])
	writeVarint(buf, uint64(o.Amount))
	buf.WriteByte(byte(len(o.Memo)))
	buf.Write(o.Memo)
	writeVarint(buf, uint64(len(o.LockScript)))
	buf.Write(o.LockScript)
}

// serializeCredit 序列化 Credit 类型输出。
// 格式：Address(48) || CredConfig(2) || [XferCount(2)] || [ExpiryHeight(varint)]
//
//	|| CreatorLen(1) || Creator || TitleLen(1) || Title || Desc
//	|| [AttachID] || LockScriptLen(varint) || LockScript
func (o *Output) serializeCredit(buf *bytes.Buffer) {
	buf.Write(o.Address[:])

	var tmp [2]byte
	binary.LittleEndian.PutUint16(tmp[:], uint16(o.CredConfig))
	buf.Write(tmp[:])

	if o.CredConfig.HasXferCount() {
		binary.LittleEndian.PutUint16(tmp[:], o.XferCount)
		buf.Write(tmp[:])
	}
	if o.CredConfig.HasExpiry() {
		writeVarint(buf, o.ExpiryHeight)
	}

	buf.WriteByte(byte(len(o.Creator)))
	buf.Write(o.Creator)
	buf.WriteByte(byte(len(o.Title)))
	buf.Write(o.Title)
	// Desc 的长度由 CredConfig 编码，直接写入数据
	buf.Write(o.Desc)

	if o.Config.HasAttachment() {
		buf.Write(o.AttachID)
	}

	writeVarint(buf, uint64(len(o.LockScript)))
	buf.Write(o.LockScript)
}

// serializeProof 序列化 Proof 类型输出。
// 格式：ProofConfig(2) || CreatorLen(1) || Creator || TitleLen(1) || Title
//
//	|| Content || [AttachID] || IdentScriptLen(varint) || IdentScript
func (o *Output) serializeProof(buf *bytes.Buffer) {
	var tmp [2]byte
	binary.LittleEndian.PutUint16(tmp[:], uint16(o.ProofConfig))
	buf.Write(tmp[:])

	buf.WriteByte(byte(len(o.Creator)))
	buf.Write(o.Creator)
	buf.WriteByte(byte(len(o.Title)))
	buf.Write(o.Title)
	// Content 的长度由 ProofConfig 编码，直接写入数据
	buf.Write(o.Content)

	if o.Config.HasAttachment() {
		buf.Write(o.AttachID)
	}

	writeVarint(buf, uint64(len(o.IdentScript)))
	buf.Write(o.IdentScript)
}

// serializeMediator 序列化 Mediator 类型输出。
// 格式：Address(48) || LockScriptLen(varint) || LockScript
func (o *Output) serializeMediator(buf *bytes.Buffer) {
	buf.Write(o.Address[:])
	writeVarint(buf, uint64(len(o.LockScript)))
	buf.Write(o.LockScript)
}

// SerializeContent 序列化输出的内容部分（用于部分签名 SIGCONTENT）。
// 根据类型返回不同内容：
//   - Coin:   Amount(varint) || MemoLen(1) || Memo
//   - Credit: CredConfig(2) || CreatorLen(1) || Creator || TitleLen(1) || Title || Desc || AttachID
//   - Proof:  ProofConfig(2) || CreatorLen(1) || Creator || TitleLen(1) || Title || Content || AttachID
func (o *Output) SerializeContent() []byte {
	var buf bytes.Buffer

	switch o.Config.Type() {
	case types.OutTypeCoin:
		writeVarint(&buf, uint64(o.Amount))
		buf.WriteByte(byte(len(o.Memo)))
		buf.Write(o.Memo)

	case types.OutTypeCredit:
		var tmp [2]byte
		binary.LittleEndian.PutUint16(tmp[:], uint16(o.CredConfig))
		buf.Write(tmp[:])
		buf.WriteByte(byte(len(o.Creator)))
		buf.Write(o.Creator)
		buf.WriteByte(byte(len(o.Title)))
		buf.Write(o.Title)
		buf.Write(o.Desc)
		buf.Write(o.AttachID)

	case types.OutTypeProof:
		var tmp [2]byte
		binary.LittleEndian.PutUint16(tmp[:], uint16(o.ProofConfig))
		buf.Write(tmp[:])
		buf.WriteByte(byte(len(o.Creator)))
		buf.Write(o.Creator)
		buf.WriteByte(byte(len(o.Title)))
		buf.Write(o.Title)
		buf.Write(o.Content)
		buf.Write(o.AttachID)
	}

	return buf.Bytes()
}

// --- Output 验证 ---

// 输出验证错误
var (
	ErrOutputZeroAddress       = errors.New("output address is zero")
	ErrOutputInvalidAmount     = errors.New("coin output amount must be positive")
	ErrOutputMemoTooLong       = errors.New("coin memo exceeds max length")
	ErrOutputEmptyLockScript   = errors.New("output lock script is empty")
	ErrOutputLockScriptTooLong = errors.New("output lock script exceeds max length")
	ErrOutputTitleTooLong      = errors.New("output title exceeds max length")
	ErrOutputCreatorTooLong    = errors.New("output creator exceeds max length")
	ErrOutputDescTooLong       = errors.New("credit description exceeds max length")
	ErrOutputContentTooLong    = errors.New("proof content exceeds max length")
	ErrOutputEmptyIdentScript  = errors.New("proof ident script is empty")
	ErrOutputUnknownType       = errors.New("unknown output type")
)

// Validate 对输出项执行基本字段验证。
func (o *Output) Validate() error {
	switch o.Config.Type() {
	case types.OutTypeCoin:
		return o.validateCoin()
	case types.OutTypeCredit:
		return o.validateCredit()
	case types.OutTypeProof:
		return o.validateProof()
	case types.OutTypeMediator:
		return o.validateMediator()
	default:
		return ErrOutputUnknownType
	}
}

// validateCoin 验证 Coin 输出。
func (o *Output) validateCoin() error {
	if o.Address.IsZero() {
		return ErrOutputZeroAddress
	}
	if o.Amount <= 0 {
		return ErrOutputInvalidAmount
	}
	if len(o.Memo) > types.MaxMemo {
		return ErrOutputMemoTooLong
	}
	if len(o.LockScript) == 0 {
		return ErrOutputEmptyLockScript
	}
	if len(o.LockScript) > types.MaxLockScript {
		return ErrOutputLockScriptTooLong
	}
	return nil
}

// validateCredit 验证 Credit 输出。
func (o *Output) validateCredit() error {
	if o.Address.IsZero() {
		return ErrOutputZeroAddress
	}
	if len(o.Creator) > types.MaxTitle {
		return ErrOutputCreatorTooLong
	}
	if len(o.Title) > types.MaxTitle {
		return ErrOutputTitleTooLong
	}
	if len(o.Desc) > types.MaxCredDesc {
		return ErrOutputDescTooLong
	}
	if len(o.LockScript) == 0 {
		return ErrOutputEmptyLockScript
	}
	if len(o.LockScript) > types.MaxLockScript {
		return ErrOutputLockScriptTooLong
	}
	return nil
}

// validateProof 验证 Proof 输出。
func (o *Output) validateProof() error {
	if len(o.Creator) > types.MaxTitle {
		return ErrOutputCreatorTooLong
	}
	if len(o.Title) > types.MaxTitle {
		return ErrOutputTitleTooLong
	}
	if len(o.Content) > types.MaxProofContent {
		return ErrOutputContentTooLong
	}
	if len(o.IdentScript) == 0 {
		return ErrOutputEmptyIdentScript
	}
	return nil
}

// validateMediator 验证 Mediator 输出。
func (o *Output) validateMediator() error {
	if o.Address.IsZero() {
		return ErrOutputZeroAddress
	}
	if len(o.LockScript) == 0 {
		return ErrOutputEmptyLockScript
	}
	if len(o.LockScript) > types.MaxLockScript {
		return ErrOutputLockScriptTooLong
	}
	return nil
}
```

### Step 4: 运行测试验证通过

```bash
go test -v ./internal/tx/ -run "TestCredConf|TestProofConf|TestOutput"
```

预期输出：全部 PASS。

### Step 5: 提交

```bash
git add internal/tx/output.go internal/tx/util.go internal/tx/output_test.go
git commit -m "feat(tx): add Output struct with Coin/Credit/Proof/Mediator serialization and validation"
```

---

## Task 4: 输入/输出哈希树与完整交易结构

**Files:**
- Create: `internal/tx/hashtree.go`
- Create: `internal/tx/tx.go`
- Test: `internal/tx/hashtree_test.go`

本 Task 实现二叉哈希树、输入/输出哈希计算、以及完整的 Tx 结构体（含 BuildHeader 和 TxID 方法）。

### Step 1: 写失败测试

创建 `internal/tx/hashtree_test.go`：

```go
package tx

import (
	"testing"

	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// --- BinaryHashTree 测试 ---

// 空输入返回零值
func TestBinaryHashTree_Empty(t *testing.T) {
	result := BinaryHashTree(nil)
	if !result.IsZero() {
		t.Error("BinaryHashTree(nil) should return zero hash")
	}

	result2 := BinaryHashTree([][]byte{})
	if !result2.IsZero() {
		t.Error("BinaryHashTree([]) should return zero hash")
	}
}

// 单个叶子：结果为该叶子的 SHA-512
func TestBinaryHashTree_SingleLeaf(t *testing.T) {
	leaf := []byte("single leaf data")
	result := BinaryHashTree([][]byte{leaf})

	expected := crypto.SHA512Sum(leaf)
	if result != expected {
		t.Error("BinaryHashTree with single leaf should return SHA512(leaf)")
	}
}

// 两个叶子：SHA-512(SHA-512(a) || SHA-512(b))
func TestBinaryHashTree_TwoLeaves(t *testing.T) {
	a := []byte("leaf A")
	b := []byte("leaf B")

	result := BinaryHashTree([][]byte{a, b})

	hashA := crypto.SHA512Sum(a)
	hashB := crypto.SHA512Sum(b)
	combined := append(hashA[:], hashB[:]...)
	expected := crypto.SHA512Sum(combined)

	if result != expected {
		t.Error("BinaryHashTree with two leaves mismatch")
	}
}

// 三个叶子（奇数）：最后一个直接提升
func TestBinaryHashTree_ThreeLeaves(t *testing.T) {
	a := []byte("leaf A")
	b := []byte("leaf B")
	c := []byte("leaf C")

	result := BinaryHashTree([][]byte{a, b, c})

	// 第一层：hashA, hashB, hashC
	hashA := crypto.SHA512Sum(a)
	hashB := crypto.SHA512Sum(b)
	hashC := crypto.SHA512Sum(c)

	// 第二层：SHA-512(hashA || hashB), hashC（直接提升）
	ab := crypto.SHA512Sum(append(hashA[:], hashB[:]...))

	// 第三层：SHA-512(ab || hashC)
	expected := crypto.SHA512Sum(append(ab[:], hashC[:]...))

	if result != expected {
		t.Error("BinaryHashTree with three leaves mismatch")
	}
}

// 确定性：同输入同输出
func TestBinaryHashTree_Deterministic(t *testing.T) {
	leaves := [][]byte{
		[]byte("data1"),
		[]byte("data2"),
		[]byte("data3"),
		[]byte("data4"),
	}

	r1 := BinaryHashTree(leaves)
	r2 := BinaryHashTree(leaves)

	if r1 != r2 {
		t.Error("BinaryHashTree should be deterministic")
	}
}

// --- ComputeInputHash 测试 ---

// 基本输入哈希计算
func TestComputeInputHash_Basic(t *testing.T) {
	lead := &LeadInput{
		Year:     2026,
		TxID:     types.Hash512{0x01, 0x02, 0x03},
		OutIndex: 0,
	}
	rest := []*RestInput{
		{Year: 2026, TxIDPart: [TxIDPartLen]byte{0x10}, OutIndex: 1, TransferIndex: -1},
		{Year: 2026, TxIDPart: [TxIDPartLen]byte{0x20}, OutIndex: 2, TransferIndex: 0},
	}

	result := ComputeInputHash(lead, rest)

	// 结果不应为零
	if result.IsZero() {
		t.Error("ComputeInputHash() should not return zero hash")
	}

	// 确定性
	result2 := ComputeInputHash(lead, rest)
	if result != result2 {
		t.Error("ComputeInputHash() should be deterministic")
	}
}

// 无 RestInput 时的输入哈希
func TestComputeInputHash_NoRestInputs(t *testing.T) {
	lead := &LeadInput{
		Year:     2026,
		TxID:     types.Hash512{0xAA},
		OutIndex: 0,
	}

	result := ComputeInputHash(lead, nil)

	// 应不为零
	if result.IsZero() {
		t.Error("ComputeInputHash() with no rest inputs should not return zero hash")
	}

	// 手动验证
	leadHash := crypto.SHA512Sum(lead.Serialize())
	restHash := crypto.SHA512Sum(nil) // SHA-512("")
	expected := crypto.SHA512Sum(append(leadHash[:], restHash[:]...))

	if result != expected {
		t.Error("ComputeInputHash() with no rest inputs mismatch")
	}
}

// --- ComputeOutputHash 测试 ---

// 基本输出哈希计算
func TestComputeOutputHash_Basic(t *testing.T) {
	var addr types.PubKeyHash
	addr[0] = 0x01

	outputs := []*Output{
		{
			Serial:     0,
			Config:     types.OutTypeCoin,
			Address:    addr,
			Amount:     1000,
			LockScript: []byte{0x01},
		},
		{
			Serial:     1,
			Config:     types.OutTypeCoin,
			Address:    addr,
			Amount:     2000,
			LockScript: []byte{0x02},
		},
	}

	result, err := ComputeOutputHash(outputs)
	if err != nil {
		t.Fatalf("ComputeOutputHash() error = %v", err)
	}

	if result.IsZero() {
		t.Error("ComputeOutputHash() should not return zero hash")
	}

	// 确定性
	result2, _ := ComputeOutputHash(outputs)
	if result != result2 {
		t.Error("ComputeOutputHash() should be deterministic")
	}
}

// 空输出列表
func TestComputeOutputHash_Empty(t *testing.T) {
	result, err := ComputeOutputHash(nil)
	if err != nil {
		t.Fatalf("ComputeOutputHash(nil) error = %v", err)
	}
	if !result.IsZero() {
		t.Error("ComputeOutputHash(nil) should return zero hash")
	}
}

// --- Tx 完整交易测试 ---

// 测试 BuildHeader 与 TxID
func TestTx_BuildHeader_And_TxID(t *testing.T) {
	var addr types.PubKeyHash
	addr[0] = 0x01

	tx := &Tx{
		Lead: &LeadInput{
			Year:     2026,
			TxID:     types.Hash512{0x01},
			OutIndex: 0,
		},
		Rest: []*RestInput{
			{Year: 2026, TxIDPart: [TxIDPartLen]byte{0x10}, OutIndex: 1, TransferIndex: -1},
		},
		Outputs: []*Output{
			{
				Serial:     0,
				Config:     types.OutTypeCoin,
				Address:    addr,
				Amount:     5000,
				LockScript: []byte{0x01, 0x02},
			},
		},
	}

	err := tx.BuildHeader(1, 1700000000000)
	if err != nil {
		t.Fatalf("BuildHeader() error = %v", err)
	}

	// Header 字段应已填充
	if tx.Header.Version != 1 {
		t.Errorf("Header.Version = %d, want 1", tx.Header.Version)
	}
	if tx.Header.Timestamp != 1700000000000 {
		t.Errorf("Header.Timestamp = %d, want 1700000000000", tx.Header.Timestamp)
	}
	if tx.Header.HashInputs.IsZero() {
		t.Error("Header.HashInputs should not be zero after BuildHeader")
	}
	if tx.Header.HashOutputs.IsZero() {
		t.Error("Header.HashOutputs should not be zero after BuildHeader")
	}

	// TxID 应不为零
	txID := tx.TxID()
	if txID.IsZero() {
		t.Error("TxID() should not be zero")
	}

	// TxID 确定性
	txID2 := tx.TxID()
	if txID != txID2 {
		t.Error("TxID() should be deterministic")
	}

	// HashInputs 应等于 ComputeInputHash 的结果
	expectedInputHash := ComputeInputHash(tx.Lead, tx.Rest)
	if tx.Header.HashInputs != expectedInputHash {
		t.Error("Header.HashInputs should match ComputeInputHash result")
	}

	// HashOutputs 应等于 ComputeOutputHash 的结果
	expectedOutputHash, _ := ComputeOutputHash(tx.Outputs)
	if tx.Header.HashOutputs != expectedOutputHash {
		t.Error("Header.HashOutputs should match ComputeOutputHash result")
	}
}

// 测试 Tx.Validate 基本校验
func TestTx_Validate(t *testing.T) {
	var addr types.PubKeyHash
	addr[0] = 0x01

	// 构造有效交易
	validTx := &Tx{
		Lead: &LeadInput{
			Year:     2026,
			TxID:     types.Hash512{0x01},
			OutIndex: 0,
		},
		Outputs: []*Output{
			{
				Serial:     0,
				Config:     types.OutTypeCoin,
				Address:    addr,
				Amount:     1000,
				LockScript: []byte{0x01},
			},
		},
	}
	validTx.BuildHeader(1, 1700000000000)

	if err := validTx.Validate(); err != nil {
		t.Errorf("Validate() error = %v for valid tx", err)
	}

	// 无 Lead 输入
	noLeadTx := &Tx{
		Outputs: []*Output{
			{
				Serial:     0,
				Config:     types.OutTypeCoin,
				Address:    addr,
				Amount:     1000,
				LockScript: []byte{0x01},
			},
		},
	}
	noLeadTx.Header = TxHeader{Version: 1, Timestamp: 1700000000000, HashInputs: types.Hash512{1}, HashOutputs: types.Hash512{2}}
	if err := noLeadTx.Validate(); err == nil {
		t.Error("Validate() should fail for tx without lead input")
	}

	// 无输出
	noOutputTx := &Tx{
		Lead: &LeadInput{
			Year:     2026,
			TxID:     types.Hash512{0x01},
			OutIndex: 0,
		},
	}
	noOutputTx.Header = TxHeader{Version: 1, Timestamp: 1700000000000, HashInputs: types.Hash512{1}, HashOutputs: types.Hash512{2}}
	if err := noOutputTx.Validate(); err == nil {
		t.Error("Validate() should fail for tx without outputs")
	}
}
```

### Step 2: 运行测试验证失败

```bash
go test -v ./internal/tx/ -run "TestBinaryHashTree|TestComputeInputHash|TestComputeOutputHash|TestTx"
```

预期输出：编译失败，`BinaryHashTree`、`ComputeInputHash`、`ComputeOutputHash`、`Tx` 等未定义。

### Step 3: 写最小实现

创建 `internal/tx/hashtree.go`：

```go
package tx

import (
	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// BinaryHashTree 计算二叉哈希树的根哈希。
// 规则：
//   - 空输入返回零值 Hash512
//   - 每个叶子先 SHA-512 哈希
//   - 逐层两两合并：SHA-512(left || right)
//   - 奇数节点直接提升（不重复）
func BinaryHashTree(leaves [][]byte) types.Hash512 {
	if len(leaves) == 0 {
		return types.Hash512{}
	}

	// 计算叶子层哈希
	hashes := make([]types.Hash512, len(leaves))
	for i, leaf := range leaves {
		hashes[i] = crypto.SHA512Sum(leaf)
	}

	// 逐层合并
	for len(hashes) > 1 {
		var next []types.Hash512
		for i := 0; i < len(hashes); i += 2 {
			if i+1 < len(hashes) {
				// 两两合并
				combined := append(hashes[i][:], hashes[i+1][:]...)
				next = append(next, crypto.SHA512Sum(combined))
			} else {
				// 奇数节点直接提升
				next = append(next, hashes[i])
			}
		}
		hashes = next
	}

	return hashes[0]
}

// ComputeInputHash 计算输入项哈希。
// 规则：
//
//	LeadHash = SHA-512(lead.Serialize())
//	RestHash = SHA-512(concat(rest[0].Serialize(), rest[1].Serialize(), ...))
//	         空时为 SHA-512("")
//	InputHash = SHA-512(LeadHash || RestHash)
func ComputeInputHash(lead *LeadInput, rest []*RestInput) types.Hash512 {
	// 计算 LeadHash
	leadHash := crypto.SHA512Sum(lead.Serialize())

	// 计算 RestHash：拼接所有 rest 序列化数据后整体哈希
	var restData []byte
	for _, r := range rest {
		restData = append(restData, r.Serialize()...)
	}
	restHash := crypto.SHA512Sum(restData)

	// InputHash = SHA-512(LeadHash || RestHash)
	combined := append(leadHash[:], restHash[:]...)
	return crypto.SHA512Sum(combined)
}

// ComputeOutputHash 计算输出项哈希。
// 将每个 output 序列化为 leaf，然后调用 BinaryHashTree。
func ComputeOutputHash(outputs []*Output) (types.Hash512, error) {
	if len(outputs) == 0 {
		return types.Hash512{}, nil
	}

	leaves := make([][]byte, len(outputs))
	for i, out := range outputs {
		leaves[i] = out.Serialize()
	}

	return BinaryHashTree(leaves), nil
}
```

创建 `internal/tx/tx.go`：

```go
package tx

import (
	"errors"
	"fmt"

	"github.com/cxio/evidcoin/pkg/types"
)

// Tx 完整交易结构。
// 包含交易头、输入项、输出项和见证数据。
// TxID 仅由 Header 计算，Witnesses 不参与 TxID。
type Tx struct {
	Header    TxHeader     // 交易头
	Lead      *LeadInput   // 首项输入
	Rest      []*RestInput // 余项输入
	Outputs   []*Output    // 输出项列表
	Witnesses [][]byte     // 见证数据（签名等），不参与 TxID
}

// BuildHeader 计算输入/输出哈希并构建交易头。
// version: 交易版本号
// timestamp: 交易时间戳（Unix 毫秒）
func (tx *Tx) BuildHeader(version uint16, timestamp int64) error {
	// 计算输入哈希
	if tx.Lead == nil {
		return errors.New("tx has no lead input")
	}
	hashInputs := ComputeInputHash(tx.Lead, tx.Rest)

	// 计算输出哈希
	hashOutputs, err := ComputeOutputHash(tx.Outputs)
	if err != nil {
		return fmt.Errorf("compute output hash: %w", err)
	}

	tx.Header = TxHeader{
		Version:     version,
		Timestamp:   timestamp,
		HashInputs:  hashInputs,
		HashOutputs: hashOutputs,
	}

	return nil
}

// TxID 返回交易 ID。
// TxID = Header.TxID() = SHA-512(Header.Serialize())
func (tx *Tx) TxID() types.Hash512 {
	return tx.Header.TxID()
}

// 交易验证错误
var (
	ErrTxNoLeadInput = errors.New("tx has no lead input")
	ErrTxNoOutputs   = errors.New("tx has no outputs")
)

// Validate 对交易执行基本校验。
// 检查：
//   - Header 字段合法
//   - Lead 输入存在且合法
//   - 至少有一个输出
//   - 所有 RestInput 合法
//   - 所有 Output 合法
func (tx *Tx) Validate() error {
	// 验证 Header
	if err := tx.Header.Validate(); err != nil {
		return fmt.Errorf("validate header: %w", err)
	}

	// Lead 输入必须存在
	if tx.Lead == nil {
		return ErrTxNoLeadInput
	}
	if err := tx.Lead.Validate(); err != nil {
		return fmt.Errorf("validate lead input: %w", err)
	}

	// 至少一个输出
	if len(tx.Outputs) == 0 {
		return ErrTxNoOutputs
	}

	// 验证所有 RestInput
	for i, ri := range tx.Rest {
		if err := ri.Validate(); err != nil {
			return fmt.Errorf("validate rest input[%d]: %w", i, err)
		}
	}

	// 验证所有 Output
	for i, out := range tx.Outputs {
		if err := out.Validate(); err != nil {
			return fmt.Errorf("validate output[%d]: %w", i, err)
		}
	}

	return nil
}
```

### Step 4: 运行测试验证通过

```bash
go test -v ./internal/tx/ -run "TestBinaryHashTree|TestComputeInputHash|TestComputeOutputHash|TestTx"
```

预期输出：全部 PASS。

### Step 5: 提交

```bash
git add internal/tx/hashtree.go internal/tx/tx.go internal/tx/hashtree_test.go
git commit -m "feat(tx): add binary hash tree, input/output hash computation, and Tx struct"
```

---

## 全量测试验证

完成 Task 1-4 后，运行完整测试套件：

```bash
go test -v -count=1 ./internal/tx/...
```

预期：全部 PASS。

运行覆盖率检查：

```bash
go test -cover ./internal/tx/...
```

预期：覆盖率 ≥ 80%。

运行格式化检查：

```bash
go fmt ./internal/tx/...
```

预期：无变更。

运行编译检查：

```bash
go build ./...
```

预期：无错误。

---

## 验收标准（Task 1-4）

1. **文件结构**：以下文件全部存在
   - `internal/tx/header.go` — TxHeader 结构、序列化、TxID、Validate
   - `internal/tx/input.go` — LeadInput、RestInput 结构、序列化、Validate
   - `internal/tx/output.go` — Output 结构、CredConf、ProofConf、Serialize、SerializeContent、Validate
   - `internal/tx/util.go` — writeVarint 帮助函数
   - `internal/tx/hashtree.go` — BinaryHashTree、ComputeInputHash、ComputeOutputHash
   - `internal/tx/tx.go` — Tx 结构、BuildHeader、TxID、Validate
   - `internal/tx/header_test.go`
   - `internal/tx/input_test.go`
   - `internal/tx/output_test.go`
   - `internal/tx/hashtree_test.go`

2. **编译通过**：`go build ./...` 无错误

3. **全量测试通过**：`go test ./internal/tx/...` 全部 PASS

4. **测试覆盖率**：核心逻辑 ≥ 80%

5. **功能完整性**：
   - TxHeader 固定 138 字节序列化，TxID = SHA-512(Serialize())
   - LeadInput 固定 70 字节，RestInput 固定 28 字节
   - Output 支持 Coin/Credit/Proof/Mediator 四种类型的序列化
   - CredConf 和 ProofConf 位域操作正确
   - SerializeContent 支持 SIGCONTENT 部分签名
   - BinaryHashTree 实现奇数节点直接提升
   - ComputeInputHash 和 ComputeOutputHash 计算正确
   - Tx.BuildHeader 自动计算 HashInputs/HashOutputs
   - 所有 Validate 方法覆盖关键约束

6. **设计约束**：
   - 仅依赖 `pkg/types` 和 `pkg/crypto`
   - 错误信息英文小写、无标点
   - 注释使用中文

---

## Task 5: CoinbaseTx（铸币交易）

**Files:**
- Create: `internal/tx/coinbase.go`
- Test: `internal/tx/coinbase_test.go`

本 Task 实现铸币交易结构 `CoinbaseTx`，包括专用输出类型 `CoinbaseOutput`、1 字节配置 `CoinbaseOutConfig`、特殊的 HashInputs 计算（基于 BlockHeight/MeritProof/TotalReward/SelfData）、BuildHeader 方法、以及完整验证逻辑。

Coinbase 交易没有输入项——从无到有创建新币。它的 HashInputs 不是从 LeadInput/RestInput 计算，而是从铸币专属数据（CoinbaseInputData）计算。HashOutputs 使用与普通交易相同的 BinaryHashTree。

### Step 1: 写失败测试

创建 `internal/tx/coinbase_test.go`：

```go
package tx

import (
	"testing"

	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// --- CoinbaseOutConfig 位域测试 ---

func TestCoinbaseOutConfig_Target(t *testing.T) {
	tests := []struct {
		name   string
		config CoinbaseOutConfig
		target byte
	}{
		{"reserved", CoinbaseOutConfig(0), 0},
		{"minter", CoinbaseTargetMinter, 1},
		{"team", CoinbaseTargetTeam, 2},
		{"blockqs", CoinbaseTargetBlockqs, 3},
		{"depots", CoinbaseTargetDepots, 4},
		{"stun", CoinbaseTargetSTUN, 5},
		{"with_upper_bits", CoinbaseOutConfig(0xF3), 3}, // 高位不影响 Target
		{"max_target_mask", CoinbaseOutConfig(0x0F), 15},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.Target(); got != tt.target {
				t.Errorf("Target() = %d, want %d", got, tt.target)
			}
		})
	}
}

// --- CoinbaseOutput 序列化测试 ---

func TestCoinbaseOutput_Serialize_Basic(t *testing.T) {
	var addr types.PubKeyHash
	addr[0] = 0xAA
	script := []byte{0x01, 0x02, 0x03}

	out := &CoinbaseOutput{
		Config:  CoinbaseTargetMinter,
		Address: addr,
		Amount:  50000,
		Script:  script,
	}

	data := out.Serialize()
	if len(data) == 0 {
		t.Fatal("Serialize() returned empty data")
	}

	// 最小预期长度：Config(1) + Address(48) + Amount(varint>=1) + ScriptLen(varint>=1) + Script
	minLen := 1 + types.PubKeyHashLen + 1 + 1 + len(script)
	if len(data) < minLen {
		t.Errorf("Serialize() length = %d, less than minimum expected %d", len(data), minLen)
	}

	// 验证首字节为 Config
	if data[0] != byte(CoinbaseTargetMinter) {
		t.Errorf("first byte = %02x, want %02x", data[0], byte(CoinbaseTargetMinter))
	}

	// 验证地址区域
	for i := 0; i < types.PubKeyHashLen; i++ {
		if data[1+i] != addr[i] {
			t.Errorf("address byte[%d] = %02x, want %02x", i, data[1+i], addr[i])
			break
		}
	}
}

func TestCoinbaseOutput_Serialize_EmptyScript(t *testing.T) {
	var addr types.PubKeyHash
	addr[0] = 0xBB

	out := &CoinbaseOutput{
		Config:  CoinbaseTargetTeam,
		Address: addr,
		Amount:  10000,
		Script:  nil,
	}

	data := out.Serialize()
	// Config(1) + Address(48) + Amount(varint) + ScriptLen(varint=0)
	if len(data) < 1+types.PubKeyHashLen+1+1 {
		t.Errorf("Serialize() length = %d, too short", len(data))
	}
}

// --- CoinbaseInputData 计算测试 ---

func TestCoinbaseInputData(t *testing.T) {
	cb := &CoinbaseTx{
		BlockHeight: 100,
		MeritProof:  []byte{0x01, 0x02, 0x03},
		TotalReward: 300000000,
		SelfData:    []byte("hello coinbase"),
	}

	data := cb.coinbaseInputData()
	if len(data) == 0 {
		t.Fatal("coinbaseInputData() returned empty")
	}

	// 确定性
	data2 := cb.coinbaseInputData()
	if string(data) != string(data2) {
		t.Error("coinbaseInputData() should be deterministic")
	}
}

func TestCoinbaseInputData_EmptySelfData(t *testing.T) {
	cb := &CoinbaseTx{
		BlockHeight: 0,
		MeritProof:  []byte{0xFF},
		TotalReward: 1,
		SelfData:    nil,
	}

	data := cb.coinbaseInputData()
	if len(data) == 0 {
		t.Fatal("coinbaseInputData() returned empty for nil SelfData")
	}
}

func TestCoinbaseInputData_DifferentHeights(t *testing.T) {
	cb1 := &CoinbaseTx{
		BlockHeight: 100,
		MeritProof:  []byte{0x01},
		TotalReward: 1000,
	}
	cb2 := &CoinbaseTx{
		BlockHeight: 101,
		MeritProof:  []byte{0x01},
		TotalReward: 1000,
	}

	data1 := cb1.coinbaseInputData()
	data2 := cb2.coinbaseInputData()

	if string(data1) == string(data2) {
		t.Error("different BlockHeight should produce different CoinbaseInputData")
	}
}

// --- CoinbaseTx.BuildHeader 与 TxID 测试 ---

func TestCoinbaseTx_BuildHeader(t *testing.T) {
	var addr types.PubKeyHash
	addr[0] = 0x01

	cb := &CoinbaseTx{
		BlockHeight: 100,
		MeritProof:  []byte{0x01, 0x02, 0x03},
		TotalReward: 300000000,
		SelfData:    []byte("test"),
		Outputs: []*CoinbaseOutput{
			{Config: CoinbaseTargetMinter, Address: addr, Amount: 30000000, Script: []byte{0x01}},
			{Config: CoinbaseTargetTeam, Address: addr, Amount: 120000000, Script: []byte{0x02}},
			{Config: CoinbaseTargetBlockqs, Address: addr, Amount: 60000000, Script: []byte{0x03}},
			{Config: CoinbaseTargetDepots, Address: addr, Amount: 60000000, Script: []byte{0x04}},
			{Config: CoinbaseTargetSTUN, Address: addr, Amount: 30000000, Script: []byte{0x05}},
		},
	}

	err := cb.BuildHeader(1, 1700000000000)
	if err != nil {
		t.Fatalf("BuildHeader() error = %v", err)
	}

	// Header 字段应已填充
	if cb.Header.Version != 1 {
		t.Errorf("Header.Version = %d, want 1", cb.Header.Version)
	}
	if cb.Header.Timestamp != 1700000000000 {
		t.Errorf("Header.Timestamp = %d, want 1700000000000", cb.Header.Timestamp)
	}
	if cb.Header.HashInputs.IsZero() {
		t.Error("Header.HashInputs should not be zero after BuildHeader")
	}
	if cb.Header.HashOutputs.IsZero() {
		t.Error("Header.HashOutputs should not be zero after BuildHeader")
	}

	// 验证 HashInputs = SHA-512(CoinbaseInputData)
	expectedInputHash := crypto.SHA512Sum(cb.coinbaseInputData())
	if cb.Header.HashInputs != expectedInputHash {
		t.Error("Header.HashInputs should equal SHA512(coinbaseInputData())")
	}

	// 验证 HashOutputs = BinaryHashTree(输出序列化列表)
	leaves := make([][]byte, len(cb.Outputs))
	for i, out := range cb.Outputs {
		leaves[i] = out.Serialize()
	}
	expectedOutputHash := BinaryHashTree(leaves)
	if cb.Header.HashOutputs != expectedOutputHash {
		t.Error("Header.HashOutputs should equal BinaryHashTree(outputs)")
	}
}

func TestCoinbaseTx_TxID(t *testing.T) {
	var addr types.PubKeyHash
	addr[0] = 0x01

	cb := &CoinbaseTx{
		BlockHeight: 50,
		MeritProof:  []byte{0xAA},
		TotalReward: 100000,
		Outputs: []*CoinbaseOutput{
			{Config: CoinbaseTargetMinter, Address: addr, Amount: 100000, Script: []byte{0x01}},
		},
	}

	cb.BuildHeader(1, 1700000000000)

	// TxID 应不为零
	txID := cb.TxID()
	if txID.IsZero() {
		t.Error("TxID() should not be zero")
	}

	// TxID 确定性
	txID2 := cb.TxID()
	if txID != txID2 {
		t.Error("TxID() should be deterministic")
	}

	// TxID = SHA-512(Header.Serialize())
	expected := crypto.SHA512Sum(cb.Header.Serialize())
	if txID != expected {
		t.Error("TxID() should equal SHA512(Header.Serialize())")
	}
}

func TestCoinbaseTx_BuildHeader_NoOutputs(t *testing.T) {
	cb := &CoinbaseTx{
		BlockHeight: 100,
		MeritProof:  []byte{0x01},
		TotalReward: 1000,
		Outputs:     nil,
	}

	err := cb.BuildHeader(1, 1700000000000)
	if err == nil {
		t.Error("BuildHeader() should fail with no outputs")
	}
}

// --- CoinbaseTx.Validate 测试 ---

func TestCoinbaseTx_Validate(t *testing.T) {
	var addr types.PubKeyHash
	addr[0] = 0x01

	// 构造有效 Coinbase 交易
	makeValid := func() *CoinbaseTx {
		cb := &CoinbaseTx{
			BlockHeight: 100,
			MeritProof:  []byte{0x01, 0x02, 0x03},
			TotalReward: 300000000,
			SelfData:    []byte("test"),
			Outputs: []*CoinbaseOutput{
				{Config: CoinbaseTargetMinter, Address: addr, Amount: 30000000, Script: []byte{0x01}},
				{Config: CoinbaseTargetTeam, Address: addr, Amount: 120000000, Script: []byte{0x02}},
				{Config: CoinbaseTargetBlockqs, Address: addr, Amount: 60000000, Script: []byte{0x03}},
				{Config: CoinbaseTargetDepots, Address: addr, Amount: 60000000, Script: []byte{0x04}},
				{Config: CoinbaseTargetSTUN, Address: addr, Amount: 30000000, Script: []byte{0x05}},
			},
		}
		cb.BuildHeader(1, 1700000000000)
		return cb
	}

	tests := []struct {
		name    string
		modify  func(cb *CoinbaseTx)
		wantErr bool
	}{
		{
			name:    "valid",
			modify:  func(cb *CoinbaseTx) {},
			wantErr: false,
		},
		{
			name: "negative_block_height",
			modify: func(cb *CoinbaseTx) {
				cb.BlockHeight = -1
			},
			wantErr: true,
		},
		{
			name: "zero_total_reward",
			modify: func(cb *CoinbaseTx) {
				cb.TotalReward = 0
			},
			wantErr: true,
		},
		{
			name: "negative_total_reward",
			modify: func(cb *CoinbaseTx) {
				cb.TotalReward = -100
			},
			wantErr: true,
		},
		{
			name: "empty_merit_proof",
			modify: func(cb *CoinbaseTx) {
				cb.MeritProof = nil
			},
			wantErr: true,
		},
		{
			name: "selfdata_too_long",
			modify: func(cb *CoinbaseTx) {
				cb.SelfData = make([]byte, types.MaxSelfData+1)
			},
			wantErr: true,
		},
		{
			name: "no_outputs",
			modify: func(cb *CoinbaseTx) {
				cb.Outputs = nil
			},
			wantErr: true,
		},
		{
			name: "output_target_zero",
			modify: func(cb *CoinbaseTx) {
				cb.Outputs[0].Config = CoinbaseOutConfig(0) // Reserved
			},
			wantErr: true,
		},
		{
			name: "output_target_too_high",
			modify: func(cb *CoinbaseTx) {
				cb.Outputs[0].Config = CoinbaseOutConfig(6)
			},
			wantErr: true,
		},
		{
			name: "output_sum_mismatch",
			modify: func(cb *CoinbaseTx) {
				cb.Outputs[0].Amount = 1 // 破坏总和
			},
			wantErr: true,
		},
		{
			name: "output_zero_amount",
			modify: func(cb *CoinbaseTx) {
				cb.Outputs[0].Amount = 0
			},
			wantErr: true,
		},
		{
			name: "output_negative_amount",
			modify: func(cb *CoinbaseTx) {
				cb.Outputs[0].Amount = -100
			},
			wantErr: true,
		},
		{
			name: "output_zero_address",
			modify: func(cb *CoinbaseTx) {
				cb.Outputs[0].Address = types.PubKeyHash{} // 全零
			},
			wantErr: true,
		},
		{
			name: "valid_height_zero",
			modify: func(cb *CoinbaseTx) {
				cb.BlockHeight = 0 // 创世区块高度 0 是合法的
			},
			wantErr: false,
		},
		{
			name:    "valid_empty_selfdata",
			modify: func(cb *CoinbaseTx) {
				cb.SelfData = nil
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := makeValid()
			tt.modify(cb)
			err := cb.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// 测试 Coinbase 输出金额总和必须等于 TotalReward
func TestCoinbaseTx_Validate_SumExact(t *testing.T) {
	var addr types.PubKeyHash
	addr[0] = 0x01

	// 精确匹配
	cb := &CoinbaseTx{
		BlockHeight: 100,
		MeritProof:  []byte{0x01},
		TotalReward: 100,
		Outputs: []*CoinbaseOutput{
			{Config: CoinbaseTargetMinter, Address: addr, Amount: 40, Script: []byte{0x01}},
			{Config: CoinbaseTargetTeam, Address: addr, Amount: 60, Script: []byte{0x02}},
		},
	}
	cb.BuildHeader(1, 1700000000000)

	if err := cb.Validate(); err != nil {
		t.Errorf("Validate() error = %v for exact sum", err)
	}

	// 超额
	cb.Outputs[1].Amount = 61
	if err := cb.Validate(); err == nil {
		t.Error("Validate() should fail when output sum > TotalReward")
	}

	// 不足
	cb.Outputs[1].Amount = 59
	if err := cb.Validate(); err == nil {
		t.Error("Validate() should fail when output sum < TotalReward")
	}
}
```

### Step 2: 运行测试验证失败

```bash
go test -v ./internal/tx/ -run "TestCoinbaseOutConfig|TestCoinbaseOutput|TestCoinbaseInputData|TestCoinbaseTx"
```

预期输出：编译失败，`CoinbaseOutConfig`、`CoinbaseOutput`、`CoinbaseTx` 等未定义。

### Step 3: 写最小实现

创建 `internal/tx/coinbase.go`：

```go
package tx

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// --- CoinbaseOutConfig 铸币输出配置 ---

// CoinbaseOutConfig Coinbase 专用输出配置（1 字节）。
// 低 4 位编码奖励目标类型。
type CoinbaseOutConfig byte

const (
	// CoinbaseTargetMask 奖励目标掩码（低 4 位）。
	CoinbaseTargetMask CoinbaseOutConfig = 0x0F

	// CoinbaseTargetMinter 铸凭者奖励（10%）。
	CoinbaseTargetMinter CoinbaseOutConfig = 1

	// CoinbaseTargetTeam 校验组奖励（40%）。
	CoinbaseTargetTeam CoinbaseOutConfig = 2

	// CoinbaseTargetBlockqs 区块查询服务奖励（20%）。
	CoinbaseTargetBlockqs CoinbaseOutConfig = 3

	// CoinbaseTargetDepots 数据驿站服务奖励（20%）。
	CoinbaseTargetDepots CoinbaseOutConfig = 4

	// CoinbaseTargetSTUN NAT 穿透服务奖励（10%）。
	CoinbaseTargetSTUN CoinbaseOutConfig = 5
)

// Target 返回奖励目标类型值（低 4 位）。
func (c CoinbaseOutConfig) Target() byte {
	return byte(c & CoinbaseTargetMask)
}

// --- CoinbaseOutput 铸币专用输出 ---

// CoinbaseOutput Coinbase 专用输出项。
// 每个输出对应一个奖励接收方。
type CoinbaseOutput struct {
	Config  CoinbaseOutConfig // 1 字节配置（低 4 位为奖励目标）
	Address types.PubKeyHash  // 接收者地址
	Amount  int64             // 金额
	Script  []byte            // 锁定脚本（含 SYS_AWARD 等特殊指令）
}

// Serialize 将 Coinbase 输出序列化为二进制格式。
// 格式：Config(1) || Address(48) || Amount(varint) || ScriptLen(varint) || Script
func (o *CoinbaseOutput) Serialize() []byte {
	var buf bytes.Buffer

	// Config: 1 字节
	buf.WriteByte(byte(o.Config))

	// Address: 48 字节
	buf.Write(o.Address[:])

	// Amount: 变长整数
	writeVarint(&buf, uint64(o.Amount))

	// ScriptLen + Script
	writeVarint(&buf, uint64(len(o.Script)))
	buf.Write(o.Script)

	return buf.Bytes()
}

// --- CoinbaseTx 铸币交易 ---

// CoinbaseTx 铸币交易结构。
// 每个区块的第一笔交易，没有输入项——从无到有创建新币。
// HashInputs 由 BlockHeight/MeritProof/TotalReward/SelfData 计算。
type CoinbaseTx struct {
	Header      TxHeader          // 标准交易头
	BlockHeight int               // 区块高度（变长整数编码）
	MeritProof  []byte            // 择优凭证（铸造者证明）
	TotalReward int64             // 收益总额
	SelfData    []byte            // 自由数据（最大 255 字节）
	Outputs     []*CoinbaseOutput // 收益分配输出
}

// coinbaseInputData 构造 Coinbase 特殊输入数据。
// 格式：BlockHeight(varint) || MeritProofLen(varint) || MeritProof || TotalReward(varint) || SelfDataLen(1) || SelfData
// 该数据用于计算 HashInputs = SHA-512(coinbaseInputData())。
func (cb *CoinbaseTx) coinbaseInputData() []byte {
	var buf bytes.Buffer

	// BlockHeight: 变长整数（使用 int64 -> uint64 zigzag 或直接 uint64）
	writeVarint(&buf, uint64(cb.BlockHeight))

	// MeritProofLen + MeritProof
	writeVarint(&buf, uint64(len(cb.MeritProof)))
	buf.Write(cb.MeritProof)

	// TotalReward: 变长整数
	writeVarint(&buf, uint64(cb.TotalReward))

	// SelfDataLen: 1 字节 + SelfData
	buf.WriteByte(byte(len(cb.SelfData)))
	buf.Write(cb.SelfData)

	return buf.Bytes()
}

// BuildHeader 计算哈希并构建 Coinbase 交易头。
// HashInputs = SHA-512(coinbaseInputData())
// HashOutputs = BinaryHashTree(输出项序列化列表)
func (cb *CoinbaseTx) BuildHeader(version uint16, timestamp int64) error {
	if len(cb.Outputs) == 0 {
		return errors.New("coinbase tx has no outputs")
	}

	// 计算 HashInputs
	hashInputs := crypto.SHA512Sum(cb.coinbaseInputData())

	// 计算 HashOutputs = BinaryHashTree(输出序列化)
	leaves := make([][]byte, len(cb.Outputs))
	for i, out := range cb.Outputs {
		leaves[i] = out.Serialize()
	}
	hashOutputs := BinaryHashTree(leaves)

	cb.Header = TxHeader{
		Version:     version,
		Timestamp:   timestamp,
		HashInputs:  hashInputs,
		HashOutputs: hashOutputs,
	}

	return nil
}

// TxID 返回 Coinbase 交易 ID。
// TxID = SHA-512(Header.Serialize()) —— 与普通交易一致。
func (cb *CoinbaseTx) TxID() types.Hash512 {
	return cb.Header.TxID()
}

// --- CoinbaseTx 验证 ---

// Coinbase 验证错误
var (
	ErrCoinbaseNegativeHeight   = errors.New("coinbase block height is negative")
	ErrCoinbaseInvalidReward    = errors.New("coinbase total reward must be positive")
	ErrCoinbaseEmptyMeritProof  = errors.New("coinbase merit proof is empty")
	ErrCoinbaseSelfDataTooLong  = errors.New("coinbase self data exceeds max length")
	ErrCoinbaseNoOutputs        = errors.New("coinbase tx has no outputs")
	ErrCoinbaseInvalidTarget    = errors.New("coinbase output target must be 1-5")
	ErrCoinbaseInvalidAmount    = errors.New("coinbase output amount must be positive")
	ErrCoinbaseZeroAddress      = errors.New("coinbase output address is zero")
	ErrCoinbaseSumMismatch      = errors.New("coinbase output sum does not equal total reward")
)

// Validate 对 Coinbase 交易执行验证。
// 检查：
//   - BlockHeight >= 0
//   - TotalReward > 0
//   - MeritProof 非空
//   - SelfData 长度 <= MaxSelfData
//   - 至少一个输出
//   - 所有输出的 Target 值在 1-5 之间
//   - 所有输出金额为正
//   - 所有输出地址非零
//   - 输出金额总和 == TotalReward
func (cb *CoinbaseTx) Validate() error {
	if cb.BlockHeight < 0 {
		return ErrCoinbaseNegativeHeight
	}
	if cb.TotalReward <= 0 {
		return ErrCoinbaseInvalidReward
	}
	if len(cb.MeritProof) == 0 {
		return ErrCoinbaseEmptyMeritProof
	}
	if len(cb.SelfData) > types.MaxSelfData {
		return ErrCoinbaseSelfDataTooLong
	}
	if len(cb.Outputs) == 0 {
		return ErrCoinbaseNoOutputs
	}

	var sum int64
	for i, out := range cb.Outputs {
		target := out.Config.Target()
		if target < 1 || target > 5 {
			return fmt.Errorf("output[%d]: %w", i, ErrCoinbaseInvalidTarget)
		}
		if out.Amount <= 0 {
			return fmt.Errorf("output[%d]: %w", i, ErrCoinbaseInvalidAmount)
		}
		if out.Address.IsZero() {
			return fmt.Errorf("output[%d]: %w", i, ErrCoinbaseZeroAddress)
		}
		sum += out.Amount
	}

	if sum != cb.TotalReward {
		return ErrCoinbaseSumMismatch
	}

	return nil
}
```

### Step 4: 运行测试验证通过

```bash
go test -v ./internal/tx/ -run "TestCoinbaseOutConfig|TestCoinbaseOutput|TestCoinbaseInputData|TestCoinbaseTx"
```

预期输出：全部 PASS。

### Step 5: 提交

```bash
git add internal/tx/coinbase.go internal/tx/coinbase_test.go
git commit -m "feat(tx): add CoinbaseTx with specialized output, input data hash, and validation"
```

---

## Task 6: AttachmentID（附件标识）

**Files:**
- Create: `internal/tx/attachment.go`
- Test: `internal/tx/attachment_test.go`

本 Task 实现附件标识结构 `AttachmentID`，包括基于 FingerprintLv 的变长指纹、ShardTreeHash 条件存在逻辑、序列化与反序列化、以及字段验证。附件标识嵌入在交易输出中，用于引用链下 Depots 网络存储的附件数据。

关键特点：
- 指纹长度由 `FingerprintLv` 决定：`FingerprintLen = 16 + FingerprintLv * 4`（范围 16-64 字节）
- `ShardTreeHash` 字段仅在 `ShardCount > 0` 时存在
- `TotalLen` 必须与实际序列化长度一致

### Step 1: 写失败测试

创建 `internal/tx/attachment_test.go`：

```go
package tx

import (
	"bytes"
	"testing"
)

// --- 指纹长度等级映射测试 ---

func TestFingerprintLen(t *testing.T) {
	tests := []struct {
		name     string
		level    uint8
		expected int
	}{
		{"level_0_min", 0, 16},
		{"level_1", 1, 20},
		{"level_2", 2, 24},
		{"level_3", 3, 28},
		{"level_6_mid", 6, 40},
		{"level_12_max", 12, 64},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FingerprintLength(tt.level)
			if got != tt.expected {
				t.Errorf("FingerprintLength(%d) = %d, want %d", tt.level, got, tt.expected)
			}
		})
	}
}

// --- AttachmentID 序列化/反序列化往返测试 ---

func TestAttachmentID_Roundtrip_NoShard(t *testing.T) {
	// ShardCount = 0：无分片，ShardTreeHash 不存在
	fingerprint := make([]byte, 16) // Level 0 = 16 字节
	fingerprint[0] = 0xAA
	fingerprint[15] = 0xBB

	original := &AttachmentID{
		MajorType:     1,
		MinorType:     2,
		FingerprintLv: 0,
		Fingerprint:   fingerprint,
		ShardCount:    0,
		DataSize:      1024,
	}
	original.TotalLen = original.computeTotalLen()

	data := original.Serialize()

	// TotalLen 应等于序列化长度
	if int(original.TotalLen) != len(data) {
		t.Errorf("TotalLen = %d, but Serialize() length = %d", original.TotalLen, len(data))
	}

	restored, err := DeserializeAttachmentID(data)
	if err != nil {
		t.Fatalf("DeserializeAttachmentID() error = %v", err)
	}

	if restored.TotalLen != original.TotalLen {
		t.Errorf("TotalLen = %d, want %d", restored.TotalLen, original.TotalLen)
	}
	if restored.MajorType != original.MajorType {
		t.Errorf("MajorType = %d, want %d", restored.MajorType, original.MajorType)
	}
	if restored.MinorType != original.MinorType {
		t.Errorf("MinorType = %d, want %d", restored.MinorType, original.MinorType)
	}
	if restored.FingerprintLv != original.FingerprintLv {
		t.Errorf("FingerprintLv = %d, want %d", restored.FingerprintLv, original.FingerprintLv)
	}
	if !bytes.Equal(restored.Fingerprint, original.Fingerprint) {
		t.Error("Fingerprint mismatch after roundtrip")
	}
	if restored.ShardCount != 0 {
		t.Errorf("ShardCount = %d, want 0", restored.ShardCount)
	}
	if restored.DataSize != original.DataSize {
		t.Errorf("DataSize = %d, want %d", restored.DataSize, original.DataSize)
	}
}

func TestAttachmentID_Roundtrip_SingleShard(t *testing.T) {
	// ShardCount = 1：ShardTreeHash 存在，值为附件本身哈希
	fingerprint := make([]byte, 20) // Level 1 = 20 字节
	fingerprint[0] = 0xCC

	var shardHash [48]byte
	for i := range shardHash {
		shardHash[i] = byte(i + 10)
	}

	original := &AttachmentID{
		MajorType:     3,
		MinorType:     4,
		FingerprintLv: 1,
		Fingerprint:   fingerprint,
		ShardCount:    1,
		ShardTreeHash: shardHash,
		DataSize:      2048000,
	}
	original.TotalLen = original.computeTotalLen()

	data := original.Serialize()

	if int(original.TotalLen) != len(data) {
		t.Errorf("TotalLen = %d, but Serialize() length = %d", original.TotalLen, len(data))
	}

	restored, err := DeserializeAttachmentID(data)
	if err != nil {
		t.Fatalf("DeserializeAttachmentID() error = %v", err)
	}

	if restored.ShardCount != 1 {
		t.Errorf("ShardCount = %d, want 1", restored.ShardCount)
	}
	if restored.ShardTreeHash != original.ShardTreeHash {
		t.Error("ShardTreeHash mismatch after roundtrip")
	}
	if restored.DataSize != original.DataSize {
		t.Errorf("DataSize = %d, want %d", restored.DataSize, original.DataSize)
	}
}

func TestAttachmentID_Roundtrip_MultiShard(t *testing.T) {
	// ShardCount > 1：正常分片
	fingerprint := make([]byte, 64) // Level 12 = 64 字节
	for i := range fingerprint {
		fingerprint[i] = byte(i)
	}

	var shardHash [48]byte
	for i := range shardHash {
		shardHash[i] = byte(i * 3)
	}

	original := &AttachmentID{
		MajorType:     10,
		MinorType:     20,
		FingerprintLv: 12,
		Fingerprint:   fingerprint,
		ShardCount:    1000,
		ShardTreeHash: shardHash,
		DataSize:      134217728, // 128 MB
	}
	original.TotalLen = original.computeTotalLen()

	data := original.Serialize()

	if int(original.TotalLen) != len(data) {
		t.Errorf("TotalLen = %d, but Serialize() length = %d", original.TotalLen, len(data))
	}

	restored, err := DeserializeAttachmentID(data)
	if err != nil {
		t.Fatalf("DeserializeAttachmentID() error = %v", err)
	}

	if restored.FingerprintLv != 12 {
		t.Errorf("FingerprintLv = %d, want 12", restored.FingerprintLv)
	}
	if len(restored.Fingerprint) != 64 {
		t.Errorf("Fingerprint length = %d, want 64", len(restored.Fingerprint))
	}
	if !bytes.Equal(restored.Fingerprint, original.Fingerprint) {
		t.Error("Fingerprint mismatch after roundtrip")
	}
	if restored.ShardCount != 1000 {
		t.Errorf("ShardCount = %d, want 1000", restored.ShardCount)
	}
	if restored.ShardTreeHash != original.ShardTreeHash {
		t.Error("ShardTreeHash mismatch after roundtrip")
	}
	if restored.DataSize != original.DataSize {
		t.Errorf("DataSize = %d, want %d", restored.DataSize, original.DataSize)
	}
}

// --- AttachmentID.Validate 测试 ---

func TestAttachmentID_Validate(t *testing.T) {
	// 构造有效附件 ID
	makeValid := func() *AttachmentID {
		fp := make([]byte, 16)
		fp[0] = 0x01
		aid := &AttachmentID{
			MajorType:     1,
			MinorType:     2,
			FingerprintLv: 0,
			Fingerprint:   fp,
			ShardCount:    0,
			DataSize:      1024,
		}
		aid.TotalLen = aid.computeTotalLen()
		return aid
	}

	tests := []struct {
		name    string
		modify  func(aid *AttachmentID)
		wantErr bool
	}{
		{
			name:    "valid_no_shard",
			modify:  func(aid *AttachmentID) {},
			wantErr: false,
		},
		{
			name: "valid_with_shard",
			modify: func(aid *AttachmentID) {
				aid.ShardCount = 5
				aid.ShardTreeHash = [48]byte{0x01}
				aid.TotalLen = aid.computeTotalLen()
			},
			wantErr: false,
		},
		{
			name: "fingerprint_lv_too_high",
			modify: func(aid *AttachmentID) {
				aid.FingerprintLv = 13
			},
			wantErr: true,
		},
		{
			name: "fingerprint_length_mismatch",
			modify: func(aid *AttachmentID) {
				// FingerprintLv = 0 需要 16 字节，但提供 20 字节
				aid.Fingerprint = make([]byte, 20)
			},
			wantErr: true,
		},
		{
			name: "fingerprint_too_short",
			modify: func(aid *AttachmentID) {
				// FingerprintLv = 0 需要 16 字节，但只有 10 字节
				aid.Fingerprint = make([]byte, 10)
			},
			wantErr: true,
		},
		{
			name: "negative_data_size",
			modify: func(aid *AttachmentID) {
				aid.DataSize = -1
			},
			wantErr: true,
		},
		{
			name: "total_len_mismatch",
			modify: func(aid *AttachmentID) {
				aid.TotalLen = 99 // 故意不匹配
			},
			wantErr: true,
		},
		{
			name: "shard_zero_but_hash_nonzero",
			modify: func(aid *AttachmentID) {
				aid.ShardCount = 0
				aid.ShardTreeHash = [48]byte{0x01} // ShardCount=0 时不应有非零哈希
			},
			wantErr: true,
		},
		{
			name: "valid_datasize_zero",
			modify: func(aid *AttachmentID) {
				aid.DataSize = 0
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aid := makeValid()
			tt.modify(aid)
			err := aid.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// --- 反序列化错误处理测试 ---

func TestDeserializeAttachmentID_Errors(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"nil", nil},
		{"empty", []byte{}},
		{"too_short_for_header", []byte{10, 1, 2}},               // 不足 4 字节头
		{"total_len_mismatch", []byte{100, 1, 2, 0, 0x01}},       // TotalLen=100 但数据不足
		{"fingerprint_lv_13", []byte{10, 1, 2, 13}},              // 非法 FingerprintLv
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DeserializeAttachmentID(tt.data)
			if err == nil {
				t.Error("DeserializeAttachmentID() should return error")
			}
		})
	}
}

// --- computeTotalLen 内部一致性测试 ---

func TestAttachmentID_ComputeTotalLen(t *testing.T) {
	tests := []struct {
		name       string
		level      uint8
		shardCount uint16
	}{
		{"level0_no_shard", 0, 0},
		{"level0_single_shard", 0, 1},
		{"level6_multi_shard", 6, 100},
		{"level12_max_fp", 12, 65535},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fpLen := FingerprintLength(tt.level)
			fp := make([]byte, fpLen)
			fp[0] = 0x01

			aid := &AttachmentID{
				MajorType:     1,
				MinorType:     2,
				FingerprintLv: tt.level,
				Fingerprint:   fp,
				ShardCount:    tt.shardCount,
				DataSize:      42,
			}
			if tt.shardCount > 0 {
				aid.ShardTreeHash = [48]byte{0x01}
			}
			aid.TotalLen = aid.computeTotalLen()

			data := aid.Serialize()
			if int(aid.TotalLen) != len(data) {
				t.Errorf("computeTotalLen() = %d, but Serialize() length = %d", aid.TotalLen, len(data))
			}
		})
	}
}
```

### Step 2: 运行测试验证失败

```bash
go test -v ./internal/tx/ -run "TestFingerprintLen|TestAttachmentID|TestDeserializeAttachmentID"
```

预期输出：编译失败，`AttachmentID`、`FingerprintLength`、`DeserializeAttachmentID` 等未定义。

### Step 3: 写最小实现

创建 `internal/tx/attachment.go`：

```go
package tx

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/cxio/evidcoin/pkg/types"
)

// 附件指纹最小长度和步长
const (
	fingerprintBaseLen = 16 // 基础指纹长度（字节）
	fingerprintStep    = 4  // 每等级增量（字节）
	maxFingerprintLv   = 12 // 最大指纹等级
)

// FingerprintLength 根据指纹等级计算指纹长度。
// 公式：16 + level * 4，范围 16-64 字节。
func FingerprintLength(level uint8) int {
	return fingerprintBaseLen + int(level)*fingerprintStep
}

// AttachmentID 附件标识结构。
// 嵌入在交易输出中，用于引用链下 Depots 网络存储的附件数据。
// ShardTreeHash 字段仅在 ShardCount > 0 时存在。
type AttachmentID struct {
	TotalLen      uint8    // ID 总长（1 字节）
	MajorType     uint8    // 附件大类（1 字节）
	MinorType     uint8    // 附件小类（1 字节）
	FingerprintLv uint8    // 指纹强度等级（0=16B, 1=20B, ..., 12=64B）
	Fingerprint   []byte   // BLAKE3 哈希指纹（16-64 字节）
	ShardCount    uint16   // 分片数量
	ShardTreeHash [48]byte // SHA3-384 片组哈希树根——仅 ShardCount > 0 时存在
	DataSize      int64    // 附件大小（变长整数编码）
}

// computeTotalLen 计算序列化后的总长度。
// 格式：TotalLen(1) || MajorType(1) || MinorType(1) || FingerprintLv(1) || Fingerprint(变长)
//
//	|| ShardCount(2,LE) || [ShardTreeHash(48)] || DataSize(varint)
func (a *AttachmentID) computeTotalLen() uint8 {
	// 固定部分：TotalLen(1) + MajorType(1) + MinorType(1) + FingerprintLv(1) = 4
	n := 4

	// 指纹长度
	n += FingerprintLength(a.FingerprintLv)

	// ShardCount: 2 字节
	n += 2

	// ShardTreeHash: 仅 ShardCount > 0 时存在
	if a.ShardCount > 0 {
		n += types.Hash384Len
	}

	// DataSize: 变长整数
	n += types.VarintSize(uint64(a.DataSize))

	return uint8(n)
}

// Serialize 将附件标识序列化为二进制格式。
// 格式：TotalLen(1) || MajorType(1) || MinorType(1) || FingerprintLv(1)
//
//	|| Fingerprint(变长) || ShardCount(2,LE) || [ShardTreeHash(48)] || DataSize(varint)
func (a *AttachmentID) Serialize() []byte {
	var buf bytes.Buffer

	// TotalLen: 1 字节
	buf.WriteByte(a.TotalLen)

	// MajorType: 1 字节
	buf.WriteByte(a.MajorType)

	// MinorType: 1 字节
	buf.WriteByte(a.MinorType)

	// FingerprintLv: 1 字节
	buf.WriteByte(a.FingerprintLv)

	// Fingerprint: 变长
	buf.Write(a.Fingerprint)

	// ShardCount: 2 字节小端序
	var tmp [2]byte
	binary.LittleEndian.PutUint16(tmp[:], a.ShardCount)
	buf.Write(tmp[:])

	// ShardTreeHash: 仅 ShardCount > 0 时写入
	if a.ShardCount > 0 {
		buf.Write(a.ShardTreeHash[:])
	}

	// DataSize: 变长整数
	writeVarint(&buf, uint64(a.DataSize))

	return buf.Bytes()
}

// DeserializeAttachmentID 从字节流反序列化附件标识。
func DeserializeAttachmentID(data []byte) (*AttachmentID, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("data too short for attachment id header: got %d, want >= 4", len(data))
	}

	a := &AttachmentID{}
	offset := 0

	// TotalLen
	a.TotalLen = data[offset]
	offset++

	// 检查数据长度是否足够
	if len(data) < int(a.TotalLen) {
		return nil, fmt.Errorf("data too short: got %d, want %d", len(data), a.TotalLen)
	}

	// MajorType
	a.MajorType = data[offset]
	offset++

	// MinorType
	a.MinorType = data[offset]
	offset++

	// FingerprintLv
	a.FingerprintLv = data[offset]
	offset++

	if a.FingerprintLv > maxFingerprintLv {
		return nil, fmt.Errorf("fingerprint level %d exceeds max %d", a.FingerprintLv, maxFingerprintLv)
	}

	// Fingerprint
	fpLen := FingerprintLength(a.FingerprintLv)
	if offset+fpLen > len(data) {
		return nil, fmt.Errorf("data too short for fingerprint: need %d more bytes at offset %d", fpLen, offset)
	}
	a.Fingerprint = make([]byte, fpLen)
	copy(a.Fingerprint, data[offset:offset+fpLen])
	offset += fpLen

	// ShardCount: 2 字节小端序
	if offset+2 > len(data) {
		return nil, fmt.Errorf("data too short for shard count at offset %d", offset)
	}
	a.ShardCount = binary.LittleEndian.Uint16(data[offset : offset+2])
	offset += 2

	// ShardTreeHash: 仅 ShardCount > 0 时存在
	if a.ShardCount > 0 {
		if offset+types.Hash384Len > len(data) {
			return nil, fmt.Errorf("data too short for shard tree hash at offset %d", offset)
		}
		copy(a.ShardTreeHash[:], data[offset:offset+types.Hash384Len])
		offset += types.Hash384Len
	}

	// DataSize: 变长整数
	if offset >= len(data) {
		return nil, fmt.Errorf("data too short for data size at offset %d", offset)
	}
	v, n := types.Varint(data[offset:])
	if n <= 0 {
		return nil, errors.New("failed to decode data size varint")
	}
	a.DataSize = int64(v)

	return a, nil
}

// --- AttachmentID 验证 ---

// 附件标识验证错误
var (
	ErrAttachFingerprintLvTooHigh = errors.New("attachment fingerprint level exceeds max 12")
	ErrAttachFingerprintLenWrong  = errors.New("attachment fingerprint length does not match level")
	ErrAttachNegativeDataSize     = errors.New("attachment data size is negative")
	ErrAttachTotalLenMismatch     = errors.New("attachment total length does not match serialized length")
	ErrAttachShardHashNotEmpty    = errors.New("attachment shard tree hash should be zero when shard count is 0")
)

// Validate 对附件标识执行验证。
// 检查：
//   - FingerprintLv <= 12
//   - len(Fingerprint) == 16 + FingerprintLv * 4
//   - ShardCount == 0 时 ShardTreeHash 应全零
//   - DataSize >= 0
//   - TotalLen 与实际序列化长度一致
func (a *AttachmentID) Validate() error {
	if a.FingerprintLv > maxFingerprintLv {
		return ErrAttachFingerprintLvTooHigh
	}

	expectedFPLen := FingerprintLength(a.FingerprintLv)
	if len(a.Fingerprint) != expectedFPLen {
		return ErrAttachFingerprintLenWrong
	}

	if a.DataSize < 0 {
		return ErrAttachNegativeDataSize
	}

	// ShardCount == 0 时 ShardTreeHash 必须全零
	if a.ShardCount == 0 {
		var zero [48]byte
		if a.ShardTreeHash != zero {
			return ErrAttachShardHashNotEmpty
		}
	}

	// TotalLen 一致性检查
	expected := a.computeTotalLen()
	if a.TotalLen != expected {
		return ErrAttachTotalLenMismatch
	}

	return nil
}
```

### Step 4: 运行测试验证通过

```bash
go test -v ./internal/tx/ -run "TestFingerprintLen|TestAttachmentID|TestDeserializeAttachmentID"
```

预期输出：全部 PASS。

### Step 5: 提交

```bash
git add internal/tx/attachment.go internal/tx/attachment_test.go
git commit -m "feat(tx): add AttachmentID with variable fingerprint, shard hash, and validation"
```

---

## 全量测试验证（Task 1-6）

完成 Task 5-6 后，运行完整测试套件：

```bash
go test -v -count=1 ./internal/tx/...
```

预期：全部 PASS。

运行覆盖率检查：

```bash
go test -cover ./internal/tx/...
```

预期：覆盖率 ≥ 80%。

运行格式化检查：

```bash
go fmt ./internal/tx/...
```

预期：无变更。

运行编译检查：

```bash
go build ./...
```

预期：无错误。

---

## 验收标准（Task 5-6）

1. **文件结构**：以下文件全部存在
   - `internal/tx/coinbase.go` — CoinbaseOutConfig、CoinbaseOutput、CoinbaseTx、BuildHeader、Validate
   - `internal/tx/coinbase_test.go`
   - `internal/tx/attachment.go` — AttachmentID、FingerprintLength、Serialize、DeserializeAttachmentID、Validate
   - `internal/tx/attachment_test.go`

2. **编译通过**：`go build ./...` 无错误

3. **全量测试通过**：`go test ./internal/tx/...` 全部 PASS

4. **测试覆盖率**：核心逻辑 ≥ 80%

5. **功能完整性**：
   - CoinbaseOutConfig 低 4 位正确提取 Target 值（1-5）
   - CoinbaseOutput 序列化格式：Config(1) || Address(48) || Amount(varint) || ScriptLen(varint) || Script
   - CoinbaseTx.coinbaseInputData 构造格式：BlockHeight(varint) || MeritProofLen(varint) || MeritProof || TotalReward(varint) || SelfDataLen(1) || SelfData
   - CoinbaseTx.BuildHeader 正确计算 HashInputs = SHA-512(coinbaseInputData) 和 HashOutputs = BinaryHashTree(输出序列化)
   - CoinbaseTx.TxID = SHA-512(Header.Serialize()) 与普通交易一致
   - CoinbaseTx.Validate 覆盖：BlockHeight >= 0、TotalReward > 0、MeritProof 非空、SelfData <= 255、输出 Target 1-5、输出金额总和 == TotalReward
   - FingerprintLength 正确映射等级到长度（16 + level * 4）
   - AttachmentID 序列化/反序列化在三种 ShardCount 情况下（0、1、>1）均正确往返
   - AttachmentID.Validate 覆盖：FingerprintLv <= 12、指纹长度一致、ShardCount=0 时哈希全零、DataSize >= 0、TotalLen 一致性

6. **设计约束**：
   - 仅依赖 `pkg/types` 和 `pkg/crypto`
   - 复用已有的 `writeVarint`、`BinaryHashTree` 等工具函数
   - 错误信息英文小写、无标点
   - 注释使用中文

---

## Task 7: SigFlag 签名数据构造

**Files:**
- Create: `internal/tx/sigflag.go`
- Test: `internal/tx/sigflag_test.go`

本 Task 实现 SigFlag 验证与签名数据（SigningData）构建逻辑。SigFlag 是 1 字节授权标志，控制签名覆盖交易的哪些部分。签名者根据 SigFlag 构建 SigningData，签名算法对此数据的哈希进行签名。

**SigFlag 位域定义**（已在 `pkg/types` 中）：

| 位 | 常量 | 类别 | 描述 |
|----|------|------|------|
| 7 | SIGIN_ALL | 独项 | 全部输入项 |
| 6 | SIGIN_SELF | 独项 | 仅当前输入项 |
| 5 | SIGOUT_ALL | 主项 | 全部输出项 |
| 4 | SIGOUT_SELF | 主项 | 与当前输入同序位的输出项 |
| 3 | SIGOUTPUT | 辅项 | 完整输出条目（receiver + content + script） |
| 2 | SIGSCRIPT | 辅项 | 输出的锁定脚本 |
| 1 | SIGCONTENT | 辅项 | 输出内容 |
| 0 | SIGRECEIVER | 辅项 | 输出的接收者 |

**组合规则：**
- 独项（Bit 7, 6）：可单独使用或与其他组合
- 主项（Bit 5, 4）：必须与辅项（3-0）组合
- 辅项（Bit 3-0）：必须与主项组合
- SIGOUTPUT 是 SIGRECEIVER|SIGCONTENT|SIGSCRIPT 的超集
- 至少要设置一个位（空 flag 无效）

### Step 1: 写失败测试

创建 `internal/tx/sigflag_test.go`：

```go
package tx

import (
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// --- ValidateSigFlag 测试 ---

func TestValidateSigFlag(t *testing.T) {
	tests := []struct {
		name    string
		flag    types.SigFlag
		wantErr bool
	}{
		{
			name:    "empty_flag",
			flag:    0,
			wantErr: true,
		},
		{
			name:    "only_SIGIN_ALL",
			flag:    types.SIGIN_ALL,
			wantErr: false,
		},
		{
			name:    "only_SIGIN_SELF",
			flag:    types.SIGIN_SELF,
			wantErr: false,
		},
		{
			name:    "SIGIN_ALL_and_SIGIN_SELF",
			flag:    types.SIGIN_ALL | types.SIGIN_SELF,
			wantErr: false,
		},
		{
			name:    "common_config",
			flag:    types.SIGIN_ALL | types.SIGOUT_ALL | types.SIGOUTPUT,
			wantErr: false,
		},
		{
			name:    "primary_without_auxiliary",
			flag:    types.SIGOUT_ALL,
			wantErr: true,
		},
		{
			name:    "auxiliary_without_primary",
			flag:    types.SIGOUTPUT,
			wantErr: true,
		},
		{
			name:    "SIGOUT_SELF_without_auxiliary",
			flag:    types.SIGOUT_SELF,
			wantErr: true,
		},
		{
			name:    "SIGRECEIVER_without_primary",
			flag:    types.SIGRECEIVER,
			wantErr: true,
		},
		{
			name:    "SIGCONTENT_without_primary",
			flag:    types.SIGCONTENT,
			wantErr: true,
		},
		{
			name:    "SIGSCRIPT_without_primary",
			flag:    types.SIGSCRIPT,
			wantErr: true,
		},
		{
			name:    "primary_SIGOUT_ALL_with_SIGRECEIVER",
			flag:    types.SIGOUT_ALL | types.SIGRECEIVER,
			wantErr: false,
		},
		{
			name:    "primary_SIGOUT_SELF_with_SIGCONTENT",
			flag:    types.SIGOUT_SELF | types.SIGCONTENT,
			wantErr: false,
		},
		{
			name:    "independent_plus_primary_plus_auxiliary",
			flag:    types.SIGIN_ALL | types.SIGOUT_ALL | types.SIGRECEIVER,
			wantErr: false,
		},
		{
			name:    "both_primary_with_auxiliary",
			flag:    types.SIGOUT_ALL | types.SIGOUT_SELF | types.SIGOUTPUT,
			wantErr: false,
		},
		{
			name:    "independent_plus_primary_no_auxiliary",
			flag:    types.SIGIN_ALL | types.SIGOUT_ALL,
			wantErr: true,
		},
		{
			name:    "independent_plus_auxiliary_no_primary",
			flag:    types.SIGIN_ALL | types.SIGOUTPUT,
			wantErr: true,
		},
		{
			name:    "all_output_subflags",
			flag:    types.SIGOUT_ALL | types.SIGRECEIVER | types.SIGCONTENT | types.SIGSCRIPT,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSigFlag(tt.flag)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSigFlag(0x%02x) error = %v, wantErr %v", byte(tt.flag), err, tt.wantErr)
			}
		})
	}
}

// --- BuildSigningData 测试 ---

// 辅助函数：构建测试用交易
func newTestTx() *Tx {
	var addr0, addr1, addr2 types.PubKeyHash
	addr0[0] = 0xAA
	addr1[0] = 0xBB
	addr2[0] = 0xCC

	return &Tx{
		Lead: &LeadInput{
			Year:     2026,
			TxID:     types.Hash512{0x01, 0x02, 0x03},
			OutIndex: 0,
		},
		Rest: []*RestInput{
			{Year: 2026, TxIDPart: [TxIDPartLen]byte{0x10}, OutIndex: 1, TransferIndex: -1},
			{Year: 2026, TxIDPart: [TxIDPartLen]byte{0x20}, OutIndex: 2, TransferIndex: 0},
		},
		Outputs: []*Output{
			{
				Serial:     0,
				Config:     types.OutTypeCoin,
				Address:    addr0,
				Amount:     5000,
				Memo:       []byte("memo0"),
				LockScript: []byte{0x01, 0x02},
			},
			{
				Serial:     1,
				Config:     types.OutTypeCredit,
				Address:    addr1,
				CredConfig: CredNew | CredConf(4),
				Creator:    []byte("alice"),
				Title:      []byte("title"),
				Desc:       []byte("desc"),
				LockScript: []byte{0x03, 0x04},
			},
			{
				Serial:      2,
				Config:      types.OutTypeProof,
				ProofConfig: ProofConf(7),
				Creator:     []byte("bob"),
				Title:       []byte("proof"),
				Content:     []byte("content"),
				IdentScript: []byte{0x05, 0x06},
			},
		},
	}
}

// 测试最常见配置：SIGIN_ALL | SIGOUT_ALL | SIGOUTPUT
func TestBuildSigningData_CommonConfig(t *testing.T) {
	tx := newTestTx()
	flag := types.SIGIN_ALL | types.SIGOUT_ALL | types.SIGOUTPUT

	data, err := BuildSigningData(tx, 0, flag)
	if err != nil {
		t.Fatalf("BuildSigningData() error = %v", err)
	}
	if len(data) == 0 {
		t.Fatal("BuildSigningData() returned empty data")
	}

	// 第一个字节应为 SigFlag
	if data[0] != byte(flag) {
		t.Errorf("first byte = 0x%02x, want 0x%02x", data[0], byte(flag))
	}
}

// 测试确定性：相同输入相同输出
func TestBuildSigningData_Deterministic(t *testing.T) {
	tx := newTestTx()
	flag := types.SIGIN_ALL | types.SIGOUT_ALL | types.SIGOUTPUT

	data1, _ := BuildSigningData(tx, 0, flag)
	data2, _ := BuildSigningData(tx, 0, flag)

	if len(data1) != len(data2) {
		t.Fatal("BuildSigningData() not deterministic: different lengths")
	}
	for i := range data1 {
		if data1[i] != data2[i] {
			t.Fatalf("BuildSigningData() not deterministic: differ at byte %d", i)
		}
	}
}

// 测试 SIGIN_SELF（仅当前输入）— Lead 输入
func TestBuildSigningData_SigInSelf_Lead(t *testing.T) {
	tx := newTestTx()
	flag := types.SIGIN_SELF | types.SIGOUT_ALL | types.SIGOUTPUT

	data, err := BuildSigningData(tx, 0, flag)
	if err != nil {
		t.Fatalf("BuildSigningData() error = %v", err)
	}

	// 第一个字节为 SigFlag
	if data[0] != byte(flag) {
		t.Errorf("first byte = 0x%02x, want 0x%02x", data[0], byte(flag))
	}

	// 不同 inputIndex 应产生不同数据
	data1, _ := BuildSigningData(tx, 1, flag)
	if len(data) == len(data1) {
		same := true
		for i := range data {
			if data[i] != data1[i] {
				same = false
				break
			}
		}
		if same {
			t.Error("BuildSigningData() with different inputIndex should produce different data")
		}
	}
}

// 测试 SIGIN_SELF — RestInput
func TestBuildSigningData_SigInSelf_Rest(t *testing.T) {
	tx := newTestTx()
	flag := types.SIGIN_SELF | types.SIGOUT_ALL | types.SIGOUTPUT

	data, err := BuildSigningData(tx, 1, flag)
	if err != nil {
		t.Fatalf("BuildSigningData(inputIndex=1) error = %v", err)
	}
	if len(data) == 0 {
		t.Fatal("BuildSigningData() returned empty data")
	}

	// inputIndex=2 也应可行
	data2, err := BuildSigningData(tx, 2, flag)
	if err != nil {
		t.Fatalf("BuildSigningData(inputIndex=2) error = %v", err)
	}

	// 不同的 rest input 应产生不同数据
	if len(data) == len(data2) {
		same := true
		for i := range data {
			if data[i] != data2[i] {
				same = false
				break
			}
		}
		if same {
			t.Error("different rest inputs should produce different signing data")
		}
	}
}

// 测试 SIGOUT_SELF | SIGCONTENT（部分输出的内容子集）
func TestBuildSigningData_SigOutSelf_Content(t *testing.T) {
	tx := newTestTx()
	flag := types.SIGIN_ALL | types.SIGOUT_SELF | types.SIGCONTENT

	// inputIndex=0, 应覆盖 Outputs[0]
	data0, err := BuildSigningData(tx, 0, flag)
	if err != nil {
		t.Fatalf("BuildSigningData(inputIndex=0) error = %v", err)
	}

	// inputIndex=1, 应覆盖 Outputs[1]
	data1, err := BuildSigningData(tx, 1, flag)
	if err != nil {
		t.Fatalf("BuildSigningData(inputIndex=1) error = %v", err)
	}

	// 两者应不同（不同输出）
	if len(data0) == len(data1) {
		same := true
		for i := range data0 {
			if data0[i] != data1[i] {
				same = false
				break
			}
		}
		if same {
			t.Error("SIGOUT_SELF with different inputIndex should produce different data")
		}
	}
}

// 测试纯独项 SIGIN_ALL（不含输出部分）
func TestBuildSigningData_PureIndependent(t *testing.T) {
	tx := newTestTx()
	flag := types.SIGIN_ALL

	data, err := BuildSigningData(tx, 0, flag)
	if err != nil {
		t.Fatalf("BuildSigningData() error = %v", err)
	}

	// 应包含 SigFlag(1) + Lead.Serialize(70) + Rest[0].Serialize(28) + Rest[1].Serialize(28) = 127 字节
	expected := 1 + LeadInputSize + RestInputSize*2
	if len(data) != expected {
		t.Errorf("len = %d, want %d", len(data), expected)
	}
}

// 测试 SIGIN_ALL | SIGIN_SELF 同时设置
func TestBuildSigningData_BothInputFlags(t *testing.T) {
	tx := newTestTx()
	flag := types.SIGIN_ALL | types.SIGIN_SELF

	data, err := BuildSigningData(tx, 0, flag)
	if err != nil {
		t.Fatalf("BuildSigningData() error = %v", err)
	}

	// 应同时写入全部输入和当前输入
	// SigFlag(1) + All(70+28+28) + Self(70) = 1 + 126 + 70 = 197
	expected := 1 + (LeadInputSize + RestInputSize*2) + LeadInputSize
	if len(data) != expected {
		t.Errorf("len = %d, want %d", len(data), expected)
	}
}

// 测试 SIGOUT_SELF 当 inputIndex 超出输出范围时静默跳过
func TestBuildSigningData_SigOutSelf_NoMatchingOutput(t *testing.T) {
	tx := newTestTx()
	// tx 有 3 个输入（Lead + 2 Rest）和 3 个输出
	// 假设只有 1 个输出
	tx.Outputs = tx.Outputs[:1]

	flag := types.SIGIN_SELF | types.SIGOUT_SELF | types.SIGOUTPUT

	// inputIndex=2 超出输出范围，但不应报错（输出部分为空）
	data, err := BuildSigningData(tx, 2, flag)
	if err != nil {
		t.Fatalf("BuildSigningData() error = %v", err)
	}

	// 应只包含 SigFlag(1) + 对应输入序列化
	if len(data) == 0 {
		t.Fatal("BuildSigningData() returned empty data")
	}
}

// 测试 inputIndex 越界错误
func TestBuildSigningData_InputIndexOutOfBounds(t *testing.T) {
	tx := newTestTx()
	flag := types.SIGIN_ALL | types.SIGOUT_ALL | types.SIGOUTPUT

	// tx 有 3 个输入 (Lead + 2 Rest)，inputIndex=3 越界
	_, err := BuildSigningData(tx, 3, flag)
	if err == nil {
		t.Error("BuildSigningData() should return error for out-of-bounds inputIndex")
	}

	// inputIndex=-1 也越界
	_, err = BuildSigningData(tx, -1, flag)
	if err == nil {
		t.Error("BuildSigningData() should return error for negative inputIndex")
	}
}

// 测试无效 SigFlag 被拒绝
func TestBuildSigningData_InvalidFlag(t *testing.T) {
	tx := newTestTx()

	_, err := BuildSigningData(tx, 0, 0)
	if err == nil {
		t.Error("BuildSigningData() should return error for zero flag")
	}

	_, err = BuildSigningData(tx, 0, types.SIGOUT_ALL) // 主项无辅项
	if err == nil {
		t.Error("BuildSigningData() should return error for primary without auxiliary")
	}
}

// 测试 SIGSCRIPT 对 Proof 输出使用 IdentScript
func TestBuildSigningData_SigScript_Proof(t *testing.T) {
	tx := newTestTx()
	// Outputs[2] 是 Proof 类型，有 IdentScript
	flag := types.SIGIN_ALL | types.SIGOUT_ALL | types.SIGSCRIPT

	data, err := BuildSigningData(tx, 0, flag)
	if err != nil {
		t.Fatalf("BuildSigningData() error = %v", err)
	}
	if len(data) == 0 {
		t.Fatal("BuildSigningData() returned empty data")
	}
}

// 测试 SIGRECEIVER 写入地址
func TestBuildSigningData_SigReceiver(t *testing.T) {
	// 构建只有一个 Coin 输出的交易
	var addr types.PubKeyHash
	addr[0] = 0xFF
	addr[47] = 0xEE

	tx := &Tx{
		Lead: &LeadInput{
			Year:     2026,
			TxID:     types.Hash512{0x01},
			OutIndex: 0,
		},
		Outputs: []*Output{
			{
				Serial:     0,
				Config:     types.OutTypeCoin,
				Address:    addr,
				Amount:     1000,
				LockScript: []byte{0x01},
			},
		},
	}

	flag := types.SIGIN_ALL | types.SIGOUT_ALL | types.SIGRECEIVER

	data, err := BuildSigningData(tx, 0, flag)
	if err != nil {
		t.Fatalf("BuildSigningData() error = %v", err)
	}

	// 数据应包含：SigFlag(1) + Lead(70) + Address(48) = 119
	expected := 1 + LeadInputSize + types.PubKeyHashLen
	if len(data) != expected {
		t.Errorf("len = %d, want %d", len(data), expected)
	}

	// 验证地址出现在末尾
	addrStart := len(data) - types.PubKeyHashLen
	if data[addrStart] != 0xFF {
		t.Errorf("address first byte = 0x%02x, want 0xFF", data[addrStart])
	}
	if data[len(data)-1] != 0xEE {
		t.Errorf("address last byte = 0x%02x, want 0xEE", data[len(data)-1])
	}
}
```

### Step 2: 运行测试验证失败

```bash
go test -v ./internal/tx/ -run "TestValidateSigFlag|TestBuildSigningData"
```

预期输出：编译失败，`ValidateSigFlag`、`BuildSigningData` 未定义。

### Step 3: 写最小实现

创建 `internal/tx/sigflag.go`：

```go
package tx

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/cxio/evidcoin/pkg/types"
)

// SigFlag 位域掩码
const (
	sigInputMask   = types.SIGIN_ALL | types.SIGIN_SELF                                      // 独项掩码（Bit 7, 6）
	sigPrimaryMask = types.SIGOUT_ALL | types.SIGOUT_SELF                                    // 主项掩码（Bit 5, 4）
	sigAuxMask     = types.SIGOUTPUT | types.SIGSCRIPT | types.SIGCONTENT | types.SIGRECEIVER // 辅项掩码（Bit 3-0）
)

// SigFlag 验证错误
var (
	ErrSigFlagEmpty        = errors.New("sig flag is empty")
	ErrSigFlagPrimaryNoAux = errors.New("sig flag has primary bits without auxiliary bits")
	ErrSigFlagAuxNoPrimary = errors.New("sig flag has auxiliary bits without primary bits")
)

// ValidateSigFlag 验证 SigFlag 的合法性。
// 规则：
//   - 至少设置一个位
//   - 主项（SIGOUT_ALL/SIGOUT_SELF）必须与辅项（SIGOUTPUT/SIGSCRIPT/SIGCONTENT/SIGRECEIVER）组合
//   - 辅项必须与主项组合
//   - 独项（SIGIN_ALL/SIGIN_SELF）可单独使用
func ValidateSigFlag(flag types.SigFlag) error {
	if flag == 0 {
		return ErrSigFlagEmpty
	}

	hasPrimary := flag&sigPrimaryMask != 0
	hasAux := flag&sigAuxMask != 0

	// 主项必须与辅项组合
	if hasPrimary && !hasAux {
		return ErrSigFlagPrimaryNoAux
	}

	// 辅项必须与主项组合
	if hasAux && !hasPrimary {
		return ErrSigFlagAuxNoPrimary
	}

	return nil
}

// BuildSigningData 根据 SigFlag 构建签名数据。
//
// 参数：
//
//	tx: 完整交易
//	inputIndex: 当前输入的序号（0 = Lead, 1+ = Rest[inputIndex-1]）
//	flag: SigFlag 授权标志
//
// 返回：签名数据的字节切片（之后对此数据哈希再签名）
func BuildSigningData(tx *Tx, inputIndex int, flag types.SigFlag) ([]byte, error) {
	// 验证 SigFlag
	if err := ValidateSigFlag(flag); err != nil {
		return nil, fmt.Errorf("invalid sig flag: %w", err)
	}

	// 验证 inputIndex 范围
	totalInputs := 1 + len(tx.Rest) // Lead + Rest
	if inputIndex < 0 || inputIndex >= totalInputs {
		return nil, fmt.Errorf("input index %d out of range [0, %d)", inputIndex, totalInputs)
	}

	var buf bytes.Buffer

	// 写入 SigFlag 本身（1 字节）
	buf.WriteByte(byte(flag))

	// === 输入部分（独项） ===

	// SIGIN_ALL：拼接所有输入的序列化
	if flag&types.SIGIN_ALL != 0 {
		buf.Write(tx.Lead.Serialize())
		for _, ri := range tx.Rest {
			buf.Write(ri.Serialize())
		}
	}

	// SIGIN_SELF：拼接 inputIndex 对应的输入序列化
	if flag&types.SIGIN_SELF != 0 {
		buf.Write(serializeInputAt(tx, inputIndex))
	}

	// === 输出部分（主项 + 辅项组合） ===

	hasPrimary := flag&sigPrimaryMask != 0
	if hasPrimary {
		// 确定输出范围
		outputIndices := collectOutputIndices(tx, inputIndex, flag)

		// 对范围内每个输出，按辅项决定写入内容
		for _, idx := range outputIndices {
			writeOutputByAux(&buf, tx.Outputs[idx], flag)
		}
	}

	return buf.Bytes(), nil
}

// serializeInputAt 序列化指定索引的输入。
// inputIndex=0 对应 Lead，1+ 对应 Rest[inputIndex-1]。
func serializeInputAt(tx *Tx, inputIndex int) []byte {
	if inputIndex == 0 {
		return tx.Lead.Serialize()
	}
	return tx.Rest[inputIndex-1].Serialize()
}

// collectOutputIndices 根据主项 flag 收集需要覆盖的输出索引列表。
// SIGOUT_ALL 选取所有输出，SIGOUT_SELF 选取与 inputIndex 同序位的输出（若存在）。
// 两者可同时出现，但 ALL 已包含 SELF，结果去重。
func collectOutputIndices(tx *Tx, inputIndex int, flag types.SigFlag) []int {
	hasAll := flag&types.SIGOUT_ALL != 0
	hasSelf := flag&types.SIGOUT_SELF != 0

	if hasAll {
		// ALL 已包含全部，无需额外处理
		indices := make([]int, len(tx.Outputs))
		for i := range tx.Outputs {
			indices[i] = i
		}
		return indices
	}

	if hasSelf && inputIndex < len(tx.Outputs) {
		return []int{inputIndex}
	}

	// SIGOUT_SELF 但 inputIndex 超出输出范围，返回空
	return nil
}

// writeOutputByAux 根据辅项 flag 将输出数据写入 buffer。
func writeOutputByAux(buf *bytes.Buffer, out *Output, flag types.SigFlag) {
	// SIGOUTPUT 是 SIGRECEIVER|SIGCONTENT|SIGSCRIPT 的超集
	if flag&types.SIGOUTPUT != 0 {
		buf.Write(out.Serialize())
		return
	}

	// 按子集分别写入
	if flag&types.SIGRECEIVER != 0 {
		buf.Write(out.Address[:])
	}
	if flag&types.SIGCONTENT != 0 {
		buf.Write(out.SerializeContent())
	}
	if flag&types.SIGSCRIPT != 0 {
		// Proof 使用 IdentScript，其他使用 LockScript
		if out.Config.Type() == types.OutTypeProof {
			buf.Write(out.IdentScript)
		} else {
			buf.Write(out.LockScript)
		}
	}
}
```

### Step 4: 运行测试验证通过

```bash
go test -v ./internal/tx/ -run "TestValidateSigFlag|TestBuildSigningData"
```

预期输出：全部 PASS。

### Step 5: 提交

```bash
git add internal/tx/sigflag.go internal/tx/sigflag_test.go
git commit -m "feat(tx): add SigFlag validation and SigningData builder"
```

---

## Task 8: 费用计算与交易优先级

**Files:**
- Create: `internal/tx/fee.go`
- Test: `internal/tx/fee_test.go`

本 Task 实现交易手续费的隐式计算、50/50 分配（销毁 + 分配）、币龄销毁计算、凭信提前销毁检测、以及交易优先级比较。

**费用模型：**
- Fee = Sum(input coin values) - Sum(output coin values)
- 仅 Coin 类型输出的 Amount 参与计算
- 50% 永久销毁，50% 分配给铸造团队

**优先级排序（降序）：**
1. HasPrematureDestroy = true 的交易最优先
2. BurnedCoinAge 越高越优先
3. Fee 越高越优先

### Step 1: 写失败测试

创建 `internal/tx/fee_test.go`：

```go
package tx

import (
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// --- CalculateFee 测试 ---

// 正常费用计算
func TestCalculateFee_Normal(t *testing.T) {
	var addr types.PubKeyHash
	addr[0] = 0x01

	tx := &Tx{
		Outputs: []*Output{
			{
				Serial:     0,
				Config:     types.OutTypeCoin,
				Address:    addr,
				Amount:     3000,
				LockScript: []byte{0x01},
			},
			{
				Serial:     1,
				Config:     types.OutTypeCoin,
				Address:    addr,
				Amount:     2000,
				LockScript: []byte{0x01},
			},
		},
	}

	// 输入总额 10000，输出总额 5000，费用 5000
	inputValues := []int64{6000, 4000}

	fee, burned, distributed, err := CalculateFee(inputValues, tx)
	if err != nil {
		t.Fatalf("CalculateFee() error = %v", err)
	}

	if fee != 5000 {
		t.Errorf("fee = %d, want 5000", fee)
	}
	if burned != 2500 {
		t.Errorf("burned = %d, want 2500", burned)
	}
	if distributed != 2500 {
		t.Errorf("distributed = %d, want 2500", distributed)
	}
	if burned+distributed != fee {
		t.Errorf("burned(%d) + distributed(%d) != fee(%d)", burned, distributed, fee)
	}
}

// 零费用
func TestCalculateFee_ZeroFee(t *testing.T) {
	var addr types.PubKeyHash
	addr[0] = 0x01

	tx := &Tx{
		Outputs: []*Output{
			{
				Serial:     0,
				Config:     types.OutTypeCoin,
				Address:    addr,
				Amount:     10000,
				LockScript: []byte{0x01},
			},
		},
	}

	inputValues := []int64{10000}

	fee, burned, distributed, err := CalculateFee(inputValues, tx)
	if err != nil {
		t.Fatalf("CalculateFee() error = %v", err)
	}

	if fee != 0 {
		t.Errorf("fee = %d, want 0", fee)
	}
	if burned != 0 {
		t.Errorf("burned = %d, want 0", burned)
	}
	if distributed != 0 {
		t.Errorf("distributed = %d, want 0", distributed)
	}
}

// 负费用（无效交易）
func TestCalculateFee_NegativeFee(t *testing.T) {
	var addr types.PubKeyHash
	addr[0] = 0x01

	tx := &Tx{
		Outputs: []*Output{
			{
				Serial:     0,
				Config:     types.OutTypeCoin,
				Address:    addr,
				Amount:     10000,
				LockScript: []byte{0x01},
			},
		},
	}

	// 输入 5000 < 输出 10000
	inputValues := []int64{5000}

	_, _, _, err := CalculateFee(inputValues, tx)
	if err == nil {
		t.Error("CalculateFee() should return error for negative fee")
	}
}

// 混合输出类型（仅 Coin 参与费用计算）
func TestCalculateFee_MixedOutputTypes(t *testing.T) {
	var addr types.PubKeyHash
	addr[0] = 0x01

	tx := &Tx{
		Outputs: []*Output{
			{
				Serial:     0,
				Config:     types.OutTypeCoin,
				Address:    addr,
				Amount:     3000,
				LockScript: []byte{0x01},
			},
			{
				Serial:     1,
				Config:     types.OutTypeCredit,
				Address:    addr,
				CredConfig: CredNew,
				Creator:    []byte("test"),
				Title:      []byte("title"),
				LockScript: []byte{0x01},
				// Credit 的 Amount 不参与计算
			},
			{
				Serial:      2,
				Config:      types.OutTypeProof,
				ProofConfig: ProofConf(4),
				Creator:     []byte("bob"),
				Title:       []byte("proof"),
				Content:     []byte("data"),
				IdentScript: []byte{0x01},
			},
		},
	}

	// 输入 10000，Coin 输出 3000，费用 7000
	inputValues := []int64{10000}

	fee, burned, distributed, err := CalculateFee(inputValues, tx)
	if err != nil {
		t.Fatalf("CalculateFee() error = %v", err)
	}

	if fee != 7000 {
		t.Errorf("fee = %d, want 7000", fee)
	}
	if burned+distributed != fee {
		t.Errorf("burned(%d) + distributed(%d) != fee(%d)", burned, distributed, fee)
	}
}

// 奇数费用的 50/50 分配
func TestCalculateFee_OddFee(t *testing.T) {
	var addr types.PubKeyHash
	addr[0] = 0x01

	tx := &Tx{
		Outputs: []*Output{
			{
				Serial:     0,
				Config:     types.OutTypeCoin,
				Address:    addr,
				Amount:     4999,
				LockScript: []byte{0x01},
			},
		},
	}

	// 费用 = 10000 - 4999 = 5001（奇数）
	inputValues := []int64{10000}

	fee, burned, distributed, err := CalculateFee(inputValues, tx)
	if err != nil {
		t.Fatalf("CalculateFee() error = %v", err)
	}

	if fee != 5001 {
		t.Errorf("fee = %d, want 5001", fee)
	}
	// burned = 5001 / 2 = 2500（整数除法）
	// distributed = 5001 - 2500 = 2501
	if burned != 2500 {
		t.Errorf("burned = %d, want 2500", burned)
	}
	if distributed != 2501 {
		t.Errorf("distributed = %d, want 2501", distributed)
	}
	if burned+distributed != fee {
		t.Errorf("burned(%d) + distributed(%d) != fee(%d)", burned, distributed, fee)
	}
}

// 偶数费用的 50/50 分配
func TestCalculateFee_EvenFee(t *testing.T) {
	var addr types.PubKeyHash
	addr[0] = 0x01

	tx := &Tx{
		Outputs: []*Output{
			{
				Serial:     0,
				Config:     types.OutTypeCoin,
				Address:    addr,
				Amount:     6000,
				LockScript: []byte{0x01},
			},
		},
	}

	// 费用 = 10000 - 6000 = 4000
	inputValues := []int64{10000}

	fee, burned, distributed, err := CalculateFee(inputValues, tx)
	if err != nil {
		t.Fatalf("CalculateFee() error = %v", err)
	}

	if fee != 4000 {
		t.Errorf("fee = %d, want 4000", fee)
	}
	if burned != 2000 {
		t.Errorf("burned = %d, want 2000", burned)
	}
	if distributed != 2000 {
		t.Errorf("distributed = %d, want 2000", distributed)
	}
}

// 输入值列表为空
func TestCalculateFee_EmptyInputValues(t *testing.T) {
	var addr types.PubKeyHash
	addr[0] = 0x01

	tx := &Tx{
		Outputs: []*Output{
			{
				Serial:     0,
				Config:     types.OutTypeCoin,
				Address:    addr,
				Amount:     1000,
				LockScript: []byte{0x01},
			},
		},
	}

	_, _, _, err := CalculateFee(nil, tx)
	if err == nil {
		t.Error("CalculateFee() should return error for empty input values")
	}
}

// --- CalculateBurnedCoinAge 测试 ---

func TestCalculateBurnedCoinAge_Normal(t *testing.T) {
	amounts := []int64{1000, 2000, 500}
	ages := []int64{10, 20, 5}

	// 1000*10 + 2000*20 + 500*5 = 10000 + 40000 + 2500 = 52500
	result := CalculateBurnedCoinAge(amounts, ages)
	if result != 52500 {
		t.Errorf("CalculateBurnedCoinAge() = %d, want 52500", result)
	}
}

func TestCalculateBurnedCoinAge_Empty(t *testing.T) {
	result := CalculateBurnedCoinAge(nil, nil)
	if result != 0 {
		t.Errorf("CalculateBurnedCoinAge(nil, nil) = %d, want 0", result)
	}
}

func TestCalculateBurnedCoinAge_SingleItem(t *testing.T) {
	result := CalculateBurnedCoinAge([]int64{5000}, []int64{100})
	if result != 500000 {
		t.Errorf("CalculateBurnedCoinAge() = %d, want 500000", result)
	}
}

func TestCalculateBurnedCoinAge_ZeroAge(t *testing.T) {
	result := CalculateBurnedCoinAge([]int64{1000}, []int64{0})
	if result != 0 {
		t.Errorf("CalculateBurnedCoinAge() = %d, want 0", result)
	}
}

// --- CheckPrematureDestroy 测试 ---

func TestCheckPrematureDestroy_WithDestroy(t *testing.T) {
	var addr types.PubKeyHash
	addr[0] = 0x01

	outputs := []*Output{
		{
			Serial:     0,
			Config:     types.OutTypeCoin,
			Address:    addr,
			Amount:     1000,
			LockScript: []byte{0x01},
		},
		{
			Serial:     1,
			Config:     types.OutTypeCredit | types.OutDestroy, // 提前销毁凭信
			Address:    addr,
			CredConfig: CredNew,
			Creator:    []byte("test"),
			Title:      []byte("title"),
			LockScript: []byte{0x01},
		},
	}

	if !CheckPrematureDestroy(outputs) {
		t.Error("CheckPrematureDestroy() should return true when credit with Destroy exists")
	}
}

func TestCheckPrematureDestroy_WithoutDestroy(t *testing.T) {
	var addr types.PubKeyHash
	addr[0] = 0x01

	outputs := []*Output{
		{
			Serial:     0,
			Config:     types.OutTypeCoin,
			Address:    addr,
			Amount:     1000,
			LockScript: []byte{0x01},
		},
		{
			Serial:     1,
			Config:     types.OutTypeCredit, // 无销毁标记
			Address:    addr,
			CredConfig: CredNew,
			Creator:    []byte("test"),
			Title:      []byte("title"),
			LockScript: []byte{0x01},
		},
	}

	if CheckPrematureDestroy(outputs) {
		t.Error("CheckPrematureDestroy() should return false when no credit with Destroy")
	}
}

func TestCheckPrematureDestroy_CoinDestroy(t *testing.T) {
	var addr types.PubKeyHash
	addr[0] = 0x01

	// 仅 Coin 被销毁，不算凭信提前销毁
	outputs := []*Output{
		{
			Serial:     0,
			Config:     types.OutTypeCoin | types.OutDestroy,
			Address:    addr,
			Amount:     1000,
			LockScript: []byte{0x01},
		},
	}

	if CheckPrematureDestroy(outputs) {
		t.Error("CheckPrematureDestroy() should return false for Coin destroy (not Credit)")
	}
}

func TestCheckPrematureDestroy_Empty(t *testing.T) {
	if CheckPrematureDestroy(nil) {
		t.Error("CheckPrematureDestroy(nil) should return false")
	}
}

// --- ComparePriority 测试 ---

func TestComparePriority_PrematureDestroyWins(t *testing.T) {
	a := &TxPriority{HasPrematureDestroy: true, BurnedCoinAge: 0, Fee: 0}
	b := &TxPriority{HasPrematureDestroy: false, BurnedCoinAge: 999999, Fee: 999999}

	if ComparePriority(a, b) <= 0 {
		t.Error("premature destroy should win regardless of other fields")
	}
	if ComparePriority(b, a) >= 0 {
		t.Error("non-destroy should lose to destroy")
	}
}

func TestComparePriority_BothPrematureDestroy(t *testing.T) {
	a := &TxPriority{HasPrematureDestroy: true, BurnedCoinAge: 100, Fee: 50}
	b := &TxPriority{HasPrematureDestroy: true, BurnedCoinAge: 200, Fee: 10}

	// 都有提前销毁，比较 BurnedCoinAge
	if ComparePriority(a, b) >= 0 {
		t.Error("higher BurnedCoinAge should win when both have premature destroy")
	}
	if ComparePriority(b, a) <= 0 {
		t.Error("lower BurnedCoinAge should lose when both have premature destroy")
	}
}

func TestComparePriority_CoinAgeWins(t *testing.T) {
	a := &TxPriority{HasPrematureDestroy: false, BurnedCoinAge: 1000, Fee: 10}
	b := &TxPriority{HasPrematureDestroy: false, BurnedCoinAge: 500, Fee: 99999}

	if ComparePriority(a, b) <= 0 {
		t.Error("higher BurnedCoinAge should win over higher Fee")
	}
}

func TestComparePriority_FeeBreaksTie(t *testing.T) {
	a := &TxPriority{HasPrematureDestroy: false, BurnedCoinAge: 100, Fee: 500}
	b := &TxPriority{HasPrematureDestroy: false, BurnedCoinAge: 100, Fee: 200}

	if ComparePriority(a, b) <= 0 {
		t.Error("higher Fee should win when BurnedCoinAge is equal")
	}
	if ComparePriority(b, a) >= 0 {
		t.Error("lower Fee should lose when BurnedCoinAge is equal")
	}
}

func TestComparePriority_Equal(t *testing.T) {
	a := &TxPriority{HasPrematureDestroy: false, BurnedCoinAge: 100, Fee: 50}
	b := &TxPriority{HasPrematureDestroy: false, BurnedCoinAge: 100, Fee: 50}

	if ComparePriority(a, b) != 0 {
		t.Error("identical priorities should return 0")
	}
}

func TestComparePriority_BothZero(t *testing.T) {
	a := &TxPriority{}
	b := &TxPriority{}

	if ComparePriority(a, b) != 0 {
		t.Error("both zero priorities should return 0")
	}
}

func TestComparePriority_AllLevels(t *testing.T) {
	// 先按 PrematureDestroy，再按 BurnedCoinAge，最后按 Fee
	priorities := []*TxPriority{
		{HasPrematureDestroy: true, BurnedCoinAge: 200, Fee: 100},   // 最高
		{HasPrematureDestroy: true, BurnedCoinAge: 100, Fee: 100},   // 次之
		{HasPrematureDestroy: false, BurnedCoinAge: 9999, Fee: 100}, // 第三
		{HasPrematureDestroy: false, BurnedCoinAge: 9999, Fee: 50},  // 第四
		{HasPrematureDestroy: false, BurnedCoinAge: 0, Fee: 0},      // 最低
	}

	for i := 0; i < len(priorities)-1; i++ {
		result := ComparePriority(priorities[i], priorities[i+1])
		if result <= 0 {
			t.Errorf("priority[%d] should be > priority[%d], got %d", i, i+1, result)
		}
	}
}
```

### Step 2: 运行测试验证失败

```bash
go test -v ./internal/tx/ -run "TestCalculateFee|TestCalculateBurnedCoinAge|TestCheckPrematureDestroy|TestComparePriority"
```

预期输出：编译失败，`CalculateFee`、`TxPriority`、`ComparePriority`、`CalculateBurnedCoinAge`、`CheckPrematureDestroy` 未定义。

### Step 3: 写最小实现

创建 `internal/tx/fee.go`：

```go
package tx

import (
	"errors"

	"github.com/cxio/evidcoin/pkg/types"
)

// 费用计算错误
var (
	ErrNegativeFee   = errors.New("transaction fee is negative")
	ErrNoInputValues = errors.New("input values list is empty")
)

// CalculateFee 计算交易费用。
// inputValues: 每个输入对应的币金金额（从 UTXO 查询获得）
// tx: 交易
// 返回：总费用、销毁部分（50%）、分配部分（50%），以及可能的错误。
//
// 计算规则：
//   - Fee = Sum(inputValues) - Sum(Coin 类型输出的 Amount)
//   - 仅 Config.Type() == OutTypeCoin 的输出参与金额计算
//   - burned = fee / 2（整数除法，向下取整）
//   - distributed = fee - burned（确保无整数损失）
func CalculateFee(inputValues []int64, tx *Tx) (fee int64, burned int64, distributed int64, err error) {
	if len(inputValues) == 0 {
		return 0, 0, 0, ErrNoInputValues
	}

	// 计算输入总额
	var sumInputs int64
	for _, v := range inputValues {
		sumInputs += v
	}

	// 计算输出总额（仅 Coin 类型）
	var sumOutputs int64
	for _, out := range tx.Outputs {
		if out.Config.Type() == types.OutTypeCoin {
			sumOutputs += out.Amount
		}
	}

	// 计算费用
	fee = sumInputs - sumOutputs
	if fee < 0 {
		return 0, 0, 0, ErrNegativeFee
	}

	// 50/50 分配
	burned = fee / 2
	distributed = fee - burned

	return fee, burned, distributed, nil
}

// TxPriority 交易优先级信息。
type TxPriority struct {
	HasPrematureDestroy bool  // 是否包含提前销毁凭信
	BurnedCoinAge       int64 // 销毁的币龄总和
	Fee                 int64 // 交易手续费
}

// ComparePriority 比较两个交易优先级。
// 返回值：正数表示 a 优先于 b，负数表示 b 优先于 a，0 表示相等。
//
// 优先级排序规则（降序）：
//  1. HasPrematureDestroy = true 的交易优先
//  2. BurnedCoinAge 越高越优先
//  3. Fee 越高越优先
func ComparePriority(a, b *TxPriority) int {
	// 第一级：提前销毁凭信
	if a.HasPrematureDestroy != b.HasPrematureDestroy {
		if a.HasPrematureDestroy {
			return 1
		}
		return -1
	}

	// 第二级：币龄销毁
	if a.BurnedCoinAge != b.BurnedCoinAge {
		if a.BurnedCoinAge > b.BurnedCoinAge {
			return 1
		}
		return -1
	}

	// 第三级：交易费用
	if a.Fee != b.Fee {
		if a.Fee > b.Fee {
			return 1
		}
		return -1
	}

	return 0
}

// CalculateBurnedCoinAge 计算交易销毁的总币龄。
// destroyedAmounts: 被销毁输出的金额列表
// destroyedAges: 对应的币龄列表（当前高度 - 创建高度）
// 返回：sum(amount[i] * age[i])
func CalculateBurnedCoinAge(destroyedAmounts []int64, destroyedAges []int64) int64 {
	var total int64
	n := len(destroyedAmounts)
	if n > len(destroyedAges) {
		n = len(destroyedAges)
	}
	for i := 0; i < n; i++ {
		total += destroyedAmounts[i] * destroyedAges[i]
	}
	return total
}

// CheckPrematureDestroy 检查交易输出中是否存在提前销毁的凭信。
// 判断条件：输出的 Config.Type() == OutTypeCredit 且 Config.IsDestroy() == true。
func CheckPrematureDestroy(outputs []*Output) bool {
	for _, out := range outputs {
		if out.Config.Type() == types.OutTypeCredit && out.Config.IsDestroy() {
			return true
		}
	}
	return false
}
```

> **注意：** `Config.IsDestroy()` 方法假设已在 `pkg/types/output_config.go` 中定义（类似于已有的 `Config.Type()` 和 `Config.HasAttachment()`）。若 Phase 1 中该方法尚未实现，需要在 `pkg/types` 中添加：
>
> ```go
> // IsDestroy 检查是否设置了销毁标记。
> func (c OutputConfig) IsDestroy() bool { return c&OutDestroy != 0 }
> ```

### Step 4: 运行测试验证通过

```bash
go test -v ./internal/tx/ -run "TestCalculateFee|TestCalculateBurnedCoinAge|TestCheckPrematureDestroy|TestComparePriority"
```

预期输出：全部 PASS。

### Step 5: 提交

```bash
git add internal/tx/fee.go internal/tx/fee_test.go
git commit -m "feat(tx): add fee calculation, burned coin-age, premature destroy check, and tx priority comparison"
```

---

## 全量测试验证（Task 7-8）

完成 Task 7-8 后，运行完整测试套件：

```bash
go test -v -count=1 ./internal/tx/...
```

预期：全部 PASS（包含 Task 1-8 的所有测试）。

运行覆盖率检查：

```bash
go test -cover ./internal/tx/...
```

预期：覆盖率 ≥ 80%。

运行格式化与编译检查：

```bash
go fmt ./internal/tx/...
go build ./...
```

预期：无变更，无错误。

---

## 验收标准（Task 7-8）

1. **文件结构**：以下文件全部存在
   - `internal/tx/sigflag.go` — ValidateSigFlag、BuildSigningData 及辅助函数
   - `internal/tx/sigflag_test.go` — 签名数据构造完整测试
   - `internal/tx/fee.go` — CalculateFee、TxPriority、ComparePriority、CalculateBurnedCoinAge、CheckPrematureDestroy
   - `internal/tx/fee_test.go` — 费用计算与优先级完整测试

2. **编译通过**：`go build ./...` 无错误

3. **全量测试通过**：`go test ./internal/tx/...` 全部 PASS

4. **测试覆盖率**：核心逻辑 ≥ 80%

5. **功能完整性**：
   - ValidateSigFlag 正确检查独项/主项/辅项组合规则
   - BuildSigningData 按 SigFlag 位域构建签名数据，首字节为 flag 本身
   - SIGIN_ALL 写入所有输入序列化，SIGIN_SELF 写入指定输入
   - SIGOUT_ALL 覆盖所有输出，SIGOUT_SELF 覆盖同序位输出
   - SIGOUTPUT 写完整 Serialize()，否则按 SIGRECEIVER/SIGCONTENT/SIGSCRIPT 子集
   - SIGSCRIPT 对 Proof 输出使用 IdentScript
   - CalculateFee 仅累加 Coin 类型输出金额，负费用报错
   - 50/50 分配中 burned = fee/2, distributed = fee - burned（无整数损失）
   - ComparePriority 严格按三级优先级排序
   - CalculateBurnedCoinAge 正确计算 sum(amount * age)
   - CheckPrematureDestroy 仅检测 Credit + Destroy 组合

6. **设计约束**：
   - 仅依赖 `pkg/types` 和 `pkg/crypto`
   - 错误信息英文小写、无标点
   - 注释使用中文
   - 费用计算不含 UTXO 查询逻辑，输入金额由外部提供
