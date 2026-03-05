# Phase 1：基础类型与密码学原语 实现计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现所有模块依赖的基础类型定义（Hash512, Hash384, PubKeyHash, OutputConfig, SigFlag, varint, 地址编码）和密码学原语封装（SHA-512, SHA3-384/512, BLAKE3, ML-DSA-65, 多重签名）。

**Architecture:** 两个公共包 `pkg/types` 和 `pkg/crypto`，零内部依赖。`types` 提供全项目共享的类型和常量；`crypto` 封装所有密码学操作并通过接口抽象签名方案。

**Tech Stack:** Go 1.25+, crypto/sha512 (stdlib), golang.org/x/crypto/sha3, lukechampine.com/blake3, github.com/cloudflare/circl (ML-DSA-65)

---

## 前置准备

在开始 Task 1 之前，确保项目依赖已就绪：

```bash
# 添加外部依赖
go get golang.org/x/crypto
go get lukechampine.com/blake3
go get github.com/cloudflare/circl
go get github.com/mr-tron/base58
go mod tidy
```

---

## pkg/types 部分

### Task 1: 基础哈希类型与常量 (pkg/types/hash.go, pkg/types/constants.go)

**Files:**
- Create: `pkg/types/constants.go`
- Create: `pkg/types/hash.go`
- Test: `pkg/types/hash_test.go`

**Step 1: 编写失败测试**

```go
// pkg/types/hash_test.go
package types

import (
	"encoding/hex"
	"testing"
)

func TestHash512_IsZero(t *testing.T) {
	tests := []struct {
		name string
		h    Hash512
		want bool
	}{
		{"zero value", Hash512{}, true},
		{"non-zero", func() Hash512 {
			var h Hash512
			h[0] = 1
			return h
		}(), false},
		{"last byte set", func() Hash512 {
			var h Hash512
			h[HashLen-1] = 0xff
			return h
		}(), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.h.IsZero(); got != tt.want {
				t.Errorf("Hash512.IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHash512_String(t *testing.T) {
	var h Hash512
	h[0] = 0xab
	h[1] = 0xcd
	// String() 应返回全 128 字符的十六进制字符串
	s := h.String()
	if len(s) != HashLen*2 {
		t.Errorf("Hash512.String() length = %d, want %d", len(s), HashLen*2)
	}
	if s[:4] != "abcd" {
		t.Errorf("Hash512.String() prefix = %q, want %q", s[:4], "abcd")
	}
}

func TestHash512_Equal(t *testing.T) {
	var a, b Hash512
	a[0] = 0x01
	b[0] = 0x01
	if !a.Equal(b) {
		t.Error("Hash512.Equal() should return true for identical hashes")
	}
	b[0] = 0x02
	if a.Equal(b) {
		t.Error("Hash512.Equal() should return false for different hashes")
	}
}

func TestHash512_Bytes(t *testing.T) {
	var h Hash512
	h[0] = 0xff
	b := h.Bytes()
	if len(b) != HashLen {
		t.Errorf("Hash512.Bytes() length = %d, want %d", len(b), HashLen)
	}
	if b[0] != 0xff {
		t.Errorf("Hash512.Bytes()[0] = %x, want %x", b[0], 0xff)
	}
	// 修改返回值不应影响原值
	b[0] = 0x00
	if h[0] != 0xff {
		t.Error("Hash512.Bytes() should return a copy")
	}
}

func TestHash384_IsZero(t *testing.T) {
	tests := []struct {
		name string
		h    Hash384
		want bool
	}{
		{"zero value", Hash384{}, true},
		{"non-zero", func() Hash384 {
			var h Hash384
			h[0] = 1
			return h
		}(), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.h.IsZero(); got != tt.want {
				t.Errorf("Hash384.IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHash384_String(t *testing.T) {
	var h Hash384
	h[0] = 0xef
	s := h.String()
	if len(s) != Hash384Len*2 {
		t.Errorf("Hash384.String() length = %d, want %d", len(s), Hash384Len*2)
	}
	if s[:2] != "ef" {
		t.Errorf("Hash384.String() prefix = %q, want %q", s[:2], "ef")
	}
}

func TestHash384_Equal(t *testing.T) {
	var a, b Hash384
	a[10] = 0xaa
	b[10] = 0xaa
	if !a.Equal(b) {
		t.Error("Hash384.Equal() should return true for identical hashes")
	}
	b[10] = 0xbb
	if a.Equal(b) {
		t.Error("Hash384.Equal() should return false for different hashes")
	}
}

func TestHash384_Bytes(t *testing.T) {
	var h Hash384
	h[47] = 0x42
	b := h.Bytes()
	if len(b) != Hash384Len {
		t.Errorf("Hash384.Bytes() length = %d, want %d", len(b), Hash384Len)
	}
	if b[47] != 0x42 {
		t.Errorf("Hash384.Bytes()[47] = %x, want %x", b[47], 0x42)
	}
}

func TestPubKeyHash_IsZero(t *testing.T) {
	var pkh PubKeyHash
	if !pkh.IsZero() {
		t.Error("zero PubKeyHash.IsZero() should return true")
	}
	pkh[0] = 1
	if pkh.IsZero() {
		t.Error("non-zero PubKeyHash.IsZero() should return false")
	}
}

func TestPubKeyHash_String(t *testing.T) {
	var pkh PubKeyHash
	pkh[0] = 0xde
	pkh[1] = 0xad
	s := pkh.String()
	if len(s) != PubKeyHashLen*2 {
		t.Errorf("PubKeyHash.String() length = %d, want %d", len(s), PubKeyHashLen*2)
	}
	if s[:4] != "dead" {
		t.Errorf("PubKeyHash.String() prefix = %q, want %q", s[:4], "dead")
	}
}

func TestPubKeyHash_Equal(t *testing.T) {
	var a, b PubKeyHash
	a[0] = 0x01
	b[0] = 0x01
	if !a.Equal(b) {
		t.Error("PubKeyHash.Equal() should return true for identical hashes")
	}
	b[0] = 0x02
	if a.Equal(b) {
		t.Error("PubKeyHash.Equal() should return false for different hashes")
	}
}

func TestPubKeyHash_Bytes(t *testing.T) {
	var pkh PubKeyHash
	pkh[0] = 0xcc
	b := pkh.Bytes()
	if len(b) != PubKeyHashLen {
		t.Errorf("PubKeyHash.Bytes() length = %d, want %d", len(b), PubKeyHashLen)
	}
	if b[0] != 0xcc {
		t.Errorf("PubKeyHash.Bytes()[0] = %x, want %x", b[0], 0xcc)
	}
}

func TestHash512_FromHex(t *testing.T) {
	// 构造已知的十六进制字符串
	var original Hash512
	for i := range original {
		original[i] = byte(i)
	}
	hexStr := hex.EncodeToString(original[:])

	h, err := Hash512FromHex(hexStr)
	if err != nil {
		t.Fatalf("Hash512FromHex() error = %v", err)
	}
	if !h.Equal(original) {
		t.Error("Hash512FromHex() round-trip failed")
	}

	// 无效长度
	_, err = Hash512FromHex("abcd")
	if err == nil {
		t.Error("Hash512FromHex() should fail on short hex")
	}

	// 无效十六进制
	_, err = Hash512FromHex(string(make([]byte, 128))) // 128 零字节不是有效 hex
	if err == nil {
		t.Error("Hash512FromHex() should fail on invalid hex chars")
	}
}

// 常量值校验
func TestConstants(t *testing.T) {
	if HashLen != 64 {
		t.Errorf("HashLen = %d, want 64", HashLen)
	}
	if Hash384Len != 48 {
		t.Errorf("Hash384Len = %d, want 48", Hash384Len)
	}
	if PubKeyHashLen != 48 {
		t.Errorf("PubKeyHashLen = %d, want 48", PubKeyHashLen)
	}
	if MaxStackHeight != 256 {
		t.Errorf("MaxStackHeight = %d, want 256", MaxStackHeight)
	}
	if MaxTxSize != 65535 {
		t.Errorf("MaxTxSize = %d, want 65535", MaxTxSize)
	}
}
```

**Step 2: 运行测试验证失败**

Run: `go test -v ./pkg/types/ -run "TestHash512|TestHash384|TestPubKeyHash|TestConstants"`
Expected: FAIL（编译错误，类型和函数尚未定义）

**Step 3: 编写最小实现**

```go
// pkg/types/constants.go
package types

import "time"

const (
	HashLen       = 64            // SHA-512 哈希长度（字节）
	Hash384Len    = 48            // SHA3-384 哈希长度（字节）
	PubKeyHashLen = 48            // 公钥哈希长度（SHA3-384）
	BlockInterval = 6 * time.Minute // 出块间隔
	BlocksPerYear = 87661         // 每年区块数

	MaxStackHeight = 256   // 脚本栈最大高度
	MaxStackItem   = 1024  // 栈数据项最大尺寸（字节）
	MaxLockScript  = 1024  // 锁定脚本最大长度（字节）
	MaxUnlockScript = 4096 // 解锁脚本最大长度（字节）
	MaxTxSize      = 65535 // 单笔交易最大尺寸（字节）

	MaxMemo         = 255  // 币金附言最大长度（字节）
	MaxTitle        = 255  // 凭信/存证标题最大长度（字节）
	MaxCredDesc     = 1023 // 凭信描述最大长度（字节）
	MaxProofContent = 4095 // 存证内容最大长度（字节）
	MaxSelfData     = 255  // Coinbase 自由数据最大长度（字节）

	BestPoolCapacity     = 20    // 择优池容量
	ForkWindowSize       = 25    // 分叉竞争窗口
	TxExpiryBlocks       = 240   // 交易过期区块数
	FeeRecalcPeriod      = 6000  // 最低手续费重算周期
	CoinbaseConfirmDepth = 25    // 新币确认深度
	MintTxMinDepth       = 25    // 铸凭交易最小深度
	MintTxMaxDepth       = 80000 // 铸凭交易最大深度
	RefBlockOffset       = 9     // 评参区块偏移
	CoinAgeOffset        = 24    // 币权源区块偏移
)
```

```go
// pkg/types/hash.go
package types

import (
	"encoding/hex"
	"fmt"
)

// Hash512 表示 64 字节的 SHA-512 哈希值。
type Hash512 [HashLen]byte

// IsZero 检查哈希是否为全零值。
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

// Equal 比较两个哈希是否相等。
func (h Hash512) Equal(other Hash512) bool {
	return h == other
}

// Bytes 返回哈希的字节切片副本。
func (h Hash512) Bytes() []byte {
	b := make([]byte, HashLen)
	copy(b, h[:])
	return b
}

// Hash512FromHex 从十六进制字符串解析 Hash512。
func Hash512FromHex(s string) (Hash512, error) {
	var h Hash512
	b, err := hex.DecodeString(s)
	if err != nil {
		return h, fmt.Errorf("decode hex: %w", err)
	}
	if len(b) != HashLen {
		return h, fmt.Errorf("invalid hash length: got %d, want %d", len(b), HashLen)
	}
	copy(h[:], b)
	return h, nil
}

// Hash384 表示 48 字节的 SHA3-384 哈希值。
type Hash384 [Hash384Len]byte

// IsZero 检查哈希是否为全零值。
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

// Equal 比较两个哈希是否相等。
func (h Hash384) Equal(other Hash384) bool {
	return h == other
}

// Bytes 返回哈希的字节切片副本。
func (h Hash384) Bytes() []byte {
	b := make([]byte, Hash384Len)
	copy(b, h[:])
	return b
}

// PubKeyHash 表示 48 字节的公钥哈希（SHA3-384）。
type PubKeyHash [PubKeyHashLen]byte

// IsZero 检查公钥哈希是否为全零值。
func (p PubKeyHash) IsZero() bool {
	for _, b := range p {
		if b != 0 {
			return false
		}
	}
	return true
}

// String 返回公钥哈希的十六进制字符串表示。
func (p PubKeyHash) String() string {
	return hex.EncodeToString(p[:])
}

// Equal 比较两个公钥哈希是否相等。
func (p PubKeyHash) Equal(other PubKeyHash) bool {
	return p == other
}

// Bytes 返回公钥哈希的字节切片副本。
func (p PubKeyHash) Bytes() []byte {
	b := make([]byte, PubKeyHashLen)
	copy(b, p[:])
	return b
}
```

**Step 4: 运行测试验证通过**

Run: `go test -v ./pkg/types/ -run "TestHash512|TestHash384|TestPubKeyHash|TestConstants"`
Expected: PASS

**Step 5: 提交**

```bash
git add pkg/types/constants.go pkg/types/hash.go pkg/types/hash_test.go
git commit -m "feat(types): add Hash512/Hash384/PubKeyHash types and constants"
```

---

### Task 2: OutputConfig 与 SigFlag (pkg/types/config.go)

**Files:**
- Create: `pkg/types/config.go`
- Test: `pkg/types/config_test.go`

**Step 1: 编写失败测试**

```go
// pkg/types/config_test.go
package types

import "testing"

// --- OutputConfig 测试 ---

func TestOutputConfig_Type(t *testing.T) {
	tests := []struct {
		name   string
		config OutputConfig
		want   OutputConfig
	}{
		{"coin", OutTypeCoin, OutTypeCoin},
		{"credit", OutTypeCredit, OutTypeCredit},
		{"proof", OutTypeProof, OutTypeProof},
		{"mediator", OutTypeMediator, OutTypeMediator},
		{"coin with flags", OutTypeCoin | OutHasAttach, OutTypeCoin},
		{"credit with destroy", OutTypeCredit | OutDestroy, OutTypeCredit},
		{"custom class masks type", OutCustomClass | 0x0F, 0x0F}, // 自定义类时低 7 位另有含义
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.Type()
			if got != tt.want {
				t.Errorf("OutputConfig(%08b).Type() = %d, want %d", tt.config, got, tt.want)
			}
		})
	}
}

func TestOutputConfig_IsCustom(t *testing.T) {
	tests := []struct {
		name   string
		config OutputConfig
		want   bool
	}{
		{"not custom", OutTypeCoin, false},
		{"custom", OutCustomClass | 5, true},
		{"custom with zero", OutCustomClass, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.IsCustom(); got != tt.want {
				t.Errorf("OutputConfig(%08b).IsCustom() = %v, want %v", tt.config, got, tt.want)
			}
		})
	}
}

func TestOutputConfig_HasAttachment(t *testing.T) {
	tests := []struct {
		name   string
		config OutputConfig
		want   bool
	}{
		{"no attachment", OutTypeCoin, false},
		{"has attachment", OutTypeCoin | OutHasAttach, true},
		{"custom class ignores attach", OutCustomClass | OutHasAttach, false}, // 自定义类下 bit 6 另有含义
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.HasAttachment(); got != tt.want {
				t.Errorf("OutputConfig(%08b).HasAttachment() = %v, want %v", tt.config, got, tt.want)
			}
		})
	}
}

func TestOutputConfig_IsDestroy(t *testing.T) {
	tests := []struct {
		name   string
		config OutputConfig
		want   bool
	}{
		{"not destroy", OutTypeCoin, false},
		{"destroy", OutTypeCoin | OutDestroy, true},
		{"destroy credit", OutTypeCredit | OutDestroy, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.IsDestroy(); got != tt.want {
				t.Errorf("OutputConfig(%08b).IsDestroy() = %v, want %v", tt.config, got, tt.want)
			}
		})
	}
}

func TestOutputConfig_HasType(t *testing.T) {
	tests := []struct {
		name   string
		config OutputConfig
		typ    OutputConfig
		want   bool
	}{
		{"coin is coin", OutTypeCoin, OutTypeCoin, true},
		{"coin is not credit", OutTypeCoin, OutTypeCredit, false},
		{"coin with flags is coin", OutTypeCoin | OutDestroy | OutHasAttach, OutTypeCoin, true},
		{"zero type", OutputConfig(0), OutTypeCoin, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.HasType(tt.typ); got != tt.want {
				t.Errorf("OutputConfig(%08b).HasType(%d) = %v, want %v", tt.config, tt.typ, got, tt.want)
			}
		})
	}
}

// --- SigFlag 测试 ---

func TestSigFlag_Has(t *testing.T) {
	flag := SIGIN_ALL | SIGOUT_ALL | SIGOUTPUT
	tests := []struct {
		name string
		f    SigFlag
		want bool
	}{
		{"has SIGIN_ALL", SIGIN_ALL, true},
		{"has SIGOUT_ALL", SIGOUT_ALL, true},
		{"has SIGOUTPUT", SIGOUTPUT, true},
		{"no SIGIN_SELF", SIGIN_SELF, false},
		{"no SIGOUT_SELF", SIGOUT_SELF, false},
		{"no SIGRECEIVER", SIGRECEIVER, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := flag.Has(tt.f); got != tt.want {
				t.Errorf("SigFlag(%08b).Has(%08b) = %v, want %v", flag, tt.f, got, tt.want)
			}
		})
	}
}

func TestSigFlag_CoveredInputs(t *testing.T) {
	tests := []struct {
		name     string
		flag     SigFlag
		wantAll  bool
		wantSelf bool
	}{
		{"all inputs", SIGIN_ALL, true, false},
		{"self input", SIGIN_SELF, false, true},
		{"both", SIGIN_ALL | SIGIN_SELF, true, true},
		{"none", SIGOUT_ALL, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAll, gotSelf := tt.flag.CoveredInputs()
			if gotAll != tt.wantAll || gotSelf != tt.wantSelf {
				t.Errorf("SigFlag(%08b).CoveredInputs() = (%v, %v), want (%v, %v)",
					tt.flag, gotAll, gotSelf, tt.wantAll, tt.wantSelf)
			}
		})
	}
}

func TestSigFlag_CoveredOutputs(t *testing.T) {
	tests := []struct {
		name     string
		flag     SigFlag
		wantAll  bool
		wantSelf bool
	}{
		{"all outputs", SIGOUT_ALL | SIGOUTPUT, true, false},
		{"self output", SIGOUT_SELF | SIGRECEIVER, false, true},
		{"both", SIGOUT_ALL | SIGOUT_SELF | SIGCONTENT, true, true},
		{"none - no primary", SIGOUTPUT, false, false}, // 辅项单独无效
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAll, gotSelf := tt.flag.CoveredOutputs()
			if gotAll != tt.wantAll || gotSelf != tt.wantSelf {
				t.Errorf("SigFlag(%08b).CoveredOutputs() = (%v, %v), want (%v, %v)",
					tt.flag, gotAll, gotSelf, tt.wantAll, tt.wantSelf)
			}
		})
	}
}

func TestSigFlag_Constants(t *testing.T) {
	// 验证各标志位不重叠
	flags := []SigFlag{
		SIGIN_ALL, SIGIN_SELF,
		SIGOUT_ALL, SIGOUT_SELF,
		SIGOUTPUT, SIGSCRIPT, SIGCONTENT, SIGRECEIVER,
	}
	for i := 0; i < len(flags); i++ {
		for j := i + 1; j < len(flags); j++ {
			if flags[i]&flags[j] != 0 {
				t.Errorf("SigFlag bit collision: %08b & %08b != 0", flags[i], flags[j])
			}
		}
	}
}
```

**Step 2: 运行测试验证失败**

Run: `go test -v ./pkg/types/ -run "TestOutputConfig|TestSigFlag"`
Expected: FAIL

**Step 3: 编写最小实现**

```go
// pkg/types/config.go
package types

// OutputConfig 输出项配置（1 字节）。
// 高 4 位为标记位，低 4 位为类型值。
type OutputConfig byte

const (
	OutCustomClass  OutputConfig = 1 << 7 // 自定义类：余下 7 位为类 ID 长度
	OutHasAttach    OutputConfig = 1 << 6 // 包含附件
	OutDestroy      OutputConfig = 1 << 5 // 销毁标记
	// bit 4: reserved

	OutTypeCoin     OutputConfig = 1 // 币金
	OutTypeCredit   OutputConfig = 2 // 凭信
	OutTypeProof    OutputConfig = 3 // 存证
	OutTypeMediator OutputConfig = 4 // 介管脚本
)

// typeMask 提取低 4 位类型值的掩码。
const typeMask OutputConfig = 0x0F

// Type 返回输出项类型（低 4 位）。
// 注意：当 IsCustom() 为 true 时，低 7 位编码类 ID 长度，
// 此方法仍返回低 4 位的值，调用者需自行判断。
func (c OutputConfig) Type() OutputConfig {
	return c & typeMask
}

// IsCustom 检查是否为自定义类（bit 7 置位）。
func (c OutputConfig) IsCustom() bool {
	return c&OutCustomClass != 0
}

// HasAttachment 检查是否包含附件（bit 6 置位）。
// 当 IsCustom() 为 true 时，bit 6 另有含义，此方法返回 false。
func (c OutputConfig) HasAttachment() bool {
	if c.IsCustom() {
		return false
	}
	return c&OutHasAttach != 0
}

// IsDestroy 检查是否为销毁输出（bit 5 置位）。
func (c OutputConfig) IsDestroy() bool {
	return c&OutDestroy != 0
}

// HasType 检查输出项是否为指定类型。
func (c OutputConfig) HasType(typ OutputConfig) bool {
	return c.Type() == typ
}

// SigFlag 签名授权标志（1 字节）。
type SigFlag byte

const (
	// 独项（可单独使用）
	SIGIN_ALL  SigFlag = 1 << 7 // 全部输入项
	SIGIN_SELF SigFlag = 1 << 6 // 仅当前输入项

	// 主项（必须与辅项组合）
	SIGOUT_ALL  SigFlag = 1 << 5 // 全部输出项
	SIGOUT_SELF SigFlag = 1 << 4 // 与当前输入同序位的输出项

	// 辅项（必须与主项组合）
	SIGOUTPUT   SigFlag = 1 << 3 // 完整输出条目
	SIGSCRIPT   SigFlag = 1 << 2 // 输出的锁定脚本
	SIGCONTENT  SigFlag = 1 << 1 // 输出内容
	SIGRECEIVER SigFlag = 1 << 0 // 输出的接收者
)

// Has 检查标志位是否包含指定标志。
func (f SigFlag) Has(flag SigFlag) bool {
	return f&flag != 0
}

// CoveredInputs 返回签名覆盖的输入范围。
// 返回 (all, self)，表示是否覆盖全部输入和/或仅当前输入。
func (f SigFlag) CoveredInputs() (all, self bool) {
	return f.Has(SIGIN_ALL), f.Has(SIGIN_SELF)
}

// CoveredOutputs 返回签名覆盖的输出范围。
// 仅当主项（SIGOUT_ALL 或 SIGOUT_SELF）存在时，辅项才有效。
// 返回 (all, self)，表示是否覆盖全部输出和/或仅同序位输出。
func (f SigFlag) CoveredOutputs() (all, self bool) {
	hasPrimary := f.Has(SIGOUT_ALL) || f.Has(SIGOUT_SELF)
	hasAuxiliary := f.Has(SIGOUTPUT) || f.Has(SIGSCRIPT) || f.Has(SIGCONTENT) || f.Has(SIGRECEIVER)

	if !hasPrimary || !hasAuxiliary {
		return false, false
	}
	return f.Has(SIGOUT_ALL), f.Has(SIGOUT_SELF)
}
```

**Step 4: 运行测试验证通过**

Run: `go test -v ./pkg/types/ -run "TestOutputConfig|TestSigFlag"`
Expected: PASS

**Step 5: 提交**

```bash
git add pkg/types/config.go pkg/types/config_test.go
git commit -m "feat(types): add OutputConfig and SigFlag types"
```

---

### Task 3: Varint 变长整数 (pkg/types/varint.go)

**Files:**
- Create: `pkg/types/varint.go`
- Test: `pkg/types/varint_test.go`

**Step 1: 编写失败测试**

```go
// pkg/types/varint_test.go
package types

import "testing"

func TestVarint_RoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		value    uint64
		wantSize int // 编码后字节数
	}{
		{"zero", 0, 1},
		{"one", 1, 1},
		{"max 1-byte", 252, 1},
		{"min 3-byte", 253, 3},
		{"mid 3-byte", 0x1234, 3},
		{"max 3-byte (0xFFFF)", 0xFFFF, 3},
		{"min 5-byte", 0x10000, 5},
		{"mid 5-byte", 0x12345678, 5},
		{"max 5-byte (0xFFFFFFFF)", 0xFFFFFFFF, 5},
		{"min 9-byte", 0x100000000, 9},
		{"large value", 0x123456789ABCDEF0, 9},
		{"max uint64", 0xFFFFFFFFFFFFFFFF, 9},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := EncodeVarint(tt.value)
			if len(encoded) != tt.wantSize {
				t.Errorf("EncodeVarint(%d) size = %d, want %d", tt.value, len(encoded), tt.wantSize)
			}

			decoded, n, err := DecodeVarint(encoded)
			if err != nil {
				t.Fatalf("DecodeVarint() error = %v", err)
			}
			if n != tt.wantSize {
				t.Errorf("DecodeVarint() bytesRead = %d, want %d", n, tt.wantSize)
			}
			if decoded != tt.value {
				t.Errorf("DecodeVarint() = %d, want %d", decoded, tt.value)
			}
		})
	}
}

func TestDecodeVarint_Errors(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"truncated 3-byte", []byte{0xFD, 0x01}},       // 需要 3 字节
		{"truncated 5-byte", []byte{0xFE, 0x01, 0x02}},  // 需要 5 字节
		{"truncated 9-byte", []byte{0xFF, 0x01, 0x02, 0x03}}, // 需要 9 字节
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := DecodeVarint(tt.data)
			if err == nil {
				t.Error("DecodeVarint() should return error for truncated data")
			}
		})
	}
}

func TestDecodeVarint_WithTrailingData(t *testing.T) {
	// 编码值 1000（3 字节编码），后面附加额外数据
	data := EncodeVarint(1000)
	data = append(data, 0xFF, 0xFF) // 额外字节

	value, n, err := DecodeVarint(data)
	if err != nil {
		t.Fatalf("DecodeVarint() error = %v", err)
	}
	if value != 1000 {
		t.Errorf("DecodeVarint() value = %d, want 1000", value)
	}
	if n != 3 {
		t.Errorf("DecodeVarint() bytesRead = %d, want 3", n)
	}
}

func TestVarint_BoundaryValues(t *testing.T) {
	// 验证编码格式标记字节
	// 0-252: 单字节（直接值）
	b := EncodeVarint(252)
	if b[0] != 252 {
		t.Errorf("EncodeVarint(252) first byte = %d, want 252", b[0])
	}

	// 253: 三字节（0xFD 前缀）
	b = EncodeVarint(253)
	if b[0] != 0xFD {
		t.Errorf("EncodeVarint(253) prefix = 0x%02X, want 0xFD", b[0])
	}

	// 65536: 五字节（0xFE 前缀）
	b = EncodeVarint(65536)
	if b[0] != 0xFE {
		t.Errorf("EncodeVarint(65536) prefix = 0x%02X, want 0xFE", b[0])
	}

	// 0x100000000: 九字节（0xFF 前缀）
	b = EncodeVarint(0x100000000)
	if b[0] != 0xFF {
		t.Errorf("EncodeVarint(0x100000000) prefix = 0x%02X, want 0xFF", b[0])
	}
}
```

**Step 2: 运行测试验证失败**

Run: `go test -v ./pkg/types/ -run TestVarint`
Expected: FAIL

**Step 3: 编写最小实现**

```go
// pkg/types/varint.go
package types

import (
	"encoding/binary"
	"errors"
)

// EncodeVarint 将无符号整数编码为 Bitcoin 风格的变长整数。
//
// 编码规则：
//   - 0 ~ 252:        1 字节（直接值）
//   - 253 ~ 0xFFFF:   3 字节（0xFD + 2 字节小端序）
//   - 0x10000 ~ 0xFFFFFFFF: 5 字节（0xFE + 4 字节小端序）
//   - 0x100000000 ~ :  9 字节（0xFF + 8 字节小端序）
func EncodeVarint(v uint64) []byte {
	switch {
	case v <= 252:
		return []byte{byte(v)}
	case v <= 0xFFFF:
		buf := make([]byte, 3)
		buf[0] = 0xFD
		binary.LittleEndian.PutUint16(buf[1:], uint16(v))
		return buf
	case v <= 0xFFFFFFFF:
		buf := make([]byte, 5)
		buf[0] = 0xFE
		binary.LittleEndian.PutUint32(buf[1:], uint32(v))
		return buf
	default:
		buf := make([]byte, 9)
		buf[0] = 0xFF
		binary.LittleEndian.PutUint64(buf[1:], v)
		return buf
	}
}

// DecodeVarint 从字节流解码变长整数。
// 返回解码后的值、读取的字节数和可能的错误。
func DecodeVarint(data []byte) (uint64, int, error) {
	if len(data) == 0 {
		return 0, 0, errors.New("empty data for varint decode")
	}

	first := data[0]
	switch {
	case first <= 252:
		return uint64(first), 1, nil
	case first == 0xFD:
		if len(data) < 3 {
			return 0, 0, errors.New("truncated 3-byte varint")
		}
		return uint64(binary.LittleEndian.Uint16(data[1:3])), 3, nil
	case first == 0xFE:
		if len(data) < 5 {
			return 0, 0, errors.New("truncated 5-byte varint")
		}
		return uint64(binary.LittleEndian.Uint32(data[1:5])), 5, nil
	default: // 0xFF
		if len(data) < 9 {
			return 0, 0, errors.New("truncated 9-byte varint")
		}
		return binary.LittleEndian.Uint64(data[1:9]), 9, nil
	}
}
```

**Step 4: 运行测试验证通过**

Run: `go test -v ./pkg/types/ -run TestVarint`
Expected: PASS

**Step 5: 提交**

```bash
git add pkg/types/varint.go pkg/types/varint_test.go
git commit -m "feat(types): add Bitcoin-style varint encoding/decoding"
```

---

### Task 4: 地址编码 (pkg/types/address.go)

**Files:**
- Create: `pkg/types/address.go`
- Test: `pkg/types/address_test.go`

**Step 1: 编写失败测试**

```go
// pkg/types/address_test.go
package types

import (
	"bytes"
	"testing"
)

func TestNewAddress(t *testing.T) {
	var pkh PubKeyHash
	pkh[0] = 0xAA
	pkh[47] = 0xBB

	addr := NewAddress(MainNetPrefix, pkh)
	if addr.Prefix != MainNetPrefix {
		t.Errorf("Address.Prefix = %q, want %q", addr.Prefix, MainNetPrefix)
	}
	if !addr.PKHash.Equal(pkh) {
		t.Error("Address.PKHash should match input")
	}
	// checksum 不应全为零
	if addr.Check == [AddressChecksumLen]byte{} {
		t.Error("Address.Check should not be zero")
	}
}

func TestAddress_Verify(t *testing.T) {
	var pkh PubKeyHash
	for i := range pkh {
		pkh[i] = byte(i)
	}

	addr := NewAddress(MainNetPrefix, pkh)
	if !addr.Verify() {
		t.Error("Address.Verify() should return true for valid address")
	}

	// 篡改 checksum
	tampered := *addr
	tampered.Check[0] ^= 0xFF
	if tampered.Verify() {
		t.Error("Address.Verify() should return false for tampered checksum")
	}

	// 篡改公钥哈希
	tampered2 := *addr
	tampered2.PKHash[0] ^= 0xFF
	if tampered2.Verify() {
		t.Error("Address.Verify() should return false for tampered PKHash")
	}

	// 篡改前缀
	tampered3 := *addr
	tampered3.Prefix = TestNetPrefix
	if tampered3.Verify() {
		t.Error("Address.Verify() should return false for wrong prefix")
	}
}

func TestAddress_Encode_Decode_RoundTrip(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		pkh    PubKeyHash
	}{
		{"mainnet zero", MainNetPrefix, PubKeyHash{}},
		{"testnet zero", TestNetPrefix, PubKeyHash{}},
		{"mainnet non-zero", MainNetPrefix, func() PubKeyHash {
			var p PubKeyHash
			for i := range p {
				p[i] = byte(i * 3)
			}
			return p
		}()},
		{"testnet non-zero", TestNetPrefix, func() PubKeyHash {
			var p PubKeyHash
			for i := range p {
				p[i] = byte(0xFF - byte(i))
			}
			return p
		}()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr := NewAddress(tt.prefix, tt.pkh)
			encoded := addr.Encode()

			// 编码后应以前缀开头
			if len(encoded) < len(tt.prefix) || encoded[:len(tt.prefix)] != tt.prefix {
				t.Errorf("Encode() = %q, should start with %q", encoded, tt.prefix)
			}

			// 解码
			decoded, err := DecodeAddress(encoded)
			if err != nil {
				t.Fatalf("DecodeAddress() error = %v", err)
			}

			if decoded.Prefix != tt.prefix {
				t.Errorf("decoded.Prefix = %q, want %q", decoded.Prefix, tt.prefix)
			}
			if !decoded.PKHash.Equal(tt.pkh) {
				t.Error("decoded.PKHash should match original")
			}
			if !bytes.Equal(decoded.Check[:], addr.Check[:]) {
				t.Error("decoded.Check should match original")
			}
			if !decoded.Verify() {
				t.Error("decoded address should verify successfully")
			}
		})
	}
}

func TestDecodeAddress_Errors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"too short", "EC"},
		{"unknown prefix", "XXabc123"},
		{"invalid base58", "EC00000OOO"},     // O 不在 Base58 字符集中
		{"truncated payload", "EC" + "1111"}, // 太短无法包含完整 PKHash + checksum
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeAddress(tt.input)
			if err == nil {
				t.Errorf("DecodeAddress(%q) should return error", tt.input)
			}
		})
	}
}

func TestAddress_DifferentPrefixes_DifferentEncoding(t *testing.T) {
	var pkh PubKeyHash
	pkh[0] = 0x42

	main := NewAddress(MainNetPrefix, pkh)
	test := NewAddress(TestNetPrefix, pkh)

	if main.Encode() == test.Encode() {
		t.Error("mainnet and testnet addresses should differ")
	}
}
```

**Step 2: 运行测试验证失败**

Run: `go test -v ./pkg/types/ -run TestAddress`
Expected: FAIL

**Step 3: 编写最小实现**

```go
// pkg/types/address.go
package types

import (
	"crypto/sha512"
	"fmt"

	"github.com/mr-tron/base58"
)

const (
	AddressChecksumLen = 4    // 地址校验码长度（字节）
	MainNetPrefix      = "EC" // 主网前缀
	TestNetPrefix      = "ET" // 测试网前缀
)

// Address 表示 Evidcoin 地址。
type Address struct {
	Prefix string                     // 网络识别前缀
	PKHash PubKeyHash                 // 48 字节公钥哈希
	Check  [AddressChecksumLen]byte   // 4 字节校验码
}

// computeChecksum 计算地址校验码。
// 规则：SHA-512(prefix || PKHash)，取最后 4 字节。
func computeChecksum(prefix string, pkh PubKeyHash) [AddressChecksumLen]byte {
	data := append([]byte(prefix), pkh[:]...)
	hash := sha512.Sum512(data)
	var check [AddressChecksumLen]byte
	copy(check[:], hash[len(hash)-AddressChecksumLen:])
	return check
}

// NewAddress 根据前缀和公钥哈希创建新地址。
func NewAddress(prefix string, pkh PubKeyHash) *Address {
	check := computeChecksum(prefix, pkh)
	return &Address{
		Prefix: prefix,
		PKHash: pkh,
		Check:  check,
	}
}

// Encode 将地址编码为人类可读的文本。
// 格式：prefix + Base58(PKHash || checksum)
func (a *Address) Encode() string {
	payload := make([]byte, 0, PubKeyHashLen+AddressChecksumLen)
	payload = append(payload, a.PKHash[:]...)
	payload = append(payload, a.Check[:]...)
	encoded := base58.Encode(payload)
	return a.Prefix + encoded
}

// Verify 验证地址的完整性。
func (a *Address) Verify() bool {
	expected := computeChecksum(a.Prefix, a.PKHash)
	return a.Check == expected
}

// DecodeAddress 从文本形式解码地址。
func DecodeAddress(s string) (*Address, error) {
	// 识别前缀
	var prefix string
	switch {
	case len(s) > len(MainNetPrefix) && s[:len(MainNetPrefix)] == MainNetPrefix:
		prefix = MainNetPrefix
	case len(s) > len(TestNetPrefix) && s[:len(TestNetPrefix)] == TestNetPrefix:
		prefix = TestNetPrefix
	default:
		return nil, fmt.Errorf("unknown or missing address prefix: %q", s)
	}

	// Base58 解码
	encoded := s[len(prefix):]
	payload, err := base58.Decode(encoded)
	if err != nil {
		return nil, fmt.Errorf("base58 decode: %w", err)
	}

	// 校验载荷长度
	expectedLen := PubKeyHashLen + AddressChecksumLen
	if len(payload) != expectedLen {
		return nil, fmt.Errorf("invalid payload length: got %d, want %d", len(payload), expectedLen)
	}

	// 解析公钥哈希和校验码
	var pkh PubKeyHash
	copy(pkh[:], payload[:PubKeyHashLen])
	var check [AddressChecksumLen]byte
	copy(check[:], payload[PubKeyHashLen:])

	addr := &Address{
		Prefix: prefix,
		PKHash: pkh,
		Check:  check,
	}

	// 验证校验码
	if !addr.Verify() {
		return nil, fmt.Errorf("address checksum mismatch")
	}

	return addr, nil
}
```

**Step 4: 运行测试验证通过**

Run: `go test -v ./pkg/types/ -run TestAddress`
Expected: PASS

**Step 5: 提交**

```bash
git add pkg/types/address.go pkg/types/address_test.go
git commit -m "feat(types): add address encoding with Base58 and checksum"
```

---

## pkg/crypto 部分

### Task 5: 哈希函数封装 (pkg/crypto/hash.go)

**Files:**
- Create: `pkg/crypto/hash.go`
- Test: `pkg/crypto/hash_test.go`

**Step 1: 编写失败测试**

```go
// pkg/crypto/hash_test.go
package crypto

import (
	"bytes"
	"crypto/sha512"
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
	"golang.org/x/crypto/sha3"
)

func TestSHA512Sum(t *testing.T) {
	data := []byte("hello evidcoin")
	result := SHA512Sum(data)

	// 与标准库对比
	expected := sha512.Sum512(data)
	if result != types.Hash512(expected) {
		t.Error("SHA512Sum() result mismatch with stdlib")
	}

	// 不同输入应产生不同输出
	other := SHA512Sum([]byte("other data"))
	if result == other {
		t.Error("SHA512Sum() should produce different hashes for different inputs")
	}

	// 相同输入应产生相同输出
	again := SHA512Sum(data)
	if result != again {
		t.Error("SHA512Sum() should be deterministic")
	}

	// 空输入
	empty := SHA512Sum(nil)
	emptyExpected := sha512.Sum512(nil)
	if empty != types.Hash512(emptyExpected) {
		t.Error("SHA512Sum(nil) should match stdlib sha512.Sum512(nil)")
	}
}

func TestSHA3_384Sum(t *testing.T) {
	data := []byte("hello evidcoin")
	result := SHA3_384Sum(data)

	// 与标准实现对比
	h := sha3.New384()
	h.Write(data)
	expected := h.Sum(nil)
	if !bytes.Equal(result[:], expected) {
		t.Error("SHA3_384Sum() result mismatch")
	}

	// 输出长度应为 48 字节
	if len(result.Bytes()) != types.Hash384Len {
		t.Errorf("SHA3_384Sum() output length = %d, want %d", len(result.Bytes()), types.Hash384Len)
	}
}

func TestSHA3_512Sum(t *testing.T) {
	data := []byte("hello evidcoin")
	result := SHA3_512Sum(data)

	// 与标准实现对比
	h := sha3.New512()
	h.Write(data)
	expected := h.Sum(nil)
	if !bytes.Equal(result[:], expected) {
		t.Error("SHA3_512Sum() result mismatch")
	}

	// 输出长度应为 64 字节
	if len(result.Bytes()) != types.HashLen {
		t.Errorf("SHA3_512Sum() output length = %d, want %d", len(result.Bytes()), types.HashLen)
	}

	// SHA3-512 与 SHA-512 应产生不同结果
	sha512Result := SHA512Sum(data)
	if result == sha512Result {
		t.Error("SHA3_512Sum() should differ from SHA512Sum()")
	}
}

func TestBlake3Sum(t *testing.T) {
	data := []byte("hello evidcoin")

	tests := []struct {
		name string
		size int
	}{
		{"16 bytes (minimum)", 16},
		{"20 bytes", 20},
		{"32 bytes", 32},
		{"48 bytes", 48},
		{"64 bytes (maximum)", 64},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Blake3Sum(data, tt.size)
			if len(result) != tt.size {
				t.Errorf("Blake3Sum() length = %d, want %d", len(result), tt.size)
			}

			// 确定性
			again := Blake3Sum(data, tt.size)
			if !bytes.Equal(result, again) {
				t.Error("Blake3Sum() should be deterministic")
			}
		})
	}

	// 不同长度的前缀应一致（BLAKE3 的可扩展输出特性）
	short := Blake3Sum(data, 16)
	long := Blake3Sum(data, 64)
	if !bytes.Equal(short, long[:16]) {
		t.Error("Blake3Sum() shorter output should be prefix of longer output")
	}
}

func TestBlake3Sum_DifferentInputs(t *testing.T) {
	a := Blake3Sum([]byte("input A"), 32)
	b := Blake3Sum([]byte("input B"), 32)
	if bytes.Equal(a, b) {
		t.Error("Blake3Sum() should produce different outputs for different inputs")
	}
}
```

**Step 2: 运行测试验证失败**

Run: `go test -v ./pkg/crypto/ -run "TestSHA512Sum|TestSHA3|TestBlake3Sum"`
Expected: FAIL

**Step 3: 编写最小实现**

```go
// pkg/crypto/hash.go
package crypto

import (
	"crypto/sha512"

	"github.com/cxio/evidcoin/pkg/types"
	"golang.org/x/crypto/sha3"
	"lukechampine.com/blake3"
)

// SHA512Sum 计算 SHA-512 哈希。
func SHA512Sum(data []byte) types.Hash512 {
	return types.Hash512(sha512.Sum512(data))
}

// SHA3_384Sum 计算 SHA3-384 哈希。
func SHA3_384Sum(data []byte) types.Hash384 {
	h := sha3.New384()
	h.Write(data)
	var result types.Hash384
	copy(result[:], h.Sum(nil))
	return result
}

// SHA3_512Sum 计算 SHA3-512 哈希。
func SHA3_512Sum(data []byte) types.Hash512 {
	h := sha3.New512()
	h.Write(data)
	var result types.Hash512
	copy(result[:], h.Sum(nil))
	return result
}

// Blake3Sum 计算 BLAKE3 哈希，支持可变长度输出。
// size 指定输出字节长度（最小 16，最大 64）。
func Blake3Sum(data []byte, size int) []byte {
	h := blake3.New(size, nil)
	h.Write(data)
	return h.Sum(nil)
}
```

**Step 4: 运行测试验证通过**

Run: `go test -v ./pkg/crypto/ -run "TestSHA512Sum|TestSHA3|TestBlake3Sum"`
Expected: PASS

**Step 5: 提交**

```bash
git add pkg/crypto/hash.go pkg/crypto/hash_test.go
git commit -m "feat(crypto): add SHA-512, SHA3-384/512, BLAKE3 hash functions"
```

---

### Task 6: 签名接口定义 (pkg/crypto/sign.go)

**Files:**
- Create: `pkg/crypto/sign.go`

此 Task 仅定义接口，不需要独立的单元测试（接口将通过 Task 7-9 的实现测试验证）。但我们验证编译通过。

**Step 1: 编写接口定义**

```go
// pkg/crypto/sign.go
package crypto

// Signer 签名者接口。
// 持有私钥，可对消息生成签名。
type Signer interface {
	// Sign 对消息进行签名。
	Sign(message []byte) ([]byte, error)

	// PublicKey 返回对应的公钥字节。
	PublicKey() []byte

	// Algorithm 返回签名算法标识。
	Algorithm() string
}

// Verifier 验证者接口。
// 无需私钥，仅用公钥验证签名。
type Verifier interface {
	// Verify 使用公钥验证消息签名。
	Verify(message, signature, publicKey []byte) (bool, error)

	// Algorithm 返回签名算法标识。
	Algorithm() string
}
```

**Step 2: 验证编译通过**

Run: `go build ./pkg/crypto/`
Expected: 编译成功

**Step 3: 提交**

```bash
git add pkg/crypto/sign.go
git commit -m "feat(crypto): define Signer and Verifier interfaces"
```

---

### Task 7: ML-DSA-65 签名实现 (pkg/crypto/mldsa.go)

**Files:**
- Create: `pkg/crypto/mldsa.go`
- Test: `pkg/crypto/mldsa_test.go`

**Step 1: 编写失败测试**

```go
// pkg/crypto/mldsa_test.go
package crypto

import (
	"testing"
)

func TestMLDSAKeyPair_Generate(t *testing.T) {
	kp, err := GenerateMLDSAKeyPair()
	if err != nil {
		t.Fatalf("GenerateMLDSAKeyPair() error = %v", err)
	}
	if kp == nil {
		t.Fatal("GenerateMLDSAKeyPair() returned nil")
	}

	// ML-DSA-65 公钥大小应为 1952 字节
	if len(kp.PublicKeyBytes) != MLDSAPubKeySize {
		t.Errorf("public key length = %d, want %d", len(kp.PublicKeyBytes), MLDSAPubKeySize)
	}
}

func TestMLDSASigner_Interface(t *testing.T) {
	kp, err := GenerateMLDSAKeyPair()
	if err != nil {
		t.Fatalf("GenerateMLDSAKeyPair() error = %v", err)
	}

	signer := kp.Signer()

	// 检查接口兼容性
	var _ Signer = signer

	if algo := signer.Algorithm(); algo != "ML-DSA-65" {
		t.Errorf("Algorithm() = %q, want %q", algo, "ML-DSA-65")
	}

	pub := signer.PublicKey()
	if len(pub) != MLDSAPubKeySize {
		t.Errorf("PublicKey() length = %d, want %d", len(pub), MLDSAPubKeySize)
	}
}

func TestMLDSA_SignVerify(t *testing.T) {
	kp, err := GenerateMLDSAKeyPair()
	if err != nil {
		t.Fatalf("GenerateMLDSAKeyPair() error = %v", err)
	}

	signer := kp.Signer()
	verifier := NewMLDSAVerifier()

	message := []byte("test message for ML-DSA-65 signing")

	// 签名
	sig, err := signer.Sign(message)
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}
	if len(sig) != MLDSASigSize {
		t.Errorf("signature length = %d, want %d", len(sig), MLDSASigSize)
	}

	// 验证成功
	valid, err := verifier.Verify(message, sig, signer.PublicKey())
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !valid {
		t.Error("Verify() should return true for valid signature")
	}
}

func TestMLDSA_VerifyFail_WrongMessage(t *testing.T) {
	kp, _ := GenerateMLDSAKeyPair()
	signer := kp.Signer()
	verifier := NewMLDSAVerifier()

	sig, _ := signer.Sign([]byte("original"))

	valid, err := verifier.Verify([]byte("tampered"), sig, signer.PublicKey())
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if valid {
		t.Error("Verify() should return false for wrong message")
	}
}

func TestMLDSA_VerifyFail_WrongKey(t *testing.T) {
	kp1, _ := GenerateMLDSAKeyPair()
	kp2, _ := GenerateMLDSAKeyPair()

	signer := kp1.Signer()
	verifier := NewMLDSAVerifier()

	message := []byte("signed by kp1")
	sig, _ := signer.Sign(message)

	valid, err := verifier.Verify(message, sig, kp2.Signer().PublicKey())
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if valid {
		t.Error("Verify() should return false for wrong public key")
	}
}

func TestMLDSA_VerifyFail_TamperedSig(t *testing.T) {
	kp, _ := GenerateMLDSAKeyPair()
	signer := kp.Signer()
	verifier := NewMLDSAVerifier()

	sig, _ := signer.Sign([]byte("test"))

	tampered := make([]byte, len(sig))
	copy(tampered, sig)
	tampered[0] ^= 0xFF

	valid, err := verifier.Verify([]byte("test"), tampered, signer.PublicKey())
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if valid {
		t.Error("Verify() should return false for tampered signature")
	}
}

func TestMLDSAVerifier_Interface(t *testing.T) {
	verifier := NewMLDSAVerifier()
	var _ Verifier = verifier

	if algo := verifier.Algorithm(); algo != "ML-DSA-65" {
		t.Errorf("Algorithm() = %q, want %q", algo, "ML-DSA-65")
	}
}

func TestMLDSA_InvalidPubKeyLen(t *testing.T) {
	verifier := NewMLDSAVerifier()
	_, err := verifier.Verify([]byte("msg"), []byte("sig"), []byte("short"))
	if err == nil {
		t.Error("Verify() should fail for invalid public key length")
	}
}
```

**Step 2: 运行测试验证失败**

Run: `go test -v ./pkg/crypto/ -run TestMLDSA`
Expected: FAIL

**Step 3: 编写最小实现**

```go
// pkg/crypto/mldsa.go
package crypto

import (
	"fmt"

	"github.com/cloudflare/circl/sign/mldsa/mldsa65"
)

const (
	MLDSAPubKeySize = mldsa65.PublicKeySize  // 1952 字节
	MLDSASigSize    = mldsa65.SignatureSize   // 3309 字节
)

// MLDSAKeyPair 表示 ML-DSA-65 密钥对。
type MLDSAKeyPair struct {
	PrivateKey     *mldsa65.PrivateKey
	PublicKey_     *mldsa65.PublicKey
	PublicKeyBytes []byte
}

// GenerateMLDSAKeyPair 生成新的 ML-DSA-65 密钥对。
func GenerateMLDSAKeyPair() (*MLDSAKeyPair, error) {
	pub, priv, err := mldsa65.GenerateKey(nil)
	if err != nil {
		return nil, fmt.Errorf("generate ML-DSA-65 key: %w", err)
	}
	pubBytes, err := pub.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("marshal ML-DSA-65 public key: %w", err)
	}
	return &MLDSAKeyPair{
		PrivateKey:     priv,
		PublicKey_:     pub,
		PublicKeyBytes: pubBytes,
	}, nil
}

// Signer 返回基于此密钥对的签名者。
func (kp *MLDSAKeyPair) Signer() *MLDSASigner {
	return &MLDSASigner{keyPair: kp}
}

// MLDSASigner 使用 ML-DSA-65 私钥进行签名。
type MLDSASigner struct {
	keyPair *MLDSAKeyPair
}

// Sign 对消息进行 ML-DSA-65 签名。
func (s *MLDSASigner) Sign(message []byte) ([]byte, error) {
	sig := make([]byte, MLDSASigSize)
	mldsa65.SignTo(sig, s.keyPair.PrivateKey, message, nil, false)
	return sig, nil
}

// PublicKey 返回对应的公钥字节。
func (s *MLDSASigner) PublicKey() []byte {
	result := make([]byte, len(s.keyPair.PublicKeyBytes))
	copy(result, s.keyPair.PublicKeyBytes)
	return result
}

// Algorithm 返回算法标识。
func (s *MLDSASigner) Algorithm() string {
	return "ML-DSA-65"
}

// MLDSAVerifier 使用 ML-DSA-65 公钥验证签名。
type MLDSAVerifier struct{}

// NewMLDSAVerifier 创建新的 ML-DSA-65 验证者。
func NewMLDSAVerifier() *MLDSAVerifier {
	return &MLDSAVerifier{}
}

// Verify 验证 ML-DSA-65 签名。
func (v *MLDSAVerifier) Verify(message, signature, publicKey []byte) (bool, error) {
	if len(publicKey) != MLDSAPubKeySize {
		return false, fmt.Errorf("invalid ML-DSA-65 public key length: got %d, want %d", len(publicKey), MLDSAPubKeySize)
	}
	var pub mldsa65.PublicKey
	if err := pub.UnmarshalBinary(publicKey); err != nil {
		return false, fmt.Errorf("unmarshal ML-DSA-65 public key: %w", err)
	}
	return mldsa65.Verify(&pub, message, nil, signature), nil
}

// Algorithm 返回算法标识。
func (v *MLDSAVerifier) Algorithm() string {
	return "ML-DSA-65"
}
```

**Step 4: 运行测试验证通过**

Run: `go test -v ./pkg/crypto/ -run TestMLDSA`
Expected: PASS

**Step 5: 提交**

```bash
git add pkg/crypto/mldsa.go pkg/crypto/mldsa_test.go
git commit -m "feat(crypto): implement ML-DSA-65 signature with Signer/Verifier"
```

---

### Task 8: 密钥管理 (pkg/crypto/keys.go)

**Files:**
- Create: `pkg/crypto/keys.go`
- Test: `pkg/crypto/keys_test.go`

**Step 1: 编写失败测试**

```go
// pkg/crypto/keys_test.go
package crypto

import (
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

func TestPubKeyHash_MLDSA(t *testing.T) {
	kp, err := GenerateMLDSAKeyPair()
	if err != nil {
		t.Fatalf("GenerateMLDSAKeyPair() error = %v", err)
	}

	pkh := ComputePubKeyHash(kp.PublicKeyBytes)

	// 长度应为 48 字节
	if len(pkh.Bytes()) != types.PubKeyHashLen {
		t.Errorf("PubKeyHash length = %d, want %d", len(pkh.Bytes()), types.PubKeyHashLen)
	}

	// 非零
	if pkh.IsZero() {
		t.Error("PubKeyHash should not be zero for valid public key")
	}

	// 确定性
	pkh2 := ComputePubKeyHash(kp.PublicKeyBytes)
	if !pkh.Equal(pkh2) {
		t.Error("PubKeyHash should be deterministic")
	}
}

func TestPubKeyHash_DifferentKeys(t *testing.T) {
	kp1, _ := GenerateMLDSAKeyPair()
	kp2, _ := GenerateMLDSAKeyPair()

	pkh1 := ComputePubKeyHash(kp1.PublicKeyBytes)
	pkh2 := ComputePubKeyHash(kp2.PublicKeyBytes)

	if pkh1.Equal(pkh2) {
		t.Error("different public keys should produce different PubKeyHash")
	}
}

func TestPubKeyHash_EmptyKey(t *testing.T) {
	// 空公钥应仍能计算（不 panic）
	pkh := ComputePubKeyHash(nil)
	if pkh.IsZero() {
		// SHA3-384 对空输入仍应产生非零哈希
		t.Error("PubKeyHash of nil should not be zero (SHA3-384 of empty produces non-zero)")
	}
}

func TestSerializePublicKey_MLDSA(t *testing.T) {
	kp, _ := GenerateMLDSAKeyPair()
	pub := kp.Signer().PublicKey()

	data := SerializePublicKey(pub)
	restored, err := DeserializePublicKey(data)
	if err != nil {
		t.Fatalf("DeserializePublicKey() error = %v", err)
	}

	if len(restored) != len(pub) {
		t.Errorf("restored length = %d, want %d", len(restored), len(pub))
	}
	for i := range pub {
		if restored[i] != pub[i] {
			t.Errorf("restored[%d] = %x, want %x", i, restored[i], pub[i])
			break
		}
	}
}

func TestDeserializePublicKey_Errors(t *testing.T) {
	// 空数据
	_, err := DeserializePublicKey(nil)
	if err == nil {
		t.Error("DeserializePublicKey(nil) should return error")
	}

	// 长度不足
	_, err = DeserializePublicKey([]byte{0x01})
	if err == nil {
		t.Error("DeserializePublicKey() should return error for truncated data")
	}
}
```

**Step 2: 运行测试验证失败**

Run: `go test -v ./pkg/crypto/ -run "TestPubKeyHash|TestSerializePublicKey|TestDeserializePublicKey"`
Expected: FAIL

**Step 3: 编写最小实现**

```go
// pkg/crypto/keys.go
package crypto

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/cxio/evidcoin/pkg/types"
)

// ComputePubKeyHash 计算公钥哈希。
// 使用 SHA3-384 算法，输出 48 字节的公钥哈希。
func ComputePubKeyHash(publicKey []byte) types.PubKeyHash {
	return types.PubKeyHash(SHA3_384Sum(publicKey))
}

// SerializePublicKey 序列化公钥为带长度前缀的字节流。
// 格式：2 字节大端序长度 + 公钥数据。
func SerializePublicKey(publicKey []byte) []byte {
	buf := make([]byte, 2+len(publicKey))
	binary.BigEndian.PutUint16(buf[:2], uint16(len(publicKey)))
	copy(buf[2:], publicKey)
	return buf
}

// DeserializePublicKey 从带长度前缀的字节流反序列化公钥。
func DeserializePublicKey(data []byte) ([]byte, error) {
	if len(data) < 2 {
		return nil, errors.New("data too short for public key deserialization")
	}
	length := int(binary.BigEndian.Uint16(data[:2]))
	if len(data) < 2+length {
		return nil, fmt.Errorf("truncated public key: need %d bytes, have %d", 2+length, len(data))
	}
	key := make([]byte, length)
	copy(key, data[2:2+length])
	return key, nil
}
```

**Step 4: 运行测试验证通过**

Run: `go test -v ./pkg/crypto/ -run "TestPubKeyHash|TestSerializePublicKey|TestDeserializePublicKey"`
Expected: PASS

**Step 5: 提交**

```bash
git add pkg/crypto/keys.go pkg/crypto/keys_test.go
git commit -m "feat(crypto): add PubKeyHash computation and key serialization"
```

---

### Task 9: 多重签名 (pkg/crypto/multisig.go)

**Files:**
- Create: `pkg/crypto/multisig.go`
- Test: `pkg/crypto/multisig_test.go`

**Step 1: 编写失败测试**

```go
// pkg/crypto/multisig_test.go
package crypto

import (
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

func TestMultiSigAddress_Basic(t *testing.T) {
	// 生成 3 个密钥对
	kp1, _ := GenerateMLDSAKeyPair()
	kp2, _ := GenerateMLDSAKeyPair()
	kp3, _ := GenerateMLDSAKeyPair()

	pkh1 := ComputePubKeyHash(kp1.Signer().PublicKey())
	pkh2 := ComputePubKeyHash(kp2.Signer().PublicKey())
	pkh3 := ComputePubKeyHash(kp3.Signer().PublicKey())

	// 2-of-3 多签地址
	addr, err := MultiSigAddress(2, 3, []types.PubKeyHash{pkh1, pkh2, pkh3})
	if err != nil {
		t.Fatalf("MultiSigAddress() error = %v", err)
	}
	if addr.IsZero() {
		t.Error("MultiSigAddress() should not return zero hash")
	}

	// 确定性：相同输入（不同顺序）应产生相同输出
	addr2, _ := MultiSigAddress(2, 3, []types.PubKeyHash{pkh3, pkh1, pkh2})
	if !addr.Equal(addr2) {
		t.Error("MultiSigAddress() should be order-independent (sorted internally)")
	}
}

func TestMultiSigAddress_Errors(t *testing.T) {
	pkh := types.PubKeyHash{}

	tests := []struct {
		name string
		m    int
		n    int
		pkhs []types.PubKeyHash
	}{
		{"m > n", 3, 2, []types.PubKeyHash{pkh, pkh}},
		{"m = 0", 0, 2, []types.PubKeyHash{pkh, pkh}},
		{"n = 0", 1, 0, nil},
		{"pkh count != n", 2, 3, []types.PubKeyHash{pkh, pkh}},
		{"m > 255", 256, 256, make([]types.PubKeyHash, 256)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := MultiSigAddress(tt.m, tt.n, tt.pkhs)
			if err == nil {
				t.Error("MultiSigAddress() should return error")
			}
		})
	}
}

func TestMultiSigAddress_DifferentM(t *testing.T) {
	kp1, _ := GenerateMLDSAKeyPair()
	kp2, _ := GenerateMLDSAKeyPair()
	kp3, _ := GenerateMLDSAKeyPair()

	pkh1 := ComputePubKeyHash(kp1.Signer().PublicKey())
	pkh2 := ComputePubKeyHash(kp2.Signer().PublicKey())
	pkh3 := ComputePubKeyHash(kp3.Signer().PublicKey())

	pkhs := []types.PubKeyHash{pkh1, pkh2, pkh3}

	addr1of3, _ := MultiSigAddress(1, 3, pkhs)
	addr2of3, _ := MultiSigAddress(2, 3, pkhs)
	addr3of3, _ := MultiSigAddress(3, 3, pkhs)

	// 不同 m 应产生不同地址
	if addr1of3.Equal(addr2of3) {
		t.Error("1-of-3 and 2-of-3 addresses should differ")
	}
	if addr2of3.Equal(addr3of3) {
		t.Error("2-of-3 and 3-of-3 addresses should differ")
	}
}

func TestVerifyMultiSig_2of3(t *testing.T) {
	// 生成 3 组混合密钥对
	kp1, _ := GenerateMLDSAKeyPair()
	kp2, _ := GenerateMLDSAKeyPair()
	kp3, _ := GenerateMLDSAKeyPair()

	pub1 := kp1.Signer().PublicKey()
	pub2 := kp2.Signer().PublicKey()
	pub3 := kp3.Signer().PublicKey()

	pkh1 := ComputePubKeyHash(pub1)
	pkh2 := ComputePubKeyHash(pub2)
	pkh3 := ComputePubKeyHash(pub3)

	expectedHash, _ := MultiSigAddress(2, 3, []types.PubKeyHash{pkh1, pkh2, pkh3})

	message := []byte("multisig test message")

	// kp1 和 kp2 签名
	sig1, _ := kp1.Signer().Sign(message)
	sig2, _ := kp2.Signer().Sign(message)

	// 补全项为 kp3 的公钥哈希
	valid, err := VerifyMultiSig(
		message,
		2,
		[][]byte{sig1, sig2},
		[][]byte{pub1, pub2},
		[]types.PubKeyHash{pkh3},
		expectedHash,
	)
	if err != nil {
		t.Fatalf("VerifyMultiSig() error = %v", err)
	}
	if !valid {
		t.Error("VerifyMultiSig() should return true for valid 2-of-3 multisig")
	}
}

func TestVerifyMultiSig_1of1(t *testing.T) {
	kp, _ := GenerateMLDSAKeyPair()
	pub := kp.Signer().PublicKey()
	pkh := ComputePubKeyHash(pub)

	expectedHash, _ := MultiSigAddress(1, 1, []types.PubKeyHash{pkh})

	message := []byte("1-of-1 test")
	sig, _ := kp.Signer().Sign(message)

	valid, err := VerifyMultiSig(
		message,
		1,
		[][]byte{sig},
		[][]byte{pub},
		nil, // 无补全项
		expectedHash,
	)
	if err != nil {
		t.Fatalf("VerifyMultiSig() error = %v", err)
	}
	if !valid {
		t.Error("VerifyMultiSig() should return true for valid 1-of-1 multisig")
	}
}

func TestVerifyMultiSig_FailWrongMessage(t *testing.T) {
	kp1, _ := GenerateMLDSAKeyPair()
	kp2, _ := GenerateMLDSAKeyPair()

	pub1 := kp1.Signer().PublicKey()
	pub2 := kp2.Signer().PublicKey()
	pkh1 := ComputePubKeyHash(pub1)
	pkh2 := ComputePubKeyHash(pub2)

	expectedHash, _ := MultiSigAddress(2, 2, []types.PubKeyHash{pkh1, pkh2})

	sig1, _ := kp1.Signer().Sign([]byte("original"))
	sig2, _ := kp2.Signer().Sign([]byte("original"))

	valid, err := VerifyMultiSig(
		[]byte("tampered"),
		2,
		[][]byte{sig1, sig2},
		[][]byte{pub1, pub2},
		nil,
		expectedHash,
	)
	if err != nil {
		t.Fatalf("VerifyMultiSig() error = %v", err)
	}
	if valid {
		t.Error("VerifyMultiSig() should return false for wrong message")
	}
}

func TestVerifyMultiSig_FailHashMismatch(t *testing.T) {
	kp1, _ := GenerateMLDSAKeyPair()
	kp2, _ := GenerateMLDSAKeyPair()

	pub1 := kp1.Signer().PublicKey()
	pub2 := kp2.Signer().PublicKey()

	message := []byte("test")
	sig1, _ := kp1.Signer().Sign(message)
	sig2, _ := kp2.Signer().Sign(message)

	// 用错误的期望哈希
	wrongHash := types.PubKeyHash{}
	wrongHash[0] = 0xFF

	valid, err := VerifyMultiSig(
		message,
		2,
		[][]byte{sig1, sig2},
		[][]byte{pub1, pub2},
		nil,
		wrongHash,
	)
	if err != nil {
		t.Fatalf("VerifyMultiSig() error = %v", err)
	}
	if valid {
		t.Error("VerifyMultiSig() should return false for hash mismatch")
	}
}

func TestVerifyMultiSig_FailWrongSigner(t *testing.T) {
	kp1, _ := GenerateMLDSAKeyPair()
	kp2, _ := GenerateMLDSAKeyPair()
	kp3, _ := GenerateMLDSAKeyPair() // 入侵者

	pub1 := kp1.Signer().PublicKey()
	pub2 := kp2.Signer().PublicKey()
	pkh1 := ComputePubKeyHash(pub1)
	pkh2 := ComputePubKeyHash(pub2)

	expectedHash, _ := MultiSigAddress(2, 2, []types.PubKeyHash{pkh1, pkh2})

	message := []byte("test")
	sig1, _ := kp1.Signer().Sign(message)
	sig3, _ := kp3.Signer().Sign(message) // 入侵者签名

	// 使用入侵者的公钥代替 kp2
	valid, err := VerifyMultiSig(
		message,
		2,
		[][]byte{sig1, sig3},
		[][]byte{pub1, kp3.Signer().PublicKey()},
		nil,
		expectedHash,
	)
	if err != nil {
		t.Fatalf("VerifyMultiSig() error = %v", err)
	}
	if valid {
		t.Error("VerifyMultiSig() should reject intruder's signature")
	}
}

func TestVerifyMultiSig_Errors(t *testing.T) {
	tests := []struct {
		name string
		m    int
		sigs [][]byte
		pubs [][]byte
		comp []types.PubKeyHash
	}{
		{"m != len(sigs)", 2, [][]byte{nil}, [][]byte{nil}, nil},
		{"m != len(pubs)", 2, [][]byte{nil, nil}, [][]byte{nil}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := VerifyMultiSig([]byte("msg"), tt.m, tt.sigs, tt.pubs, tt.comp, types.PubKeyHash{})
			if err == nil {
				t.Error("VerifyMultiSig() should return error")
			}
		})
	}
}
```

**Step 2: 运行测试验证失败**

Run: `go test -v ./pkg/crypto/ -run "TestMultiSig|TestVerifyMultiSig"`
Expected: FAIL

**Step 3: 编写最小实现**

```go
// pkg/crypto/multisig.go
package crypto

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/cxio/evidcoin/pkg/types"
)

// MultiSigAddress 计算多重签名地址。
// 公式：SHA3-384( [m, N] || sort(PKH_1 || PKH_2 || ... || PKH_N) )
// 公钥哈希按字典序排序，确保相同参与者集合产生相同地址。
func MultiSigAddress(m, n int, pubKeyHashes []types.PubKeyHash) (types.PubKeyHash, error) {
	if n <= 0 || m <= 0 {
		return types.PubKeyHash{}, fmt.Errorf("invalid m/n: m=%d, n=%d (both must be > 0)", m, n)
	}
	if m > n {
		return types.PubKeyHash{}, fmt.Errorf("invalid m/n: m=%d > n=%d", m, n)
	}
	if m > 255 || n > 255 {
		return types.PubKeyHash{}, fmt.Errorf("invalid m/n: m=%d, n=%d (max 255)", m, n)
	}
	if len(pubKeyHashes) != n {
		return types.PubKeyHash{}, fmt.Errorf("pubKeyHashes count %d != n=%d", len(pubKeyHashes), n)
	}

	// 排序公钥哈希（字典序）
	sorted := make([]types.PubKeyHash, n)
	copy(sorted, pubKeyHashes)
	sort.Slice(sorted, func(i, j int) bool {
		return bytes.Compare(sorted[i][:], sorted[j][:]) < 0
	})

	// 构造预像：[m, N] || PKH_1 || PKH_2 || ... || PKH_N
	preimage := make([]byte, 0, 2+n*types.PubKeyHashLen)
	preimage = append(preimage, byte(m), byte(n))
	for _, pkh := range sorted {
		preimage = append(preimage, pkh[:]...)
	}

	return types.PubKeyHash(SHA3_384Sum(preimage)), nil
}

// VerifyMultiSig 验证多重签名。
//
// 验证流程：
// 1. 推导 m, N（m = len(pubKeys)，N = len(pubKeys) + len(complement)）
// 2. 对每个公钥计算哈希
// 3. 与补全哈希合并、排序、前置 [m, N]、计算复合哈希
// 4. 与期望哈希对比
// 5. 逐一验证签名
func VerifyMultiSig(
	message []byte,
	m int,
	sigs [][]byte,
	pubKeys [][]byte,
	complement []types.PubKeyHash,
	expectedHash types.PubKeyHash,
) (bool, error) {
	if m != len(sigs) {
		return false, fmt.Errorf("m=%d != len(sigs)=%d", m, len(sigs))
	}
	if m != len(pubKeys) {
		return false, fmt.Errorf("m=%d != len(pubKeys)=%d", m, len(pubKeys))
	}

	n := len(pubKeys) + len(complement)

	// 步骤 2: 对每个公钥计算哈希
	allHashes := make([]types.PubKeyHash, 0, n)
	for _, pub := range pubKeys {
		pkh := ComputePubKeyHash(pub)
		allHashes = append(allHashes, pkh)
	}
	// 合并补全哈希
	allHashes = append(allHashes, complement...)

	// 步骤 3: 排序、前置 [m, N]、计算复合哈希
	computedHash, err := MultiSigAddress(m, n, allHashes)
	if err != nil {
		return false, fmt.Errorf("compute multi-sig address: %w", err)
	}

	// 步骤 4: 与期望哈希对比
	if !computedHash.Equal(expectedHash) {
		return false, nil
	}

	// 步骤 5: 逐一验证签名
	verifier := NewMLDSAVerifier()
	for i := 0; i < m; i++ {
		valid, err := verifier.Verify(message, sigs[i], pubKeys[i])
		if err != nil {
			return false, fmt.Errorf("verify signature %d: %w", i, err)
		}
		if !valid {
			return false, nil
		}
	}

	return true, nil
}
```

**Step 4: 运行测试验证通过**

Run: `go test -v ./pkg/crypto/ -run "TestMultiSig|TestVerifyMultiSig"`
Expected: PASS

**Step 5: 提交**

```bash
git add pkg/crypto/multisig.go pkg/crypto/multisig_test.go
git commit -m "feat(crypto): implement multi-signature address and verification"
```

---

## 验收标准

### 编译与测试

```bash
# 全量编译
go build ./...

# 全量测试
go test -v ./pkg/types/ ./pkg/crypto/

# 测试覆盖率
go test -cover ./pkg/types/ ./pkg/crypto/
```

### 功能验收

| 编号 | 验收项 | 验证命令 |
|------|--------|----------|
| A1 | `pkg/types` 编译通过 | `go build ./pkg/types/` |
| A2 | `pkg/crypto` 编译通过 | `go build ./pkg/crypto/` |
| A3 | 全部测试通过 | `go test ./pkg/types/ ./pkg/crypto/` |
| A4 | 核心测试覆盖率 ≥ 80% | `go test -cover ./pkg/types/ ./pkg/crypto/` |
| A5 | 代码格式正确 | `gofmt -l ./pkg/types/ ./pkg/crypto/`（无输出） |
| A6 | 无循环依赖 | `go vet ./pkg/types/ ./pkg/crypto/` |

### 类型验收

| 编号 | 类型 | 包 | 文件 |
|------|------|-----|------|
| T1 | `Hash512 [64]byte` | `types` | `hash.go` |
| T2 | `Hash384 [48]byte` | `types` | `hash.go` |
| T3 | `PubKeyHash [48]byte` | `types` | `hash.go` |
| T4 | `OutputConfig byte` | `types` | `config.go` |
| T5 | `SigFlag byte` | `types` | `config.go` |
| T6 | `Address` struct | `types` | `address.go` |

### 函数验收

| 编号 | 函数 | 包 | 文件 |
|------|------|-----|------|
| F1 | `EncodeVarint / DecodeVarint` | `types` | `varint.go` |
| F2 | `NewAddress / DecodeAddress` | `types` | `address.go` |
| F3 | `SHA512Sum` | `crypto` | `hash.go` |
| F4 | `SHA3_384Sum / SHA3_512Sum` | `crypto` | `hash.go` |
| F5 | `Blake3Sum` | `crypto` | `hash.go` |
| F6 | `GenerateMLDSAKeyPair` + Signer/Verifier | `crypto` | `mldsa.go` |
| F7 | `ComputePubKeyHash` | `crypto` | `keys.go` |
| F8 | `MultiSigAddress / VerifyMultiSig` | `crypto` | `multisig.go` |

### 接口验收

| 编号 | 接口 | 实现者 |
|------|------|--------|
| I1 | `Signer` | `MLDSASigner` |
| I2 | `Verifier` | `MLDSAVerifier` |

### 依赖验收

| 编号 | 检查项 |
|------|--------|
| D1 | `pkg/types` 零外部依赖（除 `github.com/mr-tron/base58` 用于地址编码） |
| D2 | `pkg/crypto` 仅依赖 `pkg/types` 和外部密码学库 |
| D3 | 无 `internal/` 依赖 |
| D4 | `go mod tidy` 无变化 |
