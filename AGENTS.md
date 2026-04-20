# AGENTS.md - Evidcoin 开发指南

This file provides guidance to AI Coding Agent when working with code in this repository.

## 项目概述

**Evidcoin（证信链）** 是一个基于区块链的通用信用载体系统，使用 Go 1.25+ 开发。目前处于**预实施阶段**——设计文档完整，但尚无生产代码。

项目的核心特色：
- **三种凭证单元**：Coin（可分割货币）、Credit（不可分割证书/合约）、Proof（不可转让存证）
- **历史证明共识（PoH）**：铸币权基于历史交易 ID，而非算力或财富
- **后量子密码学**：使用 ML-DSA-65 签名算法

## 常用命令

```bash
# 构建
go build ./...

# 运行所有测试
go test ./...

# 运行单个测试
go test -v ./path/to/package -run TestName

# 查看测试覆盖率
go test -cover ./...

# 格式化代码
go fmt ./... && gofmt -s -w .

# 静态分析
golangci-lint run

# 整理依赖
go mod tidy && go mod verify
```

各阶段验收标准：`go build ./...` 通过、`go test ./...` 通过、核心逻辑覆盖率 ≥80%、`go fmt` 无变更、`golangci-lint` 无警告。

## 文档三层体系

这是本项目最关键的架构特征，所有实现必须追溯到文档：

| 层级 | 目录 | 作者 | 内容 |
|------|------|------|------|
| 构想层（Tier 1） | `docs/conception/` | 人工编写 | 原始设计思想（中文） |
| 提案层（Tier 2） | `docs/proposal/` | AI 生成 | 详细技术规格，追溯自构想层 |
| 方案层（Tier 3） | `docs/plan/` | AI 生成 | 按阶段的实施计划，含代码骨架 |

实现任何功能前，应先阅读对应的 `docs/plan/` 文件，再追溯到 `docs/proposal/`，如有疑问再查 `docs/conception/`。

## 代码架构

### 分层结构（严格单向依赖）

```
Layer 5 集成层:    cmd/evidcoin/, test/
Layer 4 共识层:    internal/consensus/      ← PoH 实现
Layer 3 脚本层:    internal/script/         ← 栈式虚拟机，254 个操作码
Layer 2 状态层:    internal/utxo/, internal/utco/
Layer 1 核心层:    internal/blockchain/, internal/tx/
Layer 0 基础层:    pkg/types/, pkg/crypto/  ← 无内部依赖
```

**禁止跨层或反向依赖**。高层依赖低层，低层不能 import 高层。

### 8 阶段实施计划

| 阶段 | 重点 | 包 |
|------|------|----|
| 1 | 基础类型与密码学 | `pkg/types/`, `pkg/crypto/` |
| 2 | 区块链核心 | `internal/blockchain/` |
| 3 | 交易模型 | `internal/tx/` |
| 4 | UTXO/UTCO 状态 | `internal/utxo/`, `internal/utco/` |
| 5 | 脚本引擎 | `internal/script/` |
| 6 | PoH 共识 | `internal/consensus/` |
| 7 | 团队验证 | 接口定义 |
| 8 | 服务接口 | 外部服务接口 |

## 关键常量与密码学

### 哈希算法分配

| 用途 | 算法 | 长度 | 备注 |
|------|------|------|----|
| 区块头 | SHA3-384 | 48B | 兼顾量子安全与数据量 |
| CheckRoot（校验根） | SHA3-384 | 48B | 由交易树根、UTXO/UTCO 指纹合并计算 |
| 交易ID（TxID） | SHA3-384 | 48B | 与区块头设计一致 |
| 树结构（分支） | BLAKE3-256 | 32B | 处理多哈希场景，提高性能并节省内存 |
| 树叶子节点 | SHA3-384 | 48B | 与树枝段采用不同算法 |
| 附件指纹 | SHA3-512 | 64B | 确保超长期安全 |
| 公钥哈希（账户地址） | SHA3-256( BLAKE2b-512 ) | 32B | 适量节省空间 |
| UTXO/UTCO 指纹 | SHA3-384 | 48B | 末端叶子节点数据摘要 |
| 铸凭哈希 | BLAKE3-512 | 64B | 大量参与者场景下的速度与信息熵兼顾 |

### 核心常量（实现时参考）

```go
BlockInterval   = 6 * time.Minute
BlocksPerYear   = 87661
MaxStackHeight  = 256
MaxStackItem    = 1024            // 栈项最大字节数
MaxLockScript   = 1024
MaxUnlockScript = 4096
MaxTxSize       = 65535
```

### 计划中的外部依赖

```
golang.org/x/crypto              # SHA3
lukechampine.com/blake3          # BLAKE3
github.com/cloudflare/circl      # ML-DSA-65（Go 1.25 可能已内置，优先用标准库）
github.com/mr-tron/base58        # Base58 地址编码
```

## 代码规范

### 命名

- 接口名：动词+er（`Validator`、`Signer`）
- 结构体/函数：PascalCase
- 私有函数：camelCase
- 常量：PascalCase 或 UPPER_SNAKE

### import 分组（空行分隔）

1. 标准库
2. 第三方库
3. 项目内部包

### 注释

- **中文**：面向作者理解的注释（行内注释、逻辑说明）
- **英文 Godoc**：所有导出符号必须有 Godoc 注释
- **英文**：程序运行时输出的日志和错误消息（`errors.New()`、`log.Println()` 等的实参）

### 并发

- 用 `context.Context` 管理生命周期
- 优先用 channel，而非 mutex
- 禁止裸 goroutine

### 测试

- 单元测试：`*_test.go` 与源文件同目录
- 集成测试：`test/` 目录
- 必须使用表驱动测试（table-driven tests）


## 附：实现边界

- P2P的节点发现和连网分享，归由外部库实现（cxio/p2p...等），本项目仅需考虑合理的接口设计。
- 组队校验中，各个角色单独实现（独立的App），它们之间通过连接相互通讯。因为联系紧密，应考虑高效的通讯方式。
- 区块/交易及附件数据的长期存储，主要由外部第三方公共服务提供。

> **提示：**
> 交易数据的检索由区块查询服务（`Blockqs`）提供。
> 附件数据的获取由数据驿站服务（`Depots`）支持（以文件P2P分享的方式）。
