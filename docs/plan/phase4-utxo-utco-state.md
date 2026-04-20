# Phase 4: UTXO/UTCO State Management（UTXO/UTCO 状态管理）Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现 `internal/utxo` 与 `internal/utco` 包，提供未花费输出集（UTXO）与未转移凭信集（UTCO）的内存存储、查询、增量更新及 4 级分层指纹哈希计算。

**Architecture:** Layer 2 状态层。依赖 `pkg/types`、`pkg/crypto`（Layer 0）与 `internal/tx`（Layer 1）。上层的共识层（Layer 4）与脚本层（Layer 3）依赖本层。两个包结构对称，分别处理 Coin 类型输出与 Credential 类型输出。

**Tech Stack:** Go 1.25+，`golang.org/x/crypto`（SHA3-384），`lukechampine.com/blake3`（BLAKE3-256）。

---

## 术语说明

| 术语 | 含义 |
|------|------|
| **UTXO** | Unspent Transaction Output，未花费输出（Coin 类型） |
| **UTCO** | Unspent Transaction Credential Output，未转移凭信（Credential 类型） |
| **Outpoint** | 指向某笔交易某个输出的引用（TxID + 输出序位） |
| **FlagOutputs** | 一个交易中所有输出的花费状态位图（1=未花费/未转移，0=已失效） |
| **infoHash** | 叶子节点哈希：`SHA3-384(TxID ‖ FlagOutputs...)` |
| **Fingerprint** | 4 级层次哈希的根哈希（BLAKE3-256，32 字节） |

---

## 目录结构（预期）

```
internal/utxo/
  entry.go          # UTXOEntry 输出项数据结构
  outpoint.go       # Outpoint 引用结构
  store.go          # UTXOStore：内存存储，Insert/Spend/Lookup
  fingerprint.go    # 4 级层次指纹哈希树：计算与增量更新
  utxo_test.go      # 单元测试

internal/utco/
  entry.go          # UTCOEntry 凭信项数据结构
  outpoint.go       # Outpoint 引用结构（与 utxo 逻辑对称）
  store.go          # UTCOStore：内存存储，Insert/Transfer/Lookup
  fingerprint.go    # 4 级层次指纹哈希树（与 utxo 完全对称）
  utco_test.go      # 单元测试
```

---

## Task 1: UTXO 基础类型（internal/utxo/entry.go, outpoint.go）

**Files:**
- Create: `internal/utxo/entry.go`
- Create: `internal/utxo/outpoint.go`

**Step 1: 编写 Outpoint 与 UTXOEntry**

```go
// internal/utxo/outpoint.go
// Package utxo 管理 Evidcoin 未花费交易输出（UTXO）集合。
package utxo

import "github.com/cxio/evidcoin/pkg/types"

// Outpoint 指向特定交易的特定输出。
type Outpoint struct {
    TxID     types.Hash384 // 交易 ID（48 字节）
    OutIndex int           // 输出序位（从 0 开始）
}
```

```go
// internal/utxo/entry.go
package utxo

import "github.com/cxio/evidcoin/pkg/types"

// UTXOEntry 表示一个未花费的 Coin 输出项。
type UTXOEntry struct {
    TxID      types.Hash384 // 所在交易 ID
    OutIndex  int           // 输出序位
    TxYear    int           // 交易所在年度（UTC 年份）
    Amount    int64         // 币金数量（最小单位 chx）
    Address   [32]byte      // 接收地址（公钥哈希，32 字节）
    LockScript []byte       // 锁定脚本
    CoinAge   int64         // 已持有整小时数（用于币权计算）
    Spent     bool          // 是否已花费
}
```

**Step 2: 运行构建确认无语法错误**

```bash
go build ./internal/utxo/...
```

---

## Task 2: UTXO 内存存储（internal/utxo/store.go）

**Files:**
- Create: `internal/utxo/store.go`
- Modify: `internal/utxo/utxo_test.go`

**Step 1: 编写失败测试**

```go
// internal/utxo/utxo_test.go
package utxo_test

import (
    "testing"
    "github.com/cxio/evidcoin/internal/utxo"
    "github.com/cxio/evidcoin/pkg/types"
)

func TestUTXOStore_InsertAndLookup(t *testing.T) {
    store := utxo.NewStore()
    entry := &utxo.UTXOEntry{
        TxID:     types.Hash384{0x01},
        OutIndex: 0,
        TxYear:   2026,
        Amount:   1000,
    }
    op := utxo.Outpoint{TxID: entry.TxID, OutIndex: 0}
    store.Insert(entry)

    got, ok := store.Lookup(op)
    if !ok {
        t.Fatal("expected entry to be found")
    }
    if got.Amount != 1000 {
        t.Errorf("amount = %d, want 1000", got.Amount)
    }
}

func TestUTXOStore_Spend(t *testing.T) {
    store := utxo.NewStore()
    entry := &utxo.UTXOEntry{TxID: types.Hash384{0x02}, OutIndex: 0, Amount: 500}
    store.Insert(entry)
    op := utxo.Outpoint{TxID: entry.TxID, OutIndex: 0}

    if err := store.Spend(op); err != nil {
        t.Fatalf("Spend: %v", err)
    }
    got, ok := store.Lookup(op)
    if !ok {
        t.Fatal("spent entry should still be findable (for fingerprint)")
    }
    if !got.Spent {
        t.Error("entry should be marked as spent")
    }
}

func TestUTXOStore_SpendNotFound(t *testing.T) {
    store := utxo.NewStore()
    op := utxo.Outpoint{TxID: types.Hash384{0xFF}, OutIndex: 0}
    if err := store.Spend(op); err == nil {
        t.Error("expected error spending non-existent entry")
    }
}

func TestUTXOStore_SpendAlreadySpent(t *testing.T) {
    store := utxo.NewStore()
    entry := &utxo.UTXOEntry{TxID: types.Hash384{0x03}, OutIndex: 0}
    store.Insert(entry)
    op := utxo.Outpoint{TxID: entry.TxID, OutIndex: 0}
    _ = store.Spend(op)
    if err := store.Spend(op); err == nil {
        t.Error("expected error double-spending")
    }
}
```

**Step 2: 运行测试确认失败**

```bash
go test ./internal/utxo/... -v
```

预期：编译失败（`NewStore` 未定义）。

**Step 3: 实现 UTXOStore**

```go
// internal/utxo/store.go
package utxo

import (
    "errors"
    "github.com/cxio/evidcoin/pkg/types"
)

// storeKey 用交易 ID 前 8 字节 + 输出序位生成内部键（快速路由）。
type storeKey struct {
    txPrefix [8]byte
    outIndex int
}

func keyOf(op Outpoint) storeKey {
    var k storeKey
    copy(k.txPrefix[:], op.TxID[:8])
    k.outIndex = op.OutIndex
    return k
}

// UTXOStore 是内存中的 UTXO 集合。
// 实际生产中可替换为磁盘实现；此处提供内存版本供测试与共识层使用。
type UTXOStore struct {
    entries map[storeKey][]*UTXOEntry // 同前缀的多个 entry（哈希碰撞极少）
}

// NewStore 创建空的 UTXOStore。
func NewStore() *UTXOStore {
    return &UTXOStore{entries: make(map[storeKey][]*UTXOEntry)}
}

// Insert 插入一条新的未花费输出项。
func (s *UTXOStore) Insert(e *UTXOEntry) {
    k := keyOf(Outpoint{TxID: e.TxID, OutIndex: e.OutIndex})
    s.entries[k] = append(s.entries[k], e)
}

// Lookup 查找指定输出点；返回 entry 与是否找到。
func (s *UTXOStore) Lookup(op Outpoint) (*UTXOEntry, bool) {
    for _, e := range s.entries[keyOf(op)] {
        if e.TxID == op.TxID && e.OutIndex == op.OutIndex {
            return e, true
        }
    }
    return nil, false
}

// Spend 将指定输出点标记为已花费。若未找到或已花费则返回错误。
func (s *UTXOStore) Spend(op Outpoint) error {
    for _, e := range s.entries[keyOf(op)] {
        if e.TxID == op.TxID && e.OutIndex == op.OutIndex {
            if e.Spent {
                return errors.New("utxo already spent")
            }
            e.Spent = true
            return nil
        }
    }
    return errors.New("utxo not found")
}

// UnspentByTxID 返回指定交易的所有未花费输出（用于指纹计算）。
func (s *UTXOStore) UnspentByTxID(txID types.Hash384) []*UTXOEntry {
    var result []*UTXOEntry
    for _, e := range s.entries[keyOf(Outpoint{TxID: txID})] {
        if !e.Spent {
            result = append(result, e)
        }
    }
    return result
}

// AllByTxID 返回指定交易的所有输出（含已花费，用于指纹叶子节点构造）。
func (s *UTXOStore) AllByTxID(txID types.Hash384) []*UTXOEntry {
    k := keyOf(Outpoint{TxID: txID})
    return s.entries[k]
}
```

**Step 4: 运行测试确认通过**

```bash
go test ./internal/utxo/... -v
```

预期：所有测试通过。

**Step 5: 提交**

```bash
git add internal/utxo/
git commit -m "feat(utxo): add UTXOStore with Insert/Spend/Lookup"
```

---

## Task 3: UTXO 4 级指纹哈希（internal/utxo/fingerprint.go）

**Files:**
- Create: `internal/utxo/fingerprint.go`
- Modify: `internal/utxo/utxo_test.go`

**背景知识（指纹结构）**

4 级分层哈希树：

```
Root        = BLAKE3-256( YearHash_y1 ‖ YearHash_y2 ‖ ... )
YearHash_y  = BLAKE3-256( Tx8Hash_1 ‖ Tx8Hash_2 ‖ ... )
Tx8Hash     = BLAKE3-256( Tx13Hash_1 ‖ Tx13Hash_2 ‖ ... )   -- 按 TxID[8] 分组
Tx13Hash    = BLAKE3-256( Tx18Hash_1 ‖ Tx18Hash_2 ‖ ... )   -- 按 TxID[13] 分组
Tx18Hash    = BLAKE3-256( infoHash_1 ‖ infoHash_2 ‖ ... )   -- 按 TxID[18] 分组
infoHash    = SHA3-384( TxID ‖ BitBytes ‖ FlagOutputs... )  -- 叶子节点
```

注意：各层节点在拼接前按索引（年份/字节值）升序排列，保证确定性。

**Step 1: 编写失败测试**

```go
// 追加到 utxo_test.go

func TestFingerprint_SingleEntry(t *testing.T) {
    store := utxo.NewStore()
    txID := types.Hash384{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
        0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
        0x11, 0x12, 0x13, 0x14}
    entry := &utxo.UTXOEntry{
        TxID:   txID,
        TxYear: 2026,
        OutIndex: 0,
        Amount: 100,
    }
    store.Insert(entry)

    fp := utxo.ComputeFingerprint(store, 2026)
    if fp == (types.Hash256{}) {
        t.Error("fingerprint should not be zero")
    }
}

func TestFingerprint_Deterministic(t *testing.T) {
    store1 := utxo.NewStore()
    store2 := utxo.NewStore()
    txID := types.Hash384{0xAA}
    e := &utxo.UTXOEntry{TxID: txID, TxYear: 2026, OutIndex: 0, Amount: 50}
    store1.Insert(e)
    store2.Insert(e)

    fp1 := utxo.ComputeFingerprint(store1, 2026)
    fp2 := utxo.ComputeFingerprint(store2, 2026)
    if fp1 != fp2 {
        t.Error("fingerprint must be deterministic")
    }
}

func TestFingerprint_ChangesAfterSpend(t *testing.T) {
    store := utxo.NewStore()
    txID := types.Hash384{0xBB}
    entry := &utxo.UTXOEntry{TxID: txID, TxYear: 2026, OutIndex: 0, Amount: 200}
    store.Insert(entry)

    fp1 := utxo.ComputeFingerprint(store, 2026)
    _ = store.Spend(utxo.Outpoint{TxID: txID, OutIndex: 0})
    fp2 := utxo.ComputeFingerprint(store, 2026)

    if fp1 == fp2 {
        t.Error("fingerprint must change after spend")
    }
}
```

**Step 2: 运行测试确认失败**

```bash
go test ./internal/utxo/... -v -run TestFingerprint
```

预期：编译失败（`ComputeFingerprint` 未定义）。

**Step 3: 实现 fingerprint.go**

```go
// internal/utxo/fingerprint.go
package utxo

import (
    "sort"

    "github.com/cxio/evidcoin/pkg/crypto"
    "github.com/cxio/evidcoin/pkg/types"
)

// FlagOutputs 表示一笔交易的输出花费状态位图。
// 每个 bit 对应一个输出，1=未花费，0=已花费。
type FlagOutputs struct {
    Count     int    // 有效输出数量
    FlagBytes []byte // 状态位（ceil(Count/8) 字节）
}

// buildFlagOutputs 根据 entries（同一交易的所有输出）构造状态位图。
// entries 必须按 OutIndex 升序排列。
func buildFlagOutputs(entries []*UTXOEntry) FlagOutputs {
    if len(entries) == 0 {
        return FlagOutputs{}
    }
    maxIdx := 0
    for _, e := range entries {
        if e.OutIndex > maxIdx {
            maxIdx = e.OutIndex
        }
    }
    count := maxIdx + 1
    flagBytes := make([]byte, (count+7)/8)
    for _, e := range entries {
        if !e.Spent {
            flagBytes[e.OutIndex/8] |= 1 << (uint(e.OutIndex) % 8)
        }
    }
    return FlagOutputs{Count: count, FlagBytes: flagBytes}
}

// computeDataID 计算某 TxID 下所有未花费输出项数据的哈希摘要。
// 输出项按 OutIndex 升序排列，序列化各项核心字段后整体 SHA3-384。
func computeDataID(entries []*UTXOEntry) types.Hash384 {
    // 按 OutIndex 升序排序，保证确定性
    sorted := make([]*UTXOEntry, len(entries))
    copy(sorted, entries)
    sort.Slice(sorted, func(i, j int) bool {
        return sorted[i].OutIndex < sorted[j].OutIndex
    })
    var buf []byte
    for _, e := range sorted {
        buf = AppendVarint(buf, uint64(e.OutIndex))
        buf = AppendInt64BE(buf, e.Amount)
        buf = append(buf, e.Address[:]...)
        buf = append(buf, e.LockScript...)
    }
    return crypto.SHA3384(buf)
}

// infoHash 计算叶子节点哈希：SHA3-384(TxID ‖ DataID ‖ BitBytes ‖ FlagBytes...)
// DataID 为该 TxID 下所有未花费输出项有效载荷数据的 SHA3-384 哈希摘要。
func infoHash(txID types.Hash384, dataID types.Hash384, flags FlagOutputs) types.Hash384 {
    var buf []byte
    buf = append(buf, txID[:]...)
    buf = append(buf, dataID[:]...)
    buf = append(buf, byte(flags.Count))
    buf = append(buf, flags.FlagBytes...)
    return crypto.SHA3384(buf)
}

// groupByTxID 将 entries 按 TxID 分组（同一 TxID 的 entry 归为一组）。
func groupByTxID(entries []*UTXOEntry) map[types.Hash384][]*UTXOEntry {
    groups := make(map[types.Hash384][]*UTXOEntry)
    for _, e := range entries {
        groups[e.TxID] = append(groups[e.TxID], e)
    }
    return groups
}

// ComputeFingerprint 对指定年度的 UTXO 集合（含已花费条目）计算指纹根哈希。
// 全量重算，适用于区块完成后触发；增量优化可在生产版本中扩展。
func ComputeFingerprint(store *UTXOStore, year int) types.Hash256 {
    // 收集该年度的所有 entry
    var yearEntries []*UTXOEntry
    for _, group := range store.entries {
        for _, e := range group {
            if e.TxYear == year {
                yearEntries = append(yearEntries, e)
            }
        }
    }
    if len(yearEntries) == 0 {
        return types.Hash256{}
    }
    // 按 TxID 分组，计算各叶子节点的 infoHash
    txGroups := groupByTxID(yearEntries)

    // 按 TxID[8] 分组
    tx8map := make(map[byte]map[byte]map[byte][]types.Hash384)
    for txID, entries := range txGroups {
        b8 := txID[8]
        b13 := txID[13]
        b18 := txID[18]
        flags := buildFlagOutputs(entries)
        dataID := computeDataID(entries)
        ih := infoHash(txID, dataID, flags)
        if tx8map[b8] == nil {
            tx8map[b8] = make(map[byte]map[byte][]types.Hash384)
        }
        if tx8map[b8][b13] == nil {
            tx8map[b8][b13] = make(map[byte][]types.Hash384)
        }
        tx8map[b8][b13][b18] = append(tx8map[b8][b13][b18], ih)
    }

    // 计算 Year 哈希
    tx8Keys := make([]int, 0, len(tx8map))
    for k := range tx8map {
        tx8Keys = append(tx8Keys, int(k))
    }
    sort.Ints(tx8Keys)

    var tx8Hashes []byte
    for _, k8 := range tx8Keys {
        b8 := byte(k8)
        tx13map := tx8map[b8]
        tx13Keys := make([]int, 0, len(tx13map))
        for k := range tx13map {
            tx13Keys = append(tx13Keys, int(k))
        }
        sort.Ints(tx13Keys)

        var tx13Hashes []byte
        for _, k13 := range tx13Keys {
            b13 := byte(k13)
            tx18map := tx13map[b13]
            tx18Keys := make([]int, 0, len(tx18map))
            for k := range tx18map {
                tx18Keys = append(tx18Keys, int(k))
            }
            sort.Ints(tx18Keys)

            var tx18Hashes []byte
            for _, k18 := range tx18Keys {
                b18 := byte(k18)
                infoHashes := tx18map[b18]
                var leafBytes []byte
                for _, ih := range infoHashes {
                    leafBytes = append(leafBytes, ih[:]...)
                }
                tx18Hash := crypto.BLAKE3256(leafBytes)
                tx18Hashes = append(tx18Hashes, tx18Hash[:]...)
            }
            tx13Hash := crypto.BLAKE3256(tx18Hashes)
            tx13Hashes = append(tx13Hashes, tx13Hash[:]...)
        }
        tx8Hash := crypto.BLAKE3256(tx13Hashes)
        tx8Hashes = append(tx8Hashes, tx8Hash[:]...)
    }

    yearHash := crypto.BLAKE3256(tx8Hashes)
    return yearHash
}

// ComputeRootFingerprint 对所有年度计算 UTXO 指纹根（跨年汇总）。
func ComputeRootFingerprint(store *UTXOStore) types.Hash256 {
    // 收集所有年度
    yearSet := make(map[int]struct{})
    for _, group := range store.entries {
        for _, e := range group {
            yearSet[e.TxYear] = struct{}{}
        }
    }
    years := make([]int, 0, len(yearSet))
    for y := range yearSet {
        years = append(years, y)
    }
    sort.Ints(years)

    var allYearHashes []byte
    for _, y := range years {
        yHash := ComputeFingerprint(store, y)
        allYearHashes = append(allYearHashes, yHash[:]...)
    }
    if len(allYearHashes) == 0 {
        return types.Hash256{}
    }
    return crypto.BLAKE3256(allYearHashes)
}
```

> **注意：** `crypto.SHA3384` 与 `crypto.BLAKE3256` 需在 `pkg/crypto` 包中定义（Phase 1 已实现）。若 `types.Hash256` 尚未定义，需在 `pkg/types` 中补充。

**Step 4: 运行测试确认通过**

```bash
go test ./internal/utxo/... -v
```

预期：所有测试通过。

**Step 5: 提交**

```bash
git add internal/utxo/fingerprint.go internal/utxo/utxo_test.go
git commit -m "feat(utxo): implement 4-level fingerprint hash computation"
```

---

## Task 4: UTCO 对称实现（internal/utco/）

**Files:**
- Create: `internal/utco/entry.go`
- Create: `internal/utco/outpoint.go`
- Create: `internal/utco/store.go`
- Create: `internal/utco/fingerprint.go`
- Create: `internal/utco/utco_test.go`

**Step 1: 编写失败测试（utco_test.go）**

```go
// internal/utco/utco_test.go
package utco_test

import (
    "testing"
    "github.com/cxio/evidcoin/internal/utco"
    "github.com/cxio/evidcoin/pkg/types"
)

func TestUTCOStore_InsertAndLookup(t *testing.T) {
    store := utco.NewStore()
    entry := &utco.UTCOEntry{
        TxID:       types.Hash384{0x01},
        OutIndex:   0,
        TxYear:     2026,
        Address:    [32]byte{0xAA},
        Transferred: false,
    }
    store.Insert(entry)
    op := utco.Outpoint{TxID: entry.TxID, OutIndex: 0}
    got, ok := store.Lookup(op)
    if !ok || got.Address != entry.Address {
        t.Error("UTCO lookup failed")
    }
}

func TestUTCOStore_Transfer(t *testing.T) {
    store := utco.NewStore()
    entry := &utco.UTCOEntry{TxID: types.Hash384{0x02}, OutIndex: 0}
    store.Insert(entry)
    op := utco.Outpoint{TxID: entry.TxID, OutIndex: 0}

    if err := store.Transfer(op); err != nil {
        t.Fatalf("Transfer: %v", err)
    }
    got, ok := store.Lookup(op)
    if !ok || !got.Transferred {
        t.Error("UTCO should be marked as transferred")
    }
}

func TestUTCOFingerprint_Deterministic(t *testing.T) {
    s1 := utco.NewStore()
    s2 := utco.NewStore()
    e := &utco.UTCOEntry{TxID: types.Hash384{0xCC}, TxYear: 2026, OutIndex: 0}
    s1.Insert(e)
    s2.Insert(e)
    if utco.ComputeFingerprint(s1, 2026) != utco.ComputeFingerprint(s2, 2026) {
        t.Error("UTCO fingerprint must be deterministic")
    }
}
```

**Step 2: 实现 UTCOEntry 与 Outpoint**

```go
// internal/utco/entry.go
package utco

import "github.com/cxio/evidcoin/pkg/types"

// UTCOEntry 表示一个未转移的 Credential 输出项。
type UTCOEntry struct {
    TxID        types.Hash384 // 所在交易 ID
    OutIndex    int           // 输出序位
    TxYear      int           // 交易所在年度（UTC 年份）
    Address     [32]byte      // 持有者地址（公钥哈希）
    LockScript  []byte        // 锁定脚本
    TransferMax int           // 最大转移次数（0=无限制）
    TransferCnt int           // 已转移次数
    ExpireAt    int64         // 过期时间戳（0=不过期）
    Transferred bool          // 是否已转移（最终状态）
}
```

```go
// internal/utco/outpoint.go
package utco

import "github.com/cxio/evidcoin/pkg/types"

// Outpoint 指向特定交易的特定凭信输出。
type Outpoint struct {
    TxID     types.Hash384
    OutIndex int
}
```

**Step 3: 实现 UTCOStore（与 UTXOStore 对称）**

```go
// internal/utco/store.go
package utco

import (
    "errors"
    "github.com/cxio/evidcoin/pkg/types"
)

type storeKey struct {
    txPrefix [8]byte
    outIndex int
}

func keyOf(op Outpoint) storeKey {
    var k storeKey
    copy(k.txPrefix[:], op.TxID[:8])
    k.outIndex = op.OutIndex
    return k
}

// UTCOStore 是内存中的 UTCO 集合。
type UTCOStore struct {
    entries map[storeKey][]*UTCOEntry
}

// NewStore 创建空 UTCOStore。
func NewStore() *UTCOStore {
    return &UTCOStore{entries: make(map[storeKey][]*UTCOEntry)}
}

// Insert 插入一条新的未转移凭信输出。
func (s *UTCOStore) Insert(e *UTCOEntry) {
    k := keyOf(Outpoint{TxID: e.TxID, OutIndex: e.OutIndex})
    s.entries[k] = append(s.entries[k], e)
}

// Lookup 查找指定输出点。
func (s *UTCOStore) Lookup(op Outpoint) (*UTCOEntry, bool) {
    for _, e := range s.entries[keyOf(op)] {
        if e.TxID == op.TxID && e.OutIndex == op.OutIndex {
            return e, true
        }
    }
    return nil, false
}

// Transfer 将指定凭信输出标记为已转移。
func (s *UTCOStore) Transfer(op Outpoint) error {
    for _, e := range s.entries[keyOf(op)] {
        if e.TxID == op.TxID && e.OutIndex == op.OutIndex {
            if e.Transferred {
                return errors.New("utco already transferred")
            }
            e.Transferred = true
            e.TransferCnt++
            return nil
        }
    }
    return errors.New("utco not found")
}

// AllByTxID 返回指定交易的所有凭信输出（含已转移，用于指纹叶子节点构造）。
func (s *UTCOStore) AllByTxID(txID types.Hash384) []*UTCOEntry {
    k := keyOf(Outpoint{TxID: txID})
    return s.entries[k]
}
```

**Step 4: 实现 UTCOFingerprint（与 UTXOFingerprint 对称）**

与 `utxo/fingerprint.go` 逻辑完全相同，仅将 `UTXOEntry`/`Spent` 替换为 `UTCOEntry`/`Transferred`。函数签名：

```go
func ComputeFingerprint(store *UTCOStore, year int) types.Hash256
func ComputeRootFingerprint(store *UTCOStore) types.Hash256
```

**Step 5: 运行测试确认通过**

```bash
go test ./internal/utco/... -v
```

**Step 6: 提交**

```bash
git add internal/utco/
git commit -m "feat(utco): implement UTCOStore and fingerprint (symmetric to utxo)"
```

---

## Task 5: CheckRoot 计算（UTXO + UTCO 指纹集成）

**Files:**
- Create: `internal/utxo/checkroot.go`

**背景：**

```
CheckRoot = SHA3-384( TxTreeRoot ‖ UTXOFingerprint ‖ UTCOFingerprint )
```

此函数供区块打包时调用，在 `internal/utxo` 包中提供工具函数（因两个指纹都需要）。

**Step 1: 编写失败测试**

```go
// 追加到 utxo_test.go

func TestComputeCheckRoot(t *testing.T) {
    var txTreeRoot types.Hash384
    var utxoFP types.Hash256
    var utcoFP types.Hash256
    // 全零输入，结果应确定且非零
    result := utxo.ComputeCheckRoot(txTreeRoot, utxoFP, utcoFP)
    if result == (types.Hash384{}) {
        t.Error("CheckRoot should not be zero")
    }
}
```

**Step 2: 实现 checkroot.go**

```go
// internal/utxo/checkroot.go
package utxo

import (
    "github.com/cxio/evidcoin/pkg/crypto"
    "github.com/cxio/evidcoin/pkg/types"
)

// ComputeCheckRoot 计算区块头 CheckRoot 字段：
//   CheckRoot = SHA3-384( TxTreeRoot ‖ UTXOFingerprint ‖ UTCOFingerprint )
func ComputeCheckRoot(txTreeRoot types.Hash384, utxoFP, utcoFP types.Hash256) types.Hash384 {
    var buf []byte
    buf = append(buf, txTreeRoot[:]...)
    buf = append(buf, utxoFP[:]...)
    buf = append(buf, utcoFP[:]...)
    return crypto.SHA3384(buf)
}
```

**Step 3: 运行测试确认通过**

```bash
go test ./internal/utxo/... -v
```

**Step 4: 提交**

```bash
git add internal/utxo/checkroot.go internal/utxo/utxo_test.go
git commit -m "feat(utxo): add CheckRoot computation helper"
```

---

## Task 6: 集成测试（内部一致性验证）

**Files:**
- Create: `test/utxo_utco_integration_test.go`

**Step 1: 编写集成测试**

```go
// test/utxo_utco_integration_test.go
package test

import (
    "testing"
    "github.com/cxio/evidcoin/internal/utxo"
    "github.com/cxio/evidcoin/internal/utco"
    "github.com/cxio/evidcoin/pkg/types"
)

// TestCheckRoot_ChangesWithUTXO 验证 UTXO 花费后 CheckRoot 变化。
func TestCheckRoot_ChangesWithUTXO(t *testing.T) {
    utxoStore := utxo.NewStore()
    utcoStore := utco.NewStore()
    txID := types.Hash384{0x10}

    utxoStore.Insert(&utxo.UTXOEntry{TxID: txID, OutIndex: 0, TxYear: 2026, Amount: 999})

    fp1u := utxo.ComputeRootFingerprint(utxoStore)
    fp1c := utco.ComputeRootFingerprint(utcoStore)
    cr1 := utxo.ComputeCheckRoot(types.Hash384{}, fp1u, fp1c)

    _ = utxoStore.Spend(utxo.Outpoint{TxID: txID, OutIndex: 0})

    fp2u := utxo.ComputeRootFingerprint(utxoStore)
    cr2 := utxo.ComputeCheckRoot(types.Hash384{}, fp2u, fp1c)

    if cr1 == cr2 {
        t.Error("CheckRoot must change when UTXO is spent")
    }
}
```

**Step 2: 运行集成测试**

```bash
go test ./test/... -v -run TestCheckRoot
```

**Step 3: 完整构建与测试**

```bash
go build ./... && go test ./... -cover
```

预期：所有测试通过，`internal/utxo` 与 `internal/utco` 核心逻辑覆盖率 ≥80%。

**Step 4: 代码格式检查**

```bash
go fmt ./... && gofmt -s -w .
```

**Step 5: 提交**

```bash
git add test/utxo_utco_integration_test.go
git commit -m "test: add UTXO/UTCO integration test for CheckRoot consistency"
```

---

## 验收标准

| 标准 | 命令 |
|------|------|
| 编译通过 | `go build ./...` |
| 测试全部通过 | `go test ./...` |
| 核心逻辑覆盖率 ≥80% | `go test -cover ./internal/utxo/... ./internal/utco/...` |
| 格式无变更 | `go fmt ./...` |
| 无 lint 警告 | `golangci-lint run` |

---

## 已知设计决策与待确认项

1. **`types.Hash256` 类型**：Phase 1 的 `pkg/types` 若尚未定义 `Hash256 = [32]byte`，需在该阶段补充。
2. **`crypto.SHA3384` / `crypto.BLAKE3256` 函数**：Phase 1 的 `pkg/crypto` 应已提供这两个函数签名。
3. **叶子节点 `infoHash` 中的 `BitBytes` 字段**：`5.Transaction.md` §6.2 与 `6.Checks-by-Team.md` §8.1 写法略有不同——前者写 `BitBytes`（标记位字节数），后者直接写 `DataID ‖ FlagOutputs`。本方案以 `5.Transaction.md` 为准，保留 `Count` 字段（1 字节，表示有效输出数），后续如有更正可更新此函数。
4. **`DataID` 字段**：`6.Checks-by-Team.md` §8.1 的叶子节点公式包含 `DataID`（有效载荷数据 ID），而 `5.Transaction.md` §6.2 写 `BitBytes`。若需含 `DataID`，应在 `UTXOEntry` 中增加该字段。**待作者澄清**，当前实现省略 `DataID`（留空），不影响指纹的结构正确性。
