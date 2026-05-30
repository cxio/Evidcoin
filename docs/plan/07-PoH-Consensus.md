# PoH Consensus Implementation Plan

**Goal:** 实现历史证明共识（PoH）的铸凭哈希、铸造资格（铸凭交易）判定、择优凭证（MintProof）、择优池与三段同步、铸造者三段身份验证、创世工件结构与初段窗口规则。**不**承载分叉选择与出块时序（见 08）。

**Architecture:** `internal/consensus/`（Layer 4）只承载铸造资格与择优逻辑，通过接口读取区块头链、交易片段、状态指纹与签名验证结果。外部公共服务返回的数据必须先验证，不得作为可信输入。本篇与 08 共用 `internal/consensus` 包但职责分离：本篇=铸造资格+择优，08=分叉+出块时序。

**Tech Stack:** Go 1.26.2、`pkg/types`、`pkg/crypto`（BLAKE3-256 `mint.hash` 域）、`pkg/hashtree`、`internal/blockchain`、`internal/tx` 接口、表驱动测试。

---

## 来源提案

- `docs/proposal/11.PoH-Consensus.md`（主源）
- 依赖 `docs/proposal/05.Blockchain-Core.md`（区块头、`Stakes`、`CheckRoot`）、`06.Transaction-Model.md`（交易头、Coinbase、`MintPKHash`/`LeadPKHash`）、`04.Hash-Trees.md`（验证路径、输入根）、`08.Signatures-And-Witness.md`（铸造者签名）、`02.Cryptography-And-Hashing.md`（域标签/哈希 profile）。
- **DEC-0301**：铸凭哈希前像字段顺序与编码、`X = BE(minimal_unsigned(BlockHeight × Mix))`（`Mix=0x517cc1b727220a95`）、`Stakes==0` 时 `X=0x00`、`MintProof` 五字段顺序、择优三级升序（`MintHash → TxID → PubKey`）、`MintPKHash`/`LeadPKHash` 两路身份、Coinbase 铸凭资格。
- **DEC-0302**：创世工件双形式（`genesis.bin`/`genesis.json`）、创世 `MintHash` 全零、初段评参高度、初段铸凭高度放宽、#1/#2 规则、初段 Coinbase 资格、边界测试点。

## 包边界

| 包 | 职责 | 禁止事项 |
|----|------|----------|
| `internal/consensus` | 铸凭哈希、铸造资格、`MintProof`、择优池/同步、铸造者验证、创世/初段窗口 | 不实现 P2P 线格式、不实现分叉选择/出块时序（见 08）、不直接 import 公共服务客户端 |

`internal/consensus` 属 Layer 4，依赖 Layer 0-3 的接口/稳定类型，不被低层包反向 import。

## 建议文件

| 文件 | 内容 |
|------|------|
| `internal/consensus/mint_hash.go` | 铸凭哈希前像、`X` 编码、择优三级排序 |
| `internal/consensus/mint_eligibility.go` | 铸凭交易高度窗口（正常期/初段）判定 |
| `internal/consensus/mint_proof.go` | `MintProof` 五字段编解码 |
| `internal/consensus/mint_identity.go` | `MintPKHash`/`LeadPKHash` 两路身份规则 |
| `internal/consensus/best_pool.go` | 择优池容量 20、升序、授权同步成员 |
| `internal/consensus/sync_pool.go` | 三段同步（裁判/预合并/终合并）、防重放 |
| `internal/consensus/minter_verify.go` | 铸造者三段验证、年初回退检索 |
| `internal/consensus/genesis.go` | 创世工件结构、初段窗口规则、占位边界 |
| `internal/consensus/errors.go` | 错误定义 |

## Task 1: 铸凭哈希与择优排序

**Files:**
- Create: `internal/consensus/mint_hash.go`
- Create: `internal/consensus/mint_hash_test.go`
- Create: `internal/consensus/errors.go`

**Step 1: 写失败测试**（proposal 11 §2 / DEC-0301）

- 前像顺序固定：`BLAKE3-256( DomainTag("mint.hash") || MintPubKey || MintTxID || Stakes(BE u64) || RefMintHash || X )`，输出 32B。
- `X = BE(minimal_unsigned(BlockHeight × Mix))`，`Mix=0x517cc1b727220a95`；测试向量覆盖：`Stakes==0` 不影响 `X`（`X` 仅由高度×Mix 决定），但**注意** DEC-0301 另规定 `Stakes==0` 时 `X` 编码为单字节 `0x00`——以 DEC-0301 文本为准，测试覆盖两条规则的边界用例并就近标注来源行。
- `RefMintHash`：评参块 Coinbase 的铸凭哈希；创世块/初段无 `Minter` 时取全零 32B。
- 择优对比：按 `MintHash` 32B 字典序升序，值小者胜；相等按完整 `TxID` 升序，再按 `MintPubKey` 字节升序（三级）。
- 排序必须按 unsigned lexicographic byte order，不得按 hex 字符串。

> **待澄清标注：** 测试中对「`Stakes==0` 时 `X` 取 `0x00`」与「`X=BE(minimal_unsigned(height×Mix))`」两条 DEC-0301 表述的精确叠加关系，须就近引用 DEC-0301 对应条目；如发现两条规则文本冲突，暂停并按文档同步流程回写 proposal/decision，不在 Plan 中私自选值。

**Step 2: 实现并提交**

实现 `MintHashPreimage.CanonicalBytes()`、`ComputeMintHash()`、`RankMintCandidates()`；按 DEC-0301 固定字段顺序与 `X` 编码，不为已决场景返回占位错误。

```bash
go test ./internal/consensus -run 'TestMintHash' -v
git add internal/consensus/mint_hash.go internal/consensus/mint_hash_test.go internal/consensus/errors.go
git commit -m "feat: add mint hash and best ranking"
```

## Task 2: 铸凭交易资格窗口

**Files:**
- Create: `internal/consensus/mint_eligibility.go`
- Create: `internal/consensus/mint_eligibility_test.go`

**Step 1: 写失败测试**（proposal 11 §1·§7 / DEC-0302）

- 正常期：`h := currentHeight - txHeight`，资格为 `h > 239 && h <= 80000`（即区块高度区间 `[-80000, -240]`）。
- 资格只依赖**交易所在区块高度**，交易自身 `Timestamp` 变化不得影响判定。
- 铸凭交易必须为**已确认**交易；当前待铸区块内交易不得自引用为铸凭交易。
- 评参区块取链末端 `-8` 号区块铸凭哈希；币权销毁取 `-32` 号区块头 `Stakes`（聪时）。
- 边界测试点：`currentHeight - txHeight ∈ {239, 240, 80000, 80001}`。

**Step 2: 实现并提交**

```bash
go test ./internal/consensus -run 'TestMintEligibility' -v
git add internal/consensus/mint_eligibility.go internal/consensus/mint_eligibility_test.go
git commit -m "feat: add mint tx eligibility window"
```

## Task 3: MintProof 编解码与铸造者身份

**Files:**
- Create: `internal/consensus/mint_proof.go`
- Create: `internal/consensus/mint_identity.go`
- Create: `internal/consensus/mint_proof_test.go`
- Create: `internal/consensus/mint_identity_test.go`

**Step 1: 写失败测试**（proposal 11 §3·§4 / DEC-0301）

`MintProof` 五字段顺序（冻结）：

- `TxHeight uint32` → `TxID [48]byte` → `MintPubKey bytes` → `MintHash [32]byte` → `Signature bytes`。
- `MintHash` 置于签名前仅便于检索/预筛选；签名验证仍以**重新计算**的铸凭哈希为准。
- 凭证签名是铸造资格证明，不是输入项花费签名，不进入见证剪枝范畴（见 04 章/见证）。

身份两路（`mint_identity.go`）：

- 交易头**含** `MintPKHash`：`MintPubKey` 的公钥哈希必须等于 `MintPKHash`；不要求该公钥参与输入根。
- 交易头**不含** `MintPKHash`：`MintPubKey` 公钥哈希作 `LeadPKHash`，必须参与输入根验证 `BLAKE3-256(ListHash || LeadPKHash)`。
- Coinbase 无输入项、无 `LeadPKHash`；只要 Coinbase 头显式设 `MintPKHash` 且已确认即可参与竞争。

**Step 2: 实现并提交**

```bash
go test ./internal/consensus -run 'Test(MintProof|MintIdentity)' -v
git add internal/consensus/mint_proof.go internal/consensus/mint_identity.go internal/consensus/mint_proof_test.go internal/consensus/mint_identity_test.go
git commit -m "feat: add mint proof and identity rules"
```

## Task 4: 择优池

**Files:**
- Create: `internal/consensus/best_pool.go`
- Create: `internal/consensus/best_pool_test.go`

**Step 1: 写失败测试**（proposal 11 §5）

- 容量最多 **20**，按 `MintHash` 升序（值小者优）。
- 新候选更优时进入池并挤出最差；重复候选去重。
- 相同 `MintHash` 不同候选按 `TxID → PubKey` 三级排序继续区分（DEC-0301）。
- 预选窗口：评参区块为 `-8` 号，候选者最多提前 7 个区块时段得知对比目标。
- 授权同步成员为池中**后 15 名**；前 5 名不具同步发起权。

**Step 2: 实现并提交**

```bash
go test ./internal/consensus -run 'TestBestPool' -v
git add internal/consensus/best_pool.go internal/consensus/best_pool_test.go
git commit -m "feat: add poh best pool"
```

## Task 5: 三段同步与防重放

**Files:**
- Create: `internal/consensus/sync_pool.go`
- Create: `internal/consensus/sync_pool_test.go`

**Step 1: 写失败测试**（proposal 11 §5）

- 分段时序：新块创建后到 `-6` 号前 **5** 个区块时段为「广播收集」（30min 裕度）；成为 `-6` 号后 **2** 个区块时段「同步优化」；成为 `-8` 号定型为评参区块。
- 仅后 15 名授权节点可发起同步；每授权节点对同一目标池仅一次同步权。
- 三段流程：裁判池（本地池副本 + 目标池合并，判断对端是否在后 15 位）→ 预合并（截止后合并暂存目标池，取后 15 位为授权集）→ 终合并（合并授权集成员目标池得最终择优池）。
- 新上线节点本地池为空不影响裁判池创建。
- 合并后仍按 `MintHash` 排序并截断到 20；签名错误拒绝；同步为**概略性**而非唯一性约束（不唯一时由 08 章分叉竞争收敛）。

**Step 2: 实现并提交**

```bash
go test ./internal/consensus -run 'TestSyncPool' -v
git add internal/consensus/sync_pool.go internal/consensus/sync_pool_test.go
git commit -m "feat: add poh sync pool"
```

## Task 6: 铸造者三段验证

**Files:**
- Create: `internal/consensus/minter_verify.go`
- Create: `internal/consensus/minter_verify_test.go`

**Step 1: 写失败测试**（proposal 11 §6）

前提：客户端持有区块头链；竞争者除 `MintProof` 外提供 TxID→`TreeRoot` 验证路径（叶含 3 字节前置序号）；若未设 `MintPKHash` 还需提供 `ListHash`。三段：

1. **交易 ID 合法：** 高度在 `[-80000,-240]`；从 `MintPubKey` 算公钥哈希；按检索的 `MintPKHash` 是否为空区分身份（空则作 `LeadPKHash` 参与输入根）；重算交易头哈希，验证与 `MintProof.TxID` 匹配。
2. **属于目标区块：** 按高度从头链取 `CheckRoot`，结合目标区块 UTXO/UTCO 指纹与验证路径核实交易 ID 属该区块（`CheckRoot` 匹配，见 05 章 §2）。
3. **铸造者身份：** 重算 `mintHash` 与 `MintProof.MintHash` 比对；验证对 `mintHash` 的签名有效。

- 年初回退检索：高度在真实年初 1 天内、当年度检索无果时，可年度减一再检索一次（区块不收录未来交易，仅需考虑年初回退）。

**Step 2: 实现接口化数据源**

```go
type MintDataSource interface {
    MintTransactionHeader(id types.TxID) (...)
    CheckRootAt(height uint32) (...)
    InclusionPath(id types.TxID) (...)
}
```

不直接 import Blockqs 客户端；外部数据先验证再使用。

**Step 3: 验证并提交**

```bash
go test ./internal/consensus -run 'TestMinterVerify' -v
git add internal/consensus/minter_verify.go internal/consensus/minter_verify_test.go
git commit -m "feat: verify minter three stages"
```

## Task 7: 创世工件与初段窗口（含 C-9 占位）

**Files:**
- Create: `internal/consensus/genesis.go`
- Create: `internal/consensus/genesis_test.go`

**Step 1: 写失败测试**（proposal 11 §7 / DEC-0302）

- **创世工件结构**（客户端硬编码、发布后不可变）：创世区块头完整编码、创世 Coinbase 完整交易体、铸造者对 Coinbase 的签名、铸造者对 `CheckRoot` 的签名、创世声明 `FreeData`；双形式 `genesis.bin`（权威）+ `genesis.json`（仅审阅）。
- 创世 Coinbase 的 `Minter` 省略；创世块 `MintHash` 定义为 32B 全零，仅用于 #1~#7 评参引用语义，不表示有效择优凭证。
- 初段评参高度：`currentHeight < 8` 取 `0`（创世块）；`>= 8` 取 `currentHeight - 8`。
- 初段铸凭高度放宽：`currentHeight < 480` 时 `txHeight < currentHeight`（仍须已确认）；`>= 480` 用正常 `h>239 && h<=80000`。
- #1 由创世 `MintPKHash` 铸造；#2 起允许基于已确认初段交易竞争；#1/#2 特殊逻辑隔离，不泄漏到正常高度逻辑。
- 初段从 #2 起，已确认 Coinbase 与普通交易都可作铸凭交易；当前待铸区块内交易不得作当前区块铸凭交易。
- 边界测试点覆盖高度 **0/1/2/7/8/239/240/479/480**。

**Step 2: 实现（C-9 占位阻塞）**

- 本任务仅固定创世**工件结构与初段窗口规则**；`genesis.bin` 的具体字节取值（创世时间戳、mainnet `Genesis-ID`）属 **C-9 待决**，裁决前以占位常量标注阻塞，不虚构具体值（见 `/memories/repo/docs-genesis-boundary.md`）。
- 与 02 章创世工件边界保持一致。

**Step 3: 验证并提交**

```bash
go test ./internal/consensus -run 'TestGenesis' -v
git add internal/consensus/genesis.go internal/consensus/genesis_test.go
git commit -m "feat: add genesis artifacts and initial window"
```

## 待决问题

- **C-9 创世具体参数（与 02 章协调）：** 创世时间戳与 mainnet `Genesis-ID` 尚未冻结。本篇承载创世工件结构与初段窗口规则（DEC-0302 已定），但创世硬编码 Task 7 与初段评参引用阻塞于占位值，裁决后回填。

## 阶段门禁/验收

进入条件：02（区块）、03（交易）、05（状态）、08（签名）接口可用。

运行：

```bash
go fmt ./...
go test ./internal/consensus
go test ./...
go build ./...
go mod tidy
go mod verify
golangci-lint run
```

通过标准：

- 铸凭哈希 `BLAKE3-256(mint.hash 域 || …)` 与择优三级排序跨平台稳定。
- 铸凭资格只依赖交易所在区块高度，不依赖交易自身时间戳；窗口边界 239/240/80000/80001 覆盖。
- `MintProof` 五字段顺序、`MintPKHash`/`LeadPKHash` 两路身份测试覆盖。
- 择优池容量 20、后 15 名授权同步、三段同步流程测试覆盖；同步为概略性，不作唯一性断言。
- 铸造者三段验证含年初回退检索覆盖。
- 初段窗口边界高度 0/1/2/7/8/239/240/479/480 覆盖；#1/#2 特殊逻辑隔离。
- 创世工件以结构+占位实现，`Genesis-ID`/创世时间戳（C-9）不虚构，相关 Task 标注阻塞。
- `internal/consensus` 不实现分叉选择/出块时序（属 08），不被低层包反向 import。
