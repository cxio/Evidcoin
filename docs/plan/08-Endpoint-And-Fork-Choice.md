# Endpoint Convention & Fork Choice Implementation Plan

**Goal:** 实现端点约定与分叉选择：出块时序（6min/15s/30s）、区块发布三段、区块竞争两步归一化（同铸造者多签 + 2 倍交易量）、分叉链段竞争（31/16/20）、长度 20 临界裁决、RandomX 平局裁决、交易回收，以及零确认/最低费/错时延迟/交易过期等端点约定。

**Architecture:** `internal/consensus/`（Layer 4）承载分叉选择与端点时序，与 07 篇共用包但职责分离：07=铸造资格+择优，本篇=分叉+出块时序。**协议**（必须严格遵守、可验证）与**共约**（自觉遵守、不可/无需验证）在类型与测试中显式区分；共约不得作为拒绝合法区块的依据。

**Tech Stack:** Go 1.26.2、`pkg/types`、`pkg/crypto`、`internal/blockchain`、`internal/consensus`（07 接口）、官方 RandomX（CGO 封装，平局裁决专用）、表驱动测试。

---

## 来源提案

- `docs/proposal/12.Endpoint-And-Fork-Choice.md`（主源）
- 依赖 `docs/proposal/03.Identifiers-And-Constants.md`（常量）、`05.Blockchain-Core.md`（区块头、`Stakes`）、`11.PoH-Consensus.md`（铸凭哈希、择优池）、`08.Signatures-And-Witness.md`（铸造者签名与 `CheckRoot` 签名者一致性）。
- **DEC-0303**：完整链段比较算法、同铸造者多签归一化、「铸造者个人可得收益」定义、2 倍交易量确定性算法（同时 `>2x`）、RandomX profile（`v2.0.1`、commit 锁定、32B 输出）、平局裁决排序。

## 包边界

| 包 | 职责 | 禁止事项 |
|----|------|----------|
| `internal/consensus` | 出块时序、区块发布三段状态、区块竞争归一化、分叉链段比较、临界裁决、RandomX 平局、交易回收、端点约定 | 不实现 P2P 线格式、不实现铸造资格/择优（见 07）、不把共约当作合法性规则 |

可选将 RandomX CGO 封装隔离到 `internal/consensus/randomx`（或 `pkg/randomx`）子包，避免 CGO 依赖污染纯 Go 测试。

## 建议文件

| 文件 | 内容 |
|------|------|
| `internal/consensus/block_time.go` | 确定性出块时序（间隔/冗余/延后） |
| `internal/consensus/publish.go` | 区块发布三段状态机 |
| `internal/consensus/normalize.go` | 多签归一化 + 2 倍交易量归一化 |
| `internal/consensus/fork_choice.go` | 31 块链段、16 分胜出、20 接收上限 |
| `internal/consensus/decision.go` | 长度 20 临界分叉裁决（前 5 名签名） |
| `internal/consensus/randomx_tiebreak.go` | RandomX 平局裁决封装与排序 |
| `internal/consensus/recycle.go` | 失败分叉交易回收 |
| `internal/consensus/endpoint.go` | 端点约定（协议/共约区分） |

## Task 1: 确定性出块时序

**Files:**
- Create: `internal/consensus/block_time.go`
- Create: `internal/consensus/block_time_test.go`

**Step 1: 写失败测试**（proposal 12 §1）

- 出块间隔固定 **6 分钟**；区块时间戳由创世时间戳与高度**精确推导**，非铸造者本机时间。
- 铸造冗余：择优池所有候选者可打包，按择优池排序、间隔 **15 秒**广播；已收到排名靠前者合法区块则不再发布自己的。
- 首块延后：首个区块在区块时间戳之后延后 **30 秒**发布。
- 这些延迟/间隔标注为**共约**，不作为区块头合法性必要条件。
- 新块构建仅依赖前一区块 ID、当前交易集与 UTXO/UTCO 指纹。

**Step 2: 实现并提交**

`BlockTime(height) = GenesisTime + height × 6min`；创世时间戳（C-9）以占位常量注入，不虚构。

```bash
go test ./internal/consensus -run 'TestBlockTime' -v
git add internal/consensus/block_time.go internal/consensus/block_time_test.go
git commit -m "feat: add deterministic block timing"
```

## Task 2: 区块发布三段状态机

**Files:**
- Create: `internal/consensus/publish.go`
- Create: `internal/consensus/publish_test.go`

**Step 1: 写失败测试**（proposal 12 §2，证明包字段见 09 章 DEC-0601）

- 阶段 1「广播区块证明」：Coinbase 合法 + 区块头合法即先行转播。
- 阶段 2「发布区块概要」：全部 TxID 序列，可优化为每个 TxID 截前 16 字节（见 11 章 DEC-0602）。
- 阶段 3「同步交易数据」：补足缺失少量交易。
- 完整合法性判断完成前，择优池候选者**不应停止**发布区块。
- 只实现状态与接口，不实现网络传输。

**Step 2: 实现并提交**

```bash
go test ./internal/consensus -run 'TestPublish' -v
git add internal/consensus/publish.go internal/consensus/publish_test.go
git commit -m "feat: add block publish stages"
```

## Task 3: 区块竞争两步归一化

**Files:**
- Create: `internal/consensus/normalize.go`
- Create: `internal/consensus/normalize_test.go`

**Step 1: 写失败测试**（proposal 12 §3 / DEC-0303）

逐高度比较前，每高度先选有效候选块，两步归一化：

**3.1 同铸造者多签归一化：**

1. 按铸造者公钥哈希分组。
2. 每组保留「铸造者个人可得收益最低」者。
3. 个人可得收益相同 → 保留交易费总额更低者。
4. 仍相同 → 保留 `BlockID` 更小者。
- 「个人可得收益」= Coinbase 中直接分配给铸造者身份（铸凭者）的金额，**不含**校验组报酬/公共服务奖励/其它第三方收益。

**3.2 交易量约束归一化（Stakes `>3x` 或 TxCount `>2x`）：**

- 仅比较同高度、同前块上的冗余出块；按铸造者择优池排名升序排列，缺位跳过、不生成空候选。
- 从当前最优 `winner` 起考察后位 `challenger`；`challenger` 满足 `challenger.Stakes > winner.Stakes×3` **或** `challenger.TxCount > winner.TxCount×2`（任一条件成立即替换）时替换并继续；否则停止。
- `TxCount` 含 Coinbase；相等不算超越（Stakes 必须严格 `>3x`，TxCount 必须严格 `>2x`）；`winner.Stakes==0` 或 `winner.TxCount==0` 时仍按公式（后位对应指标 `>0` 即满足超越条件）。

**Step 2: 实现**

通过接口注入收益与 `Stakes`/`TxCount`，不在共识包重复交易/状态计算。

**Step 3: 验证并提交**

```bash
go test ./internal/consensus -run 'TestNormalize' -v
git add internal/consensus/normalize.go internal/consensus/normalize_test.go
git commit -m "feat: add block candidate normalization"
```

## Task 4: 分叉链段竞争

**Files:**
- Create: `internal/consensus/fork_choice.go`
- Create: `internal/consensus/fork_choice_test.go`

**Step 1: 写失败测试**（proposal 12 §4 / DEC-0303）

- 比较范围：分叉点之后最多 **31** 个区块。
- 逐高度先经 §3 归一化选有效候选块，再逐高度比较两链 `MintHash`；单高度较小者得 1 分，完全相等双方都不得分。
- 任一链先达 **16** 分即胜出，提前结束。
- 分叉接收上限：新分叉长度必须 `<= 20`，否则不接收；超出 31 块的分叉不再评比。
- 未被接纳参与评比的分叉被无视（即便更优），为既成事实社会性共识。
- 铸凭哈希在 Coinbase 中由签名验证；铸凭哈希签名者必须与区块头 `CheckRoot` 签名者一致（见 04 章/签名）。

**Step 2: 实现并提交**

```bash
go test ./internal/consensus -run 'TestForkChoice' -v
git add internal/consensus/fork_choice.go internal/consensus/fork_choice_test.go
git commit -m "feat: add fork segment comparison"
```

## Task 5: 长度 20 临界裁决与平局裁决

**Files:**
- Create: `internal/consensus/decision.go`
- Create: `internal/consensus/randomx_tiebreak.go`
- Create: `internal/consensus/decision_test.go`
- Create: `internal/consensus/randomx_tiebreak_test.go`

**Step 1: 写失败测试**（proposal 12 §5·§6 / DEC-0303）

临界裁决（长度恰 20 的分叉）：

- 本链当前区块（#21）择优池**前 5 名**自由决定是否接纳，签名并广播。
- 按择优池成员顺序，**最靠前**成员的裁决有效；5 名都未签名 → 默认否决。
- 裁决消息绑定分叉点、本链末端、支链末端、当前高度、目标择优池引用、域标签、防重放字段。

平局裁决（31 高度比完仍平局）：

- `score = RandomX(seed=ForkPointBlockID(48B), input=FirstForkBlockID(48B))`，输出 32B。
- `score` 字典序升序较小者胜；`score` 相同则比较分叉首块 ID 较小者胜。
- RandomX profile：官方 `https://github.com/tevador/RandomX`、版本 `v2.0.1`、commit `aaafe71322df6602c21a5c72937ac284724ae561`、输出 32B（`RANDOMX_HASH_SIZE`）、完整 VM 语义，可经 CGO 封装；**禁止**变体实现或参数变体。
- RandomX 仅作低概率平局裁决，不进入常规出块/共识主路径。

**Step 2: 实现**

RandomX 封装隔离到子包；提供可在无 CGO 环境跳过/打桩的测试入口（向量测试覆盖排序逻辑），但生产路径必须用冻结官方版本。

**Step 3: 验证并提交**

```bash
go test ./internal/consensus -run 'Test(Decision|RandomX)' -v
git add internal/consensus/decision.go internal/consensus/randomx_tiebreak.go internal/consensus/decision_test.go internal/consensus/randomx_tiebreak_test.go
git commit -m "feat: add critical decision and randomx tiebreak"
```

## Task 6: 交易回收与端点约定

**Files:**
- Create: `internal/consensus/recycle.go`
- Create: `internal/consensus/endpoint.go`
- Create: `internal/consensus/recycle_test.go`
- Create: `internal/consensus/endpoint_test.go`

**Step 1: 写失败测试**（proposal 12 §7·§8）

交易回收：

- 分叉竞争结束后，失败分叉交易合并回主链。
- 失败分叉上「过早使用新币」的交易失效（无效新币输入源排除即可）。

端点约定（**协议**严格验证 / **共约**不得拒绝合法区块）：

| 约定 | 类型 | 规则 |
|------|------|------|
| 交易过期 | 协议 | 超 **240** 块（24h）未收录作废，按交易时间戳判断；公共验证节点严格遵守 |
| 区块不收录未来交易 | 协议 | 区块不收录时间戳晚于区块时间戳的交易 |
| 零确认 | 简化处理 | 发现双花应 App 警告；大额应等足够确认 |
| 最低交易费 | 共约 | 前 **6000** 块平均交易费的 **1/4**；不得据此拒绝合法区块或阻止低费交易转播 |
| 错时延迟 | 共约 | 出块时间到达后，错时交易暂停转播至该区块确定 |
| 新币 31 确认后使用 | 共约 | 规避分叉风险，不强制 |

**Step 2: 实现并提交**

协议项作为合法性验证；共约项作为本地策略/配置，类型与注释中明确标注，不进入区块合法性路径。

```bash
go test ./internal/consensus -run 'Test(Recycle|Endpoint)' -v
git add internal/consensus/recycle.go internal/consensus/endpoint.go internal/consensus/recycle_test.go internal/consensus/endpoint_test.go
git commit -m "feat: add tx recycle and endpoint conventions"
```

## 待决问题

- 本篇无 conception/decision 未覆盖的协议待决项。`Stakes` 在分叉比较中的口径为「有效候选块的 `Stakes`」（B-5 第三义），与 03 章 §5、07 章币权销毁口径区分，禁止混用。
- 出块时序的创世时间戳依赖 **C-9**（与 02/07 章一致），以占位注入。

## 阶段门禁/验收

进入条件：07（PoH 铸凭/择优池）接口稳定。

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

- 出块时序由创世时间戳与高度推导，不读本机时间；冗余/延后标注共约。
- 分叉比较前必须先完成两步归一化（多签 + 2 倍交易量）。
- 链段比较范围 31、过半 16 胜出、接收上限 20；`MintHash` 相等双方不得分。
- 测试覆盖：同铸造者多签、`MintHash` 相等、31 块平局、RandomX 裁决、连续后位超越、`Stakes=0`、`TxCount=0` 与相等边界（DEC-0303）。
- RandomX 使用冻结官方 `v2.0.1`（commit 锁定）、32B 输出、不得变体。
- 协议项（交易过期 240 块、不收录未来交易）严格验证；共约项（最低费/错时/新币 31 确认）不作为拒绝合法区块依据。
- 不实现 P2P 线格式；不实现铸造资格/择优（属 07）。
