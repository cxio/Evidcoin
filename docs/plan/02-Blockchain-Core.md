# Blockchain Core Implementation Plan

**Goal:** 实现只负责区块头链、链身份、年块边界和最小入块验证的区块链核心层。

**Architecture:** `internal/blockchain/` 依赖 `pkg/types` 和 `pkg/crypto`，不执行交易、脚本、状态转移、PoH 或网络逻辑。它只验证区块头自身编码、`BlockID`、高度连续性、`PrevBlock` 衔接和显式链身份。

**Tech Stack:** Go 1.26.2、`pkg/types`、`pkg/crypto`、内存 store 测试替身、表驱动测试。

---

## 来源提案

- `docs/proposal/05.Blockchain-Core.md`
- 依赖 `docs/proposal/01.Types-And-Encoding.md`、`02.Cryptography-And-Hashing.md`、`03.Identifiers-And-Constants.md`、`04.Hash-Trees.md`
- DEC-0003：区块头字段编码顺序、`YearBlock` 省略规则、常规 112B / 年块 160B。
- DEC-0002（引用）：`block.header`、`checkroot` 域标签 + SHA3-384 profile。
- CheckRoot 状态根口径（proposal 05 §2）：`UTXORoot`/`UTCORoot` 取**前一区块完成后**的状态指纹；创世 `h==0` 关联 UTXO/UTCO 为空根（第 05 章空根规则）。

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
| `internal/blockchain/sizelimit.go` | 区块尺寸限额曲线（含解锁脚本、不含见证） |
| `internal/blockchain/genesis.go` | 创世区块头工件（边界） |
| `internal/blockchain/errors.go` | 错误定义 |

## 数据设计

`BlockHeader` 字段顺序必须固定（DEC-0003）：

| 序 | 字段 | 类型/宽度 | 说明 |
|----|------|-----------|------|
| 1 | `Version` | `uint32` 大端 | 创世固定 1 |
| 2 | `Height` | `uint32` 大端 | 代替时间戳 |
| 3 | `PrevBlock` | `[48]byte` | 前块 ID |
| 4 | `CheckRoot` | `[48]byte` | 校验根 |
| 5 | `Stakes` | `uint64` 大端 | 区块收录交易币权（币量×币龄）合计，单位「聊时」，溢出截断 |
| 6 | `YearBlock` | `[48]byte` | **仅 `Height % BlocksPerYear == 0` 时存在**，否则完全省略（不编码全零占位） |

- 区块头尺寸：常规 `4+4+48+48+8 = 112` 字节；年块多 48 字节 = **160** 字节（DEC-0003）。
- 无时间戳字段：时间戳由高度与出块间隔（6min）从创世时间戳推导。
- `Stakes` 经济语义（币龄 ≥1 小时、花费归零、`-63` 区块铸凭因子）由第 07 章承载；本层只固定编码与字段语义，不实现经济计算。

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

实现查询和验证辅助。`YearBlock` 仅在 `Height % BlocksPerYear == 0` 时存在并参与 BlockID 前像；非年块前像不含该字段。创世（高度 0）`YearBlock` **存在但全零**（`0 % BlocksPerYear == 0` 故字段存在，无前一年块故值全零，proposal 05 §9）。

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

- 输入 `TransactionTreeRoot || UTXORoot || UTCORoot` 得到 48B `CheckRoot`；三者均为 32B 树根。
- 改变任一输入会改变结果。
- UTXO 与 UTCO 输入顺序调换会改变结果。
- `h == 0` 时调用方传入空状态指纹，组合结果稳定可复现。
- 普通区块高度 `H > 0` 的 CheckRoot 测试必须从状态指纹提供者读取上一高度 `H-1` 的 UTXO/UTCO 指纹，而不是当前区块执行后的指纹。

**Step 2: 实现**

只组合已给定的根，不在核心层计算交易树或状态树。组合固定为 `SHA3-384(DomainTag("checkroot") || TreeRoot || UTXORoot || UTCORoot)`（proposal 05 §2）。若提供高度感知辅助函数，`h == 0` 使用空状态指纹（第 05 章空根），普通区块读取上一高度 `H-1` 完成后的状态指纹。

**Step 3: 验证并提交**

```bash
go test ./internal/blockchain -run 'TestCheckRoot' -v
git add internal/blockchain/checkroot.go internal/blockchain/checkroot_test.go
git commit -m "feat: add check root composition"
```

## Task 7: 区块尺寸限额曲线

**Files:**
- Create: `internal/blockchain/sizelimit.go`
- Create: `internal/blockchain/sizelimit_test.go`

**Step 1: 写失败测试**

测试（proposal 05 §8，限额仅约束数据量，包含解锁脚本但不含见证）：

- 第 1~3 月（`7305×3 = 21915` 块）固定 **1MB**。
- 第 4~12 月每月递增 1MB，至 **10MB**；月块数 `87661/12 ≈ 7305`，末月容纳尾数（`7305×11 + 7306 = 87661`）。
- 第 2 年起每恒星年 87661 块逐年递增 **1MB**。
- 边界高度（0/21914/21915/87660/87661/175321 等）限额取值准确。

**Step 2: 实现并提交**

实现 `BlockSizeLimit(height uint32) int` 等辅助；尺寸口径含解锁脚本、不含见证。

```bash
go test ./internal/blockchain -run 'TestBlockSizeLimit' -v
git add internal/blockchain/sizelimit.go internal/blockchain/sizelimit_test.go
git commit -m "feat: add block size limit curve"
```

## 创世工件与手动切链（边界）

- **创世区块头工件（proposal 05 §9）：** `Version=1`、`Height=0`、`PrevBlock` 全零、`Stakes=0`、`YearBlock` 存在但全零；`CheckRoot` 按常规计算（仅一笔 Coinbase，关联 UTXO/UTCO 为空根）。创世 BlockID（`Genesis-ID`）与创世时间戳属 **C-9 待决**，裁决前以占位标注阻塞（见 `/memories/repo/docs-genesis-boundary.md`：不虚构创世时间戳/初始输出/启动归属）。本层仅固定创世**实现边界与验证规则**。
- **手动切链（proposal 05 §7）：** 全球分区 >2 小时（>20 块分叉）时两链均可能合法，由用户手动选择认可主链；人工切链是社会性「用脚投票」，**非算法逻辑**，本包不自动重组长期分叉。

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
