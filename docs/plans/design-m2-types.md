# M2: types 模块设计

> **模块路径:** `pkg/types`
> **依赖:** `pkg/crypto`
> **预估工时:** 2 天

## 概述

基础类型定义模块，提供区块链核心数据类型和常量定义。作为公共包，可被外部项目引用。

## 功能清单

| 功能 | 文件 | 说明 |
|------|------|------|
| 哈希类型 | `hash.go` | Hash512, Hash256 固定长度哈希 |
| 地址类型 | `address.go` | Address 48字节地址 |
| 变长整数 | `varint.go` | VarInt 编解码 |
| 金额类型 | `amount.go` | Amount 类型及单位转换 |
| 常量定义 | `constants.go` | 系统常量 |
| 时间相关 | `time.go` | 区块时间戳计算 |

---

## 详细设计

### 1. constants.go - 系统常量

```go
package types

import "time"

const (
    // === 哈希长度 ===
    HashLength      = 64 // SHA-512 哈希长度（字节）
    AddressLength   = 48 // 地址长度（字节）
    ShortHashLength = 20 // 短哈希长度（用于非首领输入引用）

    // === 时间参数 ===
    BlockInterval     = 6 * time.Minute // 出块间隔
    BlocksPerYear     = 87661           // 每年区块数（恒星年 365.25636 日）
    BlocksPerMonth    = 7305            // 每月区块数（近似）
    GenesisTimestamp  = 0               // 创世区块时间戳（待设定）

    // === 脚本限制 ===
    MaxStackHeight   = 256  // 脚本栈最大高度
    MaxStackItemSize = 1024 // 栈数据项最大尺寸（字节）
    MaxLockScript    = 1024 // 锁定脚本最大长度
    MaxUnlockScript  = 4096 // 解锁脚本最大长度

    // === 交易限制 ===
    MaxTxSize        = 8192 // 单笔交易最大尺寸（字节，不含解锁）
    MaxOutputCount   = 1024 // 最大输出项数量
    MaxInputCount    = 256  // 最大输入项数量
    TxExpireBlocks   = 240  // 交易过期区块数（24小时）
    MaxMemoLength    = 255  // 附言最大长度

    // === 凭信限制 ===
    MaxCredentialTitle   = 255  // 凭信标题最大长度
    MaxCredentialContent = 1024 // 凭信描述最大长度

    // === 存证限制 ===
    MaxEvidenceTitle   = 255  // 存证标题最大长度
    MaxEvidenceContent = 2048 // 存证内容最大长度

    // === 共识参数 ===
    EvalBlockOffset     = 9     // 评参区块偏移（-9号区块）
    UTXOFingerprintOffset = 24  // UTXO 指纹区块偏移（-24号区块）
    MintTxMinHeight     = 25    // 铸凭交易最小高度偏移
    MintTxMaxHeight     = 80000 // 铸凭交易最大高度偏移（约11个月）
    PreferencePoolSize  = 20    // 择优池容量
    SyncAuthCount       = 15    // 有权同步的后段成员数
    ForkCompeteBlocks   = 25    // 分叉竞争区块数
    CoinbaseConfirms    = 25    // 新币确认数

    // === 铸造冗余 ===
    MintDelayFirst = 30 * time.Second // 首个区块延迟发布
    MintDelayStep  = 15 * time.Second // 候选者间隔

    // === 初段规则 ===
    InitialPhaseBlocks = 9 // 初段区块数（无评参区块）

    // === 组队校验 ===
    RedundancyFactor = 2  // 冗余校验系数
    ReviewBlocks     = 48 // 兑奖评估区块数
    MinConfirms      = 2  // 最低确认数
    BlacklistDuration = 24 * time.Hour // 黑名单冻结时间

    // === 激励参数（百分比）===
    MinterShare   = 50 // 铸造者分成（%），其中校验组40%，铸凭者10%
    TeamShare     = 40 // 校验组分成（%）
    MintOwnerShare = 10 // 铸凭者分成（%）
    DepotsShare   = 20 // depots 分成（%）
    BlockqsShare  = 20 // blockqs 分成（%）
    Stun2pShare   = 10 // stun2p 分成（%）
    TxFeeBurn     = 50 // 交易费销毁比例（%）

    // === 附件参数 ===
    MaxAttachmentSliceSize = 2 * 1024 * 1024 // 分片最大尺寸（2MB）
    MinAttachmentFPLength  = 16              // 附件指纹最小长度
    MaxAttachmentFPLength  = 64              // 附件指纹最大长度

    // === EMBED/GOTO 限制 ===
    MaxEmbedDepth = 5 // EMBED 嵌入最大次数
    MaxGotoDepth  = 3 // GOTO 跳转最大深度
    MaxGotoCount  = 2 // 主脚本中 GOTO 最大次数

    // === 交易费 ===
    MinFeeCalcBlocks = 6000 // 最低交易费统计区块数（约25天）
    MinFeeDivisor    = 4    // 最低交易费除数（取平均值的1/4）
)
```

### 2. hash.go - 哈希类型

```go
package types

import (
    "encoding/hex"
    "errors"
)

var (
    ErrInvalidHashLength = errors.New("invalid hash length")
)

// Hash512 64字节哈希类型
type Hash512 [HashLength]byte

// EmptyHash512 空哈希
var EmptyHash512 Hash512

// NewHash512 从字节切片创建 Hash512
func NewHash512(data []byte) (Hash512, error) {
    var h Hash512
    if len(data) != HashLength {
        return h, ErrInvalidHashLength
    }
    copy(h[:], data)
    return h, nil
}

// MustHash512 从字节切片创建 Hash512，失败时 panic
func MustHash512(data []byte) Hash512 {
    h, err := NewHash512(data)
    if err != nil {
        panic(err)
    }
    return h
}

// Bytes 返回字节切片
func (h Hash512) Bytes() []byte {
    return h[:]
}

// String 返回十六进制字符串
func (h Hash512) String() string {
    return hex.EncodeToString(h[:])
}

// Short 返回短格式字符串（前16字符...后8字符）
func (h Hash512) Short() string {
    s := h.String()
    if len(s) <= 24 {
        return s
    }
    return s[:16] + "..." + s[len(s)-8:]
}

// IsEmpty 检查是否为空哈希
func (h Hash512) IsEmpty() bool {
    return h == EmptyHash512
}

// Equal 比较两个哈希是否相等
func (h Hash512) Equal(other Hash512) bool {
    return h == other
}

// Compare 比较两个哈希（用于排序）
// 返回: -1 (h < other), 0 (h == other), 1 (h > other)
func (h Hash512) Compare(other Hash512) int {
    for i := 0; i < HashLength; i++ {
        if h[i] < other[i] {
            return -1
        }
        if h[i] > other[i] {
            return 1
        }
    }
    return 0
}

// Hash256 32字节哈希类型（用于某些场景）
type Hash256 [32]byte

// EmptyHash256 空哈希
var EmptyHash256 Hash256

// NewHash256 从字节切片创建 Hash256
func NewHash256(data []byte) (Hash256, error) {
    var h Hash256
    if len(data) != 32 {
        return h, ErrInvalidHashLength
    }
    copy(h[:], data)
    return h, nil
}

func (h Hash256) Bytes() []byte {
    return h[:]
}

func (h Hash256) String() string {
    return hex.EncodeToString(h[:])
}

// ShortHash 20字节短哈希类型（用于非首领输入引用）
type ShortHash [ShortHashLength]byte

func NewShortHash(data []byte) (ShortHash, error) {
    var h ShortHash
    if len(data) != ShortHashLength {
        return h, ErrInvalidHashLength
    }
    copy(h[:], data)
    return h, nil
}

func (h ShortHash) Bytes() []byte {
    return h[:]
}

func (h ShortHash) String() string {
    return hex.EncodeToString(h[:])
}
```

### 3. address.go - 地址类型

```go
package types

import (
    "encoding/hex"
    "errors"
)

var (
    ErrInvalidAddressLength = errors.New("invalid address length")
)

// Address 48字节地址类型
type Address [AddressLength]byte

// NullAddress 空地址（用于销毁）
var NullAddress Address

// NewAddress 从字节切片创建 Address
func NewAddress(data []byte) (Address, error) {
    var a Address
    if len(data) != AddressLength {
        return a, ErrInvalidAddressLength
    }
    copy(a[:], data)
    return a, nil
}

// MustAddress 从字节切片创建 Address，失败时 panic
func MustAddress(data []byte) Address {
    a, err := NewAddress(data)
    if err != nil {
        panic(err)
    }
    return a
}

// Bytes 返回字节切片
func (a Address) Bytes() []byte {
    return a[:]
}

// String 返回十六进制字符串
func (a Address) String() string {
    return hex.EncodeToString(a[:])
}

// Short 返回短格式字符串
func (a Address) Short() string {
    s := a.String()
    if len(s) <= 20 {
        return s
    }
    return s[:12] + "..." + s[len(s)-8:]
}

// IsNull 检查是否为空地址（销毁地址）
func (a Address) IsNull() bool {
    return a == NullAddress
}

// Equal 比较两个地址是否相等
func (a Address) Equal(other Address) bool {
    return a == other
}
```

### 4. varint.go - 变长整数

```go
package types

import (
    "encoding/binary"
    "errors"
    "io"
)

var (
    ErrVarIntOverflow = errors.New("varint overflow")
    ErrVarIntTooLong  = errors.New("varint too long")
)

// VarInt 变长整数类型
type VarInt int64

// Encode 编码变长整数到字节切片
func (v VarInt) Encode() []byte {
    var buf [binary.MaxVarintLen64]byte
    n := binary.PutVarint(buf[:], int64(v))
    return buf[:n]
}

// Size 返回编码后的字节长度
func (v VarInt) Size() int {
    var buf [binary.MaxVarintLen64]byte
    return binary.PutVarint(buf[:], int64(v))
}

// DecodeVarInt 从字节切片解码变长整数
func DecodeVarInt(data []byte) (VarInt, int, error) {
    val, n := binary.Varint(data)
    if n == 0 {
        return 0, 0, io.ErrUnexpectedEOF
    }
    if n < 0 {
        return 0, 0, ErrVarIntOverflow
    }
    return VarInt(val), n, nil
}

// ReadVarInt 从 Reader 读取变长整数
func ReadVarInt(r io.ByteReader) (VarInt, error) {
    val, err := binary.ReadVarint(r)
    if err != nil {
        return 0, err
    }
    return VarInt(val), nil
}

// UVarInt 无符号变长整数类型
type UVarInt uint64

// Encode 编码无符号变长整数到字节切片
func (v UVarInt) Encode() []byte {
    var buf [binary.MaxVarintLen64]byte
    n := binary.PutUvarint(buf[:], uint64(v))
    return buf[:n]
}

// Size 返回编码后的字节长度
func (v UVarInt) Size() int {
    var buf [binary.MaxVarintLen64]byte
    return binary.PutUvarint(buf[:], uint64(v))
}

// DecodeUVarInt 从字节切片解码无符号变长整数
func DecodeUVarInt(data []byte) (UVarInt, int, error) {
    val, n := binary.Uvarint(data)
    if n == 0 {
        return 0, 0, io.ErrUnexpectedEOF
    }
    if n < 0 {
        return 0, 0, ErrVarIntOverflow
    }
    return UVarInt(val), n, nil
}

// ReadUVarInt 从 Reader 读取无符号变长整数
func ReadUVarInt(r io.ByteReader) (UVarInt, error) {
    val, err := binary.ReadUvarint(r)
    if err != nil {
        return 0, err
    }
    return UVarInt(val), nil
}
```

### 5. amount.go - 金额类型

```go
package types

import (
    "errors"
    "fmt"
    "math"
)

const (
    // 货币单位
    // 1 Bi = 1,000,000 Chx
    ChxPerBi = 1_000_000

    // 最大金额（防止溢出）
    MaxAmount = math.MaxInt64
)

var (
    ErrAmountOverflow  = errors.New("amount overflow")
    ErrAmountNegative  = errors.New("amount cannot be negative")
    ErrAmountPrecision = errors.New("amount precision loss")
)

// Amount 金额类型（以 Chx 为单位）
// Chx 是最小货币单位，1 Bi = 1,000,000 Chx
type Amount int64

// NewAmount 创建金额
func NewAmount(chx int64) (Amount, error) {
    if chx < 0 {
        return 0, ErrAmountNegative
    }
    return Amount(chx), nil
}

// FromBi 从 Bi 单位创建金额
func FromBi(bi float64) (Amount, error) {
    if bi < 0 {
        return 0, ErrAmountNegative
    }
    chx := bi * ChxPerBi
    if chx > float64(MaxAmount) {
        return 0, ErrAmountOverflow
    }
    return Amount(int64(chx)), nil
}

// ToBi 转换为 Bi 单位
func (a Amount) ToBi() float64 {
    return float64(a) / ChxPerBi
}

// ToChx 返回 Chx 值
func (a Amount) ToChx() int64 {
    return int64(a)
}

// String 返回格式化字符串
func (a Amount) String() string {
    bi := a.ToBi()
    if bi == float64(int64(bi)) {
        return fmt.Sprintf("%d Bi", int64(bi))
    }
    return fmt.Sprintf("%.6f Bi", bi)
}

// Add 加法
func (a Amount) Add(b Amount) (Amount, error) {
    result := int64(a) + int64(b)
    if result < 0 || (int64(a) > 0 && int64(b) > 0 && result < int64(a)) {
        return 0, ErrAmountOverflow
    }
    return Amount(result), nil
}

// Sub 减法
func (a Amount) Sub(b Amount) (Amount, error) {
    if b > a {
        return 0, ErrAmountNegative
    }
    return Amount(int64(a) - int64(b)), nil
}

// Mul 乘法
func (a Amount) Mul(n int64) (Amount, error) {
    if n < 0 {
        return 0, ErrAmountNegative
    }
    result := int64(a) * n
    if n != 0 && result/n != int64(a) {
        return 0, ErrAmountOverflow
    }
    return Amount(result), nil
}

// Div 除法
func (a Amount) Div(n int64) Amount {
    if n == 0 {
        panic("division by zero")
    }
    return Amount(int64(a) / n)
}

// Percent 计算百分比
func (a Amount) Percent(p int) Amount {
    return Amount(int64(a) * int64(p) / 100)
}

// IsZero 检查是否为零
func (a Amount) IsZero() bool {
    return a == 0
}

// Compare 比较
func (a Amount) Compare(b Amount) int {
    if a < b {
        return -1
    }
    if a > b {
        return 1
    }
    return 0
}
```

### 6. time.go - 时间相关

```go
package types

import (
    "time"
)

// BlockHeight 区块高度类型
type BlockHeight uint64

// Year 年度类型（从创世年开始）
type Year uint16

// Timestamp 时间戳类型（毫秒）
type Timestamp int64

// Now 返回当前时间戳（毫秒）
func Now() Timestamp {
    return Timestamp(time.Now().UnixMilli())
}

// FromTime 从 time.Time 创建时间戳
func FromTime(t time.Time) Timestamp {
    return Timestamp(t.UnixMilli())
}

// ToTime 转换为 time.Time
func (t Timestamp) ToTime() time.Time {
    return time.UnixMilli(int64(t))
}

// BlockTimestamp 计算区块时间戳
// 每个区块的时间戳由创世块时间戳和区块高度计算
func BlockTimestamp(genesisTime Timestamp, height BlockHeight) Timestamp {
    return genesisTime + Timestamp(int64(height)*int64(BlockInterval/time.Millisecond))
}

// HeightAtTime 计算给定时间对应的区块高度
func HeightAtTime(genesisTime Timestamp, t Timestamp) BlockHeight {
    if t < genesisTime {
        return 0
    }
    elapsed := int64(t - genesisTime)
    intervalMs := int64(BlockInterval / time.Millisecond)
    return BlockHeight(elapsed / intervalMs)
}

// YearFromTimestamp 从时间戳计算年度
func YearFromTimestamp(t Timestamp) Year {
    tm := t.ToTime()
    return Year(tm.Year())
}

// YearFromHeight 从区块高度计算年度（链内年度）
func YearFromHeight(height BlockHeight) Year {
    if height == 0 {
        return 0
    }
    return Year((height - 1) / BlocksPerYear)
}

// IsYearBlock 检查是否为年块（每年第一个区块）
func IsYearBlock(height BlockHeight) bool {
    if height == 0 {
        return true
    }
    return height%BlocksPerYear == 0
}

// CoinDays 币权（币天）计算
// 金额 * 持有天数
type CoinDays int64

// CalcCoinDays 计算币权
func CalcCoinDays(amount Amount, createdHeight, currentHeight BlockHeight) CoinDays {
    if currentHeight <= createdHeight {
        return 0
    }
    blocks := currentHeight - createdHeight
    days := float64(blocks) / float64(BlocksPerYear) * 365.25636
    return CoinDays(float64(amount) * days)
}
```

---

## 测试用例

### types_test.go

```go
package types

import (
    "bytes"
    "testing"
)

func TestHash512(t *testing.T) {
    data := make([]byte, HashLength)
    for i := range data {
        data[i] = byte(i)
    }

    h, err := NewHash512(data)
    if err != nil {
        t.Fatalf("failed to create Hash512: %v", err)
    }

    if !bytes.Equal(h.Bytes(), data) {
        t.Error("hash bytes mismatch")
    }

    // 测试空哈希
    if !EmptyHash512.IsEmpty() {
        t.Error("EmptyHash512 should be empty")
    }
    if h.IsEmpty() {
        t.Error("non-zero hash should not be empty")
    }
}

func TestVarInt(t *testing.T) {
    tests := []int64{0, 1, 127, 128, 255, 256, 65535, 1<<20, -1, -128, -129}

    for _, val := range tests {
        v := VarInt(val)
        encoded := v.Encode()

        decoded, n, err := DecodeVarInt(encoded)
        if err != nil {
            t.Errorf("failed to decode %d: %v", val, err)
            continue
        }

        if int(decoded) != int(val) {
            t.Errorf("expected %d, got %d", val, decoded)
        }

        if n != len(encoded) {
            t.Errorf("size mismatch: encoded %d bytes, decoded %d", len(encoded), n)
        }
    }
}

func TestAmount(t *testing.T) {
    // 测试从 Bi 创建
    a, err := FromBi(1.5)
    if err != nil {
        t.Fatalf("failed to create amount: %v", err)
    }

    if a.ToChx() != 1_500_000 {
        t.Errorf("expected 1500000 Chx, got %d", a.ToChx())
    }

    // 测试加法
    b, _ := FromBi(0.5)
    sum, err := a.Add(b)
    if err != nil {
        t.Fatalf("failed to add: %v", err)
    }

    if sum.ToBi() != 2.0 {
        t.Errorf("expected 2.0 Bi, got %f", sum.ToBi())
    }

    // 测试百分比
    half := sum.Percent(50)
    if half.ToBi() != 1.0 {
        t.Errorf("expected 1.0 Bi, got %f", half.ToBi())
    }
}

func TestBlockTimestamp(t *testing.T) {
    genesis := Timestamp(0)

    // 第0块
    ts0 := BlockTimestamp(genesis, 0)
    if ts0 != genesis {
        t.Errorf("block 0 timestamp should be genesis")
    }

    // 第1块（6分钟后）
    ts1 := BlockTimestamp(genesis, 1)
    expected := genesis + Timestamp(6*60*1000)
    if ts1 != expected {
        t.Errorf("expected %d, got %d", expected, ts1)
    }
}

func TestYearFromHeight(t *testing.T) {
    tests := []struct {
        height BlockHeight
        year   Year
    }{
        {0, 0},
        {1, 0},
        {87660, 0},
        {87661, 1},
        {87662, 1},
        {175322, 2},
    }

    for _, tt := range tests {
        got := YearFromHeight(tt.height)
        if got != tt.year {
            t.Errorf("height %d: expected year %d, got %d", tt.height, tt.year, got)
        }
    }
}
```

---

## 实现步骤

### Step 1: 创建包结构

```bash
mkdir -p pkg/types
touch pkg/types/constants.go
touch pkg/types/hash.go
touch pkg/types/address.go
touch pkg/types/varint.go
touch pkg/types/amount.go
touch pkg/types/time.go
touch pkg/types/types_test.go
```

### Step 2: 按顺序实现

1. `constants.go` - 常量定义（无依赖）
2. `hash.go` - 哈希类型
3. `address.go` - 地址类型
4. `varint.go` - 变长整数
5. `amount.go` - 金额类型
6. `time.go` - 时间相关

### Step 3: 测试验证

```bash
go test -v ./pkg/types/...
```

---

## 注意事项

1. **精度**: Amount 使用 int64，避免浮点数精度问题
2. **溢出**: 所有算术操作需检查溢出
3. **时间**: 时间戳使用毫秒精度，与设计文档一致
4. **编码**: VarInt 使用 Go 标准库的 varint 编码
