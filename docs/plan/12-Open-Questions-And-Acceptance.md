# Open Questions And Acceptance Implementation Plan

**Goal:** 汇总当前仍开放的待决项（C-6/C-7/C-9/C-10）及其承载章与阻塞策略，列明已裁决移出待决的项，并定义全项目验收标准、阶段门禁汇总、单元/集成测试标准与文档同步流程。

**Architecture:** 待决项必须在编码时显式隔离（开放规格项/策略参数/接口注入/明确拒绝），直到 conception/decision 固定，不得用魔法常量悄悄固化。验收分为阶段验收、跨层依赖验收、协议向量验收与集成验收。

**Tech Stack:** Markdown、Go 1.26.2、`go test`、`go build`、`go fmt`、`go mod verify`、`golangci-lint`。

---

## 来源提案

- `docs/proposal/00.Project-Scope.md`「全局待决问题汇总」（与本篇一致）
- 各章「待决问题」小节：`02·06·07·11·13·14·15`（C-6/C-7/C-9/C-10 承载章）
- 已裁决依据：DEC-0401（兑奖槽/HashInputs）、DEC-0004（哈希树证明路径）、proposal 01/03（单位 C-8）、proposal 03/11/12（Stakes 三义）

## 当前待决项总表

> 待决项严格限于 C-6/C-7/C-9/C-10。已裁决项不再列为待决（见下一节）。

| 编号 | 领域 | 状态 | 待决项 | 承载 Plan | 阻塞策略 |
|------|------|------|--------|-----------|----------|
| C-6 | Script | 开放（DEC-0504） | 脚本成本数值（opcode `base_cost`、正则/随机最坏上界、三层上限） | 06、12 | 实现预算框架与保守拒绝；成本数值以策略参数注入，不固化；相关 Task 标注阻塞 |
| C-7 | Script | 开放（DEC-0505） | 禁用指令解除方式（逐项 vs 统一版本激活） | 06、12 | 禁用指令解锁段限制按 DEC-0505 当前定值；解除方式以策略开关隔离，不预设激活路径 |
| C-9 | Block/PoH | 开放 | 创世具体参数（创世时间戳、mainnet `Genesis-ID`） | 02、07、12 | 仅固定创世块实现边界与验证规则；具体字节取值以占位常量阻塞，不虚构（见 `/memories/repo/docs-genesis-boundary.md`） |
| C-10 | Services/Validation | 开放（外包/未抽象） | P2P 线格式 / 版本分叉治理 / 通用子链派生协议 | 09、11、12 | 仅声明接口边界，不规格化传输线格式；归外部库（`cxio/p2p`/`depots`/`blockqs`/`stun2p`） |

## 已裁决、移出待决

| 旧待决表述 | 新状态 | 依据 |
|------------|--------|------|
| 货币单位金额单位未定 | 已裁决 `1 Bi = 10^8 chx`、`chx` 最小承载单位 | C-8 已裁决，proposal 01·03（见 10 章） |
| 兑奖槽 bit 顺序未决 | 已固定 Coinbase 序列化/兑奖槽（18B、三服务各 6B、`bit0=H-1`） | DEC-0401（见 10 章） |
| Stakes 精确定义未定 | 已锚定三种语义（区块头累计 / 铸凭取 -63 块 / 分叉比较有效候选块） | proposal 03·11·12（见 02/07/08 章） |
| 证明路径编码未决 | 已固定哈希树与证明边界（空根/单叶/奇数层、路径禁带 `leafIndex`） | DEC-0004（见 01 章） |
| Coinbase `HashInputs` 特殊规则未决 | 已裁决 Coinbase 省略 `HashInputs` | DEC-0401 / conception（见 10 章） |

## 编码时的待决项规则

- 待决项不能用魔法常量悄悄实现。
- 待决项不能产生最终测试向量（C-6 成本向量、C-9 创世字节向量延后）。
- 待决项可通过显式策略参数、接口注入、`ErrSpecIncomplete`/开放规格项标注处理。
- 若某任务无法在不固定待决项的情况下推进，先询问用户（每次 1~3 个问题）。
- 一旦用户确认规则，必须先回写对应 conception/decision，再改 proposal、Plan 或代码。

## 单元测试标准

所有包都必须满足：

- 使用 table-driven tests。
- 覆盖成功路径、失败路径与边界值。
- 协议编码测试必须检查字节级输出。
- Hash 测试必须检查输出长度与用途隔离（域标签）。
- 状态转移测试必须检查前后状态与错误类型。
- 脚本 VM 测试必须检查执行状态、栈、资源计数与公共/私有模式（成本 C-6 以策略参数）。
- 共识测试必须检查排序、窗口、边界高度与防重放（PoH 窗口、分叉 31/16/20、RandomX 平局）。

## 集成测试建议

后续创建 `test/` 目录，按以下顺序添加集成测试：

| 文件 | 场景 |
|------|------|
| `test/foundation_encoding_test.go` | 基础编码与 Hash 跨包一致性（DEC-0001/0002/0004） |
| `test/header_chain_test.go` | 创世头、连续头、年块、`CheckRoot`（创世具体值 C-9 占位） |
| `test/transaction_state_test.go` | Coin 创建、消费、销毁、重复消费拒绝 |
| `test/credit_lifecycle_test.go` | Credit 创建、转移、到期、重复转移拒绝 |
| `test/script_validation_test.go` | unlock + lock script 公共验证（成本 C-6 框架） |
| `test/poh_pool_test.go` | 铸凭哈希排序、择优池、三段同步 |
| `test/fork_choice_test.go` | 20/31/16 分叉规则、两步归一化、RandomX 平局 |
| `test/reward_redemption_test.go` | Coinbase 序列化、奖励分配、兑奖槽、回收 |
| `test/proof_package_test.go` | 区块证明包 9 步快速预验证 |

## 覆盖率目标

核心逻辑覆盖率目标 80% 以上。阶段目标：

| 包 | 最低覆盖率 |
|----|------------|
| `pkg/types` | 90% |
| `pkg/crypto` | 85% |
| `pkg/hashtree` | 85% |
| `internal/blockchain` | 85% |
| `internal/tx` | 85% |
| `internal/utxo` / `internal/utco` | 85% |
| `internal/script` | 80% |
| `internal/consensus` | 80% |
| `internal/validation` / `internal/services` / `internal/rewards` | 80% |

覆盖率命令：

```bash
go test -cover ./...
```

## 阶段门禁汇总

| 阶段 | Plan 文件 | 进入条件 | 退出条件 |
|------|-----------|----------|----------|
| Foundation | 01 | `go.mod` 存在 | `pkg/types`、`pkg/crypto`、`pkg/hashtree` 测试通过 |
| Blockchain | 02 | Foundation 完成 | 区块头链最小验证通过（创世参数 C-9 阻塞项除外） |
| Transaction | 03 | Blockchain 可引用基础 ID | 交易头、输入输出、payload 测试通过 |
| Signatures | 04 | Transaction 类型稳定 | 签名消息、见证容器、多签验证测试通过 |
| State | 05 | 交易类型与签名稳定 | UTXO/UTCO 宽成员树与状态转移测试通过 |
| Script | 06 | 基础类型与状态接口可用 | VM 基础与公共验证测试通过（成本 C-6/禁用 C-7 隔离） |
| PoH | 07 | 区块、交易、状态接口可用 | 铸凭哈希、择优池、铸造者验证单元测试通过 |
| Fork-Choice | 08 | PoH 接口稳定 | 出块时序、区块竞争归一化、分叉选择测试通过 |
| Team-Validation | 09 | 区块/交易/状态/共识边界稳定 | 角色接口、区块证明包预验证测试通过 |
| Incentives | 10 | 区块与共识稳定 | Coinbase 序列化、奖励分配、兑奖槽测试通过 |
| Services | 11 | 区块/状态/证明接口稳定 | 服务可验证数据 profile 接口测试通过 |
| Integration | 12 | 所有单元测试通过 | `test/` 集成测试和构建通过 |

## 全局验收命令

完整阶段结束后运行：

```bash
go fmt ./...
go test ./...
go test -cover ./...
go build ./...
go mod tidy
go mod verify
golangci-lint run
```

验收输出要求：

- `go fmt ./...` 后无额外 diff。
- `go test ./...` 全部通过。
- `go build ./...` 全部通过。
- `go mod tidy` 后 `go.mod`、`go.sum` 无非预期变更。
- `go mod verify` 通过。
- `golangci-lint run` 无 warning；如工具未安装，记录为环境阻塞。

## 依赖检查

每个阶段后检查 import 方向：

```bash
go list -deps ./...
```

人工确认：

- `pkg/types` 不依赖 `pkg/crypto` 或 `internal/*`。
- `pkg/crypto` 不依赖 `internal/*`。
- `internal/blockchain` 不依赖交易、状态、脚本、共识。
- `internal/tx` 不依赖状态、脚本、共识。
- `internal/utxo` / `internal/utco` 不依赖脚本具体实现或共识。
- `internal/script` 不依赖共识。
- `internal/consensus` 不被低层包 import。
- `internal/validation`、`internal/services`、`internal/rewards` 属 Layer 5，不被 Layer 0-4 import。

## 文档同步要求

编码中若发现 proposal 与实现计划冲突：

1. 暂停相关任务。
2. 在本文件更新对应待决项状态。
3. 更新对应 conception/decision，再更新 `docs/proposal/*.md`。
4. 更新对应 `docs/plan/*.md` 的任务说明。
5. 再继续编码。

> 权威顺序：conception > decision > proposal > plan（遇冲突以 conception 为准）。

## 最终完成定义

整个项目从 Plan 进入可编码状态的条件：

- `docs/plan/` 下存在 1 篇路线图（00）+ 11 篇阶段（01~11）+ 1 篇待决验收（12），共 13 篇。
- 每个阶段计划引用对应 proposal（新编号），列出建议文件、TDD 任务、测试命令与验收标准。
- 全程无 `ADR-xxxx` 残留，决策引用为 `DEC-NNNN` 且主题匹配。
- 待决项严格限于 C-6/C-7/C-9/C-10，相关 Task 显式标注阻塞/占位；已裁决项不再列为待决。
- 用户已确认无需继续拆分更细的阶段方案。
