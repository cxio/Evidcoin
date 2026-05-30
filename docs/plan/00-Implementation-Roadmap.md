# Evidcoin Implementation Roadmap Implementation Plan

**Goal:** 将重写后的 `docs/proposal/`（00~15 + Instruction，追溯自 conception+decision）技术规格转化为后续 Go 编码可直接执行的阶段化实施路线。

**Architecture:** 按项目既定 Layer 0 到 Layer 5 单向依赖推进，先固定基础类型、编码、Hash 与签名抽象，再逐步实现区块、交易、签名见证、状态、脚本、共识和外部接口边界。所有共识级字节输入必须追溯到 `docs/proposal/` 的规范化编码（最终依据为 conception 与 decision），所有待决协议细节必须显式隔离，不能在代码中悄悄固化。

**Tech Stack:** Go 1.26.2、标准库、`golang.org/x/crypto`、`lukechampine.com/blake3`、`github.com/cloudflare/circl`（ML-DSA-65，DEC-0104）、`github.com/mr-tron/base58`、表驱动测试、`go test ./...`、`go build ./...`、`golangci-lint run`。

---

## 范围

本目录是 `Proposal => Plan` 层。它不编写生产代码，只定义后续编码顺序、包边界、文件清单、TDD 任务、测试策略和验收标准。

> proposal 层是「由 conception+decision 重生的可实施技术规格」，其权威性低于 conception 与 decision；本 Plan 引用 proposal，最终依据回溯至 conception/decision（遇冲突以 conception 为准）。

本实施路线覆盖以下 Proposal（新 16 篇结构）：

- `docs/proposal/00.Project-Scope.md` 到 `docs/proposal/15.Public-Service-Interfaces.md`
- `docs/proposal/Instruction/*.md`

## 总体分层

| 层级 | 包 | 主要职责 | 依赖方向 |
|------|----|----------|----------|
| Layer 0 | `pkg/types/`, `pkg/crypto/`, `pkg/hashtree/` | 基础类型、编码、Hash、域标签、签名抽象、通用哈希树 | 不依赖 `internal/*` |
| Layer 1 | `internal/blockchain/`, `internal/tx/` | 区块头链、交易头、输入输出、信元 payload、签名见证 | 依赖 Layer 0 |
| Layer 2 | `internal/utxo/`, `internal/utco/` | Coin 与 Credit 状态、宽成员状态指纹、回滚 | 依赖 Layer 0-1 |
| Layer 3 | `internal/script/` | 栈式 VM、公共验证、254 指令注册表 | 依赖 Layer 0-2 的接口，不反向依赖 |
| Layer 4 | `internal/consensus/` | PoH 铸凭、择优池、出块时序、分叉选择 | 依赖 Layer 0-3 的接口 |
| Layer 5 | `internal/validation/`, `internal/rewards/`, `internal/services/`, `cmd/evidcoin/`, `test/` | 组队校验接口、激励与 Coinbase 结算、公共服务接口、组装、集成测试、命令入口 | 依赖所有内部层，不能被 Layer 0-4 反向依赖 |

## 方案文件

| 文件 | 覆盖 Proposal | 主要输出 |
|------|---------------|----------|
| `docs/plan/01-Foundation-Types-Crypto.md` | 01·02·03·04 | `pkg/types/`、`pkg/crypto/`、`pkg/hashtree/` |
| `docs/plan/02-Blockchain-Core.md` | 05 | `internal/blockchain/` |
| `docs/plan/03-Transaction-And-Units.md` | 06·07 | `internal/tx/`、Coin/Credit/Proof payload |
| `docs/plan/04-Signatures-And-Witness.md` | 08 | `internal/tx/` 签名消息、见证容器、多签 |
| `docs/plan/05-UTXO-UTCO-State.md` | 09 | `internal/utxo/`、`internal/utco/` |
| `docs/plan/06-Script-System.md` | 10 + Instruction/ | `internal/script/` |
| `docs/plan/07-PoH-Consensus.md` | 11 | `internal/consensus/`（铸凭、择优池、创世初段窗口） |
| `docs/plan/08-Endpoint-And-Fork-Choice.md` | 12 | `internal/consensus/`（出块时序、分叉竞争、RandomX 平局） |
| `docs/plan/09-Team-Validation.md` | 13 | `internal/validation/` 接口、区块证明包 |
| `docs/plan/10-Incentives-And-Coinbase.md` | 14 | `internal/rewards/`、Coinbase 序列化、兑奖槽 |
| `docs/plan/11-Public-Service-Interfaces.md` | 15 | `internal/services/` 接口、Blockqs/Depots 验证数据 |
| `docs/plan/12-Open-Questions-And-Acceptance.md` | 全部 | 待决项（C-6/C-7/C-9/C-10）、全局验收、阶段门禁 |

## 推荐实施顺序

1. 完成 `pkg/types/` 的固定长度类型、ID 类型、常量和规范化编码工具（DEC-0001）。
2. 完成 `pkg/crypto/` 的 Hash API、14 项域标签、地址/多签复合公钥哈希、ML-DSA-65 抽象（DEC-0002/0104）。
3. 完成 `pkg/hashtree/` 通用二叉树与专用树规则（DEC-0004）；附件片组树走免域标签路径。
4. 完成区块头、`BlockID`、CheckRoot、头链存储接口和最小衔接验证（DEC-0003）；创世参数（C-9）以占位阻塞。
5. 完成交易头、输入输出 envelope、`TxID` 和 Coin/Credit/Proof payload（DEC-0101/0003）。
6. 完成签名消息布局、见证容器与剪枝、多签 M-of-N（DEC-0102/0103/0104）。
7. 完成 UTXO/UTCO entry、宽成员树分层、空根、过期处理和状态指纹（DEC-0201）。
8. 完成脚本 VM 基础运行时、公共/私有模式和先导指令子集（DEC-0501~0505）；成本数值（C-6）、禁用解除（C-7）以策略参数隔离。
9. 完成 PoH 铸凭哈希、择优池、铸造者验证、创世初段窗口（DEC-0301/0302）。
10. 完成出块时序、区块竞争归一化、分叉选择与 RandomX 平局（DEC-0303）。
11. 完成组队校验接口、区块证明包、激励与 Coinbase 结算、公共服务可验证数据接口（DEC-0601/0401/0602/0603）。
12. 完成集成测试和命令入口。

## 全局编码原则

- 所有导出符号必须有英文 Godoc。
- 面向作者理解的源码注释使用中文。
- 程序输出、日志和 error 文本使用英文。
- 所有测试使用 table-driven tests。
- 所有协议字节序列必须由显式编码函数生成，不使用 JSON、反射、map 遍历顺序或平台字节序。
- 低层包不能 import 高层包。
- 待决协议细节必须表现为开放规格项、策略参数、接口注入或明确拒绝，不能默认选一个值并当作协议事实；已由 decision 关闭的规则（如 DEC-0001 ULEB128 最短编码、DEC-0002 域标签、DEC-0004 哈希树边界、DEC-0104 地址/ML-DSA profile）必须按对应 DEC 执行。

## 全局验证命令

每个阶段完成后运行：

```bash
go fmt ./...
go test ./...
go build ./...
go mod tidy
go mod verify
golangci-lint run
```

若本地尚未安装 `golangci-lint`，该项记为环境阻塞，不得用“通过 lint”描述。

## 提交建议

提交步骤只在用户明确要求提交时执行。若用户要求按计划提交，则每个 Task 完成并通过局部测试后提交一次，提交信息采用简洁英文前缀：

```bash
git add <files>
git commit -m "feat: add canonical encoding types"
```

不要在同一提交中混合多个层级。不要提交 `.DS_Store`、临时日志、覆盖率文件或本地 IDE 配置。

## 阶段门禁

| 阶段 | Plan 文件 | 进入条件 | 退出条件 |
|------|-----------|----------|----------|
| Foundation | 01 | `go.mod` 存在 | `pkg/types`、`pkg/crypto`、`pkg/hashtree` 测试通过 |
| Blockchain | 02 | Foundation 完成 | 区块头链最小验证通过（创世参数 C-9 阻塞项除外） |
| Transaction | 03 | Blockchain 可引用基础 ID | 交易头、输入输出和 payload 测试通过 |
| Signatures | 04 | Transaction 类型稳定 | 签名消息、见证容器、多签验证测试通过 |
| State | 05 | 交易类型与签名稳定 | UTXO/UTCO 宽成员树与状态转移测试通过 |
| Script | 06 | 基础类型与状态接口可用 | VM 基础与公共验证测试通过（成本 C-6/禁用 C-7 隔离） |
| PoH | 07 | 区块、交易、状态接口可用 | 铸凭哈希、择优池、铸造者验证单元测试通过 |
| Fork-Choice | 08 | PoH 接口稳定 | 出块时序、区块竞争归一化、分叉选择测试通过 |
| Team-Validation | 09 | 区块/交易/状态/共识边界稳定 | 角色接口、区块证明包预验证测试通过 |
| Incentives | 10 | 区块与共识稳定 | Coinbase 序列化、奖励分配、兑奖槽测试通过 |
| Services | 11 | 区块/状态/证明接口稳定 | 服务可验证数据 profile 接口测试通过 |
| Integration | 12 | 所有单元测试通过 | `test/` 集成测试和构建通过 |

## 主要风险

- `canonical unsigned varint`（ULEB128）由 DEC-0001 关闭，必须实现最短编码并拒绝非最短编码；BigInt 按 `slen||magnitude`（DEC-0001）。
- Hash domain tag 策略由 DEC-0002 关闭，14 项域标签全集（含 `utxo.empty`/`utco.empty`）必须使用固定字符串前缀；附件片组树是唯一免域标签例外。
- 哈希树空根、单叶根和奇数层提升策略由 DEC-0004 关闭，必须按 DEC-0004 实现；验证路径禁止携带 `leafIndex`。
- 地址与多签复合公钥哈希、ML-DSA-65 profile 由 DEC-0104 关闭，当前固定 `cloudflare/circl`；A-2 仅作为未来兼容观察项，不属于全局待决项，编码时不得混用标准库实现。
- PoH 铸凭哈希、铸凭交易窗口与 `Stakes` 取值由 DEC-0301 固定；分叉竞争归一化与 RandomX 平局由 DEC-0303 固定。`Stakes` 三种语义（区块头累计值/铸凭取 -32 块/分叉比较）须分章分上下文区分，禁止混用。
- Coinbase `HashInputs` 省略、奖励分配余数、兑奖槽 bit 顺序由 DEC-0401 关闭；金额一律以 `chx` 整数承载（`1 Bi = 10^8 chx`）。
- 脚本 VM 254 指令全集很大，必须按元数据和公共验证安全边界分批实现；成本数值（C-6）、禁用指令解除方式（C-7）为待决，须以策略参数隔离，不得固化。
- 创世具体参数（创世时间戳、mainnet `Genesis-ID`）为待决 C-9；Plan 只固定创世块实现边界与验证规则，不伪造具体值，相关硬编码任务阻塞。

## 全局待决问题指针

待决问题在 `12-Open-Questions-And-Acceptance.md` 集中承载，与 proposal 00「全局待决问题汇总」一致：

| 编号 | 待决问题 | 承载 Plan |
|------|---------|-----------|
| C-6 | 脚本成本数值（opcode base_cost、正则/随机最坏上界、三层上限）；DEC-0504 开放 | 06、12 |
| C-7 | 禁用指令解除方式（逐项 vs 统一版本激活）；DEC-0505 开放 | 06、12 |
| C-9 | 创世具体参数（创世时间戳、mainnet Genesis-ID） | 02、07、12 |
| C-10 | P2P 线格式 / 版本分叉治理 / 通用子链派生协议（外包或未抽象） | 09、11、12 |

> 已裁决、移出待决：单位体系（`1 Bi = 10^8 chx`）、兑奖槽 bit 顺序（DEC-0401）、Stakes 三义（proposal 03/11/12）、哈希树证明路径（DEC-0004）、Coinbase 省略 HashInputs（DEC-0401）。

## 完成定义

本 Plan 被视为完成时，应满足：

- `docs/plan/` 下存在 1 篇总览路线图（00）、11 个阶段计划（01~11）和 1 篇待决项验收文档（12），共 13 篇。
- 每个阶段计划都引用对应 Proposal（新编号）。
- 每个阶段计划都列出建议文件、TDD 任务、测试命令和验收标准。
- 全程无 `ADR-xxxx` 残留，决策引用为 `DEC-NNNN` 且主题匹配。
- 待决项限于 C-6/C-7/C-9/C-10，相关任务显式标注阻塞/占位。
