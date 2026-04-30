# Evidcoin Implementation Roadmap Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 将 `docs/proposal/` 的技术提案转化为后续 Go 编码可直接执行的阶段化实施路线。

**Architecture:** 按项目既定 Layer 0 到 Layer 5 单向依赖推进，先固定基础类型、编码、Hash 与签名抽象，再逐步实现区块、交易、状态、脚本、共识和外部接口边界。所有共识级字节输入必须追溯到 `docs/proposal/` 的规范化编码，所有未决协议细节必须显式隔离，不能在代码中悄悄固化。

**Tech Stack:** Go 1.26.2、标准库、`golang.org/x/crypto`、`lukechampine.com/blake3`、可插拔 ML-DSA-65 签名适配、表驱动测试、`go test ./...`、`go build ./...`、`golangci-lint run`。

---

## 范围

本目录是 `Proposal => Plan` 层。它不编写生产代码，只定义后续编码顺序、包边界、文件清单、TDD 任务、测试策略和验收标准。

本实施路线覆盖以下 Proposal：

- `docs/proposal/00.Project-Scope.md` 到 `docs/proposal/14.Incentives-And-Coinbase-Rewards.md`
- `docs/proposal/Instruction/*.md`

## 总体分层

| 层级 | 包 | 主要职责 | 依赖方向 |
|------|----|----------|----------|
| Layer 0 | `pkg/types/`, `pkg/crypto/` | 基础类型、编码、Hash、签名抽象 | 不依赖 `internal/*` |
| Layer 1 | `internal/blockchain/`, `internal/tx/` | 区块头链、交易头、输入输出、Coinbase | 依赖 Layer 0 |
| Layer 2 | `internal/utxo/`, `internal/utco/` | Coin 与 Credit 状态、状态指纹、回滚 | 依赖 Layer 0-1 |
| Layer 3 | `internal/script/` | 栈式 VM、公共验证、指令注册表 | 依赖 Layer 0-2 的接口，不反向依赖 |
| Layer 4 | `internal/consensus/` | PoH、择优池、同步池、分叉选择 | 依赖 Layer 0-3 的接口 |
| Layer 5 | `internal/validation/`, `internal/services/`, `internal/rewards/`, `cmd/evidcoin/`, `test/` | 校验组接口、公共服务接口、激励结算、组装、集成测试、命令入口 | 依赖所有内部层，不能被 Layer 0-4 反向依赖 |

## 方案文件

| 文件 | 覆盖范围 | 主要输出 |
|------|----------|----------|
| `docs/plan/01-Foundation-Types-Crypto.md` | Proposal 00-04 | `pkg/types/`、`pkg/crypto/`、基础哈希树 |
| `docs/plan/02-Blockchain-Core.md` | Proposal 05 | `internal/blockchain/` |
| `docs/plan/03-Transaction-And-Units.md` | Proposal 06-07、14 部分 | `internal/tx/`、Coin/Credit/Proof、Coinbase 骨架 |
| `docs/plan/04-UTXO-UTCO-State.md` | Proposal 08 | `internal/utxo/`、`internal/utco/` |
| `docs/plan/05-Script-System.md` | Proposal 09、Instruction 0-18 | `internal/script/` |
| `docs/plan/06-PoH-Consensus-And-Fork-Choice.md` | Proposal 10-11 | `internal/consensus/` |
| `docs/plan/07-Team-Validation-Services-Incentives.md` | Proposal 12-14 | 校验组接口、公共服务接口、激励与兑奖 |
| `docs/plan/08-Open-Questions-And-Acceptance.md` | 全部 Proposal | 未决项、全局验收、阶段门禁 |

## 推荐实施顺序

1. 完成 `pkg/types/` 的固定长度类型、ID 类型、常量和规范化编码工具。
2. 完成 `pkg/crypto/` 的 Hash API、地址哈希、签名抽象和测试替身。
3. 完成基础哈希树工具，但对空树、单叶、奇数叶等未决规则只提供显式策略参数。
4. 完成区块头、`BlockID`、头链存储接口和最小衔接验证。
5. 完成交易头、输入输出 envelope、`TxID`、签名消息和 Coin/Credit/Proof payload。
6. 完成 UTXO/UTCO entry、引用解析、重复消费检查和状态指纹。
7. 完成脚本 VM 基础运行时、公共/私有模式和先导指令子集。
8. 完成 PoH `MintHash`、择优池、同步池、确定性区块时间和分叉选择。
9. 完成校验组接口、公共服务可验证数据接口、Coinbase 奖励和兑奖状态。
10. 完成集成测试和命令入口。

## 全局编码原则

- 所有导出符号必须有英文 Godoc。
- 面向作者理解的源码注释使用中文。
- 程序输出、日志和 error 文本使用英文。
- 所有测试使用 table-driven tests。
- 所有协议字节序列必须由显式编码函数生成，不使用 JSON、反射、map 遍历顺序或平台字节序。
- 低层包不能 import 高层包。
- 未决协议细节必须表现为 `TODO(spec)`、策略参数、接口注入或明确拒绝，不能默认选一个值并当作协议事实。

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

| 阶段 | 进入条件 | 退出条件 |
|------|----------|----------|
| Foundation | `go.mod` 存在 | `pkg/types`、`pkg/crypto` 测试通过 |
| Blockchain | Foundation 完成 | 区块头链最小验证通过 |
| Transaction | Blockchain 可引用基础 ID | 交易头、输入输出和 payload 测试通过 |
| State | Transaction 类型稳定 | UTXO/UTCO 状态转移测试通过 |
| Script | 基础类型与状态接口可用 | VM 基础与公共验证测试通过 |
| Consensus | 区块、交易、状态接口可用 | PoH、池、分叉选择单元测试通过 |
| Interfaces | Consensus 边界稳定 | 校验组、服务、激励接口测试通过 |
| Integration | 所有单元测试通过 | `test/` 集成测试和构建通过 |

## 主要风险

- `canonical unsigned varint` 算法尚未最终确认，涉及所有结构体编码和测试向量。
- Hash domain tag 策略未完全固定，不能生成最终跨实现测试向量。
- 哈希树空根、单叶根和奇数叶策略未固定，必须延迟或参数化。
- PoH 内层 Hash、整数宽度、`Stakes` 定义和 tie-breaker 未固定。
- Coinbase 字段、输出顺序、奖励余数、兑奖槽 bit 顺序未固定。
- 脚本 VM 指令全集很大，必须按元数据和公共验证安全边界分批实现。

## 完成定义

本 Plan 被视为完成时，应满足：

- `docs/plan/` 下存在总览、7 个阶段计划和未决项验收文档。
- 每个阶段计划都引用对应 Proposal。
- 每个阶段计划都列出建议文件、TDD 任务、测试命令和验收标准。
- `docs/AGENTS.md` 的 `Proposal => Plan` 对应表已更新。
