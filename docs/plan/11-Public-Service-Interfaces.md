# Public Service Interfaces Implementation Plan

**Goal:** 以**外部接口边界**形式落地公共服务（Layer 5/外部接口层）：Depots/Blockqs/STUN/基网四类服务职责边界、Blockqs 6 类响应、数据量边界（`<10MB`/`>=10MB`）、区块概要 TxID profile 与碰撞回退、响应验证与服务密钥约束、初始主链验证支持。

**Architecture:** `internal/services/`（Layer 5）只定义接口边界、可验证数据 profile 与服务约束，不实现 Depots/Blockqs/STUN/基网内部逻辑与 P2P 线格式（C-10 外包）。**核心原则：** 公共服务与共识无关；对验证节点而言所有数据都需验证，Blockqs/Depots **不是信任根**，仅查询加速层。

**Tech Stack:** Go 1.26.2、接口驱动设计、`internal/blockchain`（头链、`CheckRoot`）、`pkg/hashtree`（验证路径）、`internal/utxo`/`internal/utco`（状态）、`internal/validation`（区块证明包）、表驱动测试。

---

## 来源提案

- `docs/proposal/15.Public-Service-Interfaces.md`（主源）
- 依赖 `docs/proposal/04.Hash-Trees.md`（验证路径）、`05.Blockchain-Core.md`（头链、`CheckRoot`）、`09.UTXO-UTCO-State.md`（状态）、`13.Team-Validation.md`（区块证明包 `RecentBlockProofs`）、`14.Incentives-And-Coinbase.md`（公共服务奖励）；呼应 00 章实现边界（C-10）。
- **DEC-0602**：网络概要交易 ID profile（`Summary = BlockID || TxCount || TxIDPrefix*`、前 16 字节、含 Coinbase、碰撞回退返回完整 48B）。
- **DEC-0603**：Blockqs 6 类响应、服务数据量边界、响应验证与独立服务密钥。

## 包边界

| 包 | 职责 | 禁止事项 |
|----|------|----------|
| `internal/services` | 服务职责边界、6 类响应可验证数据 profile、区块概要、响应验证约束 | 不实现 Depots/Blockqs/STUN/基网内部逻辑、不实现 P2P 线格式、不把服务返回值当信任根 |

`internal/services` 属 Layer 5，可依赖 Layer 0-4 接口/稳定类型，不被反向 import。接口返回 `Data + Proof`，验证函数本地执行；不要把 HTTP/RPC/P2P 客户端写进核心接口。

## 建议文件

| 文件 | 内容 |
|------|------|
| `internal/services/decomposition.go` | 四类服务职责与数据量边界 |
| `internal/services/depots.go` | Depots 接口（存储/查询/紧缺性/心跳，边界声明） |
| `internal/services/blockqs.go` | Blockqs 6 类响应接口 |
| `internal/services/summary.go` | 区块概要 TxID profile 与碰撞回退 |
| `internal/services/verify.go` | 响应验证与服务密钥约束 |
| `internal/services/stun.go` | STUN/基网边界声明 |
| `internal/services/errors.go` | 错误定义 |

## Task 1: 服务分解与数据量边界

**Files:**
- Create: `internal/services/decomposition.go`
- Create: `internal/services/depots.go`
- Create: `internal/services/stun.go`
- Create: `internal/services/decomposition_test.go`

**Step 1: 写失败测试**（proposal 15 §1·§2 / DEC-0603）

- 四类服务职责边界：基网（节点发现，`cxio/p2p`）、STUN（NAT 探测，`cxio/stun2p`）、Depots（`>=10MB` 附件/完整区块/分片，`cxio/depots`）、Blockqs（`<10MB` 附件/分片索引，`cxio/blockqs`）。
- 数据量边界以 `10MB` 划分：`<10MB` 附件与分片索引归 Blockqs，`>=10MB` 附件/完整区块/分片归 Depots；边界为建议，二者对同一数据提供可能重叠，但**验证口径必须相同**。
- Depots 接口仅声明职责（存储/数据查询经基网广播/紧缺性感知/数据心跳/数据上传），不实现内部逻辑；服务节点连接应用节点时提供区块链账户地址以接收奖励分配（见 14 章）。
- STUN/基网不参与区块/交易/PoH/脚本验证；服务不可达不改变区块合法性。

**Step 2: 实现并提交**

```bash
go test ./internal/services -run 'TestDecomposition' -v
git add internal/services/decomposition.go internal/services/depots.go internal/services/stun.go internal/services/decomposition_test.go
git commit -m "feat: define service decomposition boundaries"
```

## Task 2: Blockqs 6 类响应

**Files:**
- Create: `internal/services/blockqs.go`
- Create: `internal/services/blockqs_test.go`

**Step 1: 写失败测试**（proposal 15 §3 / DEC-0603）

6 类响应可验证数据结构（与链上 canonical encoding 一致；响应外壳格式可演进）：

| 响应类型 | 内容 |
|----------|------|
| `TxLookup` | 按年度和 TxID 返回完整交易、区块高度、区块内序位 |
| `TxProof` | 交易到区块交易树根的验证路径（见 04 章） |
| `BlockTxList` | 区块完整 TxID 序列或网络概要 |
| `StateProof` | UTXO/UTCO 状态位证明与输出详情（见 09 章） |
| `RecentBlockProofs` | 最近至少 31 个区块证明包（可 240 或更多，见 13 章） |
| `AttachmentIndex` | 小附件或大附件分片索引 |

- 可验证材料必须使用链上 canonical encoding 或明确提供链上原始字节；服务响应外壳可用 JSON/CBOR 等传输格式（外壳不进入共识）。
- 初始同步依赖 `RecentBlockProofs` 完整性（≥31 块证明包覆盖分叉安全窗口）。

**Step 2: 实现接口**

只定义响应类型与可验证数据 profile，不实现 Blockqs 内部存储/查询逻辑。

**Step 3: 验证并提交**

```bash
go test ./internal/services -run 'TestBlockqs' -v
git add internal/services/blockqs.go internal/services/blockqs_test.go
git commit -m "feat: define blockqs response profiles"
```

## Task 3: 区块概要 TxID profile 与碰撞回退

**Files:**
- Create: `internal/services/summary.go`
- Create: `internal/services/summary_test.go`

**Step 1: 写失败测试**（proposal 15 §4 / DEC-0602）

- 基础格式：`Summary = BlockID || TxCount || TxIDPrefix*`。
- `TxIDPrefix` 固定为完整 TxID 的**前 16 字节**，不设 `TxIDPrefixLen` 字段、不协商其它长度。
- `TxIDPrefix` 按区块交易序列顺序排列，**包含 Coinbase**。
- 碰撞回退：接收方发现本地候选交易有多个匹配时，按交易序位请求碰撞回退信息；发布方对指定序位返回完整 48 字节 TxID（不属于基础 `Summary` 本体）。
- 区块概要不需发布方单独签名，只作网络同步优化；最终验证必须用完整 TxID 序列计算交易树根。
- 节点不得因短前缀无法解析就接受不完整区块；错误摘要在完整交易树验证阶段失败。

**Step 2: 实现并提交**

```bash
go test ./internal/services -run 'TestSummary' -v
git add internal/services/summary.go internal/services/summary_test.go
git commit -m "feat: add block summary profile"
```

## Task 4: 响应验证与服务密钥

**Files:**
- Create: `internal/services/verify.go`
- Create: `internal/services/verify_test.go`

**Step 1: 写失败测试**（proposal 15 §5·§6 / DEC-0603·DEC-0601）

- 所有响应必须可由区块头链、`CheckRoot`、TxID 或附件指纹验证。
- 服务节点签名使用**独立服务密钥**，只证明服务来源，**不证明数据真实**；独立服务密钥不需与链上收益地址做协议级绑定；收益地址声明不作为响应真实性依据。
- 客户端应向多个 Blockqs 节点**交叉查询**关键数据。
- 初始主链验证支持：新上线节点请求区块头链 + 末端局部区块 + 当前 UTXO/UTCO 集即可大致确定目标主链是否合法（UTXO/UTCO 指纹是全链历史当前总结，三路耦合见 09 章）；区块头链可借年块链缩小体积（见 05 章）。
- 服务身份与收益地址不做协议级强绑定。

**Step 2: 实现并提交**

验证函数本地执行；服务密钥仅证来源不证真实，不得据收益地址声明判定真实性。

```bash
go test ./internal/services -run 'TestVerify' -v
git add internal/services/verify.go internal/services/verify_test.go internal/services/errors.go
git commit -m "feat: add response verification rules"
```

## 待决问题

- **C-10 实现边界：** P2P 线格式、版本治理、通用子链派生协议属外包或未抽象（基网 `cxio/p2p`、`depots`、`blockqs`、`stun2p`）。本篇仅定义接口边界与可验证数据 profile，不规格化传输线格式（呼应 00 章非目标），不强行规格化。

## 阶段门禁/验收

进入条件：区块/状态/证明接口（02、05、09）稳定。

运行：

```bash
go fmt ./...
go test ./internal/services
go test ./...
go build ./...
go mod tidy
go mod verify
golangci-lint run
```

通过标准：

- 公共服务以外部接口边界形式落地，不实现 Depots/Blockqs/STUN/基网内部逻辑与 P2P 线格式。
- 区块概要 `BlockID || TxCount || TxIDPrefix*`，前缀固定 16 字节、含 Coinbase、按序排列；碰撞回退返回完整 48B；最终验证用完整 TxID 序列重算交易树根。
- Blockqs 6 类响应可验证数据结构与链上 canonical encoding 一致；响应外壳格式可演进。
- 响应验证必须可由头链/`CheckRoot`/TxID/附件指纹验证；服务密钥仅证来源不证真实；关键数据交叉查询多节点。
- 数据量边界 `10MB` 划分 Blockqs/Depots，验证口径统一；服务不可达不改变区块合法性。
- 初始同步依赖 `RecentBlockProofs`（≥31 块证明包），与 09 章一致；`internal/services` 不被 Layer 0-4 反向 import。
