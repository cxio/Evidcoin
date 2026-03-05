# Evidcoin 实施方案：总体架构

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 全量实现 Evidcoin 区块链系统——涵盖核心数据结构、交易模型、脚本引擎、PoH 共识与组队校验框架。

**Architecture:** 模块化分层架构，Core 极简化（仅管理区块头链），共识/校验/脚本作为外部可插拔组件。采用 UTXO + UTCO 双集模型，三种信元（Coin/Credit/Proof），栈式脚本系统，ML-DSA-65 抗量子签名方案。

**Tech Stack:** Go 1.25+, SHA-512, SHA3-384/512, BLAKE3, ML-DSA-65 (Dilithium3), circl (crypto library)

---

## 1. 项目目录结构

```
evidcoin/
├── cmd/
│   └── evidcoin/
│       └── main.go                 # 主程序入口
├── internal/
│   ├── blockchain/                 # 区块头链管理（Core）
│   │   ├── blockchain.go           # Blockchain 结构与链操作
│   │   ├── header.go               # BlockHeader 结构与验证
│   │   ├── store.go                # HeaderStore 接口与内存实现
│   │   ├── yearblock.go            # 年块机制
│   │   ├── bootstrap.go            # 初始验证与新节点引导
│   │   └── blockchain_test.go
│   ├── consensus/                  # PoH 共识机制
│   │   ├── minting.go              # 铸凭哈希计算
│   │   ├── bestpool.go             # 择优池管理
│   │   ├── schedule.go             # 铸造时间表（铸币计划）
│   │   ├── fork.go                 # 分叉检测与解决
│   │   ├── bootstrap.go            # 引导启动规则
│   │   ├── coinday.go              # 币权（币天）计算
│   │   └── consensus_test.go
│   ├── script/                     # 栈式脚本引擎
│   │   ├── engine.go               # ScriptEngine 核心
│   │   ├── opcode.go               # 指令定义与枚举
│   │   ├── stack.go                # 数据栈实现
│   │   ├── argspace.go             # 实参区实现
│   │   ├── scope.go                # 局部域与全局变量
│   │   ├── value.go                # 值指令 (NIL, TRUE, FALSE, DATA{}, ...)
│   │   ├── flow.go                 # 流程控制 (IF, ELSE, SWITCH, EACH, ...)
│   │   ├── arithmetic.go           # 运算指令
│   │   ├── compare.go              # 比较与逻辑指令
│   │   ├── crypto.go               # 密码学函数指令 (FN_HASH*, FN_CHECKSIG, ...)
│   │   ├── system.go               # 系统指令 (SYS_CHKPASS, SYS_TIME, SYS_AWARD)
│   │   ├── result.go               # 结果指令 (PASS, CHECK, GOTO, EMBED, EXIT, ...)
│   │   ├── pattern.go              # 模式匹配 (MODEL{}, 通配符, ...)
│   │   ├── environment.go          # 环境指令 (ENV, IN, OUT, VAR, ...)
│   │   ├── pipeline.go             # 解锁→锁定执行管道
│   │   └── script_test.go
│   ├── tx/                         # 交易结构
│   │   ├── header.go               # TxHeader 与 TxID 计算
│   │   ├── input.go                # LeadInput, RestInput
│   │   ├── output.go               # Output（Coin/Credit/Proof/Mediator）
│   │   ├── coinbase.go             # CoinbaseTx
│   │   ├── attachment.go           # AttachmentID 与分片验证
│   │   ├── hashtree.go             # 输入/输出哈希树
│   │   ├── sigflag.go              # SigFlag 授权标志
│   │   ├── fee.go                  # 费用计算与优先级
│   │   └── tx_test.go
│   ├── utxo/                       # UTXO 集与指纹
│   │   ├── set.go                  # UTXO 集管理
│   │   ├── fingerprint.go          # 4 级层次哈希指纹
│   │   ├── entry.go                # UTXOEntry 结构
│   │   ├── cache.go                # UTXO 缓存器实现
│   │   └── utxo_test.go
│   └── utco/                       # UTCO 集与指纹
│       ├── set.go                  # UTCO 集管理
│       ├── fingerprint.go          # 4 级层次哈希指纹
│       ├── entry.go                # UTCOEntry 结构
│       ├── expiry.go               # 凭信过期与高度索引
│       ├── cache.go                # UTCO 缓存器实现
│       └── utco_test.go
├── pkg/
│   ├── types/                      # 公共类型定义
│   │   ├── hash.go                 # Hash512, Hash384 等
│   │   ├── address.go              # 地址编码与校验
│   │   ├── config.go               # OutputConfig, SigFlag 等配置类型
│   │   ├── constants.go            # 全局常量
│   │   ├── varint.go               # 变长整数编解码
│   │   └── types_test.go
│   └── crypto/                     # 密码学原语
│       ├── hash.go                 # SHA-512, SHA3-384, SHA3-512, BLAKE3 封装
│       ├── sign.go                 # 签名接口（Signer/Verifier）
│       ├── mldsa.go                # ML-DSA-65 实现
│       ├── multisig.go             # 多重签名逻辑
│       ├── keys.go                 # 密钥对生成与管理
│       └── crypto_test.go
├── config/
│   └── genesis.go                  # 创世块配置
├── test/                           # 集成测试
│   ├── blockchain_integration_test.go
│   ├── tx_integration_test.go
│   └── script_integration_test.go
├── go.mod
├── go.sum
└── README.md
```


## 2. 模块依赖关系

```
Layer 0 (无依赖):
  pkg/types          ← 基础类型、常量
  pkg/crypto         ← 密码学原语（仅依赖 types）

Layer 1 (依赖 Layer 0):
  internal/tx        ← 交易结构（依赖 types, crypto）
  internal/blockchain ← 区块头链（依赖 types, crypto）

Layer 2 (依赖 Layer 0-1):
  internal/utxo      ← UTXO 集（依赖 types, crypto, tx）
  internal/utco      ← UTCO 集（依赖 types, crypto, tx）

Layer 3 (依赖 Layer 0-2):
  internal/script    ← 脚本引擎（依赖 types, crypto, tx, utxo, utco）

Layer 4 (依赖 Layer 0-3):
  internal/consensus ← PoH 共识（依赖 types, crypto, tx, blockchain, utxo, utco）

Layer 5 (集成层):
  cmd/evidcoin       ← 主程序（集成所有模块）
  test/              ← 集成测试
```


## 3. 实施阶段规划

### Phase 1：基础层（Foundation）
**目标**：建立所有模块依赖的基础类型和密码学原语。

| 任务 | 文件 | 描述 |
|------|------|------|
| 1.1 | `pkg/types/*.go` | Hash512, Hash384, PubKeyHash, 常量, OutputConfig, SigFlag, varint |
| 1.2 | `pkg/crypto/hash.go` | SHA-512, SHA3-384, SHA3-512, BLAKE3 封装 |
| 1.3 | `pkg/crypto/sign.go` | Signer/Verifier 接口定义 |
| 1.4 | `pkg/crypto/mldsa.go` | ML-DSA-65 签名实现 |
| 1.5 | `pkg/crypto/multisig.go` | 多重签名逻辑 |
| 1.6 | `pkg/types/address.go` | 地址编码/解码/验证 |

**详细方案**：`plan/01-foundation-types-crypto.md`

### Phase 2：区块链核心（Blockchain Core）
**目标**：实现极简的区块头链管理。

| 任务 | 文件 | 描述 |
|------|------|------|
| 2.1 | `internal/blockchain/header.go` | BlockHeader 结构与 BlockID 计算 |
| 2.2 | `internal/blockchain/store.go` | HeaderStore 接口与内存实现 |
| 2.3 | `internal/blockchain/blockchain.go` | Blockchain 核心逻辑 |
| 2.4 | `internal/blockchain/yearblock.go` | 年块机制 |
| 2.5 | `internal/blockchain/bootstrap.go` | 初始验证 |

**详细方案**：`plan/02-blockchain-core.md`

### Phase 3：交易模型（Transaction Model）
**目标**：实现完整的交易数据结构。

| 任务 | 文件 | 描述 |
|------|------|------|
| 3.1 | `internal/tx/header.go` | TxHeader 与 TxID 计算 |
| 3.2 | `internal/tx/input.go` | LeadInput, RestInput |
| 3.3 | `internal/tx/output.go` | 三类信元输出 |
| 3.4 | `internal/tx/hashtree.go` | 输入/输出哈希树 |
| 3.5 | `internal/tx/coinbase.go` | Coinbase 交易 |
| 3.6 | `internal/tx/attachment.go` | 附件 ID 与分片 |
| 3.7 | `internal/tx/sigflag.go` | 签名授权标志 |
| 3.8 | `internal/tx/fee.go` | 费用与优先级 |

**详细方案**：`plan/03-transaction-model.md`

### Phase 4：UTXO/UTCO 集（State Management）
**目标**：实现 UTXO/UTCO 集管理与 4 级层次哈希指纹。

| 任务 | 文件 | 描述 |
|------|------|------|
| 4.1 | `internal/utxo/entry.go` | UTXOEntry 结构 |
| 4.2 | `internal/utxo/set.go` | UTXO 集管理 |
| 4.3 | `internal/utxo/fingerprint.go` | 4 级哈希指纹 |
| 4.4 | `internal/utxo/cache.go` | UTXO 缓存器 |
| 4.5 | `internal/utco/entry.go` | UTCOEntry 结构 |
| 4.6 | `internal/utco/set.go` | UTCO 集管理 |
| 4.7 | `internal/utco/fingerprint.go` | 4 级哈希指纹 |
| 4.8 | `internal/utco/expiry.go` | 凭信过期索引 |
| 4.9 | `internal/utco/cache.go` | UTCO 缓存器 |

**详细方案**：`plan/04-utxo-utco.md`

### Phase 5：脚本引擎（Script Engine）
**目标**：实现脚本引擎框架与约 30 条核心指令。

| 任务 | 文件 | 描述 |
|------|------|------|
| 5.1 | `internal/script/opcode.go` | Opcode 枚举（全 254 条定义，核心子集实现） |
| 5.2 | `internal/script/engine.go` | ScriptEngine 核心结构 |
| 5.3 | `internal/script/stack.go` | 数据栈实现 |
| 5.4 | `internal/script/argspace.go` | 实参区实现 |
| 5.5 | `internal/script/value.go` | 值指令（NIL, TRUE, FALSE, DATA{}, ...） |
| 5.6 | `internal/script/flow.go` | 流程控制（IF, ELSE, EACH, ...） |
| 5.7 | `internal/script/result.go` | 结果指令（PASS, CHECK, EXIT, ...） |
| 5.8 | `internal/script/crypto.go` | FN_HASH*, FN_CHECKSIG |
| 5.9 | `internal/script/system.go` | SYS_CHKPASS, SYS_TIME, SYS_AWARD |
| 5.10 | `internal/script/pipeline.go` | 解锁→锁定执行管道 |

**详细方案**：`plan/05-script-engine.md`

### Phase 6：PoH 共识（Consensus）
**目标**：实现历史证明共识机制。

| 任务 | 文件 | 描述 |
|------|------|------|
| 6.1 | `internal/consensus/minting.go` | 铸凭哈希计算 |
| 6.2 | `internal/consensus/bestpool.go` | 择优池管理 |
| 6.3 | `internal/consensus/schedule.go` | 铸造时间表 |
| 6.4 | `internal/consensus/fork.go` | 分叉处理 |
| 6.5 | `internal/consensus/bootstrap.go` | 引导启动 |
| 6.6 | `internal/consensus/coinday.go` | 币权计算 |

**详细方案**：`plan/06-consensus-poh.md`

### Phase 7：组队校验框架（Verification Framework）
**目标**：定义组队校验的接口与协作协议。

| 任务 | 文件 | 描述 |
|------|------|------|
| 7.1 | 接口定义 | Guard/Verifier/Dispatcher/Broadcaster 接口 |
| 7.2 | 首领校验 | Leader Verification 实现 |
| 7.3 | 冗余校验 | 扩展复核机制 |

**详细方案**：`plan/07-team-verification.md`

### Phase 8：公共服务接口（Service Interfaces）
**目标**：定义与第三方服务交互的接口。

| 任务 | 文件 | 描述 |
|------|------|------|
| 8.1 | 接口定义 | BlockqsClient, DepotsClient, StunClient |
| 8.2 | 奖励分配 | 收益分配与兑换机制 |

**详细方案**：`plan/08-services-interfaces.md`


## 4. 关键设计决策

### 4.1 签名方案：ML-DSA-65

| 属性 | ML-DSA-65 |
|------|-----------|
| 公钥大小 | 1,952 B |
| 签名大小 | 3,293 B |
| 安全级别 | NIST Level 3（抗量子） |

**实现要点**：
- 公钥哈希为 SHA3-384(MLDSAPubKey) = 48 字节，外观不暴露签名方案
- 签名数据与交易 ID 分离，不影响交易大小限制
- `SYS_CHKPASS` 指令从环境而非数据栈获取签名，确保签名不进入脚本流

### 4.2 哈希算法分工

| 用途 | 算法 | 输出长度 | 理由 |
|------|------|----------|------|
| 区块哈希 / TxID / 铸凭哈希 | SHA-512 | 64 B | 默认高安全哈希 |
| UTXO/UTCO 指纹 | SHA3-512 | 64 B | 与 SHA-512 区分，增强安全 |
| 公钥哈希 / 分片树 | SHA3-384 | 48 B | 空间与安全的折中 |
| 附件指纹 | BLAKE3 | 16–64 B | 高速流式，可变长度 |

### 4.3 外部依赖

| 依赖 | 用途 |
|------|------|
| `github.com/cloudflare/circl` | ML-DSA-65 (Dilithium3) |
| `lukechampine.com/blake3` | BLAKE3 哈希 |
| `golang.org/x/crypto` | SHA3 (如标准库不含) |

> **注**：Go 1.25 可能已原生支持 ML-DSA，届时优先使用标准库。


## 5. 验收标准

每个 Phase 完成后应满足：

1. **编译通过**：`go build ./...` 无错误
2. **测试通过**：`go test ./...` 全部 PASS
3. **覆盖率**：核心逻辑 ≥ 80%
4. **格式规范**：`go fmt ./...` 无变更
5. **静态检查**：`golangci-lint run` 无警告（如已安装）


## 6. 后续方案索引

| 方案文件 | 内容 |
|----------|------|
| `plan/01-foundation-types-crypto.md` | 基础类型与密码学原语 |
| `plan/02-blockchain-core.md` | 区块头链管理 |
| `plan/03-transaction-model.md` | 交易结构与模型 |
| `plan/04-utxo-utco.md` | UTXO/UTCO 集与指纹 |
| `plan/05-script-engine.md` | 脚本引擎框架与核心指令 |
| `plan/06-consensus-poh.md` | PoH 共识与铸造 |
| `plan/07-team-verification.md` | 组队校验接口 |
| `plan/08-services-interfaces.md` | 公共服务接口 |
