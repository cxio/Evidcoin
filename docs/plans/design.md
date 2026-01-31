# Evidcoin 实现方案（Implementation Plan）

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**目标：** 实现一个基于历史证明（PoH）共识机制的通用信用载体区块链系统，支持币金、凭信、存证三种基本信元。

**架构：** 采用模块化设计，各模块职责明确、接口清晰。核心模块包括密码学、类型定义、脚本系统、交易、UTXO、共识、组队校验和区块链。P2P 网络层和公共服务（depots, blockqs, stun2p）作为独立项目开发，本项目仅定义接口。

**技术栈：**
- Go 1.21+
- SHA-512（64字节哈希）/ BLAKE3（附件哈希）
- Ed25519 签名（预留后量子算法扩展）
- Base58Check 地址编码

---

## 目录

1. [架构概览](#架构概览)
2. [模块依赖关系](#模块依赖关系)
3. [项目结构](#项目结构)
4. [模块详细设计](#模块详细设计)
5. [开发路线图](#开发路线图)
6. [关键常量定义](#关键常量定义)

---

## 架构概览

```graph
┌─────────────────────────────────────────────────────────────────┐
│                        cmd/evidcoin                             │
│                        (主程序入口)                             │
└─────────────────────────────────────────────────────────────────┘
                                │
                                V
┌─────────────────────────────────────────────────────────────────┐
│                     internal/blockchain                         │
│                     (区块链核心逻辑)                            │
└─────────────────────────────────────────────────────────────────┘
           │              │              │              │
           V              V              V              V
┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐
│  consensus  │  │  checkteam  │  │    utxo     │  │     tx      │
│  (PoH共识)  │  │ (组队校验)  │  │  (UTXO管理) │  │   (交易)    │
└─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘
           │              │              │              │
           └──────────────┴──────────────┴──────────────┘
                                │
                                V
                    ┌─────────────────────┐
                    │       script        │
                    │     (脚本系统)      │
                    └─────────────────────┘
                                │
                                V
          ┌─────────────────────┴─────────────────────┐
          │                                           │
          V                                           V
┌─────────────────────┐                   ┌─────────────────────┐
│     pkg/types       │                   │     pkg/crypto      │
│   (基础类型定义)    │                   │   (密码学基础库)    │
└─────────────────────┘                   └─────────────────────┘
```

### 外部依赖（独立项目）

```graph
┌──────────────────────────────────────────────────────────────────┐
│                      External Projects                           │
├─────────────────┬─────────────────┬──────────────────────────────┤
│   cxio/p2p      │  cxio/depots    │  cxio/blockqs  │ cxio/stun2p │
│  (节点发现)     │  (数据驿站)     │  (区块查询)    │ (NAT服务)   │
└─────────────────┴─────────────────┴──────────────────────────────┘
```

---

## 模块依赖关系

```graph
M1:crypto ◄───────────────────────────┐
    │                                  │
    V                                  │
M2:types ◄────────────────────────────┤
    │                                  │
    V                                  │
M3:script ◄───────────────────────────┤
    │                                  │
    V                                  │
M4:tx ◄───────────────────────────────┤
    │                                  │
    V                                  │
M5:utxo ◄─────────────────────────────┤
    │                                  │
    V                                  │
M6:consensus ◄────────────────────────┤
    │                                  │
    V                                  │
M7:checkteam ◄────────────────────────┤
    │                                  │
    V                                  │
M8:blockchain ─────────────────────────┘
```

| 模块 | 包路径 | 说明 | 预估复杂度 |
|------|--------|------|------------|
| M1 | `pkg/crypto` | 密码学基础库：哈希、签名、地址编码 | 中 |
| M2 | `pkg/types` | 基础类型：Hash512、Address、VarInt 等 | 低 |
| M3 | `internal/script` | 栈式脚本系统：~170个基础指令 | 高 |
| M4 | `internal/tx` | 交易结构：币金、凭信、存证、输入输出 | 高 |
| M5 | `internal/utxo` | UTXO 集合管理与指纹计算 | 中 |
| M6 | `internal/consensus` | PoH 共识：铸凭哈希、择优池、分叉处理 | 高 |
| M7 | `internal/checkteam` | 组队校验：管理层、守卫者、校验员 | 高 |
| M8 | `internal/blockchain` | 区块链核心：区块、链管理、存储 | 高 |

---

## 项目结构

```
evidcoin/
├── cmd/
│   └── evidcoin/
│       └── main.go              # 主程序入口
│
├── pkg/                         # 公共包（可被外部引用）
│   ├── crypto/
│   │   ├── hash.go              # SHA-512, BLAKE3 哈希
│   │   ├── signature.go         # Ed25519 签名（预留扩展）
│   │   ├── address.go           # Base58Check 地址编码
│   │   └── crypto_test.go
│   │
│   └── types/
│       ├── hash.go              # Hash512, Hash256 类型
│       ├── address.go           # Address 类型（48字节）
│       ├── varint.go            # 变长整数编解码
│       ├── amount.go            # 金额类型
│       └── types_test.go
│
├── internal/                    # 私有包
│   ├── script/
│   │   ├── opcode.go            # 指令码定义
│   │   ├── stack.go             # 数据栈实现
│   │   ├── engine.go            # 脚本执行引擎
│   │   ├── ops_value.go         # 值指令 [0-19]
│   │   ├── ops_capture.go       # 截取指令 [20-24]
│   │   ├── ops_stack.go         # 栈操作指令 [25-35]
│   │   ├── ops_collection.go    # 集合指令 [36-45]
│   │   ├── ops_io.go            # 交互指令 [46-49]
│   │   ├── ops_result.go        # 结果指令 [50-56]
│   │   ├── ops_flow.go          # 流程控制 [57-66]
│   │   ├── ops_convert.go       # 转换指令 [67-79]
│   │   ├── ops_math.go          # 运算指令 [80-103]
│   │   ├── ops_compare.go       # 比较指令 [104-111]
│   │   ├── ops_logic.go         # 逻辑指令 [112-119]
│   │   ├── ops_pattern.go       # 模式指令 [120-134]
│   │   ├── ops_env.go           # 环境指令 [135-144]
│   │   ├── ops_util.go          # 工具指令 [145-159]
│   │   ├── ops_system.go        # 系统指令 [160-169]
│   │   ├── ops_function.go      # 函数指令 [170-209]
│   │   ├── ops_module.go        # 模块指令 [210-249]
│   │   ├── ops_extend.go        # 扩展指令 [250-254]
│   │   ├── compiler.go          # 脚本编译器
│   │   ├── disasm.go            # 反汇编器
│   │   └── script_test.go
│   │
│   ├── tx/
│   │   ├── header.go            # 交易头
│   │   ├── input.go             # 输入项
│   │   ├── output.go            # 输出项（币金、凭信、存证）
│   │   ├── coinbase.go          # 铸币交易
│   │   ├── transaction.go       # 完整交易结构
│   │   ├── validation.go        # 交易验证
│   │   ├── attachment.go        # 附件ID结构
│   │   └── tx_test.go
│   │
│   ├── utxo/
│   │   ├── utxo.go              # UTXO 条目
│   │   ├── set.go               # UTXO 集合
│   │   ├── fingerprint.go       # UTXO 指纹计算
│   │   ├── tree.go              # 哈希校验树
│   │   └── utxo_test.go
│   │
│   ├── consensus/
│   │   ├── poh.go               # 历史证明核心
│   │   ├── mint_hash.go         # 铸凭哈希计算
│   │   ├── preference_pool.go   # 择优池
│   │   ├── minter.go            # 铸造者逻辑
│   │   ├── fork.go              # 分叉处理
│   │   ├── params.go            # 共识参数
│   │   └── consensus_test.go
│   │
│   ├── checkteam/
│   │   ├── team.go              # 校验组定义
│   │   ├── manager.go           # 管理层（广播+调度）
│   │   ├── guardian.go          # 守卫者
│   │   ├── validator.go         # 校验员
│   │   ├── lead_check.go        # 首领校验
│   │   ├── redundancy.go        # 冗余校验与扩展复核
│   │   ├── blacklist.go         # 黑名单管理
│   │   └── checkteam_test.go
│   │
│   ├── blockchain/
│   │   ├── block.go             # 区块结构
│   │   ├── header.go            # 区块头
│   │   ├── chain.go             # 链管理
│   │   ├── genesis.go           # 创世区块
│   │   ├── storage.go           # 存储接口
│   │   ├── mempool.go           # 交易池
│   │   └── blockchain_test.go
│   │
│   └── network/
│       ├── interface.go         # P2P 网络接口定义
│       ├── message.go           # 消息类型定义
│       └── mock.go              # 测试用 Mock 实现
│
├── test/                        # 集成测试
│   ├── integration_test.go
│   └── scenarios/
│
├── docs/
│   └── plans/
│       ├── design.md            # 本文档
│       ├── design-m1-crypto.md
│       ├── design-m2-types.md
│       ├── design-m3-script.md
│       ├── design-m4-tx.md
│       ├── design-m5-utxo.md
│       ├── design-m6-consensus.md
│       ├── design-m7-checkteam.md
│       └── design-m8-blockchain.md
│
├── conception/                  # 设计构想文档（已有）
│
├── go.mod
├── go.sum
├── AGENTS.md
├── LICENSE
└── README.md
```

---

## 模块详细设计

各模块的详细设计请参阅：

| 模块 | 设计文档 | 状态 |
|------|----------|------|
| M1: crypto | [design-m1-crypto.md](design-m1-crypto.md) | ✅ |
| M2: types | [design-m2-types.md](design-m2-types.md) | ✅ |
| M3: script | [design-m3-script.md](design-m3-script.md) | ✅ |
| M4: tx | [design-m4-tx.md](design-m4-tx.md) | ✅ |
| M5: utxo | [design-m5-utxo.md](design-m5-utxo.md) | ✅ |
| M6: consensus | [design-m6-consensus.md](design-m6-consensus.md) | ✅ |
| M7: checkteam | [design-m7-checkteam.md](design-m7-checkteam.md) | ✅ |
| M8: blockchain | [design-m8-blockchain.md](design-m8-blockchain.md) | ✅ |

---

## 开发路线图

### 阶段一：基础设施（2-3 周）

```
Week 1-2: M1 crypto + M2 types
Week 2-3: M3 script (核心框架 + 基础指令)
```

- [ ] M1: 完成密码学基础库
- [ ] M2: 完成基础类型定义
- [ ] M3: 完成脚本引擎框架和值指令、栈操作指令

### 阶段二：交易与验证（2-3 周）

```
Week 3-4: M3 script (完整指令集)
Week 4-5: M4 tx + M5 utxo
```

- [ ] M3: 完成全部 ~170 个脚本指令
- [ ] M4: 完成交易结构与验证
- [ ] M5: 完成 UTXO 管理与指纹

### 阶段三：共识机制（2-3 周）

```
Week 5-6: M6 consensus
Week 6-7: M7 checkteam
```

- [ ] M6: 完成 PoH 共识机制
- [ ] M7: 完成组队校验系统

### 阶段四：区块链核心（2 周）

```
Week 7-8: M8 blockchain
Week 8: 集成测试与优化
```

- [ ] M8: 完成区块链核心逻辑
- [ ] 集成测试
- [ ] 性能优化

### 阶段五：网络集成（待定）

```
依赖 cxio/p2p 项目进度
```

- [ ] 集成 P2P 网络层
- [ ] 实现节点通信协议
- [ ] 端到端测试

---

## 关键常量定义

以下常量应在 `pkg/types/constants.go` 中定义：

```go
package types

import "time"

const (
    // 哈希长度
    HashLength      = 64                  // SHA-512 哈希长度（字节）
    AddressLength   = 48                  // 地址长度（字节）
    ShortHashLength = 20                  // 短哈希长度（用于非首领输入）

    // 时间参数
    BlockInterval   = 6 * time.Minute     // 出块间隔
    BlocksPerYear   = 87661               // 每年区块数（恒星年）

    // 脚本限制
    MaxStackHeight  = 256                 // 脚本栈最大高度
    MaxStackItem    = 1024                // 栈数据项最大尺寸（字节）
    MaxLockScript   = 1024                // 锁定脚本最大长度
    MaxUnlockScript = 4096                // 解锁脚本最大长度

    // 交易限制
    MaxTxSize       = 8192                // 单笔交易最大尺寸（字节，不含解锁）
    MaxOutputCount  = 2048                // 最大输出项数量
    TxExpireBlocks  = 240                 // 交易过期区块数（24小时）

    // 共识参数
    EvalBlockOffset    = 9                // 评参区块偏移
    MintTxMinHeight    = 25               // 铸凭交易最小高度偏移
    MintTxMaxHeight    = 80000            // 铸凭交易最大高度偏移（约11个月）
    PreferencePoolSize = 20               // 择优池容量
    ForkCompeteBlocks  = 25               // 分叉竞争区块数
    CoinbaseConfirms   = 25               // 新币确认数

    // 铸造冗余
    MintDelayFirst     = 30 * time.Second // 首个区块延迟发布
    MintDelayStep      = 15 * time.Second // 候选者间隔

    // 组队校验
    RedundancyFactor   = 2                // 冗余校验系数
    ReviewBlocks       = 48               // 兑奖评估区块数
    MinConfirms        = 2                // 最低确认数

    // 激励参数
    MinterShare        = 50               // 铸造者分成（%）
    DepotsShare        = 20               // depots 分成（%）
    BlockqsShare       = 20               // blockqs 分成（%）
    Stun2pShare        = 10               // stun2p 分成（%）
    TxFeeBurn          = 50               // 交易费销毁比例（%）

    // 附件参数
    MaxAttachmentSlice = 2 * 1024 * 1024  // 分片最大尺寸（2MB）
)
```

---

## 下一步

请选择您希望开始实现的模块，或者查看各模块的详细设计文档。

建议从 **M1: crypto** 和 **M2: types** 开始，因为它们是其他所有模块的基础依赖。
