# Blockchain Core Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现只负责区块头链、链身份、年块边界和最小入块验证的区块链核心层。

**Architecture:** `internal/blockchain/` 依赖 `pkg/types` 和 `pkg/crypto`，不执行交易、脚本、状态转移、PoH 或网络逻辑。它只验证区块头自身编码、`BlockID`、高度连续性、`PrevBlock` 衔接和显式链身份。

**Tech Stack:** Go 1.26.2、`pkg/types`、`pkg/crypto`、内存 store 测试替身、表驱动测试。

---

## 来源提案

- `docs/proposal/05.Blockchain-Core.md`
- 依赖 `docs/proposal/01.Types-And-Encoding.md`
- 依赖 `docs/proposal/02.Cryptography-And-Hashing.md`
- 依赖 `docs/proposal/03.Identifiers-And-Constants.md`

## 非目标

- 不验证交易合法性。
- 不计算交易树。
- 不执行脚本。
- 不判断 PoH 铸造资格。
- 不自动重组长期分叉。
- 不实现 P2P 同步。

## 建议文件

| 文件 | 内容 |
|------|------|
| `internal/blockchain/header.go` | `BlockHeader` 字段、规范化编码 |
| `internal/blockchain/hash.go` | `BlockID` 计算 |
| `internal/blockchain/identity.go` | `ProtocolID`、`ChainID`、`GenesisID`、`BoundID` |
| `internal/blockchain/store.go` | 区块头存储接口 |
| `internal/blockchain/memstore_test.go` | 测试用内存 store |
| `internal/blockchain/chain.go` | tip、按高度和 ID 查询、入块 |
| `internal/blockchain/validate.go` | 最小头链验证 |
| `internal/blockchain/yearblock.go` | 年块边界和 `YearBlock` 查询 |
| `internal/blockchain/checkroot.go` | `CheckRoot` 组合函数 |
| `internal/blockchain/errors.go` | 错误定义 |

## 数据设计

`BlockHeader` 字段顺序必须固定：

1. `Version`
2. `Height`
3. `PrevBlock`
4. `CheckRoot`
5. `Stakes`
6. `YearBlock`

`Stakes` 的具体单位和宽度如果在 Proposal 中仍未固定，先定义为显式类型并用 `TODO(spec)` 标记，测试只覆盖编码稳定性，不覆盖经济语义。

## Task 1: 区块头编码与 BlockID

**Files:**
- Create: `internal/blockchain/header.go`
- Create: `internal/blockchain/hash.go`
- Create: `internal/blockchain/header_test.go`
- Create: `internal/blockchain/hash_test.go`

**Step 1: 写失败测试**

测试：

- 相同头字段得到相同规范化字节。
- 改变任一字段会改变 `BlockID`。
- 字段顺序固定，手工拼接向量与实现一致。
- `BlockID` 输出 48B。

**Step 2: 运行测试确认失败**

```bash
go test ./internal/blockchain -run 'Test(BlockHeader|BlockID)' -v
```

**Step 3: 最小实现**

实现 `BlockHeader.CanonicalBytes()` 和 `BlockHeader.ID()`。不要让 `BlockHeader` import 交易、状态或共识包。

**Step 4: 验证并提交**

```bash
go test ./internal/blockchain -run 'Test(BlockHeader|BlockID)' -v
git add internal/blockchain/header.go internal/blockchain/hash.go internal/blockchain/header_test.go internal/blockchain/hash_test.go
git commit -m "feat: add block header hashing"
```

## Task 2: 链身份

**Files:**
- Create: `internal/blockchain/identity.go`
- Create: `internal/blockchain/identity_test.go`

**Step 1: 写失败测试**

测试：

- 链身份编码包含 `ProtocolID`、`ChainID`、`GenesisID`。
- `BoundID` absent/present 编码不同。
- 签名消息调用方可以取得稳定的 identity bytes。

**Step 2: 实现**

定义 `ChainIdentity`。不要在核心层定义签名消息语义，只提供身份材料。

**Step 3: 验证并提交**

```bash
go test ./internal/blockchain -run TestChainIdentity -v
git add internal/blockchain/identity.go internal/blockchain/identity_test.go
git commit -m "feat: add chain identity encoding"
```

## Task 3: HeaderStore 接口与内存实现测试

**Files:**
- Create: `internal/blockchain/store.go`
- Create: `internal/blockchain/memstore_test.go`
- Create: `internal/blockchain/store_test.go`

**Step 1: 写失败测试**

测试：

- 按高度查询。
- 按 `BlockID` 查询。
- 查询 tip。
- 缺失头返回 `ErrHeaderNotFound`。

**Step 2: 实现接口**

只定义接口，生产存储延后。测试内存 store 放 `_test.go`，避免误当生产存储。

**Step 3: 验证并提交**

```bash
go test ./internal/blockchain -run 'TestHeaderStore' -v
git add internal/blockchain/store.go internal/blockchain/memstore_test.go internal/blockchain/store_test.go
git commit -m "feat: define block header store"
```

## Task 4: 最小入块验证

**Files:**
- Create: `internal/blockchain/chain.go`
- Create: `internal/blockchain/validate.go`
- Create: `internal/blockchain/errors.go`
- Create: `internal/blockchain/chain_test.go`

**Step 1: 写失败测试**

测试：

- 创世头可初始化。
- 新头高度必须为 tip + 1。
- `PrevBlock` 必须等于当前 tip ID。
- 同高度不同 ID 不自动替换。
- `BlockID` 重算不匹配拒绝，如果 API 接收外部 ID。
- 缺失中间头时拒绝衔接。

**Step 2: 实现**

实现 `Chain.AddHeader`、`Chain.Tip`、`Chain.HeaderByHeight`、`Chain.HeaderByID`。不要加入分叉选择逻辑。

**Step 3: 验证并提交**

```bash
go test ./internal/blockchain -run 'TestChain' -v
git add internal/blockchain/chain.go internal/blockchain/validate.go internal/blockchain/errors.go internal/blockchain/chain_test.go
git commit -m "feat: add minimal header chain validation"
```

## Task 5: 年块边界与恢复衔接

**Files:**
- Create: `internal/blockchain/yearblock.go`
- Create: `internal/blockchain/yearblock_test.go`

**Step 1: 写失败测试**

测试：

- `height % BlocksPerYear == 0` 识别年度边界。
- 非年度边界返回最近年块引用。
- 缺失年块时返回明确错误。
- 恢复头必须与前后头衔接。

**Step 2: 实现**

实现查询和验证辅助。`YearBlock` 在创世和非边界高度的确切含义如仍未定，代码用文档注释说明采用的临时规则，并在 `08-Open-Questions-And-Acceptance.md` 关联未决项。

**Step 3: 验证并提交**

```bash
go test ./internal/blockchain -run 'TestYearBlock' -v
git add internal/blockchain/yearblock.go internal/blockchain/yearblock_test.go
git commit -m "feat: add year block helpers"
```

## Task 6: CheckRoot 组合函数

**Files:**
- Create: `internal/blockchain/checkroot.go`
- Create: `internal/blockchain/checkroot_test.go`

**Step 1: 写失败测试**

测试：

- 输入 `TransactionTreeRoot || UTXORoot || UTCORoot` 得到 48B `CheckRoot`。
- 改变任一输入会改变结果。
- UTXO 与 UTCO 输入顺序调换会改变结果。

**Step 2: 实现**

只组合已给定的根，不在核心层计算交易树或状态树。

**Step 3: 验证并提交**

```bash
go test ./internal/blockchain -run 'TestCheckRoot' -v
git add internal/blockchain/checkroot.go internal/blockchain/checkroot_test.go
git commit -m "feat: add check root composition"
```

## 阶段验收

运行：

```bash
go fmt ./...
go test ./internal/blockchain ./pkg/types ./pkg/crypto
go test ./...
go build ./...
go mod tidy
go mod verify
golangci-lint run
```

通过标准：

- `internal/blockchain` 不 import `internal/tx`、`internal/utxo`、`internal/utco`、`internal/script`、`internal/consensus`。
- 区块核心测试覆盖头编码、ID、tip、衔接、年块和 CheckRoot。
- 同高度冲突不自动切换主链。
- 长期分叉裁决不在本包实现。
