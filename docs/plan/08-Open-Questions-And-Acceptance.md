# Open Questions And Acceptance Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 汇总阻塞最终协议实现的未决项，并定义全项目验收、测试覆盖、集成测试和文档同步标准。

**Architecture:** 未决项必须在编码时显式隔离，直到 Proposal 或 ADR 固定。验收分为阶段验收、跨层依赖验收、协议向量验收和集成验收。

**Tech Stack:** Markdown、Go 1.26.2、`go test`、`go build`、`go fmt`、`go mod verify`、`golangci-lint`。

---

## 未决项总表

| 编号 | 领域 | 未决项 | 阻塞范围 | 建议处理 |
|------|------|--------|----------|----------|
| OQ-001 | 编码 | canonical unsigned varint 具体算法 | 所有可变长度编码最终向量 | 实施前询问用户或新增 ADR |
| OQ-002 | 编码 | signed integer 是否进入协议编码 | 脚本外的负数语义 | 未确认前只在 VM 值内部使用 |
| OQ-003 | Hash | domain tag 是否进入协议 Hash 输入 | 跨实现 Hash 向量 | 代码提供 domain API，但协议 Hash 不擅自加 tag |
| OQ-004 | Hash Tree | 空树根 | 交易树、状态树、附件树 | 未确认前空树返回 `ErrSpecIncomplete` |
| OQ-005 | Hash Tree | 单叶树根规则 | 所有树 | 通过策略参数显式选择 |
| OQ-006 | Hash Tree | 奇数叶处理策略 | 二元树 | 通过策略参数显式选择 |
| OQ-007 | Block | `YearBlock` 在创世、边界、非边界高度取值 | 区块头编码和验证 | 先实现查询接口并标注规则来源 |
| OQ-008 | Block | `CheckRoot` 是否包含 domain tag、版本、高度 | 区块最终验证向量 | 只实现当前 Proposal 的三根组合 |
| OQ-009 | Tx | Coinbase `HashInputs` 特殊规则 | Coinbase TxID | Coinbase 编码延后最终向量 |
| OQ-010 | Tx | 交易大小是否包含解锁脚本和外层长度 | `MaxTxSize` 验证 | 先只提供配置和结构检查 |
| OQ-011 | Tx | 多签排序、重复公钥/签名处理 | 签名验证 | 先实现接口，不固定多签协议 |
| OQ-012 | State | 四层宽成员树节点组合、空根、年度分区编码 | UTXO/UTCO root | 先实现叶子和分组，不宣称最终 root |
| OQ-013 | Script | VM 初始 pass 状态和 `CHECK` 覆盖规则 | 脚本结果验证 | 先按测试写明临时语义 |
| OQ-014 | Script | 成本公式 | 公共验证 DoS 防护 | 先实现预算框架和保守拒绝 |
| OQ-015 | Address | 文本地址编码、前缀、checksum | CLI/钱包 | 基础层只固定 32B `AddressHash` |
| OQ-016 | PoH | 内层 `Hash(...)` 算法和 domain tag | `MintHash` 最终向量 | 注入 hasher 或返回未决错误 |
| OQ-017 | PoH | `timeStamp * Stakes * Mix` 宽度、溢出和编码 | `MintHash` 最终向量 | 注入 X encoder 或新增 ADR |
| OQ-018 | PoH | `Stakes` 精确定义和单位 | 区块头和 PoH | 独立类型，不参与经济语义测试 |
| OQ-019 | PoH | `MintHash` 相等 tie-breaker | pool/fork 排序 | 相等时返回未决错误 |
| OQ-020 | Fork | 3 倍币权销毁比较是否包含等于 | 区块竞争 | 策略参数化 |
| OQ-021 | Incentive | 发行递减取整规则 | 奖励测试向量 | 正式期向量延后 |
| OQ-022 | Incentive | 交易费奇数最小单位余数归属 | 费用分配 | 策略参数化 |
| OQ-023 | Incentive | 奖励分配余数归属 | Coinbase 输出 | 策略参数化 |
| OQ-024 | Incentive | 兑奖槽 bit 顺序和分叉重组重算 | 公共服务兑奖 | 仅实现 18B 骨架 |
| OQ-025 | Services | 服务奖励地址是否签名绑定服务身份 | 服务奖励安全性 | 接口预留证明字段 |
| OQ-026 | Validation | 首领输入黑名单是否协议规则 | 校验组行为 | 作为 convention 配置 |
| OQ-027 | Validation | 首笔输入币权最大是否协议规则 | 交易合法性 | 作为 convention 检查 |

## 编码时的未决项规则

- 未决项不能用魔法常量悄悄实现。
- 未决项不能产生最终测试向量。
- 未决项可以通过显式策略参数、接口注入、`ErrSpecIncomplete` 或 `TODO(spec)` 标注处理。
- 如果某任务无法在不固定未决项的情况下推进，先询问用户，每次 1 到 3 个问题。
- 一旦用户确认规则，必须回写到对应 Proposal 或新增 ADR，再改 Plan 或代码。

## 单元测试标准

所有包都必须满足：

- 使用 table-driven tests。
- 覆盖成功路径、失败路径和边界值。
- 协议编码测试必须检查字节级输出。
- Hash 测试必须检查输出长度和用途隔离。
- 状态转移测试必须检查前后状态和错误类型。
- 脚本 VM 测试必须检查执行状态、栈、资源计数和公共/私有模式。
- 共识测试必须检查排序、窗口、边界高度和重放防护。

## 集成测试建议

后续创建 `test/` 目录，按以下顺序添加集成测试：

| 文件 | 场景 |
|------|------|
| `test/foundation_encoding_test.go` | 基础编码和 Hash 跨包一致性 |
| `test/header_chain_test.go` | 创世头、连续头、年块、CheckRoot |
| `test/transaction_state_test.go` | Coin 创建、消费、销毁、重复消费拒绝 |
| `test/credit_lifecycle_test.go` | Credit 创建、转移、到期、不可变字段拒绝 |
| `test/script_validation_test.go` | unlock + lock script 公共验证 |
| `test/poh_pool_test.go` | MintHash 排序、BestPool、SyncPool |
| `test/fork_choice_test.go` | 20/29/15 分叉规则 |
| `test/reward_redemption_test.go` | Coinbase 成熟期、兑奖窗口、回收 |

## 覆盖率目标

核心逻辑覆盖率目标为 80% 以上。阶段目标：

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
- `internal/validation`、`internal/services`、`internal/rewards` 属于 Layer 5，不被 Layer 0-4 import。

## 文档同步要求

编码中若发现 Proposal 与实现计划冲突：

1. 暂停相关任务。
2. 在本文件新增未决项或更新对应项状态。
3. 更新对应 `docs/proposal/*.md` 或新增 ADR。
4. 更新对应 `docs/plan/*.md` 的任务说明。
5. 再继续编码。

## 最终完成定义

整个项目从 Plan 进入可编码状态的条件：

- 所有方案文件已写入 `docs/plan/`。
- `docs/AGENTS.md` 已列出 Proposal 到 Plan 的追溯关系。
- 每个阶段的文件清单、任务、测试和验收命令明确。
- 所有当前已知未决项已记录在本文件。
- 用户已确认无需继续拆分更细的阶段方案。
