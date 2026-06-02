# Team Validation Implementation Plan

**Goal:** 以**接口定义**形式落地组队校验（Layer 5）：角色分工与协作协议、首领校验与黑名单、安全保障（冗余/两级复核/组间反馈）、铸造协作信息分离流程、区块证明包（DEC-0601）与 9 步快速预验证、交易收录优先级与交易量约束（Stakes 严格 `>3x`）、最优规模参考。

**Architecture:** `internal/validation/`（Layer 5）只定义接口、任务状态与协议约束，不实现各角色 App 与组内/组间 P2P 线格式（C-10 外包）。校验组与公共服务**不是**新的共识信任源；区块证明包仅支持快速预验证与转播，不能替代完整区块验证、不证明 UTXO/UTCO 状态真实性。

**Tech Stack:** Go 1.26.2、接口驱动设计、`internal/blockchain`、`internal/tx`、`internal/utxo`、`internal/utco`、`internal/consensus` 的接口/稳定类型、表驱动测试。

---

## 来源提案

- `docs/proposal/13.Team-Validation.md`（主源）
- 依赖 `docs/proposal/05.Blockchain-Core.md`（区块头、`CheckRoot`）、`06.Transaction-Model.md`（交易、Coinbase）、`09.UTXO-UTCO-State.md`（指纹）、`11.PoH-Consensus.md`（择优凭证、铸造者验证）、`12.Endpoint-And-Fork-Choice.md`（区块竞争、交易量约束）；被 14（激励）、15（服务）引用。
- **DEC-0601**：区块证明包八字段、`CoinbaseTxIndex==0`、不单独携带 `CheckRoot`/`MintProof`/公钥/`ChainScope`/状态证明路径、9 步快速预验证顺序、初始同步至少 31 块证明包。

## 包边界

| 包 | 职责 | 禁止事项 |
|----|------|----------|
| `internal/validation` | 角色接口、首领校验、复核流程、铸造协作、区块证明包与预验证、收录优先级 | 不实现组内 RPC/组间 P2P 线格式、不实现信誉系统、不让校验/服务结果直接改变区块合法性 |

`internal/validation` 属 Layer 5，可依赖 Layer 0-4 接口/稳定类型，不被 Layer 0-4 反向 import。

## 建议文件

| 文件 | 内容 |
|------|------|
| `internal/validation/roles.go` | 管理层/守卫者/校验员角色与连接关系接口 |
| `internal/validation/task.go` | 校验任务与结果类型 |
| `internal/validation/leader_check.go` | 首领校验与黑名单 |
| `internal/validation/review.go` | 冗余与两级扩展复核、组间反馈 |
| `internal/validation/block_building.go` | 铸造协作（信息分离）接口 |
| `internal/validation/proof_package.go` | 区块证明包编解码 |
| `internal/validation/pre_verify.go` | 9 步快速预验证 |
| `internal/validation/priority.go` | 交易收录优先级与交易量约束（共约） |
| `internal/validation/errors.go` | 错误定义 |

## Task 1: 角色分工与任务模型

**Files:**
- Create: `internal/validation/roles.go`
- Create: `internal/validation/task.go`
- Create: `internal/validation/roles_test.go`
- Create: `internal/validation/task_test.go`

**Step 1: 写失败测试**（proposal 13 §1）

- 三角色职责与连接：管理层（分发/冗余控制/汇总/与外部区块交互）；守卫者（接收外部交易、执行首领校验、提交准合法交易、传递其它组守卫者）；校验员（请求任务、完整校验、无条件反馈、向其它组守卫者传送合法交易）。
- 组内公共服务接口：UTXO/UTCO 缓存器（未花费/未转出查询、计算指纹后通知管理层拉取）、外部脚本引用缓存（`GOTO`/`EMBED`/`SCRIPT`，及时清理/实时更新）。
- 任务含交易 ID、任务类型、分配时间、候选验证上下文；结果区分合法/非法/拒绝任务/验证错误；可追溯到校验员标识但不定义信誉系统。

**Step 2: 实现并提交**

```bash
go test ./internal/validation -run 'Test(Roles|Task)' -v
git add internal/validation/roles.go internal/validation/task.go internal/validation/roles_test.go internal/validation/task_test.go
git commit -m "feat: define team roles and tasks"
```

## Task 2: 首领校验与黑名单

**Files:**
- Create: `internal/validation/leader_check.go`
- Create: `internal/validation/leader_check_test.go`

**Step 1: 写失败测试**（proposal 13 §2）

- 仅验证交易首笔输入，合法即通过（加快全网传播）。
- 约束：首笔输入必须是**币金**，且为全部币金输入中**币权最大者**。
- 黑名单：首领校验通过但最终完整验证失败的交易，其首笔输入进入黑名单临时冻结（约 **24** 小时）。
- 黑名单冻结时长为本地策略/配置，区块验证不得因首领输入处于本地黑名单冻结期而拒绝区块。
- 快速广播折衷：随机抉择（约 50% 概率先校验通过再转发，其余直接转发）为共约性优化。

**Step 2: 实现并提交**

```bash
go test ./internal/validation -run 'TestLeaderCheck' -v
git add internal/validation/leader_check.go internal/validation/leader_check_test.go
git commit -m "feat: add leader check and blacklist"
```

## Task 3: 冗余校验与两级复核

**Files:**
- Create: `internal/validation/review.go`
- Create: `internal/validation/review_test.go`

**Step 1: 写失败测试**（proposal 13 §3）

- 冗余度 `>= 2`；全部反馈合法则合法；至少一名判非法进入扩展复核。
- 一级复核：零报错→合法；超半数报错→非法；低于半数→进入二级复核。
- 二级复核：只要有报错即非法。
- 复核判非法的交易仍有机会（其它组验证合法并入块、同步回本组时重新验证）。
- 组间反馈：守卫者收到被本组判非法的交易即时反馈来源（它组校验员），触发其管理层重复两级复核；管理层须建立守卫者递送记录。

**Step 2: 实现并提交**

```bash
go test ./internal/validation -run 'TestReview' -v
git add internal/validation/review.go internal/validation/review_test.go
git commit -m "feat: add redundancy and review flow"
```

## Task 4: 铸造协作（信息分离）

**Files:**
- Create: `internal/validation/block_building.go`
- Create: `internal/validation/block_building_test.go`

**Step 1: 写失败测试**（proposal 13 §4）

- 铸造者只能是择优池成员；铸造流程三步：①铸币申请（提交签名择优凭证，管理层返回交易费/校验组收益地址/公共服务地址/铸币量/兑奖截留等）②构造 Coinbase（铸造者构造并提交，管理层返回含 `TreeRoot` 与 UTXO/UTCO 指纹的校验路径）③签署区块（铸造者验证自己 Coinbase 在其中、对 `CheckRoot` 签名提交，管理层验证后构造区块头发布）。
- 信息分离：铸造者除非自己是管理层，否则不知全局 `CheckRoot`，只能验证自己 Coinbase 是否在其中。
- 铸造者对 Coinbase 的签名用于管理层验证，**不进入** TxID 计算（见 06、08 章）。
- 铸造者收益地址（铸凭者）自由，不与铸造身份（公钥哈希）绑定。
- 公共服务受奖地址由管理层推荐，铸造者可自选；管理层不接受其自选则不提供签名信息（铸造者可能需向多个组申请，见 14 章）。

**Step 2: 实现接口与数据结构**

只定义协作消息与验证钩子，不实现网络 RPC。

**Step 3: 验证并提交**

```bash
go test ./internal/validation -run 'TestBlockBuilding' -v
git add internal/validation/block_building.go internal/validation/block_building_test.go
git commit -m "feat: define minting collaboration"
```

## Task 5: 区块证明包与 9 步快速预验证

**Files:**
- Create: `internal/validation/proof_package.go`
- Create: `internal/validation/pre_verify.go`
- Create: `internal/validation/proof_package_test.go`
- Create: `internal/validation/pre_verify_test.go`

**Step 1: 写失败测试**（proposal 13 §5 / DEC-0601）

证明包八字段顺序（冻结）：

1. `BlockHeader` → 2. `CoinbaseTx`（含 `Minter`/`MintProof`）→ 3. `CoinbaseTxIndex`（必须为 `0`）→ 4. `CoinbaseMerklePath` → 5. `TreeRoot` → 6. `UTXORoot`（上一区块完成后）→ 7. `UTCORoot`（上一区块完成后）→ 8. `MinterCheckRootSignature`。

字段规则：不单独携带 `CheckRoot`（以 `BlockHeader.CheckRoot` 为准）、`MintProof`（从 `CoinbaseTx.Minter` 读取）、铸造者公钥（从 `Minter.MintPubKey` 读取）、`ChainScope`（隐含）、UTXO/UTCO 状态证明路径（`UTXORoot`/`UTCORoot` 仅用于与本地当前状态快速比较并重算 `CheckRoot`）。

9 步快速预验证（廉价检查优先）：

1. `BlockHeader.PrevBlock` 与本地末端区块 ID 衔接。
2. `CoinbaseTx.Minter` 存在且铸造者是本地当前择优池成员（最廉价关键检查，**前置**）。
3. `UTXORoot`/`UTCORoot` 与本地当前指纹一致。
4. `CoinbaseTxIndex == 0`。
5. 计算 `CoinbaseTxID`。
6. 用 `CoinbaseTxID`/`CoinbaseTxIndex`/`CoinbaseMerklePath` 重算交易树根，比对 `TreeRoot`。
7. 用 `TreeRoot || UTXORoot || UTCORoot` 重算 `CheckRoot`，比对 `BlockHeader.CheckRoot`。
8. 验证 `Minter.MintHash` 与签名（择优池成员进入时已验证，可略过）。
9. 用 `Minter.MintPubKey` 验证 `MinterCheckRootSignature`。

- 证明包不能替代完整区块验证、不证明状态本身；初始同步至少需最近 **31** 块证明包覆盖分叉安全窗口。
- 择优池成员检查须前置于哈希树/签名等高成本验证。

**Step 2: 实现并提交**

```bash
go test ./internal/validation -run 'Test(ProofPackage|PreVerify)' -v
git add internal/validation/proof_package.go internal/validation/pre_verify.go internal/validation/proof_package_test.go internal/validation/pre_verify_test.go
git commit -m "feat: add block proof package and pre-verify"
```

## Task 6: 交易收录优先级与交易量约束

**Files:**
- Create: `internal/validation/priority.go`
- Create: `internal/validation/priority_test.go`

**Step 1: 写失败测试**（proposal 13 §6·§8）

- 收录优先级（**共约**）：币权销毁较多 > 交易费更多 > 有凭信消费（凭信消费优先级最低）；为共约，不得作为拒绝合法区块依据。
- 交易量约束（Stakes 严格 `>3x`，交易数量不参与判定）：抑制自私铸造，详细确定性算法引用 08 章 §3.2（DEC-0303），本篇只承载语义与接口对接。
- 最优规模参考（说明性）：约 182 笔/秒负载、50–60 节点可能足够；校验员对交易认领自由、自适应。

**Step 2: 实现并提交**

```bash
go test ./internal/validation -run 'TestPriority' -v
git add internal/validation/priority.go internal/validation/priority_test.go
git commit -m "feat: add inclusion priority convention"
```

## 待决问题

- **C-10 实现边界：** 各角色为独立 App，组内/组间通讯线格式属外部 App 实现范畴（P2P 线格式外包，见 00、11 章边界声明）。本篇仅定义接口与协议约束，不规格化传输线格式，不规定具体 App 实现。

## 阶段门禁/验收

进入条件：区块/交易/状态/共识（02~08）边界稳定。

运行：

```bash
go fmt ./...
go test ./internal/validation
go test ./...
go build ./...
go mod tidy
go mod verify
golangci-lint run
```

通过标准：

- 团队验证以接口形式落地，不实现各角色 App 与 P2P 线格式。
- 区块证明包八字段编解码、`CoinbaseTxIndex` 固定 0；9 步预验证顺序正确，择优池成员检查前置于高成本验证。
- 首领校验约束（首输入为币权最大币金）与黑名单（约 24h，本地策略）测试覆盖；黑名单不作为区块合法性依据。
- 冗余 `>=2` 与两级扩展复核、组间反馈触发重复核测试覆盖。
- 铸造协作信息分离：Coinbase 签名不进入 TxID；`CheckRoot` 由管理层提供路径供铸造者验证后签名。
- 收录优先级（共约）不拒绝合法区块；交易量约束（Stakes `>3x`）引用 08 章 DEC-0303。
- 初始同步依赖最近至少 31 块证明包；`internal/validation` 不被 Layer 0-4 反向 import。
