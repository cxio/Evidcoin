# Evidcoin 实现方案（Implementation Plan）

## 一、项目概述

Evidcoin（证信链）是基于区块链技术的通用信用载体系统，采用历史证明（PoH）共识机制，固定 6 分钟出块时间。

**本方案范围**：区块链核心功能 + 信用结构 + 脚本系统

**不含**：公共服务网络（depots, blockqs, stun2p）

---

## 二、技术选型

| 领域 | 技术选择 | 说明 |
|------|---------|------|
| 语言 | Go 1.25+ | 强类型、高并发原生支持 |
| 哈希算法 | SHA-512（主）、BLAKE3（附件） | 64 字节哈希 |
| 签名算法 | Ed25519 | 未来可扩展抗量子算法 |
| 序列化 | 自定义二进制格式 | 紧凑高效 |
| 存储 | Badger 或 LevelDB | 键值存储 |
| P2P 网络 | github.com/cxio/p2p（外部） | 假设可用 |

---

## 三、模块架构

```
evidcoin/
├── cmd/
│   └── evidcoin/          # 主程序入口
├── internal/
│   ├── blockchain/        # 区块链核心（区块、区块头、链管理）
│   ├── consensus/         # 共识机制（PoH、择优池、铸造）
│   ├── tx/                # 交易处理（交易头、输入输出、验证）
│   ├── credit/            # 信用结构（币金、凭信、存证）
│   ├── script/            # 脚本虚拟机（解析、执行、指令集）
│   ├── utxo/              # UTXO 管理（集合、指纹）
│   ├── crypto/            # 加密原语（哈希、签名、地址编码）
│   ├── storage/           # 持久化存储
│   └── mempool/           # 未确认交易池
├── pkg/
│   ├── types/             # 公共类型定义（Hash512、Address 等）
│   └── encoding/          # 编解码工具
└── test/                  # 集成测试
```

### 模块依赖关系

```
              +-----------------+
              │       cmd/      │
              +-----------------+
                       │
      +----------------+----------------+
      |                |                |
      V                V                V
+------------+   +------------+   +------------+
| blockchain |   | consensus  |   |  mempool   |
+-----+------+   +-----+------+   +-----+------+
      |                |                |
      V                V                V
+----------------------+----------------------+
│                     tx                      │
+----------------------+----------------------+
                       │
     +-----------------+-----------------+
     |                 |                 |
     V                 V                 V
+----------+     +----------+     +----------+
|  credit  |     |  script  |     |   utxo   |
+-----+----+     +-----+----+     +-----+----+
      |                |                |
      +----------------+----------------+
                       |
                       V
              +----------------+
              |     crypto     |
              +--------+-------+
                       |
                       V
              +----------------+
              |    pkg/types   |
              +----------------+
```

---

## 四、分阶段实施计划

### 第一阶段：基础类型与加密原语

**目标**：构建项目基础设施

**模块**：
- `pkg/types/` - 核心类型定义
- `internal/crypto/` - 加密原语

**关键类型**：

```go
// pkg/types/hash.go
type Hash512 [64]byte         // SHA-512 哈希
type Hash256 [32]byte         // BLAKE3 哈希（附件用）
type Address [48]byte         // 公钥地址
type Signature []byte         // 签名数据

// pkg/types/constants.go
const (
    HashLength      = 64
    AddressLength   = 48
    BlockInterval   = 6 * time.Minute
    BlocksPerYear   = 87661
    MaxStackHeight  = 256
    MaxStackItem    = 1024
    MaxLockScript   = 1024
    MaxUnlockScript = 4096
    MaxTxSize       = 8192
)
```

**里程碑**：
- 基础类型定义完成
- SHA-512、BLAKE3 哈希封装
- Ed25519 签名/验签封装
- 地址编码/解码（Base32）
- 单元测试覆盖

---

### 第二阶段：信用结构实现

**目标**：实现三种基本信元

**模块**：`internal/credit/`

**数据结构**：

```go
// 币金输出
type CoinOutput struct {
    Amount     int64     // 金额（最小单位）
    Receiver   Address   // 接收者地址
    Note       []byte    // 附言（≤255 字节）
    LockScript []byte    // 锁定脚本
}

// 凭信输出
type CredentialOutput struct {
    Receiver   Address   // 接收者
    Creator    []byte    // 创建者或引用
    Config     uint16    // 配置位
    Title      string    // 标题
    Desc       []byte    // 描述（≤1KB）
    AttachID   []byte    // 附件 ID（可选）
    LockScript []byte    // 锁定脚本
}

// 存证输出
type EvidenceOutput struct {
    Creator      []byte    // 创建者
    Title        string    // 标题
    Content      []byte    // 内容（≤2KB）
    AttachID     []byte    // 附件 ID（可选）
    IdentScript  []byte    // 识别脚本
}
```

**里程碑**：
- 三种信元结构定义
- 信元序列化/反序列化
- 凭信转移规则验证
- 输出配置解析

---

### 第三阶段：交易系统

**目标**：完成交易数据结构与验证

**模块**：`internal/tx/`

**数据结构**：

```go
// 交易头
type TxHeader struct {
    Version   int       // 版本号
    Timestamp int64     // 时间戳（Unix 纳秒）
    HashBody  Hash512   // 数据体哈希
}

// 输入项
type TxInput struct {
    Year     int       // 交易年度
    TxID     []byte    // 交易 ID（首领 64 字节，其余 20 字节）
    OutIndex int       // 输出序位
    TransferIndex int  // 转出序位（凭信专用）
    UnlockData []byte  // 解锁数据
    UnlockScript []byte // 解锁脚本
}

// 完整交易
type Transaction struct {
    Header  TxHeader      // 交易头
    Inputs  []TxInput     // 输入项
    Outputs []Output      // 输出项（多态）
}

// 铸币交易（Coinbase）
type Coinbase struct {
    Height       int           // 区块高度
    MintProof    MintProof     // 择优凭证
    TotalReward  int64         // 收益总额
    Outputs      []CoinOutput  // 收益分成
    FreeData     []byte        // 自由数据（≤256 字节）
}
```

**里程碑**：
- 交易头与交易体结构
- 输入/输出项处理
- 交易 ID 计算（哈希）
- 交易序列化格式
- 首领输入校验逻辑
- 交易过期检查（240 区块）

---

### 第四阶段：脚本虚拟机

**目标**：实现栈式脚本解释器

**模块**：`internal/script/`

**核心组件**：

```go
// 执行环境
type ScriptVM struct {
    stack    []any       // 数据栈（≤256 条目）
    args     []any       // 实参区
    locals   []any       // 局部域（≤128）
    globals  map[string]any // 全局域
    passed   bool        // 通关状态
}

// 指令定义
type Opcode byte

const (
    OP_NIL Opcode = iota
    OP_TRUE
    OP_FALSE
    OP_BYTE
    OP_RUNE
    OP_INT
    // ... 170+ 基础指令
)

// 指令执行器接口
type OpcodeHandler func(vm *ScriptVM, params []byte) error
```

**实施分步**：

1. **值指令**（0-19）：NIL, TRUE, FALSE, 数值类型, DATA{} 等
2. **截取指令**（20-24）：@, ~, $, $[n], $X
3. **栈操作指令**（25-35）：PUSH, POP, TOP, PEEK 等
4. **集合指令**（36-45）：SLICE, MERGE, SPREAD 等
5. **交互指令**（46-49）：INPUT, OUTPUT, BUFDUMP, PRINT
6. **结果指令**（50-56）：PASS, CHECK, GOTO, EXIT 等
7. **流程控制**（57-66）：IF, ELSE, SWITCH, EACH, BREAK 等
8. **转换指令**（67-79）：类型转换系列
9. **运算指令**（80-103）：数学运算、位运算
10. **比较/逻辑指令**（104-119）
11. **环境/工具指令**（120-160）
12. **系统/函数指令**（SYS_*, FN_*）
13. **模块/扩展指令**（MO_*, EX_*）

**里程碑**：
- 词法分析器（源码 -> Token）
- 语法分析器（Token -> 指令序列）
- 虚拟机执行引擎
- 基础指令集实现（170 个）
- 函数指令（FN_HASH256, FN_CHECKSIG 等）
- SYS_CHKPASS 内置验证
- 脚本执行沙箱与资源限制

---

### 第五阶段：UTXO 管理

**目标**：实现 UTXO 集合与指纹

**模块**：`internal/utxo/`

**数据结构**：

```go
// UTXO 条目
type UTXOEntry struct {
    TxID     Hash512   // 交易 ID
    OutIndex int       // 输出序位
    Output   Output    // 输出内容
    Height   int       // 所在区块高度
}

// UTXO 集合
type UTXOSet struct {
    entries map[string]*UTXOEntry  // 按 TxID+Index 索引
    // 四层哈希树用于指纹计算
    yearTree map[int]*HashNode
}

// UTXO 指纹
func (s *UTXOSet) Fingerprint() Hash512
```

**里程碑**：
- UTXO 集合增删查
- 四层哈希指纹树（年度 -> 区块高度区间 -> 交易 -> 输出）
- 指纹增量更新
- 双花检测
- 持久化存储接口

---

### 第六阶段：区块与区块链

**目标**：实现区块结构与链管理

**模块**：`internal/blockchain/`

**数据结构**：

```go
// 区块头
type BlockHeader struct {
    Version   int       // 版本号
    PrevBlock Hash512   // 前一区块哈希
    CheckRoot Hash512   // 校验根
    Stakes    int64     // 币权销毁
    Height    int       // 区块高度
    Timestamp int64     // 时间戳（固定计算）
    YearBlock Hash512   // 年块引用（年度首块）
}

// 完整区块
type Block struct {
    Header    BlockHeader
    Coinbase  Coinbase
    Txs       []*Transaction
    TxTree    *HashTree     // 交易哈希校验树
}

// 区块链
type Blockchain struct {
    headers   []BlockHeader  // 区块头链
    utxoSet   *UTXOSet       // 当前 UTXO 集
    storage   Storage        // 持久化存储
}
```

**里程碑**：
- 区块头/区块体结构
- 区块 ID 计算
- 哈希校验树（含序）
- 区块验证流程
- 区块链遍历与查询
- 区块限额检查（按月/年递增）

---

### 第七阶段：共识机制（PoH）

**目标**：实现历史证明共识

**模块**：`internal/consensus/`

**核心组件**：

```go
// 铸凭交易
type MintCredential struct {
    Year      int       // 交易年度
    TxID      Hash512   // 交易 ID
    Minter    Address   // 铸造者地址（首领输入接收者）
    Signature []byte    // 签名数据
}

// 铸凭哈希计算
func ComputeMintHash(
    txID Hash512,
    refBlockMintHash Hash512,
    utxoFingerprint Hash512,
    blockTimestamp int64,
    privateKey PrivateKey,
) (Hash512, []byte)

// 择优池
type CandidatePool struct {
    candidates [20]*MintCredential  // 按铸凭哈希排序
}
```

**里程碑**：
- 铸凭交易范围验证（-80000 ~ -25）
- 铸凭哈希计算逻辑
- 择优池管理（20 名）
- 择优池同步协议
- 铸造冗余机制（15 秒间隔）
- 分叉处理（25 区块竞争）

---

### 第八阶段：内存池与交易广播

**目标**：未确认交易管理

**模块**：`internal/mempool/`

**里程碑**：
- 未确认交易存储
- 交易费排序
- 过期交易清理
- 双花交易检测与警告
- 交易广播接口（对接 P2P）

---

### 第九阶段：存储层

**目标**：持久化数据管理

**模块**：`internal/storage/`

**存储分类**：
- 区块头链存储
- 区块体存储（含交易）
- UTXO 集快照
- 交易索引（年度 + TxID）

**里程碑**：
- 存储接口抽象
- Badger/LevelDB 实现
- 区块链初始化与恢复
- UTXO 快照与增量同步

---

### 第十阶段：整合与主程序

**目标**：整合各模块，实现可运行节点

**模块**：`cmd/evidcoin/`

**里程碑**：
- 命令行参数解析
- 节点启动流程
- P2P 网络接入
- 区块同步
- 交易提交接口
- 铸造参与流程
- 日志与监控

---

## 五、关键接口设计

### 脚本验证接口

```go
// 验证交易输入
func ValidateInput(
    input *TxInput,
    prevOutput *Output,
    tx *Transaction,
    inputIndex int,
) error
```

### 区块验证接口

```go
// 验证区块合法性
func ValidateBlock(
    block *Block,
    prevBlock *BlockHeader,
    utxoSet *UTXOSet,
    candidatePool *CandidatePool,
) error
```

### 交易创建接口

```go
// 创建币金转账交易
func CreateCoinTransfer(
    inputs []UTXOEntry,
    outputs []CoinOutput,
    feeRate int64,
    privateKey PrivateKey,
) (*Transaction, error)
```

---

## 六、测试策略

| 层级 | 测试类型 | 覆盖目标 |
|-----|---------|---------|
| 单元 | 各模块内部 | 核心算法、边界条件 |
| 集成 | 模块间交互 | 交易验证全流程、区块生成 |
| E2E | 多节点模拟 | 共识达成、分叉处理 |

**关键测试场景**：
- 脚本指令全覆盖
- 多重签名验证
- 铸凭哈希竞争
- 25 区块分叉竞争
- 交易过期处理
- UTXO 指纹一致性

---

## 七、里程碑时间线（建议）

| 阶段 | 内容 | 预估工作量 |
|-----|------|-----------|
| Phase 1 | 基础类型与加密原语 | 1 周 |
| Phase 2 | 信用结构 | 1 周 |
| Phase 3 | 交易系统 | 2 周 |
| Phase 4 | 脚本虚拟机 | 4-6 周 |
| Phase 5 | UTXO 管理 | 1-2 周 |
| Phase 6 | 区块与区块链 | 2 周 |
| Phase 7 | 共识机制 | 2-3 周 |
| Phase 8 | 内存池 | 1 周 |
| Phase 9 | 存储层 | 1-2 周 |
| Phase 10 | 整合主程序 | 2 周 |

**总计**：约 17-22 周（4-5 个月）

---

## 八、风险与注意事项

1. **脚本系统复杂度**：170+ 指令需要逐一实现和测试，是最耗时模块
2. **哈希校验树实现**：需确保与设计文档一致，影响跨节点验证
3. **时间戳精度**：区块时间戳由高度计算，需处理时区和精度问题
4. **外部依赖协调**：假设外部 P2P 服务可用，需定义清晰接口边界
5. **安全审计**：加密相关代码需额外关注
6. **性能优化**：初期实现以正确性为主，后续需进行性能调优
