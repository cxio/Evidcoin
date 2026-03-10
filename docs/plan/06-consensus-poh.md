# Phase 6：PoH 共识机制 实现计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现历史证明（PoH）共识机制——包括铸凭哈希计算、择优池管理、铸造时间表（铸币计划）、分叉检测与解决、引导启动规则、币权（币天）计算、以及端点约定。

**Architecture:** `internal/consensus` 包，依赖 `pkg/types`、`pkg/crypto` 和 `internal/blockchain`。共识模块不直接操作区块链，而是提供铸造竞争、分叉判断和奖励计算的独立逻辑，由上层应用组合调用。

**Tech Stack:** Go 1.25+, pkg/types (Hash512, constants), pkg/crypto (SHA512Sum, Sign, Verify), internal/blockchain (BlockHeader, BlockTime)

---

## 前置依赖

本 Phase 假设 Phase 1 和 Phase 2 已完成，以下类型和函数已可用：

```go
// pkg/types
type Hash512 [64]byte              // SHA-512（64 字节）
func (h Hash512) IsZero() bool     // 判断是否全零
func (h Hash512) String() string   // 十六进制字符串
const HashLen = 64                 // 哈希长度（字节）
const BlocksPerYear = 87661        // 每年区块数
const ForkWindowSize = 25          // 分叉竞争窗口（区块数）
const BestPoolCapacity = 20        // 择优池容量

// pkg/crypto
func SHA512Sum(data []byte) types.Hash512          // SHA-512 哈希
func Sign(privateKey PrivateKey, data []byte) []byte  // 私钥签名
func Verify(publicKey PublicKey, data, signature []byte) bool // 公钥验签
type PrivateKey                                     // 私钥类型
type PublicKey                                      // 公钥类型
func GenerateKeyPair() (PublicKey, PrivateKey, error) // 生成密钥对

// internal/blockchain
func BlockTime(height int32) int64  // 按高度计算区块时间戳（Unix 毫秒）
```

> **注意：** 如果前置 Phase 的具体 API 与以上描述有差异，请在实现时以 `pkg/types`、`pkg/crypto` 和 `internal/blockchain` 的实际导出符号为准，适当调整本计划中的调用方式。

---

## Task 1: 铸凭哈希计算 (internal/consensus/minting.go)

**Files:**
- Create: `internal/consensus/minting.go`
- Test: `internal/consensus/minting_test.go`

本 Task 实现铸凭哈希计算的核心算法，包括 `MintingHash` 函数、`CompareMintHash` 比较函数、`encodeInt64` 大端编码辅助、以及 `IsMintTxEligible` 铸凭交易合法性判断。

### Step 1: 写失败测试

创建 `internal/consensus/minting_test.go`：

```go
package consensus

import (
	"bytes"
	"testing"

	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// --- encodeInt64 测试 ---

func TestEncodeInt64(t *testing.T) {
	tests := []struct {
		name string
		val  int64
		want []byte
	}{
		{
			name: "zero",
			val:  0,
			want: []byte{0, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			name: "one",
			val:  1,
			want: []byte{0, 0, 0, 0, 0, 0, 0, 1},
		},
		{
			name: "max int64",
			val:  0x7FFFFFFFFFFFFFFF,
			want: []byte{0x7F, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
		},
		{
			name: "negative one",
			val:  -1,
			want: []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
		},
		{
			name: "specific value",
			val:  0x0102030405060708,
			want: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := encodeInt64(tt.val)
			if !bytes.Equal(got, tt.want) {
				t.Errorf("encodeInt64(%d) = %x, want %x", tt.val, got, tt.want)
			}
		})
	}
}

// 验证 encodeInt64 始终返回 8 字节
func TestEncodeInt64_Length(t *testing.T) {
	values := []int64{0, 1, -1, 12345678, 0x7FFFFFFFFFFFFFFF}
	for _, v := range values {
		got := encodeInt64(v)
		if len(got) != 8 {
			t.Errorf("encodeInt64(%d) length = %d, want 8", v, len(got))
		}
	}
}

// --- MintingHash 测试 ---

// 测试相同输入产生相同哈希
func TestMintingHash_Deterministic(t *testing.T) {
	pub, priv, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error: %v", err)
	}
	_ = pub

	txID := types.Hash512{0xAA, 0xBB, 0xCC}
	refMintHash := types.Hash512{0x11, 0x22, 0x33}
	var stakes int64 = 100000
	var blockTimestamp int64 = 1780317296789

	h1 := MintingHash(txID, refMintHash, stakes, blockTimestamp, priv)
	h2 := MintingHash(txID, refMintHash, stakes, blockTimestamp, priv)

	if h1 != h2 {
		t.Error("MintingHash() should be deterministic for identical inputs")
	}
	if h1.IsZero() {
		t.Error("MintingHash() should not return zero hash")
	}
}

// 测试不同交易 ID 产生不同哈希
func TestMintingHash_DifferentTxID(t *testing.T) {
	_, priv, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error: %v", err)
	}

	txID1 := types.Hash512{0x01}
	txID2 := types.Hash512{0x02}
	refMintHash := types.Hash512{0x11}
	var stakes int64 = 50000
	var blockTimestamp int64 = 1780317296789

	h1 := MintingHash(txID1, refMintHash, stakes, blockTimestamp, priv)
	h2 := MintingHash(txID2, refMintHash, stakes, blockTimestamp, priv)

	if h1 == h2 {
		t.Error("different txID should produce different MintingHash")
	}
}

// 测试不同参考铸凭哈希产生不同结果
func TestMintingHash_DifferentRefMintHash(t *testing.T) {
	_, priv, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error: %v", err)
	}

	txID := types.Hash512{0xAA}
	ref1 := types.Hash512{0x11}
	ref2 := types.Hash512{0x22}
	var stakes int64 = 50000
	var blockTimestamp int64 = 1780317296789

	h1 := MintingHash(txID, ref1, stakes, blockTimestamp, priv)
	h2 := MintingHash(txID, ref2, stakes, blockTimestamp, priv)

	if h1 == h2 {
		t.Error("different refMintHash should produce different MintingHash")
	}
}

// 测试不同私钥产生不同哈希（签名致盲）
func TestMintingHash_DifferentPrivateKey(t *testing.T) {
	_, priv1, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error: %v", err)
	}
	_, priv2, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error: %v", err)
	}

	txID := types.Hash512{0xAA}
	refMintHash := types.Hash512{0x11}
	var stakes int64 = 50000
	var blockTimestamp int64 = 1780317296789

	h1 := MintingHash(txID, refMintHash, stakes, blockTimestamp, priv1)
	h2 := MintingHash(txID, refMintHash, stakes, blockTimestamp, priv2)

	if h1 == h2 {
		t.Error("different private keys should produce different MintingHash (signing blindness)")
	}
}

// 测试不同时间戳产生不同哈希
func TestMintingHash_DifferentTimestamp(t *testing.T) {
	_, priv, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error: %v", err)
	}

	txID := types.Hash512{0xAA}
	refMintHash := types.Hash512{0x11}
	var stakes int64 = 50000

	h1 := MintingHash(txID, refMintHash, stakes, 1780317296789, priv)
	h2 := MintingHash(txID, refMintHash, stakes, 1780317296790, priv)

	if h1 == h2 {
		t.Error("different blockTimestamp should produce different MintingHash")
	}
}

// 测试不同币权产生不同哈希
func TestMintingHash_DifferentStakes(t *testing.T) {
	_, priv, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error: %v", err)
	}

	txID := types.Hash512{0xAA}
	refMintHash := types.Hash512{0x11}
	var blockTimestamp int64 = 1780317296789

	h1 := MintingHash(txID, refMintHash, 100, blockTimestamp, priv)
	h2 := MintingHash(txID, refMintHash, 200, blockTimestamp, priv)

	if h1 == h2 {
		t.Error("different stakes should produce different MintingHash")
	}
}

// --- CompareMintHash 测试 ---

func TestCompareMintHash(t *testing.T) {
	tests := []struct {
		name string
		a, b types.Hash512
		want int
	}{
		{
			name: "equal hashes",
			a:    types.Hash512{0x01, 0x02, 0x03},
			b:    types.Hash512{0x01, 0x02, 0x03},
			want: 0,
		},
		{
			name: "a less than b at first byte",
			a:    types.Hash512{0x01},
			b:    types.Hash512{0x02},
			want: -1,
		},
		{
			name: "a greater than b at first byte",
			a:    types.Hash512{0xFF},
			b:    types.Hash512{0x01},
			want: 1,
		},
		{
			name: "differ at second byte",
			a:    types.Hash512{0x01, 0x01},
			b:    types.Hash512{0x01, 0x02},
			want: -1,
		},
		{
			name: "differ at last byte",
			a: func() types.Hash512 {
				var h types.Hash512
				h[63] = 0x01
				return h
			}(),
			b: func() types.Hash512 {
				var h types.Hash512
				h[63] = 0x02
				return h
			}(),
			want: -1,
		},
		{
			name: "both zero",
			a:    types.Hash512{},
			b:    types.Hash512{},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CompareMintHash(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("CompareMintHash() = %d, want %d", got, tt.want)
			}
		})
	}
}

// 测试比较的对称性
func TestCompareMintHash_Symmetry(t *testing.T) {
	a := types.Hash512{0x01, 0x02}
	b := types.Hash512{0x03, 0x04}

	ab := CompareMintHash(a, b)
	ba := CompareMintHash(b, a)

	if ab == 0 || ba == 0 {
		t.Fatal("test hashes should not be equal")
	}
	if ab+ba != 0 {
		t.Errorf("CompareMintHash symmetry broken: %d + %d != 0", ab, ba)
	}
}

// --- IsMintTxEligible 测试 ---

func TestIsMintTxEligible(t *testing.T) {
	tests := []struct {
		name          string
		currentHeight int
		txHeight      int
		want          bool
	}{
		// 初段：currentHeight < 28，所有合法交易均可参与
		{
			name:          "early stage height=0",
			currentHeight: 0,
			txHeight:      0,
			want:          true,
		},
		{
			name:          "early stage height=1",
			currentHeight: 1,
			txHeight:      0,
			want:          true,
		},
		{
			name:          "early stage height=27",
			currentHeight: 27,
			txHeight:      0,
			want:          true,
		},
		{
			name:          "early stage height=27 tx at 26",
			currentHeight: 27,
			txHeight:      26,
			want:          true,
		},
		// 正常阶段：currentHeight >= 28
		// depth = currentHeight - txHeight，需满足 depth > 27 && depth <= 80000
		{
			name:          "boundary: height=28, tx at 1, depth=27 (not eligible)",
			currentHeight: 28,
			txHeight:      1,
			want:          false, // depth=27，需要 > 27
		},
		{
			name:          "boundary: height=28, tx at 1, depth=27 (not eligible)",
			currentHeight: 28,
			txHeight:      1,
			want:          false, // depth=27，需要 > 27
		},
		{
			name:          "boundary: height=28, tx at 0, depth=28 (eligible)",
			currentHeight: 28,
			txHeight:      0,
			want:          true, // depth=28 > 27 && <= 80000
		},
		{
			name:          "boundary: height=29, tx at 1, depth=28 (eligible)",
			currentHeight: 29,
			txHeight:      1,
			want:          true,
		},
		{
			name:          "normal eligible",
			currentHeight: 1000,
			txHeight:      500,
			want:          true, // depth=500
		},
		{
			name:          "depth exactly 80000 (eligible)",
			currentHeight: 80028,
			txHeight:      28,
			want:          true, // depth=80000
		},
		{
			name:          "depth 80001 (not eligible)",
			currentHeight: 80029,
			txHeight:      28,
			want:          false, // depth=80001 > 80000
		},
		{
			name:          "same height (not eligible in normal stage)",
			currentHeight: 100,
			txHeight:      100,
			want:          false, // depth=0
		},
		{
			name:          "depth=1 (not eligible)",
			currentHeight: 100,
			txHeight:      99,
			want:          false,
		},
		{
			name:          "depth=27 boundary",
			currentHeight: 100,
			txHeight:      73,
			want:          false, // depth=27，需 > 27
		},
		{
			name:          "depth=28 just eligible",
			currentHeight: 100,
			txHeight:      72,
			want:          true, // depth=28
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsMintTxEligible(tt.currentHeight, tt.txHeight)
			if got != tt.want {
				t.Errorf("IsMintTxEligible(%d, %d) = %v, want %v",
					tt.currentHeight, tt.txHeight, got, tt.want)
			}
		})
	}
}
```

### Step 2: 运行测试验证失败

```bash
go test -v ./internal/consensus/ -run "TestEncodeInt64|TestMintingHash|TestCompareMintHash|TestIsMintTxEligible"
```

预期输出：编译失败，`encodeInt64`、`MintingHash`、`CompareMintHash`、`IsMintTxEligible` 等未定义。

### Step 3: 写最小实现

创建 `internal/consensus/minting.go`：

```go
package consensus

import (
	"bytes"
	"encoding/binary"

	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// Mix 混合常数：增加 Bit 位复杂性，降低可塑性。
const Mix int64 = 0x517cc1b727220a95

// MintTxMaxDepth 铸凭交易最大回溯深度（约 11 个月）。
const MintTxMaxDepth = 80000

// MintTxMinDepth 铸凭交易最小深度（排除尾部区块，防塑造）。
const MintTxMinDepth = 28

// MintingHash 计算铸凭哈希。
// 算法共四个阶段：
//   1. 构造动态因子 X = encodeInt64(blockTimestamp * stakes * Mix)
//   2. source = concat(txID[:], refMintHash[:], X)，hashData = SHA512(source)
//   3. signData = Sign(privateKey, hashData)
//   4. return SHA512(signData)
//
// 参数说明：
//   txID:           铸凭交易 ID
//   refMintHash:    评参区块（Height-9）的铸凭哈希
//   stakes:         第 Height-27 区块的币权总值
//   blockTimestamp: 当前区块时间戳（由公式计算，非本地时钟）
//   privateKey:     铸造者的私钥
func MintingHash(txID types.Hash512, refMintHash types.Hash512, stakes int64,
	blockTimestamp int64, privateKey crypto.PrivateKey) types.Hash512 {

	// 阶段一：构造动态因子 X
	// X 将时间戳、币权总值与混合常数结合，可塑性低且成本高昂
	x := encodeInt64(blockTimestamp * stakes * Mix)

	// 阶段二：组装源数据并哈希
	source := make([]byte, 0, types.HashLen+types.HashLen+8)
	source = append(source, txID[:]...)
	source = append(source, refMintHash[:]...)
	source = append(source, x...)
	hashData := crypto.SHA512Sum(source)

	// 阶段三：用铸造者私钥签名（结果对外不可预测）
	signData := crypto.Sign(privateKey, hashData[:])

	// 阶段四：对签名结果取哈希，得到铸凭哈希
	return crypto.SHA512Sum(signData)
}

// CompareMintHash 按字节序比较两个铸凭哈希。
// 返回值：-1 表示 a < b，0 表示 a == b，1 表示 a > b。
// 最小值获胜（即 -1 表示 a 更优）。
func CompareMintHash(a, b types.Hash512) int {
	return bytes.Compare(a[:], b[:])
}

// IsMintTxEligible 判断铸凭交易是否在合法区块范围内。
// 在初段（currentHeight < 28）所有合法交易均可参与。
// 正常阶段要求 depth > 27 && depth <= 80000。
//
// 参数说明：
//   currentHeight: 当前区块高度
//   txHeight:      铸凭交易所在区块的高度
func IsMintTxEligible(currentHeight, txHeight int) bool {
	// 初段：所有合法交易均可参与
	if currentHeight < MintTxMinDepth {
		return true
	}

	// 正常阶段：depth > 27 && depth <= 80000
	depth := currentHeight - txHeight
	return depth > MintTxMinDepth-1 && depth <= MintTxMaxDepth
}

// encodeInt64 将 int64 值编码为 8 字节大端字节序。
func encodeInt64(v int64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(v))
	return buf
}
```

### Step 4: 运行测试验证通过

```bash
go test -v ./internal/consensus/ -run "TestEncodeInt64|TestMintingHash|TestCompareMintHash|TestIsMintTxEligible"
```

预期输出：全部 PASS。

### Step 5: 提交

```bash
git add internal/consensus/minting.go internal/consensus/minting_test.go
git commit -m "feat(consensus): add MintingHash computation, CompareMintHash and IsMintTxEligible"
```

---

## Task 2: 择优池管理 (internal/consensus/bestpool.go)

**Files:**
- Create: `internal/consensus/bestpool.go`
- Test: `internal/consensus/bestpool_test.go`

本 Task 实现择优池 `BestPool`、铸造候选者 `MintCandidate` 结构体、池的插入/查询/排序逻辑、同步授权判断和重复检查。

### Step 1: 写失败测试

创建 `internal/consensus/bestpool_test.go`：

```go
package consensus

import (
	"testing"

	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// 辅助函数：创建测试用 MintCandidate，通过 mintHashByte 控制排序
func makeCandidate(txByte byte, mintHashByte byte) MintCandidate {
	var txID types.Hash512
	txID[0] = txByte

	var mintHash types.Hash512
	mintHash[0] = mintHashByte

	pub, _, _ := crypto.GenerateKeyPair()

	return MintCandidate{
		TxYear:    2026,
		TxID:      txID,
		MinterPub: pub,
		SignData:  []byte{0x01, 0x02},
		MintHash:  mintHash,
	}
}

// --- NewBestPool 测试 ---

func TestNewBestPool(t *testing.T) {
	pool := NewBestPool(100)
	if pool == nil {
		t.Fatal("NewBestPool() returned nil")
	}
	if pool.RefHeight != 100 {
		t.Errorf("RefHeight = %d, want 100", pool.RefHeight)
	}
	if pool.Size != 0 {
		t.Errorf("Size = %d, want 0", pool.Size)
	}
}

// --- Insert 测试 ---

// 空池插入
func TestBestPool_Insert_Empty(t *testing.T) {
	pool := NewBestPool(100)
	c := makeCandidate(0x01, 0x50)

	ok := pool.Insert(c)
	if !ok {
		t.Error("Insert() to empty pool should return true")
	}
	if pool.Size != 1 {
		t.Errorf("Size = %d, want 1", pool.Size)
	}
}

// 多次插入保持升序
func TestBestPool_Insert_Ordering(t *testing.T) {
	pool := NewBestPool(100)

	// 以不同 mintHash 顺序插入
	pool.Insert(makeCandidate(0x03, 0x30)) // 中间
	pool.Insert(makeCandidate(0x01, 0x10)) // 最小
	pool.Insert(makeCandidate(0x05, 0x50)) // 最大

	if pool.Size != 3 {
		t.Fatalf("Size = %d, want 3", pool.Size)
	}

	// 验证升序排列
	for i := 0; i < pool.Size-1; i++ {
		cmp := CompareMintHash(pool.Candidates[i].MintHash, pool.Candidates[i+1].MintHash)
		if cmp >= 0 {
			t.Errorf("Candidates[%d].MintHash >= Candidates[%d].MintHash", i, i+1)
		}
	}
}

// 填满池后插入更优条目
func TestBestPool_Insert_FullPool_BetterEntry(t *testing.T) {
	pool := NewBestPool(100)

	// 填满池
	for i := byte(0); i < types.BestPoolCapacity; i++ {
		pool.Insert(makeCandidate(i+0x10, i+0x10)) // mintHash 从 0x10 到 0x23
	}

	if pool.Size != types.BestPoolCapacity {
		t.Fatalf("pool not full: Size = %d, want %d", pool.Size, types.BestPoolCapacity)
	}

	// 记录当前最差
	worst := pool.Worst()
	if worst == nil {
		t.Fatal("Worst() returned nil for full pool")
	}

	// 插入更优的候选者（比当前最差更小）
	better := makeCandidate(0x01, 0x01)
	ok := pool.Insert(better)
	if !ok {
		t.Error("Insert() with better hash should return true for full pool")
	}

	// 池仍然满
	if pool.Size != types.BestPoolCapacity {
		t.Errorf("Size = %d, want %d", pool.Size, types.BestPoolCapacity)
	}

	// 新最优应小于或等于新条目
	newBest := pool.Best()
	if newBest == nil {
		t.Fatal("Best() returned nil")
	}
	cmp := CompareMintHash(newBest.MintHash, better.MintHash)
	if cmp > 0 {
		t.Error("Best().MintHash should be <= inserted better hash")
	}
}

// 满池插入更差条目
func TestBestPool_Insert_FullPool_WorseEntry(t *testing.T) {
	pool := NewBestPool(100)

	// 填满池（mintHash 从 0x01 到 0x14）
	for i := byte(0); i < types.BestPoolCapacity; i++ {
		pool.Insert(makeCandidate(i+0x01, i+0x01))
	}

	// 插入更差的候选者（比所有已有成员都大）
	worse := makeCandidate(0xFF, 0xFF)
	ok := pool.Insert(worse)
	if ok {
		t.Error("Insert() with worse hash should return false for full pool")
	}

	// 池大小不变
	if pool.Size != types.BestPoolCapacity {
		t.Errorf("Size = %d, want %d", pool.Size, types.BestPoolCapacity)
	}
}

// --- Best 和 Worst 测试 ---

func TestBestPool_Best_Empty(t *testing.T) {
	pool := NewBestPool(100)
	if pool.Best() != nil {
		t.Error("Best() should return nil for empty pool")
	}
}

func TestBestPool_Worst_Empty(t *testing.T) {
	pool := NewBestPool(100)
	if pool.Worst() != nil {
		t.Error("Worst() should return nil for empty pool")
	}
}

func TestBestPool_Best_Worst(t *testing.T) {
	pool := NewBestPool(100)
	pool.Insert(makeCandidate(0x03, 0x30))
	pool.Insert(makeCandidate(0x01, 0x10))
	pool.Insert(makeCandidate(0x05, 0x50))

	best := pool.Best()
	worst := pool.Worst()

	if best == nil || worst == nil {
		t.Fatal("Best() or Worst() returned nil")
	}

	// best 应为 0x10，worst 应为 0x50
	if best.MintHash[0] != 0x10 {
		t.Errorf("Best().MintHash[0] = 0x%02x, want 0x10", best.MintHash[0])
	}
	if worst.MintHash[0] != 0x50 {
		t.Errorf("Worst().MintHash[0] = 0x%02x, want 0x50", worst.MintHash[0])
	}
}

// --- IsSyncAuthorized 测试 ---

func TestBestPool_IsSyncAuthorized(t *testing.T) {
	tests := []struct {
		name  string
		index int
		want  bool
	}{
		{"rank 0 (top 1)", 0, false},
		{"rank 1 (top 2)", 1, false},
		{"rank 2 (top 3)", 2, false},
		{"rank 3 (top 4)", 3, false},
		{"rank 4 (top 5)", 4, false},
		{"rank 5 (6th, authorized)", 5, true},
		{"rank 6 (7th, authorized)", 6, true},
		{"rank 10 (11th, authorized)", 10, true},
		{"rank 15 (16th, authorized)", 15, true},
		{"rank 19 (20th, authorized)", 19, true},
		{"rank 20 (out of pool)", 20, false},
		{"negative index", -1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsSyncAuthorized(tt.index)
			if got != tt.want {
				t.Errorf("IsSyncAuthorized(%d) = %v, want %v", tt.index, got, tt.want)
			}
		})
	}
}

// --- Contains 测试 ---

func TestBestPool_Contains(t *testing.T) {
	pool := NewBestPool(100)

	txID := types.Hash512{0xAA}
	c := makeCandidate(0xAA, 0x50)
	pool.Insert(c)

	// 已存在的 txID
	if !pool.Contains(txID) {
		t.Error("Contains() should return true for existing txID")
	}

	// 不存在的 txID
	missingID := types.Hash512{0xBB}
	if pool.Contains(missingID) {
		t.Error("Contains() should return false for missing txID")
	}
}

// 测试重复 txID 不重复插入
func TestBestPool_Insert_DuplicateTxID(t *testing.T) {
	pool := NewBestPool(100)

	c1 := makeCandidate(0x01, 0x10)
	c2 := makeCandidate(0x01, 0x05) // 同一 txID（txByte 相同），更优的哈希

	pool.Insert(c1)
	ok := pool.Insert(c2)

	if ok {
		t.Error("Insert() should return false for duplicate txID")
	}
	if pool.Size != 1 {
		t.Errorf("Size = %d, want 1 (no duplicate insertion)", pool.Size)
	}
}

// --- 综合测试：渐进填充与淘汰 ---

func TestBestPool_Progressive(t *testing.T) {
	pool := NewBestPool(100)

	// 先填满池（mintHash 从 0x10 到 0x23）
	for i := byte(0); i < types.BestPoolCapacity; i++ {
		pool.Insert(makeCandidate(i+0x40, i+0x10))
	}

	if pool.Size != types.BestPoolCapacity {
		t.Fatalf("pool not full: Size = %d", pool.Size)
	}

	// 当前最差应为 0x23
	worst := pool.Worst()
	if worst.MintHash[0] != 0x10+types.BestPoolCapacity-1 {
		t.Errorf("initial worst = 0x%02x, want 0x%02x",
			worst.MintHash[0], byte(0x10+types.BestPoolCapacity-1))
	}

	// 插入比 0x10 更优的 0x05
	betterC := makeCandidate(0xA0, 0x05)
	ok := pool.Insert(betterC)
	if !ok {
		t.Error("insert better candidate should succeed")
	}

	// 最优应为 0x05
	best := pool.Best()
	if best.MintHash[0] != 0x05 {
		t.Errorf("Best().MintHash[0] = 0x%02x, want 0x05", best.MintHash[0])
	}

	// 最差应已被淘汰，新最差应为 0x22（倒数第二）
	newWorst := pool.Worst()
	if newWorst.MintHash[0] != 0x10+types.BestPoolCapacity-2 {
		t.Errorf("new worst = 0x%02x, want 0x%02x",
			newWorst.MintHash[0], byte(0x10+types.BestPoolCapacity-2))
	}
}
```

### Step 2: 运行测试验证失败

```bash
go test -v ./internal/consensus/ -run "TestNewBestPool|TestBestPool"
```

预期输出：编译失败，`MintCandidate`、`BestPool`、`NewBestPool`、`IsSyncAuthorized` 等未定义。

### Step 3: 写最小实现

创建 `internal/consensus/bestpool.go`：

```go
package consensus

import (
	"github.com/cxio/evidcoin/pkg/crypto"
	"github.com/cxio/evidcoin/pkg/types"
)

// 同步授权范围常量
const (
	SyncAuthStart = 5  // 同步授权起始索引（第 6 名，索引从 0 开始）
	SyncAuthEnd   = 19 // 同步授权结束索引（第 20 名）
)

// MintCandidate 择优凭证：铸造候选者的证明信息。
type MintCandidate struct {
	TxYear    int              // 交易所在年度
	TxID      types.Hash512    // 铸凭交易 ID
	MinterPub crypto.PublicKey // 铸造者公钥（首笔输入的接收者）
	SignData  []byte           // 铸造者对铸凭哈希源数据的签名
	MintHash  types.Hash512    // 铸凭哈希（由 SignData 计算）
}

// BestPool 择优池：按铸凭哈希升序排列的铸造候选者集合。
// 固定容量为 BestPoolCapacity（20），使用数组存储。
type BestPool struct {
	RefHeight  int                                        // 对应的评参区块高度
	Candidates [types.BestPoolCapacity]MintCandidate      // 按铸凭哈希升序排列
	Size       int                                        // 当前候选者数量
}

// NewBestPool 创建一个新的择优池。
func NewBestPool(refHeight int) *BestPool {
	return &BestPool{
		RefHeight: refHeight,
		Size:      0,
	}
}

// Insert 插入候选者到择优池。
// 返回 true 表示插入成功，false 表示拒绝（重复或不够优秀）。
//
// 规则：
//   - 若 txID 已存在，拒绝插入
//   - 若池未满，按序插入
//   - 若池已满且新候选者优于最差候选者，替换最差并按序插入
//   - 若池已满且新候选者不够优秀，拒绝
func (p *BestPool) Insert(candidate MintCandidate) bool {
	// 检查 txID 是否已存在
	if p.Contains(candidate.TxID) {
		return false
	}

	// 池已满
	if p.Size >= types.BestPoolCapacity {
		// 与最差条目比较
		worst := &p.Candidates[p.Size-1]
		if CompareMintHash(candidate.MintHash, worst.MintHash) >= 0 {
			// 新候选者不优于最差条目
			return false
		}
		// 替换最差条目（先移除最差，再插入新条目）
		p.Size--
	}

	// 查找插入位置（升序，二分查找）
	pos := p.findInsertPos(candidate.MintHash)

	// 后移元素以腾出位置
	if pos < p.Size {
		copy(p.Candidates[pos+1:p.Size+1], p.Candidates[pos:p.Size])
	}

	// 插入
	p.Candidates[pos] = candidate
	p.Size++

	return true
}

// findInsertPos 使用二分查找定位铸凭哈希的插入位置（升序）。
func (p *BestPool) findInsertPos(mintHash types.Hash512) int {
	lo, hi := 0, p.Size
	for lo < hi {
		mid := lo + (hi-lo)/2
		if CompareMintHash(p.Candidates[mid].MintHash, mintHash) < 0 {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo
}

// Best 返回最优（最小铸凭哈希）候选者。
// 若池为空则返回 nil。
func (p *BestPool) Best() *MintCandidate {
	if p.Size == 0 {
		return nil
	}
	return &p.Candidates[0]
}

// Worst 返回最差（最大铸凭哈希）候选者。
// 若池为空则返回 nil。
func (p *BestPool) Worst() *MintCandidate {
	if p.Size == 0 {
		return nil
	}
	return &p.Candidates[p.Size-1]
}

// Contains 检查指定交易 ID 是否已在池中。
func (p *BestPool) Contains(txID types.Hash512) bool {
	for i := 0; i < p.Size; i++ {
		if p.Candidates[i].TxID == txID {
			return true
		}
	}
	return false
}

// IsSyncAuthorized 判断指定排名（索引从 0 开始）是否有权发起同步。
// 仅后 15 名（排名 6-20，索引 5-19）的候选者可发起同步。
// 基于"利益无关"原则：低排名者无动机操控池状态。
func IsSyncAuthorized(index int) bool {
	return index >= SyncAuthStart && index <= SyncAuthEnd
}
```

### Step 4: 运行测试验证通过

```bash
go test -v ./internal/consensus/ -run "TestNewBestPool|TestBestPool"
```

预期输出：全部 PASS。

### Step 5: 提交

```bash
git add internal/consensus/bestpool.go internal/consensus/bestpool_test.go
git commit -m "feat(consensus): add BestPool management with MintCandidate insertion and sync authorization"
```

---

## Task 3: 铸造时间表 (internal/consensus/schedule.go)

**Files:**
- Create: `internal/consensus/schedule.go`
- Test: `internal/consensus/schedule_test.go`

本 Task 实现铸币计划——按年份计算铸币率、区块奖励、手续费拆分、收入分配（五类接收者）、最低交易费、以及交易过期常量。

### Step 1: 写失败测试

创建 `internal/consensus/schedule_test.go`：

```go
package consensus

import (
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// --- YearFromHeight 测试 ---

func TestYearFromHeight(t *testing.T) {
	tests := []struct {
		name   string
		height int
		want   int
	}{
		{"genesis block", 0, 1},
		{"first block", 1, 1},
		{"last of year 1", types.BlocksPerYear - 1, 1},
		{"first of year 2", types.BlocksPerYear, 2},
		{"middle of year 2", types.BlocksPerYear + 1000, 2},
		{"first of year 3", types.BlocksPerYear * 2, 3},
		{"first of year 10", types.BlocksPerYear * 9, 10},
		{"first of year 26", types.BlocksPerYear * 25, 26},
		{"beyond year 26", types.BlocksPerYear * 30, 31},
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

// --- MintingRate 测试 ---

func TestMintingRate(t *testing.T) {
	tests := []struct {
		name string
		year int
		want int64
	}{
		{"year 1", 1, 10},
		{"year 2", 2, 20},
		{"year 3", 3, 30},
		{"year 4", 4, 40},
		{"year 5", 5, 40},
		{"year 6", 6, 32},
		{"year 7", 7, 32},
		{"year 8", 8, 25},
		{"year 9", 9, 25},
		{"year 10", 10, 20},
		{"year 11", 11, 20},
		{"year 12", 12, 16},
		{"year 13", 13, 16},
		{"year 14", 14, 12},
		{"year 15", 15, 12},
		{"year 16", 16, 9},
		{"year 17", 17, 9},
		{"year 18", 18, 7},
		{"year 19", 19, 7},
		{"year 20", 20, 5},
		{"year 21", 21, 5},
		{"year 22", 22, 4},
		{"year 23", 23, 4},
		{"year 24", 24, 3},
		{"year 25", 25, 3},
		{"year 26 permanent", 26, 3},
		{"year 50 permanent", 50, 3},
		{"year 100 permanent", 100, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MintingRate(tt.year)
			if got != tt.want {
				t.Errorf("MintingRate(%d) = %d, want %d", tt.year, got, tt.want)
			}
		})
	}
}

// 测试无效年份
func TestMintingRate_InvalidYear(t *testing.T) {
	if MintingRate(0) != 0 {
		t.Error("MintingRate(0) should return 0")
	}
	if MintingRate(-1) != 0 {
		t.Error("MintingRate(-1) should return 0")
	}
}

// --- BlockReward 测试 ---

func TestBlockReward(t *testing.T) {
	tests := []struct {
		name   string
		height int
		want   int64
	}{
		{"genesis (year 1)", 0, 10},
		{"year 1 last block", types.BlocksPerYear - 1, 10},
		{"year 2 first block", types.BlocksPerYear, 20},
		{"year 4 first block", types.BlocksPerYear * 3, 40},
		{"year 26+ permanent", types.BlocksPerYear * 25, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BlockReward(tt.height)
			if got != tt.want {
				t.Errorf("BlockReward(%d) = %d, want %d", tt.height, got, tt.want)
			}
		})
	}
}

// --- 手续费拆分测试 ---

func TestRetainedFees(t *testing.T) {
	tests := []struct {
		name      string
		totalFees int64
		want      int64
	}{
		{"even", 100, 50},
		{"odd", 101, 50},
		{"zero", 0, 0},
		{"large", 999999, 499999},
		{"one", 1, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RetainedFees(tt.totalFees)
			if got != tt.want {
				t.Errorf("RetainedFees(%d) = %d, want %d", tt.totalFees, got, tt.want)
			}
		})
	}
}

func TestBurnedFees(t *testing.T) {
	tests := []struct {
		name      string
		totalFees int64
		want      int64
	}{
		{"even", 100, 50},
		{"odd", 101, 51},
		{"zero", 0, 0},
		{"one", 1, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BurnedFees(tt.totalFees)
			if got != tt.want {
				t.Errorf("BurnedFees(%d) = %d, want %d", tt.totalFees, got, tt.want)
			}
		})
	}
}

// 验证保留 + 销毁 = 总手续费
func TestFeesSplitConsistency(t *testing.T) {
	values := []int64{0, 1, 2, 3, 99, 100, 101, 999999, 1000000}
	for _, total := range values {
		retained := RetainedFees(total)
		burned := BurnedFees(total)
		if retained+burned != total {
			t.Errorf("RetainedFees(%d) + BurnedFees(%d) = %d + %d = %d, want %d",
				total, total, retained, burned, retained+burned, total)
		}
	}
}

// --- TotalRevenue 测试 ---

func TestTotalRevenue(t *testing.T) {
	// height=0 (year 1, rate=10), totalFees=100, retained=50
	// revenue = 10 + 50 = 60
	got := TotalRevenue(0, 100)
	if got != 60 {
		t.Errorf("TotalRevenue(0, 100) = %d, want 60", got)
	}

	// 零手续费
	got = TotalRevenue(0, 0)
	if got != 10 {
		t.Errorf("TotalRevenue(0, 0) = %d, want 10", got)
	}
}

// --- 收入分配测试 ---

func TestRevenueDistribution(t *testing.T) {
	tests := []struct {
		name    string
		revenue int64
	}{
		{"small", 10},
		{"medium", 100},
		{"large", 1000000},
		{"prime", 97},
		{"one", 1},
		{"zero", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := ValidatorShare(tt.revenue)
			minter := MinterShare(tt.revenue)
			depots := DepotsShare(tt.revenue)
			blockqs := BlockqsShare(tt.revenue)
			stun := StunShare(tt.revenue)

			total := validator + minter + depots + blockqs + stun
			if total != tt.revenue {
				t.Errorf("shares sum = %d, want %d (v=%d, m=%d, d=%d, b=%d, s=%d)",
					total, tt.revenue, validator, minter, depots, blockqs, stun)
			}
		})
	}
}

// 验证各份额比例大致正确
func TestRevenueDistribution_Ratios(t *testing.T) {
	revenue := int64(10000)

	validator := ValidatorShare(revenue) // 40%
	minter := MinterShare(revenue)       // 10%
	depots := DepotsShare(revenue)       // 20%
	blockqs := BlockqsShare(revenue)     // 20%
	stun := StunShare(revenue)           // 10%

	if validator != 4000 {
		t.Errorf("ValidatorShare(10000) = %d, want 4000", validator)
	}
	if minter != 1000 {
		t.Errorf("MinterShare(10000) = %d, want 1000", minter)
	}
	if depots != 2000 {
		t.Errorf("DepotsShare(10000) = %d, want 2000", depots)
	}
	if blockqs != 2000 {
		t.Errorf("BlockqsShare(10000) = %d, want 2000", blockqs)
	}
	if stun != 1000 {
		t.Errorf("StunShare(10000) = %d, want 1000", stun)
	}
}

// 验证余数分配给最后一个接收者（stun）
func TestRevenueDistribution_Remainder(t *testing.T) {
	// revenue=97 不能被 100 整除
	// validator: 97 * 40 / 100 = 38
	// minter:    97 * 10 / 100 = 9
	// depots:    97 * 20 / 100 = 19
	// blockqs:   97 * 20 / 100 = 19
	// stun:      97 - 38 - 9 - 19 - 19 = 12（含余数）
	revenue := int64(97)
	stun := StunShare(revenue)
	expected := revenue - ValidatorShare(revenue) - MinterShare(revenue) -
		DepotsShare(revenue) - BlockqsShare(revenue)

	if stun != expected {
		t.Errorf("StunShare(%d) = %d, want %d (remainder)", revenue, stun, expected)
	}
}

// --- MinTransactionFee 测试 ---

func TestMinTransactionFee(t *testing.T) {
	tests := []struct {
		name string
		avg  int64
		want int64
	}{
		{"average 100", 100, 25},
		{"average 4", 4, 1},
		{"average 3", 3, 0},
		{"average 0", 0, 0},
		{"average 1000000", 1000000, 250000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MinTransactionFee(tt.avg)
			if got != tt.want {
				t.Errorf("MinTransactionFee(%d) = %d, want %d", tt.avg, got, tt.want)
			}
		})
	}
}

// --- 常量测试 ---

func TestScheduleConstants(t *testing.T) {
	if FeeRecalcPeriod != 6000 {
		t.Errorf("FeeRecalcPeriod = %d, want 6000", FeeRecalcPeriod)
	}
	if TxExpiryBlocks != 240 {
		t.Errorf("TxExpiryBlocks = %d, want 240", TxExpiryBlocks)
	}
}
```

### Step 2: 运行测试验证失败

```bash
go test -v ./internal/consensus/ -run "TestYearFromHeight|TestMintingRate|TestBlockReward|TestRetainedFees|TestBurnedFees|TestFeesSplit|TestTotalRevenue|TestRevenueDistribution|TestMinTransactionFee|TestScheduleConstants"
```

预期输出：编译失败，`YearFromHeight`、`MintingRate`、`BlockReward` 等未定义。

### Step 3: 写最小实现

创建 `internal/consensus/schedule.go`：

```go
package consensus

import (
	"github.com/cxio/evidcoin/pkg/types"
)

// 铸造时间表常量
const (
	// FeeRecalcPeriod 最低交易费重新计算周期（区块数，约 25 天）。
	FeeRecalcPeriod = 6000

	// TxExpiryBlocks 未确认交易过期时间（区块数，约 24 小时）。
	TxExpiryBlocks = 240
)

// 收入分配比例（百分比）
const (
	ValidatorPercent = 40 // 验证者份额
	MinterPercent    = 10 // 铸造者份额
	DepotsPercent    = 20 // 数据驿站份额
	BlockqsPercent   = 20 // 区块查询服务份额
	// StunPercent 网络穿透服务份额为剩余部分（10% + 余数）
)

// mintingRateTable 铸币率表：key 为年份，value 为每块铸币量。
// 年份从 1 开始。未在表中的年份采用最近定义值，26+ 年永久为 3。
var mintingRateTable = [...]struct {
	yearStart int
	yearEnd   int
	rate      int64
}{
	{1, 1, 10},
	{2, 2, 20},
	{3, 3, 30},
	{4, 5, 40},
	{6, 7, 32},
	{8, 9, 25},
	{10, 11, 20},
	{12, 13, 16},
	{14, 15, 12},
	{16, 17, 9},
	{18, 19, 7},
	{20, 21, 5},
	{22, 23, 4},
	{24, 25, 3},
}

// PermanentMintingRate 永久低通胀铸币率（Year 26+）。
const PermanentMintingRate int64 = 3

// YearFromHeight 从区块高度计算年份（从 1 开始）。
// height 0 到 BlocksPerYear-1 为 Year 1，以此类推。
func YearFromHeight(height int) int {
	if height < 0 {
		return 0
	}
	return height/types.BlocksPerYear + 1
}

// MintingRate 根据年份返回每块铸币量。
// 年份从 1 开始，无效年份返回 0。
func MintingRate(year int) int64 {
	if year <= 0 {
		return 0
	}

	// 在表中查找
	for _, entry := range mintingRateTable {
		if year >= entry.yearStart && year <= entry.yearEnd {
			return entry.rate
		}
	}

	// Year 26+：永久低通胀
	return PermanentMintingRate
}

// BlockReward 计算指定高度的区块铸币奖励。
func BlockReward(height int) int64 {
	year := YearFromHeight(height)
	return MintingRate(year)
}

// RetainedFees 计算保留的手续费（50%，向下取整）。
func RetainedFees(totalFees int64) int64 {
	return totalFees / 2
}

// BurnedFees 计算销毁的手续费（50%，向上取整）。
// 确保 RetainedFees + BurnedFees = totalFees。
func BurnedFees(totalFees int64) int64 {
	return totalFees - RetainedFees(totalFees)
}

// TotalRevenue 计算区块总收入。
// 总收入 = 铸币奖励 + 保留手续费
func TotalRevenue(height int, totalFees int64) int64 {
	return BlockReward(height) + RetainedFees(totalFees)
}

// ValidatorShare 计算验证者份额（40%）。
func ValidatorShare(revenue int64) int64 {
	return revenue * ValidatorPercent / 100
}

// MinterShare 计算铸造者份额（10%）。
func MinterShare(revenue int64) int64 {
	return revenue * MinterPercent / 100
}

// DepotsShare 计算数据驿站份额（20%）。
func DepotsShare(revenue int64) int64 {
	return revenue * DepotsPercent / 100
}

// BlockqsShare 计算区块查询服务份额（20%）。
func BlockqsShare(revenue int64) int64 {
	return revenue * BlockqsPercent / 100
}

// StunShare 计算网络穿透服务份额（剩余部分，含整数余数）。
// 确保所有份额之和 = revenue。
func StunShare(revenue int64) int64 {
	return revenue - ValidatorShare(revenue) - MinterShare(revenue) -
		DepotsShare(revenue) - BlockqsShare(revenue)
}

// MinTransactionFee 计算最低交易费（共约规则）。
// 每 FeeRecalcPeriod 个区块重新计算一次。
// 最低费 = 上周期平均费用 / 4
func MinTransactionFee(avgFeeLastPeriod int64) int64 {
	return avgFeeLastPeriod / 4
}
```

### Step 4: 运行测试验证通过

```bash
go test -v ./internal/consensus/ -run "TestYearFromHeight|TestMintingRate|TestBlockReward|TestRetainedFees|TestBurnedFees|TestFeesSplit|TestTotalRevenue|TestRevenueDistribution|TestMinTransactionFee|TestScheduleConstants"
```

预期输出：全部 PASS。

### Step 5: 提交

```bash
git add internal/consensus/schedule.go internal/consensus/schedule_test.go
git commit -m "feat(consensus): add minting schedule with reward rates, fee split and revenue distribution"
```

---

## Task 4: 分叉检测与解决 (internal/consensus/fork.go)

**Files:**
- Create: `internal/consensus/fork.go`
- Test: `internal/consensus/fork_test.go`

本 Task 实现分叉竞争窗口比较 `ResolveFork`、`ForkState` 分叉状态管理、区块竞争规则 `SameMinterCompete` 和 `DifferentMinterCompete`。

### Step 1: 写失败测试

创建 `internal/consensus/fork_test.go`：

```go
package consensus

import (
	"testing"

	"github.com/cxio/evidcoin/pkg/types"
)

// 辅助函数：构造 ForkWindowSize 个铸凭哈希数组
func makeHashes(values [types.ForkWindowSize]byte) [types.ForkWindowSize]types.Hash512 {
	var hashes [types.ForkWindowSize]types.Hash512
	for i, v := range values {
		hashes[i][0] = v
	}
	return hashes
}

// --- ResolveFork 测试 ---

// challenger 全胜
func TestResolveFork_ChallengerWinsAll(t *testing.T) {
	var mainVals, challVals [types.ForkWindowSize]byte
	for i := 0; i < types.ForkWindowSize; i++ {
		mainVals[i] = byte(0x80 + i)  // 大值
		challVals[i] = byte(0x01 + i) // 小值
	}

	main := makeHashes(mainVals)
	challenger := makeHashes(challVals)

	if !ResolveFork(main, challenger) {
		t.Error("ResolveFork() should return true when challenger wins all blocks")
	}
}

// main 全胜
func TestResolveFork_MainWinsAll(t *testing.T) {
	var mainVals, challVals [types.ForkWindowSize]byte
	for i := 0; i < types.ForkWindowSize; i++ {
		mainVals[i] = byte(0x01 + i)  // 小值
		challVals[i] = byte(0x80 + i) // 大值
	}

	main := makeHashes(mainVals)
	challenger := makeHashes(challVals)

	if ResolveFork(main, challenger) {
		t.Error("ResolveFork() should return false when main wins all blocks")
	}
}

// 刚好 13/25 胜出（多数）
func TestResolveFork_ExactMajority(t *testing.T) {
	var mainVals, challVals [types.ForkWindowSize]byte
	// 前 13 个区块 challenger 胜
	for i := 0; i < 13; i++ {
		mainVals[i] = 0x80
		challVals[i] = 0x01
	}
	// 后 12 个区块 main 胜
	for i := 13; i < types.ForkWindowSize; i++ {
		mainVals[i] = 0x01
		challVals[i] = 0x80
	}

	main := makeHashes(mainVals)
	challenger := makeHashes(challVals)

	if !ResolveFork(main, challenger) {
		t.Error("ResolveFork() should return true when challenger wins 13/25")
	}
}

// 12/25 不胜出（未过半）
func TestResolveFork_NotEnoughWins(t *testing.T) {
	var mainVals, challVals [types.ForkWindowSize]byte
	// 前 12 个区块 challenger 胜
	for i := 0; i < 12; i++ {
		mainVals[i] = 0x80
		challVals[i] = 0x01
	}
	// 后 13 个区块 main 胜
	for i := 12; i < types.ForkWindowSize; i++ {
		mainVals[i] = 0x01
		challVals[i] = 0x80
	}

	main := makeHashes(mainVals)
	challenger := makeHashes(challVals)

	if ResolveFork(main, challenger) {
		t.Error("ResolveFork() should return false when challenger only wins 12/25")
	}
}

// 提前终止（前 13 个全胜，无需比较后续）
func TestResolveFork_EarlyTermination(t *testing.T) {
	var mainVals, challVals [types.ForkWindowSize]byte
	for i := 0; i < types.ForkWindowSize; i++ {
		mainVals[i] = 0x80
		challVals[i] = 0x01
	}

	main := makeHashes(mainVals)
	challenger := makeHashes(challVals)

	// 应提前在第 13 次比较后返回 true
	// 功能正确即可，此处仅验证结果
	if !ResolveFork(main, challenger) {
		t.Error("ResolveFork() with early termination should still return true")
	}
}

// 完全平局（相同哈希）
func TestResolveFork_AllEqual(t *testing.T) {
	var vals [types.ForkWindowSize]byte
	for i := 0; i < types.ForkWindowSize; i++ {
		vals[i] = byte(0x42)
	}

	main := makeHashes(vals)
	challenger := makeHashes(vals)

	if ResolveFork(main, challenger) {
		t.Error("ResolveFork() should return false when all hashes are equal (no wins)")
	}
}

// --- ForkState 测试 ---

func TestNewForkState(t *testing.T) {
	fs := NewForkState(1000)
	if fs == nil {
		t.Fatal("NewForkState() returned nil")
	}
	if fs.ForkHeight != 1000 {
		t.Errorf("ForkHeight = %d, want 1000", fs.ForkHeight)
	}
	if fs.Resolved {
		t.Error("new ForkState should not be resolved")
	}
	if len(fs.MainHashes) != 0 {
		t.Errorf("MainHashes length = %d, want 0", len(fs.MainHashes))
	}
	if len(fs.ChallengerHashes) != 0 {
		t.Errorf("ChallengerHashes length = %d, want 0", len(fs.ChallengerHashes))
	}
}

func TestForkState_Append(t *testing.T) {
	fs := NewForkState(1000)

	h1 := types.Hash512{0x01}
	h2 := types.Hash512{0x02}

	fs.AppendMain(h1)
	fs.AppendChallenger(h2)

	if len(fs.MainHashes) != 1 {
		t.Errorf("MainHashes length = %d, want 1", len(fs.MainHashes))
	}
	if len(fs.ChallengerHashes) != 1 {
		t.Errorf("ChallengerHashes length = %d, want 1", len(fs.ChallengerHashes))
	}
	if fs.MainHashes[0] != h1 {
		t.Error("MainHashes[0] mismatch")
	}
	if fs.ChallengerHashes[0] != h2 {
		t.Error("ChallengerHashes[0] mismatch")
	}
}

// 双方累积到 ForkWindowSize 时评估
func TestForkState_Evaluate_ChallengerWins(t *testing.T) {
	fs := NewForkState(1000)

	for i := 0; i < types.ForkWindowSize; i++ {
		fs.AppendMain(types.Hash512{0x80})
		fs.AppendChallenger(types.Hash512{0x01})
	}

	resolved, challengerWins := fs.Evaluate()
	if !resolved {
		t.Error("Evaluate() should resolve when both have ForkWindowSize hashes")
	}
	if !challengerWins {
		t.Error("Evaluate() challenger should win with all smaller hashes")
	}
	if !fs.Resolved {
		t.Error("ForkState.Resolved should be true after evaluation")
	}
}

// 双方未累积满时不评估
func TestForkState_Evaluate_NotReady(t *testing.T) {
	fs := NewForkState(1000)

	for i := 0; i < types.ForkWindowSize-1; i++ {
		fs.AppendMain(types.Hash512{0x80})
		fs.AppendChallenger(types.Hash512{0x01})
	}

	resolved, _ := fs.Evaluate()
	if resolved {
		t.Error("Evaluate() should not resolve before ForkWindowSize hashes accumulated")
	}
}

// main 胜出
func TestForkState_Evaluate_MainWins(t *testing.T) {
	fs := NewForkState(1000)

	for i := 0; i < types.ForkWindowSize; i++ {
		fs.AppendMain(types.Hash512{0x01})
		fs.AppendChallenger(types.Hash512{0x80})
	}

	resolved, challengerWins := fs.Evaluate()
	if !resolved {
		t.Error("Evaluate() should resolve")
	}
	if challengerWins {
		t.Error("Evaluate() main should win with all smaller hashes")
	}
}

// --- IsStale 测试 ---

func TestForkState_IsStale(t *testing.T) {
	fs := NewForkState(1000)

	tests := []struct {
		name          string
		currentHeight int
		want          bool
	}{
		{"at fork height", 1000, false},
		{"within window", 1000 + types.ForkWindowSize - 1, false},
		{"at window boundary", 1000 + types.ForkWindowSize, false},
		{"just past window (stale)", 1000 + types.ForkWindowSize + 1, true},
		{"far past window", 1000 + types.ForkWindowSize + 100, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fs.IsStale(tt.currentHeight)
			if got != tt.want {
				t.Errorf("IsStale(%d) = %v, want %v", tt.currentHeight, got, tt.want)
			}
		})
	}
}

// --- SameMinterCompete 测试 ---

func TestSameMinterCompete(t *testing.T) {
	tests := []struct {
		name                string
		blockAFees, blockBFees int64
		want                int
	}{
		{"A has lower fees (A wins)", 50, 100, -1},
		{"B has lower fees (B wins)", 100, 50, 1},
		{"equal fees (tie)", 100, 100, 0},
		{"both zero", 0, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SameMinterCompete(tt.blockAFees, tt.blockBFees)
			if got != tt.want {
				t.Errorf("SameMinterCompete(%d, %d) = %d, want %d",
					tt.blockAFees, tt.blockBFees, got, tt.want)
			}
		})
	}
}

// --- DifferentMinterCompete 测试 ---

func TestDifferentMinterCompete(t *testing.T) {
	tests := []struct {
		name                             string
		primaryStakes, candidateStakes int64
		want                             bool
	}{
		{"candidate exactly 3x (wins)", 100, 300, true},
		{"candidate above 3x (wins)", 100, 301, true},
		{"candidate below 3x (loses)", 100, 299, false},
		{"candidate equal (loses)", 100, 100, false},
		{"candidate 2x (loses)", 100, 200, false},
		{"primary zero, candidate nonzero (wins)", 0, 1, true},
		{"both zero (loses)", 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DifferentMinterCompete(tt.primaryStakes, tt.candidateStakes)
			if got != tt.want {
				t.Errorf("DifferentMinterCompete(%d, %d) = %v, want %v",
					tt.primaryStakes, tt.candidateStakes, got, tt.want)
			}
		})
	}
}

// 边界：primaryStakes = 0 时任何正值候选者都应胜出
func TestDifferentMinterCompete_ZeroPrimary(t *testing.T) {
	// 0 * 3 = 0，candidateStakes >= 0 + 某个正值
	if !DifferentMinterCompete(0, 1) {
		t.Error("candidate with 1 stake should beat primary with 0 stakes")
	}
	if DifferentMinterCompete(0, 0) {
		t.Error("candidate with 0 stake should not beat primary with 0 stakes")
	}
}
```

### Step 2: 运行测试验证失败

```bash
go test -v ./internal/consensus/ -run "TestResolveFork|TestNewForkState|TestForkState|TestSameMinterCompete|TestDifferentMinterCompete"
```

预期输出：编译失败，`ResolveFork`、`ForkState`、`SameMinterCompete`、`DifferentMinterCompete` 等未定义。

### Step 3: 写最小实现

创建 `internal/consensus/fork.go`：

```go
package consensus

import (
	"bytes"

	"github.com/cxio/evidcoin/pkg/types"
)

// DifferentMinterThreshold 不同铸造者竞争时，挑战者币权需达到主块的倍数。
const DifferentMinterThreshold = 3

// ResolveFork 分叉评比：逐区块比较铸凭哈希。
// 返回 true 表示 challenger 胜出（获半数以上胜利）。
// 采用提前终止优化：wins > ForkWindowSize/2 时直接返回。
func ResolveFork(main, challenger [types.ForkWindowSize]types.Hash512) bool {
	wins := 0
	threshold := types.ForkWindowSize / 2

	for i := 0; i < types.ForkWindowSize; i++ {
		if bytes.Compare(challenger[i][:], main[i][:]) < 0 {
			wins++
		}
		// 提前终止：已过半胜出
		if wins > threshold {
			return true
		}
	}
	return false
}

// ForkState 分叉状态：管理一次分叉的铸凭哈希收集与评估。
type ForkState struct {
	ForkHeight       int            // 分叉起始高度
	MainHashes       []types.Hash512 // 主链铸凭哈希
	ChallengerHashes []types.Hash512 // 挑战链铸凭哈希
	Resolved         bool           // 是否已解决
}

// NewForkState 创建新的分叉状态。
func NewForkState(forkHeight int) *ForkState {
	return &ForkState{
		ForkHeight:       forkHeight,
		MainHashes:       make([]types.Hash512, 0, types.ForkWindowSize),
		ChallengerHashes: make([]types.Hash512, 0, types.ForkWindowSize),
		Resolved:         false,
	}
}

// AppendMain 向主链添加一个铸凭哈希。
func (fs *ForkState) AppendMain(mintHash types.Hash512) {
	if len(fs.MainHashes) < types.ForkWindowSize {
		fs.MainHashes = append(fs.MainHashes, mintHash)
	}
}

// AppendChallenger 向挑战链添加一个铸凭哈希。
func (fs *ForkState) AppendChallenger(mintHash types.Hash512) {
	if len(fs.ChallengerHashes) < types.ForkWindowSize {
		fs.ChallengerHashes = append(fs.ChallengerHashes, mintHash)
	}
}

// Evaluate 当双方都累积到 ForkWindowSize 个区块时评估分叉。
// 返回 resolved 表示是否已可评估，challengerWins 表示挑战者是否胜出。
func (fs *ForkState) Evaluate() (resolved bool, challengerWins bool) {
	// 双方都必须累积到 ForkWindowSize
	if len(fs.MainHashes) < types.ForkWindowSize || len(fs.ChallengerHashes) < types.ForkWindowSize {
		return false, false
	}

	// 构造固定大小数组
	var main, challenger [types.ForkWindowSize]types.Hash512
	copy(main[:], fs.MainHashes[:types.ForkWindowSize])
	copy(challenger[:], fs.ChallengerHashes[:types.ForkWindowSize])

	challengerWins = ResolveFork(main, challenger)
	fs.Resolved = true

	return true, challengerWins
}

// IsStale 判断分叉是否已超出竞争窗口（隐蔽分叉）。
// 若当前高度已超过分叉起始高度 + ForkWindowSize，则为隐蔽分叉，
// 应被视为独立合法的侧链并被忽略。
func (fs *ForkState) IsStale(currentHeight int) bool {
	return currentHeight > fs.ForkHeight+types.ForkWindowSize
}

// SameMinterCompete 同一铸造者多块竞争。
// 交易总手续费更低的区块获胜（抑制多签竞赛）。
// 返回：-1 表示 blockA 胜，0 表示平局，1 表示 blockB 胜。
func SameMinterCompete(blockAFees, blockBFees int64) int {
	switch {
	case blockAFees < blockBFees:
		return -1
	case blockAFees > blockBFees:
		return 1
	default:
		return 0
	}
}

// DifferentMinterCompete 不同铸造者竞争。
// 挑战者的币天销毁量需达到主块的 3 倍或以上才能获胜。
// 返回 true 表示挑战者（candidate）胜出。
func DifferentMinterCompete(primaryStakes, candidateStakes int64) bool {
	return candidateStakes >= primaryStakes*DifferentMinterThreshold
}
```

### Step 4: 运行测试验证通过

```bash
go test -v ./internal/consensus/ -run "TestResolveFork|TestNewForkState|TestForkState|TestSameMinterCompete|TestDifferentMinterCompete"
```

预期输出：全部 PASS。

### Step 5: 提交

```bash
git add internal/consensus/fork.go internal/consensus/fork_test.go
git commit -m "feat(consensus): add fork detection and resolution with block competition rules"
```

---

## Task 5: 引导启动规则 (internal/consensus/bootstrap.go)

**Files:**
- Create: `internal/consensus/bootstrap.go`
- Test: `internal/consensus/bootstrap_test.go`

本 Task 实现引导阶段的特殊规则，包括评参区块高度 `RefBlockHeight`、引导阶段识别 `BootstrapPhase`、输出数限制 `MaxOutputsPerTx` 和 `CoinbaseOutputCount`、确认窗口常量和奖励兑换 `RewardRedeemable`。

### Step 1: 写失败测试

创建 `internal/consensus/bootstrap_test.go`：

```go
package consensus

import (
	"math"
	"testing"
)

// --- RefBlockHeight 测试 ---

func TestRefBlockHeight(t *testing.T) {
	tests := []struct {
		name          string
		currentHeight int
		want          int
	}{
		{"height 0 (genesis)", 0, 0},
		{"height 1", 1, 0},
		{"height 5", 5, 0},
		{"height 8", 8, 0},
		{"height 9 (first normal)", 9, 0},
		{"height 10", 10, 1},
		{"height 18", 18, 9},
		{"height 100", 100, 91},
		{"height 1000", 1000, 991},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RefBlockHeight(tt.currentHeight)
			if got != tt.want {
				t.Errorf("RefBlockHeight(%d) = %d, want %d", tt.currentHeight, got, tt.want)
			}
		})
	}
}

// --- BootstrapPhase 测试 ---

func TestBootstrapPhase(t *testing.T) {
	tests := []struct {
		name   string
		height int
		want   string
	}{
		{"genesis", 0, "genesis"},
		{"key expansion start", 1, "key_expansion"},
		{"key expansion end", 10, "key_expansion"},
		{"accumulation start", 11, "accumulation"},
		{"accumulation middle", 3600, "accumulation"},
		{"accumulation end", 7200, "accumulation"},
		{"lottery expansion start", 7201, "lottery_expansion"},
		{"lottery expansion middle", 15000, "lottery_expansion"},
		{"lottery expansion end", 24000, "lottery_expansion"},
		{"open market start", 24001, "open_market"},
		{"open market far", 100000, "open_market"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BootstrapPhase(tt.height)
			if got != tt.want {
				t.Errorf("BootstrapPhase(%d) = %q, want %q", tt.height, got, tt.want)
			}
		})
	}
}

// --- MaxOutputsPerTx 测试 ---

func TestMaxOutputsPerTx(t *testing.T) {
	tests := []struct {
		name       string
		height     int
		inputCount int
		wantMax    int
	}{
		// 引导期：min(inputCount * 2, MaxOutputsAllowed)
		{"bootstrap 1 input", 100, 1, 2},
		{"bootstrap 3 inputs", 100, 3, 6},
		{"bootstrap 10 inputs", 100, 10, 20},
		{"bootstrap 24000 height", 24000, 5, 10},
		// 引导期大输入数应受全局上限约束
		{"bootstrap large inputs", 100, 10000, MaxOutputsAllowed},
		// 开放市场：返回全局上限
		{"open market 1 input", 24001, 1, MaxOutputsAllowed},
		{"open market 5 inputs", 24001, 5, MaxOutputsAllowed},
		{"open market far future", 100000, 1, MaxOutputsAllowed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MaxOutputsPerTx(tt.height, tt.inputCount)
			if got != tt.wantMax {
				t.Errorf("MaxOutputsPerTx(%d, %d) = %d, want %d",
					tt.height, tt.inputCount, got, tt.wantMax)
			}
		})
	}
}

// --- CoinbaseOutputCount 测试 ---

func TestCoinbaseOutputCount(t *testing.T) {
	tests := []struct {
		name   string
		height int
		want   int
	}{
		{"bootstrap genesis", 0, 1},
		{"bootstrap early", 100, 1},
		{"bootstrap 24000", 24000, 1},
		{"open market start", 24001, 5},
		{"open market far", 100000, 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CoinbaseOutputCount(tt.height)
			if got != tt.want {
				t.Errorf("CoinbaseOutputCount(%d) = %d, want %d", tt.height, got, tt.want)
			}
		})
	}
}

// --- 确认窗口常量测试 ---

func TestConfirmationConstants(t *testing.T) {
	if ConfirmationWindowSize != 48 {
		t.Errorf("ConfirmationWindowSize = %d, want 48", ConfirmationWindowSize)
	}
	if RequiredConfirmations != 2 {
		t.Errorf("RequiredConfirmations = %d, want 2", RequiredConfirmations)
	}
	if ConfirmationBitmapSize != 18 {
		t.Errorf("ConfirmationBitmapSize = %d, want 18", ConfirmationBitmapSize)
	}
}

// --- RewardRedeemable 测试 ---

func TestRewardRedeemable(t *testing.T) {
	tests := []struct {
		name                string
		confirmations       int
		blocksSinceCreation int
		wantPortion         float64
		wantDelta           float64 // 允许的浮点精度偏差
	}{
		// 不足所需确认数
		{"zero confirmations", 0, 100, 0.0, 0.0},
		{"one confirmation (not enough)", 1, 100, 0.0, 0.0},
		// 刚好满足最低确认
		{"exact required confirmations", 2, 100, 0.5, 0.01},
		// 所有服务都确认满窗口
		{"full confirmations 3 services", 6, 200, 1.0, 0.01},
		// 部分确认
		{"partial confirmations", 3, 100, 0.5, 0.1},
		// 区块不足
		{"blocks since creation = 0", 2, 0, 0.5, 0.01},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RewardRedeemable(tt.confirmations, tt.blocksSinceCreation)
			if math.Abs(got-tt.wantPortion) > tt.wantDelta {
				t.Errorf("RewardRedeemable(%d, %d) = %f, want %f (±%f)",
					tt.confirmations, tt.blocksSinceCreation, got, tt.wantPortion, tt.wantDelta)
			}
		})
	}
}

// 验证 RewardRedeemable 的返回值在 [0.0, 1.0] 范围内
func TestRewardRedeemable_Range(t *testing.T) {
	inputs := []struct {
		confirmations       int
		blocksSinceCreation int
	}{
		{0, 0}, {1, 0}, {2, 0}, {3, 50}, {6, 100}, {6, 1000},
		{0, 100}, {10, 200},
	}
	for _, in := range inputs {
		got := RewardRedeemable(in.confirmations, in.blocksSinceCreation)
		if got < 0.0 || got > 1.0 {
			t.Errorf("RewardRedeemable(%d, %d) = %f, out of [0.0, 1.0]",
				in.confirmations, in.blocksSinceCreation, got)
		}
	}
}
```

### Step 2: 运行测试验证失败

```bash
go test -v ./internal/consensus/ -run "TestRefBlockHeight|TestBootstrapPhase|TestMaxOutputsPerTx|TestCoinbaseOutputCount|TestConfirmationConstants|TestRewardRedeemable"
```

预期输出：编译失败，`RefBlockHeight`、`BootstrapPhase` 等未定义。

### Step 3: 写最小实现

创建 `internal/consensus/bootstrap.go`：

```go
package consensus

// 引导阶段相关常量
const (
	// BootstrapEndHeight 引导期结束高度。
	BootstrapEndHeight = 24000

	// KeyExpansionEnd 密钥扩展阶段结束高度。
	KeyExpansionEnd = 10

	// AccumulationEnd 累积观察阶段结束高度。
	AccumulationEnd = 7200

	// LotteryExpansionEnd 抽奖扩张阶段结束高度。
	LotteryExpansionEnd = 24000

	// MaxOutputsAllowed 全局最大输出数（交易级别）。
	MaxOutputsAllowed = 256

	// CoinbaseBootstrapOutputs 引导期 Coinbase 输出数。
	CoinbaseBootstrapOutputs = 1

	// CoinbaseNormalOutputs 正常阶段 Coinbase 输出数（含服务奖励）。
	CoinbaseNormalOutputs = 5
)

// 确认与奖励兑换常量
const (
	// ConfirmationWindowSize 确认窗口大小（区块数）。
	ConfirmationWindowSize = 48

	// RequiredConfirmations 所需最低确认数（每类服务 2 次）。
	RequiredConfirmations = 2

	// ConfirmationBitmapSize 确认位图大小（字节）。
	// 3 类服务 × 48 区块 = 144 bits = 18 字节
	ConfirmationBitmapSize = 18

	// ServiceCount 参与确认的服务类别数量。
	ServiceCount = 3
)

// RefBlockHeight 获取评参区块高度。
// 当 currentHeight < 9 时返回 0（使用创世块作为评参区块）。
// 否则返回 currentHeight - 9。
func RefBlockHeight(currentHeight int) int {
	if currentHeight < 9 {
		return 0
	}
	return currentHeight - 9
}

// BootstrapPhase 返回指定高度对应的引导阶段名称。
//   - 0:           "genesis"
//   - 1-10:        "key_expansion"
//   - 11-7200:     "accumulation"
//   - 7201-24000:  "lottery_expansion"
//   - 24001+:      "open_market"
func BootstrapPhase(height int) string {
	switch {
	case height == 0:
		return "genesis"
	case height <= KeyExpansionEnd:
		return "key_expansion"
	case height <= AccumulationEnd:
		return "accumulation"
	case height <= LotteryExpansionEnd:
		return "lottery_expansion"
	default:
		return "open_market"
	}
}

// MaxOutputsPerTx 返回指定高度下单笔交易的最大输出数。
// 引导期（height <= 24000）：min(inputCount * 2, MaxOutputsAllowed)
// 开放市场（height > 24000）：MaxOutputsAllowed
func MaxOutputsPerTx(height int, inputCount int) int {
	if height > BootstrapEndHeight {
		return MaxOutputsAllowed
	}

	// 引导期限制
	limit := inputCount * 2
	if limit > MaxOutputsAllowed {
		return MaxOutputsAllowed
	}
	return limit
}

// CoinbaseOutputCount 返回指定高度下 Coinbase 交易的输出数限制。
// 引导期（height <= 24000）：1（仅单一输出）
// 开放市场（height > 24000）：5（含服务奖励）
func CoinbaseOutputCount(height int) int {
	if height > BootstrapEndHeight {
		return CoinbaseNormalOutputs
	}
	return CoinbaseBootstrapOutputs
}

// RewardRedeemable 计算奖励的可兑换比例。
// 基于确认数和创建后经过的区块数计算可兑换比例（0.0 到 1.0）。
//
// 规则：
//   - confirmations < RequiredConfirmations 时返回 0.0
//   - 每类服务达到 RequiredConfirmations 视为该服务已确认
//   - 可兑换比例 = 已确认服务数 / ServiceCount
//
// 参数说明：
//   confirmations:       累计确认次数（所有服务的总和）
//   blocksSinceCreation: 奖励创建后经过的区块数
func RewardRedeemable(confirmations int, blocksSinceCreation int) float64 {
	if confirmations < RequiredConfirmations {
		return 0.0
	}

	// 计算已达标的服务数量
	// 每类服务需要 RequiredConfirmations 次确认
	confirmedServices := confirmations / RequiredConfirmations
	if confirmedServices > ServiceCount {
		confirmedServices = ServiceCount
	}

	return float64(confirmedServices) / float64(ServiceCount)
}
```

### Step 4: 运行测试验证通过

```bash
go test -v ./internal/consensus/ -run "TestRefBlockHeight|TestBootstrapPhase|TestMaxOutputsPerTx|TestCoinbaseOutputCount|TestConfirmationConstants|TestRewardRedeemable"
```

预期输出：全部 PASS。

### Step 5: 提交

```bash
git add internal/consensus/bootstrap.go internal/consensus/bootstrap_test.go
git commit -m "feat(consensus): add bootstrap rules with phase detection, output limits and reward redeemable"
```

---

## Task 6: 币权计算 (internal/consensus/coinday.go)

**Files:**
- Create: `internal/consensus/coinday.go`
- Test: `internal/consensus/coinday_test.go`

本 Task 实现币权（聪*秒）与币天的计算，包括 `CoinStakes`、`CoinDays`、`HoldDuration`、`BlockStakes` 及相关常量。

### Step 1: 写失败测试

创建 `internal/consensus/coinday_test.go`：

```go
package consensus

import (
	"testing"

	"github.com/cxio/evidcoin/internal/blockchain"
)

// --- 常量测试 ---

func TestCoindayConstants(t *testing.T) {
	if SatoshiPerCoin != 100_000_000 {
		t.Errorf("SatoshiPerCoin = %d, want 100000000", SatoshiPerCoin)
	}
	if SecondsPerDay != 86_400 {
		t.Errorf("SecondsPerDay = %d, want 86400", SecondsPerDay)
	}
	// CoinDayUnit = SatoshiPerCoin * SecondsPerDay
	expectedUnit := int64(100_000_000) * int64(86_400)
	if CoinDayUnit != expectedUnit {
		t.Errorf("CoinDayUnit = %d, want %d", CoinDayUnit, expectedUnit)
	}
	// 验证具体数值
	if CoinDayUnit != 8_640_000_000_000 {
		t.Errorf("CoinDayUnit = %d, want 8640000000000", CoinDayUnit)
	}
}

// --- CoinStakes 测试 ---

func TestCoinStakes(t *testing.T) {
	tests := []struct {
		name           string
		amountSatoshi  int64
		holdSeconds    int64
		want           int64
	}{
		{"1 coin 1 day", SatoshiPerCoin, SecondsPerDay, CoinDayUnit},
		{"1 satoshi 1 second", 1, 1, 1},
		{"zero amount", 0, 86400, 0},
		{"zero time", 100_000_000, 0, 0},
		{"both zero", 0, 0, 0},
		{"2 coins 3 days", 2 * SatoshiPerCoin, 3 * SecondsPerDay, 6 * CoinDayUnit},
		{"small values", 100, 200, 20000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CoinStakes(tt.amountSatoshi, tt.holdSeconds)
			if got != tt.want {
				t.Errorf("CoinStakes(%d, %d) = %d, want %d",
					tt.amountSatoshi, tt.holdSeconds, got, tt.want)
			}
		})
	}
}

// --- CoinDays 测试 ---

func TestCoinDays(t *testing.T) {
	tests := []struct {
		name          string
		amountSatoshi int64
		holdSeconds   int64
		want          int64
	}{
		{"1 coin 1 day = 1 coinday", SatoshiPerCoin, SecondsPerDay, 1},
		{"2 coins 3 days = 6 coindays", 2 * SatoshiPerCoin, 3 * SecondsPerDay, 6},
		{"half day = 0 (integer division)", SatoshiPerCoin, SecondsPerDay / 2, 0},
		{"zero amount", 0, SecondsPerDay, 0},
		{"zero time", SatoshiPerCoin, 0, 0},
		{"10 coins 10 days", 10 * SatoshiPerCoin, 10 * SecondsPerDay, 100},
		{"small amount short time", 100, 100, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CoinDays(tt.amountSatoshi, tt.holdSeconds)
			if got != tt.want {
				t.Errorf("CoinDays(%d, %d) = %d, want %d",
					tt.amountSatoshi, tt.holdSeconds, got, tt.want)
			}
		})
	}
}

// 验证 CoinDays = CoinStakes / CoinDayUnit
func TestCoinDays_Consistency(t *testing.T) {
	amount := int64(5 * SatoshiPerCoin)
	hold := int64(7 * SecondsPerDay)

	stakes := CoinStakes(amount, hold)
	days := CoinDays(amount, hold)

	if days != stakes/CoinDayUnit {
		t.Errorf("CoinDays() = %d, want CoinStakes() / CoinDayUnit = %d",
			days, stakes/CoinDayUnit)
	}
}

// --- HoldDuration 测试 ---

func TestHoldDuration(t *testing.T) {
	tests := []struct {
		name            string
		creationHeight  int32
		spendHeight     int32
		wantPositive    bool
	}{
		{"same height", 0, 0, false},
		{"one block apart", 0, 1, true},
		{"100 blocks apart", 100, 200, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HoldDuration(tt.creationHeight, tt.spendHeight)
			if tt.wantPositive && got <= 0 {
				t.Errorf("HoldDuration(%d, %d) = %d, want positive",
					tt.creationHeight, tt.spendHeight, got)
			}
			if !tt.wantPositive && got != 0 {
				t.Errorf("HoldDuration(%d, %d) = %d, want 0",
					tt.creationHeight, tt.spendHeight, got)
			}
		})
	}
}

// 验证 HoldDuration 使用 BlockTime 的差值
func TestHoldDuration_UsesBlockTime(t *testing.T) {
	var creationHeight int32 = 10
	var spendHeight int32 = 20

	duration := HoldDuration(creationHeight, spendHeight)
	expectedDuration := blockchain.BlockTime(spendHeight) - blockchain.BlockTime(creationHeight)

	if duration != expectedDuration {
		t.Errorf("HoldDuration(%d, %d) = %d, want BlockTime diff = %d",
			creationHeight, spendHeight, duration, expectedDuration)
	}
}

// --- BlockStakes 测试 ---

func TestBlockStakes(t *testing.T) {
	tests := []struct {
		name     string
		txStakes []int64
		want     int64
	}{
		{"empty", nil, 0},
		{"single", []int64{CoinDayUnit}, CoinDayUnit},
		{"multiple", []int64{CoinDayUnit, 2 * CoinDayUnit, 3 * CoinDayUnit}, 6 * CoinDayUnit},
		{"with zeros", []int64{0, CoinDayUnit, 0}, CoinDayUnit},
		{"all zeros", []int64{0, 0, 0}, 0},
		{"single zero", []int64{0}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BlockStakes(tt.txStakes)
			if got != tt.want {
				t.Errorf("BlockStakes(%v) = %d, want %d", tt.txStakes, got, tt.want)
			}
		})
	}
}

// 验证 BlockStakes 为简单求和
func TestBlockStakes_IsSum(t *testing.T) {
	stakes := []int64{100, 200, 300, 400, 500}
	var expectedSum int64
	for _, s := range stakes {
		expectedSum += s
	}

	got := BlockStakes(stakes)
	if got != expectedSum {
		t.Errorf("BlockStakes() = %d, want sum = %d", got, expectedSum)
	}
}
```

### Step 2: 运行测试验证失败

```bash
go test -v ./internal/consensus/ -run "TestCoindayConstants|TestCoinStakes|TestCoinDays|TestHoldDuration|TestBlockStakes"
```

预期输出：编译失败，`CoinStakes`、`CoinDays`、`HoldDuration`、`BlockStakes` 等未定义。

### Step 3: 写最小实现

创建 `internal/consensus/coinday.go`：

```go
package consensus

import (
	"github.com/cxio/evidcoin/internal/blockchain"
)

// 币权计算常量
const (
	// SatoshiPerCoin 每币的最小单位数（聪）。
	SatoshiPerCoin int64 = 100_000_000

	// SecondsPerDay 每天秒数。
	SecondsPerDay int64 = 86_400

	// CoinDayUnit 一币天对应的聪*秒数。
	// 1 币天 = 100,000,000 聪 × 86,400 秒 = 8,640,000,000,000
	CoinDayUnit int64 = SatoshiPerCoin * SecondsPerDay
)

// CoinStakes 计算币权（聪*秒）。
// 币权 = 金额（聪） × 持有秒数
func CoinStakes(amountSatoshi int64, holdSeconds int64) int64 {
	return amountSatoshi * holdSeconds
}

// CoinDays 计算币天数。
// 币天 = 币权 / CoinDayUnit（整数除法，向下取整）
func CoinDays(amountSatoshi int64, holdSeconds int64) int64 {
	return CoinStakes(amountSatoshi, holdSeconds) / CoinDayUnit
}

// HoldDuration 根据区块高度计算持有时长（毫秒）。
// 使用 BlockTime 函数计算两个高度之间的时间差。
func HoldDuration(creationHeight, spendHeight int32) int64 {
	return blockchain.BlockTime(spendHeight) - blockchain.BlockTime(creationHeight)
}

// BlockStakes 汇总区块内全部交易的币权。
// 各笔交易的币权已由调用方预先计算，此处仅做求和。
func BlockStakes(txStakes []int64) int64 {
	var total int64
	for _, s := range txStakes {
		total += s
	}
	return total
}
```

### Step 4: 运行测试验证通过

```bash
go test -v ./internal/consensus/ -run "TestCoindayConstants|TestCoinStakes|TestCoinDays|TestHoldDuration|TestBlockStakes"
```

预期输出：全部 PASS。

### Step 5: 提交

```bash
git add internal/consensus/coinday.go internal/consensus/coinday_test.go
git commit -m "feat(consensus): add coin-day calculation with CoinStakes, CoinDays, HoldDuration and BlockStakes"
```

---

## 全量集成验证

完成所有 6 个 Task 后，运行完整的共识包测试：

```bash
go test -v -cover ./internal/consensus/...
```

预期：全部 PASS，覆盖率 > 85%。

### 文件清单

| 文件 | 用途 |
|------|------|
| `internal/consensus/minting.go` | 铸凭哈希计算（MintingHash, CompareMintHash, IsMintTxEligible） |
| `internal/consensus/minting_test.go` | 铸凭哈希测试 |
| `internal/consensus/bestpool.go` | 择优池管理（BestPool, MintCandidate, Insert, IsSyncAuthorized） |
| `internal/consensus/bestpool_test.go` | 择优池测试 |
| `internal/consensus/schedule.go` | 铸造时间表（MintingRate, BlockReward, 收入分配） |
| `internal/consensus/schedule_test.go` | 铸造时间表测试 |
| `internal/consensus/fork.go` | 分叉检测与解决（ResolveFork, ForkState, 区块竞争） |
| `internal/consensus/fork_test.go` | 分叉测试 |
| `internal/consensus/bootstrap.go` | 引导启动规则（RefBlockHeight, BootstrapPhase, 输出限制） |
| `internal/consensus/bootstrap_test.go` | 引导启动测试 |
| `internal/consensus/coinday.go` | 币权计算（CoinStakes, CoinDays, HoldDuration, BlockStakes） |
| `internal/consensus/coinday_test.go` | 币权计算测试 |
