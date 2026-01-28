## Assistant (Plan · claude-opus-4.5 · 90.0s)

明白了。我将为 Evidcoin 核心部分编写一份分阶段的实现方案，聚焦于区块链核心、信用结构和脚本系统，假设外部 P2P 依赖可用。

---

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
                    ┌─────────────┐
                    │   cmd/      │
                    └──────┬──────┘
                           │
              ┌────────────┼────────────┐
              ▼            ▼            ▼
        ┌──────────┐ ┌──────────┐ ┌──────────┐
        │blockchain│ │consensus │ │ mempool  │
        └────┬─────┘ └────┬─────┘ └────┬─────┘
             │            │            │
        ┌────┴────────────┴────────────┴────┐
        │                tx                  │
        └───────────────┬────────────────────┘
                        │
           ┌────────────┼────────────┐
           ▼            ▼            ▼
      ┌────────┐   ┌────────┐   ┌────────┐
      │ credit │   │ script │   │  utxo  │
      └────┬───┘   └────┬───┘   └────┬───┘
           │            │            │
           └────────────┼────────────┘
                        ▼
               ┌────────────────┐
               │    crypto      │
               └────────┬───────┘
                        ▼
               ┌────────────────┐
               │  pkg/types     │
               └────────────────┘
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
4. **外部依赖协调**：假设 P2P 可用，需定义清晰接口边界
5. **安全审计**：加密相关代码需额外关注

---

以上是 Evidcoin 核心部分的实现方案。请问您是否有以下方面需要我进一步细化：

1. 某个具体模块的详细设计？
2. 特定数据结构的字段定义？
3. 脚本指令的具体实现策略？
4. 开始实施第一阶段？

---

## User

3

---

## Assistant (Plan · claude-opus-4.5 · 158.1s)

好的，让我详细阐述脚本指令的具体实现策略。

---

# 脚本虚拟机实现策略

## 一、整体架构

### 1.1 核心组件

```
┌─────────────────────────────────────────────────────────┐
│                      Script Engine                       │
├─────────────────────────────────────────────────────────┤
│  ┌──────────┐  ┌──────────┐  ┌──────────┐              │
│  │  Lexer   │──▶│  Parser  │──▶│ Compiler │              │
│  │ (词法器) │  │ (解析器) │  │ (编译器) │              │
│  └──────────┘  └──────────┘  └──────────┘              │
│        │              │              │                  │
│        ▼              ▼              ▼                  │
│  ┌──────────────────────────────────────────┐          │
│  │            Bytecode (指令码序列)          │          │
│  └──────────────────────────────────────────┘          │
│                         │                               │
│                         ▼                               │
│  ┌──────────────────────────────────────────┐          │
│  │              VM (虚拟机)                  │          │
│  │  ┌────────┐ ┌────────┐ ┌────────┐       │          │
│  │  │ Stack  │ │  Args  │ │ Locals │       │          │
│  │  │ 数据栈 │ │ 实参区 │ │ 局部域 │       │          │
│  │  └────────┘ └────────┘ └────────┘       │          │
│  │  ┌────────┐ ┌────────┐ ┌────────┐       │          │
│  │  │Globals │ │ Import │ │ Export │       │          │
│  │  │ 全局域 │ │导入缓存│ │导出缓存│       │          │
│  │  └────────┘ └────────┘ └────────┘       │          │
│  └──────────────────────────────────────────┘          │
└─────────────────────────────────────────────────────────┘
```

### 1.2 执行流程

```
源码文本 ──▶ Token 流 ──▶ AST ──▶ 字节码 ──▶ 执行结果
   │           │          │         │          │
   │           │          │         │          ▼
   │           │          │         │     true/false
   └───────────┴──────────┴─────────┘     (通关状态)
              编译时                  运行时
```

---

## 二、指令编码格式

### 2.1 字节码结构

每条指令的编码格式：

```
┌──────────┬──────────────┬────────────────┐
│  Opcode  │   附参区     │    关联数据    │
│  (1字节) │  (变长)      │    (变长)      │
└──────────┴──────────────┴────────────────┘
```

### 2.2 指令类型分类

根据附参和关联数据的特征，指令可分为以下几类：

| 类型 | 特征 | 示例 |
|-----|------|------|
| 纯指令 | 无附参无数据 | `NIL`, `TRUE`, `POP`, `PASS` |
| 带附参 | 固定/变长附参 | `POPS(1)`, `IF{}(~)` |
| 带数据 | 携带关联数据 | `DATA{}`, `""`, `{}` |
| 复合型 | 附参 + 数据 | `STRING(1){}`, `GOTO(~,64,~)` |

### 2.3 附参编码规则

```go
// 附参长度标记
// (~) 变长整数 varint
// (1) 固定 1 字节
// (2) 固定 2 字节
// (64) 固定 64 字节

// 变长整数编码（类似 protobuf varint）
func EncodeVarint(n int64) []byte
func DecodeVarint(data []byte) (int64, int)
```

---

## 三、虚拟机数据结构

### 3.1 核心状态

```go
// internal/script/vm.go

type VM struct {
    // 执行状态
    code     []byte    // 当前执行的字节码
    pc       int       // 程序计数器
    passed   bool      // 通关状态
    finished bool      // 是否结束

    // 数据空间
    stack    *Stack    // 数据栈（≤256 条目）
    args     *ArgQueue // 实参区（FIFO 队列）
    locals   *Locals   // 局部域栈（每层 ≤128）
    globals  *Globals  // 全局域

    // 交互缓存
    importBuf *Buffer  // 导入缓存区
    exportBuf *Buffer  // 导出缓存区

    // 执行上下文
    env       *Environment  // 环境信息（交易、区块等）
    loopVars  *LoopContext  // 循环变量上下文
    ifState   []bool        // IF 状态栈

    // 资源限制
    gasUsed   int64         // 已消耗 gas
    maxGas    int64         // 最大 gas
    depth     int           // 嵌入/跳转深度
}
```

### 3.2 数据栈实现

```go
// internal/script/stack.go

type Stack struct {
    items []any
    max   int  // 最大 256
}

func (s *Stack) Push(v any) error {
    if len(s.items) >= s.max {
        return ErrStackOverflow
    }
    s.items = append(s.items, v)
    return nil
}

func (s *Stack) Pop() (any, error) {
    if len(s.items) == 0 {
        return nil, ErrStackUnderflow
    }
    v := s.items[len(s.items)-1]
    s.items = s.items[:len(s.items)-1]
    return v, nil
}

func (s *Stack) Peek(offset int) (any, error)      // 引用不弹出
func (s *Stack) PeekN(n int) ([]any, error)        // 引用多项
func (s *Stack) PopN(n int) ([]any, error)         // 弹出多项
func (s *Stack) Top() (any, error)                 // 栈顶项
func (s *Stack) Size() int                         // 栈高度
```

### 3.3 实参区实现

```go
// internal/script/args.go

type ArgQueue struct {
    items []any
}

// 添加条目（持续添加）
func (a *ArgQueue) Add(v any) {
    a.items = append(a.items, v)
}

// 展开添加多项
func (a *ArgQueue) Extend(vs []any) {
    a.items = append(a.items, vs...)
}

// 一次性取出全部并清空
func (a *ArgQueue) Drain() []any {
    items := a.items
    a.items = nil
    return items
}

// 是否有值
func (a *ArgQueue) HasValues() bool {
    return len(a.items) > 0
}
```

### 3.4 局部域实现

```go
// internal/script/locals.go

// 每个语法块的局部域
type LocalScope struct {
    items []any  // 最大 128
}

// 局部域栈（嵌套语法块）
type Locals struct {
    scopes []*LocalScope
}

func (l *Locals) Push() *LocalScope     // 进入新块
func (l *Locals) Pop()                  // 离开块
func (l *Locals) Current() *LocalScope  // 当前块
func (l *Locals) Set(v any) error       // 添加到当前
func (l *Locals) Get(idx int) (any, error) // 按索引取值
```

---

## 四、指令分组实现策略

### 4.1 值指令（Opcode 0-19）

**实现重点**：解析附参和关联数据，构造具体类型值

```go
// 值指令处理器
var valueHandlers = map[Opcode]func(vm *VM) error{
    OP_NIL:    opNil,     // => nil
    OP_TRUE:   opTrue,    // => true
    OP_FALSE:  opFalse,   // => false
    OP_BYTE:   opByte,    // 读1字节 => byte
    OP_RUNE:   opRune,    // 读4字节 => rune
    OP_INT:    opInt,     // 读变长 => int64
    OP_BIGINT: opBigInt,  // 读附参长度 + 数据 => *big.Int
    OP_FLOAT:  opFloat,   // 读8字节 => float64
    OP_STRING_SHORT: opStringShort, // 附参1字节长度
    OP_STRING_LONG:  opStringLong,  // 附参2字节长度
    OP_VALUESET:     opValueSet,    // 值集 {1, 2, 3}
    OP_DATA_SHORT:   opDataShort,   // 小字节序列
    OP_DATA_LONG:    opDataLong,    // 大字节序列
    OP_REGEXP:       opRegexp,      // 正则表达式
    OP_DATE:         opDate,        // 时间对象
    OP_CODE:         opCode,        // 代码/脚本
    OP_SCRIPT:       opScript,      // 脚本引用
    OP_VALUE:        opValue,       // 通用取值
}

// 示例：整数值处理
func opInt(vm *VM) error {
    val, n := DecodeVarint(vm.code[vm.pc:])
    vm.pc += n
    return vm.stack.Push(val)
}

// 示例：DATA{} 处理
func opDataShort(vm *VM) error {
    length := int(vm.code[vm.pc])
    vm.pc++
    data := make([]byte, length)
    copy(data, vm.code[vm.pc:vm.pc+length])
    vm.pc += length
    return vm.stack.Push(data)
}
```

**类型提升规则**（值集内）：

```go
// 类型提升优先级：Byte < Rune < Int < BigInt
//                 Byte < Rune < Int < Float
func promoteTypes(items []any) ([]any, error) {
    maxType := detectMaxType(items)
    result := make([]any, len(items))
    for i, item := range items {
        result[i] = convertTo(item, maxType)
    }
    return result, nil
}
```

---

### 4.2 截取指令（Opcode 20-24）

**实现重点**：修改下一指令的返回值目标

```go
// 截取模式
type InterceptMode int
const (
    InterceptNone   InterceptMode = iota
    InterceptToArgs               // @ => 实参区
    InterceptDirect               // ~ => 跳过实参区
    InterceptToLocal              // $ => 局部域
)

// 截取指令不直接执行，而是设置标记
func opIntercept(vm *VM) error {
    vm.interceptMode = InterceptToArgs
    return nil
}

func opLocalStore(vm *VM) error {
    vm.interceptMode = InterceptToLocal
    return nil
}

func opLocalGet(vm *VM) error {
    idx := int(int8(vm.code[vm.pc])) // 有符号，支持负数
    vm.pc++
    val, err := vm.locals.Get(idx)
    if err != nil {
        return err
    }
    vm.args.Add(val)  // 放入实参区
    return nil
}

func opLoopVar(vm *VM) error {
    varType := vm.code[vm.pc]
    vm.pc++
    var val any
    switch varType {
    case 0: val = vm.loopVars.Value
    case 1: val = vm.loopVars.Index
    case 2: val = vm.loopVars.Collection
    case 3: val = vm.loopVars.Size
    }
    vm.args.Add(val)
    return nil
}
```

**返回值分发逻辑**：

```go
// 指令执行后的返回值处理
func (vm *VM) dispatchResult(results []any) error {
    switch vm.interceptMode {
    case InterceptToArgs:
        vm.args.Extend(results)
    case InterceptToLocal:
        for _, r := range results {
            vm.locals.Current().Add(r)
        }
    default:
        // 默认入栈
        for _, r := range results {
            if err := vm.stack.Push(r); err != nil {
                return err
            }
        }
    }
    vm.interceptMode = InterceptNone
    return nil
}
```

---

### 4.3 栈操作指令（Opcode 25-35）

**实现重点**：精确处理附参定义的数量，区分引用和弹出

```go
func opPop(vm *VM) error {
    val, err := vm.stack.Pop()
    if err != nil {
        return err
    }
    return vm.dispatchResult([]any{val})
}

func opPops(vm *VM) error {
    n := int(vm.code[vm.pc])
    vm.pc++
    if n == 0 {
        n = vm.stack.Size() // 0 表示全部
    }
    vals, err := vm.stack.PopN(n)
    if err != nil {
        return err
    }
    return vm.dispatchResult(vals) // 自动展开
}

func opTop(vm *VM) error {
    val, err := vm.stack.Top()
    if err != nil {
        return err
    }
    return vm.dispatchResult([]any{val})
}

func opShift(vm *VM) error {
    n := int(vm.code[vm.pc])
    vm.pc++
    if n == 0 {
        n = vm.stack.Size()
    }
    vals, err := vm.stack.PopN(n)
    if err != nil {
        return err
    }
    // SHIFT 返回切片而非展开
    return vm.dispatchResult([]any{vals})
}

func opClone(vm *VM) error {
    n := int(vm.code[vm.pc])
    vm.pc++
    if n == 0 {
        n = vm.stack.Size()
    }
    vals, err := vm.stack.PeekN(n) // 引用不弹出
    if err != nil {
        return err
    }
    cloned := shallowCopy(vals)
    return vm.dispatchResult([]any{cloned})
}
```

---

### 4.4 集合指令（Opcode 36-45）

**实现重点**：切片/字典的基本操作

```go
func opSlice(vm *VM) error {
    size := int(vm.code[vm.pc])
    vm.pc++
    
    args := vm.getArgs(2) // 取2个实参
    slice := args[0].([]any)
    start := toInt(args[1])
    
    // 支持负数索引
    if start < 0 {
        start = len(slice) + start
    }
    
    end := start + size
    if size == 0 {
        end = len(slice)
    }
    
    if start < 0 || end > len(slice) {
        return ErrIndexOutOfRange
    }
    
    result := slice[start:end]
    return vm.dispatchResult([]any{result})
}

func opSpread(vm *VM) error {
    args := vm.getArgs(1)
    slice := args[0].([]any)
    return vm.dispatchResult(slice) // 展开为多项
}

func opMerge(vm *VM) error {
    args := vm.getArgs(-1) // 不定数量
    var result []any
    for _, arg := range args {
        if s, ok := arg.([]any); ok {
            result = append(result, s...)
        }
    }
    return vm.dispatchResult([]any{result})
}
```

---

### 4.5 结果指令（Opcode 50-56）

**实现重点**：控制脚本执行流和通关状态

```go
func opPass(vm *VM) error {
    args := vm.getArgs(1)
    val := toBool(args[0])
    if val {
        vm.passed = true
        return nil
    }
    // 失败立即结束
    vm.passed = false
    vm.finished = true
    return nil
}

func opCheck(vm *VM) error {
    args := vm.getArgs(1)
    val := toBool(args[0])
    vm.passed = val // 覆盖状态，但不退出
    return nil
}

func opExit(vm *VM) error {
    vm.finished = true
    // 可选返回值
    if vm.args.HasValues() {
        args := vm.args.Drain()
        return vm.dispatchResult(args)
    }
    return nil
}

func opGoto(vm *VM) error {
    if vm.jumpDepth >= MaxJumpDepth {
        return ErrMaxJumpDepthExceeded
    }
    
    // 读取附参
    year, n := DecodeVarint(vm.code[vm.pc:])
    vm.pc += n
    txID := vm.code[vm.pc : vm.pc+64]
    vm.pc += 64
    outIndex, n := DecodeVarint(vm.code[vm.pc:])
    vm.pc += n
    
    // 获取目标脚本
    script, err := vm.env.FetchScript(int(year), txID, int(outIndex))
    if err != nil {
        return err
    }
    
    // 传递实参作为新脚本的初始栈
    initStack := vm.args.Drain()
    
    // 创建新 VM 执行
    newVM := NewVM(script, vm.env)
    newVM.jumpDepth = vm.jumpDepth + 1
    for _, v := range initStack {
        newVM.stack.Push(v)
    }
    
    err = newVM.Execute()
    vm.passed = newVM.passed
    vm.finished = true
    return err
}

func opEmbed(vm *VM) error {
    if vm.embedCount >= MaxEmbedCount {
        return ErrMaxEmbedCountExceeded
    }
    
    // 读取附参（同 GOTO）
    year, n := DecodeVarint(vm.code[vm.pc:])
    vm.pc += n
    txID := vm.code[vm.pc : vm.pc+64]
    vm.pc += 64
    outIndex, n := DecodeVarint(vm.code[vm.pc:])
    vm.pc += n
    
    script, err := vm.env.FetchScript(int(year), txID, int(outIndex))
    if err != nil {
        return err
    }
    
    // 嵌入执行：共享当前 VM 状态
    savedPC := vm.pc
    savedCode := vm.code
    
    vm.code = script
    vm.pc = 0
    vm.embedCount++
    
    err = vm.run() // 执行嵌入脚本
    
    vm.code = savedCode
    vm.pc = savedPC
    return err
}
```

---

### 4.6 流程控制（Opcode 57-66）

**实现重点**：子语句块的跳过与执行

```go
func opIf(vm *VM) error {
    // 读取块长度
    blockLen, n := DecodeVarint(vm.code[vm.pc:])
    vm.pc += n
    
    // 取条件
    args := vm.getArgs(1)
    cond := toBool(args[0])
    
    // 记录 IF 状态（供 ELSE 使用）
    vm.pushIfState(cond)
    
    if cond {
        // 执行块内代码
        vm.locals.Push() // 新建局部域
        err := vm.executeBlock(int(blockLen))
        vm.locals.Pop()
        return err
    }
    // 跳过块
    vm.pc += int(blockLen)
    return nil
}

func opElse(vm *VM) error {
    blockLen, n := DecodeVarint(vm.code[vm.pc:])
    vm.pc += n
    
    ifState := vm.popIfState()
    
    if !ifState { // IF 为假时执行 ELSE
        vm.locals.Push()
        err := vm.executeBlock(int(blockLen))
        vm.locals.Pop()
        return err
    }
    vm.pc += int(blockLen)
    return nil
}

func opEach(vm *VM) error {
    blockLen, n := DecodeVarint(vm.code[vm.pc:])
    vm.pc += n
    blockStart := vm.pc
    
    args := vm.getArgs(1)
    collection := args[0]
    
    switch c := collection.(type) {
    case []any:
        for i, v := range c {
            vm.loopVars = &LoopContext{
                Value:      v,
                Index:      i,
                Collection: c,
                Size:       len(c),
            }
            vm.pc = blockStart
            vm.locals.Push()
            
            err := vm.executeBlock(int(blockLen))
            vm.locals.Pop()
            
            if vm.breakFlag {
                vm.breakFlag = false
                break
            }
            if vm.continueFlag {
                vm.continueFlag = false
                continue
            }
            if err != nil {
                return err
            }
        }
    case map[string]any:
        keys := getSortedKeys(c)
        for i, k := range keys {
            vm.loopVars = &LoopContext{
                Value:      c[k],
                Index:      k, // 字典用键名
                Collection: c,
                Size:       len(c),
            }
            // ... 同上
        }
    }
    
    vm.pc = blockStart + int(blockLen)
    return nil
}

func opBreak(vm *VM) error {
    // 可选条件
    if vm.args.HasValues() {
        args := vm.args.Drain()
        if !toBool(args[0]) {
            return nil
        }
    }
    vm.breakFlag = true
    return nil
}

func opSwitch(vm *VM) error {
    blockLen, n := DecodeVarint(vm.code[vm.pc:])
    vm.pc += n
    
    args := vm.getArgs(2)
    target := args[0]       // 标的值
    caseValues := args[1].([]any) // 分支值列表
    
    vm.switchContext = &SwitchContext{
        Target:     target,
        CaseValues: caseValues,
        CaseIndex:  0,
        Matched:    false,
    }
    
    vm.locals.Push()
    err := vm.executeBlock(int(blockLen))
    vm.locals.Pop()
    vm.switchContext = nil
    return err
}

func opCase(vm *VM) error {
    blockLen, n := DecodeVarint(vm.code[vm.pc:])
    vm.pc += n
    
    ctx := vm.switchContext
    if ctx.Matched {
        // 已有 CASE 匹配，跳过
        vm.pc += int(blockLen)
        return nil
    }
    
    // 取对应位置的分支值比较
    caseVal := ctx.CaseValues[ctx.CaseIndex]
    ctx.CaseIndex++
    
    if equal(ctx.Target, caseVal) {
        ctx.Matched = true
        err := vm.executeBlock(int(blockLen))
        return err
    }
    vm.pc += int(blockLen)
    return nil
}
```

---

### 4.7 转换指令（Opcode 67-79）

**实现重点**：类型安全转换，边界检查

```go
func opBool(vm *VM) error {
    args := vm.getArgs(1)
    result := toBoolValue(args[0])
    return vm.dispatchResult([]any{result})
}

func toBoolValue(v any) bool {
    switch x := v.(type) {
    case nil:
        return false
    case bool:
        return x
    case byte:
        return x != 0
    case rune:
        return x != 0
    case int64:
        return x != 0
    case *big.Int:
        return x.Sign() != 0
    case float64:
        return x > math.SmallestNonzeroFloat64
    case string:
        return x != ""
    default:
        return true
    }
}

func opInt(vm *VM) error {
    args := vm.getArgs(1)
    result, err := toIntValue(args[0])
    if err != nil {
        return err
    }
    return vm.dispatchResult([]any{result})
}

func toIntValue(v any) (int64, error) {
    switch x := v.(type) {
    case nil:
        return 0, nil
    case bool:
        if x { return 1, nil }
        return 0, nil
    case byte:
        return int64(x), nil
    case rune:
        return int64(x), nil
    case int64:
        return x, nil
    case *big.Int:
        if !x.IsInt64() {
            return 0, ErrIntOverflow
        }
        return x.Int64(), nil
    case float64:
        return int64(x), nil
    case string:
        return strconv.ParseInt(x, 0, 64)
    default:
        return 0, ErrInvalidConversion
    }
}

func opString(vm *VM) error {
    // 附参：进制或格式
    format := vm.code[vm.pc]
    vm.pc++
    
    args := vm.getArgs(1)
    result := toStringValue(args[0], format)
    return vm.dispatchResult([]any{result})
}

func opDict(vm *VM) error {
    args := vm.getArgs(2)
    keys := args[0].([]string)
    vals := args[1].([]any)
    
    dict := make(map[string]any)
    for i, k := range keys {
        if i < len(vals) {
            dict[k] = vals[i]
        } else {
            dict[k] = nil
        }
    }
    return vm.dispatchResult([]any{dict})
}
```

---

### 4.8 运算指令（Opcode 80-103）

**实现重点**：表达式解析与求值

```go
// 表达式求值器
func opExpression(vm *VM) error {
    exprLen := int(vm.code[vm.pc])
    vm.pc++
    
    exprEnd := vm.pc + exprLen
    result, err := vm.evaluateExpression(exprEnd)
    if err != nil {
        return err
    }
    return vm.dispatchResult([]any{result})
}

func (vm *VM) evaluateExpression(end int) (float64, error) {
    // 使用调度场算法 (Shunting Yard) 解析中缀表达式
    var output []float64
    var operators []Opcode
    
    for vm.pc < end {
        op := Opcode(vm.code[vm.pc])
        vm.pc++
        
        switch op {
        case OP_INT, OP_FLOAT, OP_BYTE:
            // 操作数
            val := vm.readValue(op)
            output = append(output, toFloat(val))
            
        case OP_POP, OP_TOP, OP_PEEK:
            // 从栈取值
            result, _ := vm.executeOp(op)
            output = append(output, toFloat(result[0]))
            
        case OP_LPAREN: // (
            operators = append(operators, op)
            
        case OP_RPAREN: // )
            for len(operators) > 0 && operators[len(operators)-1] != OP_LPAREN {
                output = applyOp(output, operators[len(operators)-1])
                operators = operators[:len(operators)-1]
            }
            operators = operators[:len(operators)-1] // 移除 (
            
        case OP_MUL, OP_DIV, OP_MOD, OP_ADD, OP_SUB:
            for len(operators) > 0 && precedence(operators[len(operators)-1]) >= precedence(op) {
                output = applyOp(output, operators[len(operators)-1])
                operators = operators[:len(operators)-1]
            }
            operators = append(operators, op)
        }
    }
    
    for len(operators) > 0 {
        output = applyOp(output, operators[len(operators)-1])
        operators = operators[:len(operators)-1]
    }
    
    return output[0], nil
}

// 运算优先级
func precedence(op Opcode) int {
    switch op {
    case OP_MUL, OP_DIV, OP_MOD:
        return 2
    case OP_ADD, OP_SUB:
        return 1
    default:
        return 0
    }
}

// 命名运算指令
func opAdd(vm *VM) error {
    args := vm.getArgs(2)
    
    // 类型检测，支持多种类型
    switch a := args[0].(type) {
    case int64:
        b := args[1].(int64)
        return vm.dispatchResult([]any{a + b})
    case float64:
        b := toFloat(args[1])
        return vm.dispatchResult([]any{a + b})
    case string:
        b := args[1].(string)
        return vm.dispatchResult([]any{a + b})
    case []byte:
        b := args[1].([]byte)
        result := make([]byte, len(a)+len(b))
        copy(result, a)
        copy(result[len(a):], b)
        return vm.dispatchResult([]any{result})
    }
    return ErrInvalidOperands
}

func opLmov(vm *VM) error {
    args := vm.getArgs(2)
    x := args[0].(int64)
    y := args[1].(int64)
    return vm.dispatchResult([]any{x << y})
}
```

---

### 4.9 比较与逻辑指令（Opcode 104-119）

```go
func opEqual(vm *VM) error {
    args := vm.getArgs(2)
    result := deepEqual(args[0], args[1])
    return vm.dispatchResult([]any{result})
}

func opLt(vm *VM) error {
    args := vm.getArgs(2)
    result := compare(args[0], args[1]) < 0
    return vm.dispatchResult([]any{result})
}

func compare(a, b any) int {
    switch x := a.(type) {
    case int64:
        y := b.(int64)
        if x < y { return -1 }
        if x > y { return 1 }
        return 0
    case float64:
        y := toFloat(b)
        if x < y { return -1 }
        if x > y { return 1 }
        return 0
    case []byte:
        y := b.([]byte)
        return bytes.Compare(x, y)
    case string:
        y := b.(string)
        return strings.Compare(x, y)
    }
    panic("incomparable types")
}

func opWithin(vm *VM) error {
    args := vm.getArgs(2)
    val := args[0]
    rangeSlice := args[1].([]any)
    min, max := rangeSlice[0], rangeSlice[1]
    
    result := compare(val, min) >= 0 && compare(val, max) < 0
    return vm.dispatchResult([]any{result})
}

func opBoth(vm *VM) error {
    args := vm.getArgs(-1) // 不定数量
    for _, arg := range args {
        if !toBool(arg) {
            return vm.dispatchResult([]any{false})
        }
    }
    return vm.dispatchResult([]any{true})
}

func opEither(vm *VM) error {
    args := vm.getArgs(-1)
    for _, arg := range args {
        if toBool(arg) {
            return vm.dispatchResult([]any{true})
        }
    }
    return vm.dispatchResult([]any{false})
}
```

---

### 4.10 系统指令（SYS_*）

**核心：内置验证 SYS_CHKPASS**

```go
func opSysChkpass(vm *VM) error {
    // 从栈取参数
    args := vm.getArgs(3) // 类型、标记、签名、公钥
    
    sigType := toInt(args[0])
    
    switch sigType {
    case 1: // 单签名
        return vm.verifySingleSig()
    case 2: // 多重签名
        return vm.verifyMultiSig()
    }
    return ErrInvalidSignatureType
}

func (vm *VM) verifySingleSig() error {
    // 从环境取实参
    flag := vm.env.Get("flag").(byte)
    sig := vm.env.Get("sig").([]byte)
    pubKey := vm.env.Get("pubKey").([]byte)
    receiver := vm.env.Get("receiver").(Address)
    
    // 1. 验证公钥哈希 == 接收者
    pubHash := Hash512(pubKey)[:AddressLength]
    if !bytes.Equal(pubHash, receiver[:]) {
        vm.passed = false
        vm.finished = true
        return nil
    }
    
    // 2. 根据 flag 构造签名消息
    message := vm.buildSignMessage(flag)
    
    // 3. 验证签名
    valid := crypto.Verify(pubKey, message, sig)
    vm.passed = valid
    if !valid {
        vm.finished = true
    }
    return nil
}

func (vm *VM) verifyMultiSig() error {
    // 解析 m/n 配置
    sigs := vm.env.Get("sigs").([][]byte)
    pubKeys := vm.env.Get("pubKeys").([][]byte)
    supplements := vm.env.Get("supplements").([][]byte) // 补全集
    receiver := vm.env.Get("receiver").(Address)
    
    m := len(pubKeys)
    n := m + len(supplements)
    
    // 1. 计算复合公钥哈希
    var pubHashes [][]byte
    for _, pk := range pubKeys {
        pubHashes = append(pubHashes, Hash512(pk)[:AddressLength])
    }
    pubHashes = append(pubHashes, supplements...)
    sort.Slice(pubHashes, func(i, j int) bool {
        return bytes.Compare(pubHashes[i], pubHashes[j]) < 0
    })
    
    prefix := []byte{byte(m), byte(n)}
    combined := append(prefix, bytes.Join(pubHashes, nil)...)
    compositeHash := Hash512(combined)[:AddressLength]
    
    // 2. 验证复合哈希 == 接收者
    if !bytes.Equal(compositeHash, receiver[:]) {
        vm.passed = false
        vm.finished = true
        return nil
    }
    
    // 3. 验证各个签名
    message := vm.buildSignMessage(vm.env.Get("flag").(byte))
    for i, sig := range sigs {
        if !crypto.Verify(pubKeys[i], message, sig) {
            vm.passed = false
            vm.finished = true
            return nil
        }
    }
    
    vm.passed = true
    return nil
}

func opSysTime(vm *VM) error {
    timeType := vm.code[vm.pc]
    vm.pc++
    
    var result any
    switch timeType {
    case 0: // Stamp - 当前区块时间戳
        result = vm.env.BlockTimestamp()
    case 1: // Height - 当前区块高度
        result = vm.env.BlockHeight()
    // ... 其他时间相关
    }
    return vm.dispatchResult([]any{result})
}
```

---

### 4.11 函数指令（FN_*）

```go
func opFnHash256(vm *VM) error {
    args := vm.getArgs(1)
    data := toBytes(args[0])
    hash := crypto.BLAKE3(data)
    return vm.dispatchResult([]any{hash})
}

func opFnHash512(vm *VM) error {
    args := vm.getArgs(1)
    data := toBytes(args[0])
    hash := crypto.SHA512(data)
    return vm.dispatchResult([]any{hash})
}

func opFnPubhash(vm *VM) error {
    args := vm.getArgs(1)
    pubKey := args[0].([]byte)
    hash := crypto.SHA512(pubKey)[:AddressLength]
    return vm.dispatchResult([]any{hash})
}

func opFnChecksig(vm *VM) error {
    args := vm.getArgs(2)
    sig := args[0].([]byte)
    pubKey := args[1].([]byte)
    
    // 默认对整个交易签名
    message := vm.env.TxSignatureMessage()
    valid := crypto.Verify(pubKey, message, sig)
    return vm.dispatchResult([]any{valid})
}

func opFnBase32(vm *VM) error {
    mode := vm.code[vm.pc]
    vm.pc++
    
    args := vm.getArgs(1)
    
    switch mode {
    case 0: // 编码
        data := args[0].([]byte)
        result := base32.Encode(data)
        return vm.dispatchResult([]any{result})
    case 1: // 解码
        str := args[0].(string)
        result, err := base32.Decode(str)
        if err != nil {
            return err
        }
        return vm.dispatchResult([]any{result})
    }
    return ErrInvalidMode
}
```

---

## 五、实参获取策略

**核心逻辑**：先从实参区取，不足则从栈取

```go
// 获取指定数量的实参
func (vm *VM) getArgs(n int) []any {
    if n == -1 {
        // 不定数量：仅取实参区
        return vm.args.Drain()
    }
    
    if n == 0 {
        return nil
    }
    
    // 检查实参区
    if vm.directMode {
        // ~ 模式：跳过实参区，直接从栈取
        vm.directMode = false
        vals, _ := vm.stack.PopN(n)
        return vals
    }
    
    argsFromQueue := vm.args.Drain()
    
    if len(argsFromQueue) == n {
        return argsFromQueue
    }
    
    if len(argsFromQueue) > 0 && len(argsFromQueue) != n {
        // 实参区有值但数量不匹配，报错
        panic("args count mismatch")
    }
    
    // 实参区为空，从栈取
    vals, _ := vm.stack.PopN(n)
    return vals
}
```

---

## 六、编译器实现要点

### 6.1 词法分析

```go
// internal/script/lexer.go

type TokenType int
const (
    TOKEN_EOF TokenType = iota
    TOKEN_IDENT         // 指令名
    TOKEN_INT           // 整数
    TOKEN_FLOAT         // 浮点数
    TOKEN_STRING        // 字符串
    TOKEN_BYTES         // 字节序列
    TOKEN_LBRACE        // {
    TOKEN_RBRACE        // }
    TOKEN_LBRACKET      // [
    TOKEN_RBRACKET      // ]
    TOKEN_LPAREN        // (
    TOKEN_RPAREN        // )
    TOKEN_AT            // @
    TOKEN_DOLLAR        // $
    TOKEN_TILDE         // ~
    // ...
)

type Token struct {
    Type    TokenType
    Value   string
    Line    int
    Column  int
}

type Lexer struct {
    input   string
    pos     int
    line    int
    column  int
}

func (l *Lexer) NextToken() Token
```

### 6.2 语法分析与编译

```go
// internal/script/compiler.go

type Compiler struct {
    lexer   *Lexer
    output  []byte
}

func (c *Compiler) Compile() ([]byte, error) {
    for {
        tok := c.lexer.NextToken()
        if tok.Type == TOKEN_EOF {
            break
        }
        
        if err := c.compileToken(tok); err != nil {
            return nil, err
        }
    }
    return c.output, nil
}

func (c *Compiler) compileToken(tok Token) error {
    switch tok.Type {
    case TOKEN_IDENT:
        return c.compileInstruction(tok.Value)
    case TOKEN_INT:
        return c.compileInt(tok.Value)
    case TOKEN_STRING:
        return c.compileString(tok.Value)
    // ...
    }
    return nil
}

func (c *Compiler) compileInstruction(name string) error {
    opcode, ok := instructionMap[name]
    if !ok {
        return fmt.Errorf("unknown instruction: %s", name)
    }
    
    c.output = append(c.output, byte(opcode))
    
    // 处理附参和关联数据
    switch opcode {
    case OP_IF, OP_ELSE, OP_EACH:
        return c.compileBlock()
    case OP_DATA_SHORT, OP_DATA_LONG:
        return c.compileData()
    // ...
    }
    return nil
}
```

---

## 七、安全边界检查

```go
// 资源限制常量
const (
    MaxStackHeight  = 256
    MaxStackItem    = 1024
    MaxLocalScope   = 128
    MaxLockScript   = 1024
    MaxUnlockScript = 4096
    MaxEmbedCount   = 5
    MaxJumpDepth    = 3
    MaxJumpCount    = 2
)

// 栈项大小检查
func (vm *VM) checkItemSize(v any) error {
    size := estimateSize(v)
    if size > MaxStackItem {
        return ErrStackItemTooLarge
    }
    return nil
}

// 脚本长度检查
func ValidateScriptLength(script []byte, isUnlock bool) error {
    maxLen := MaxLockScript
    if isUnlock {
        maxLen = MaxUnlockScript
    }
    if len(script) > maxLen {
        return ErrScriptTooLong
    }
    return nil
}

// 解锁脚本指令限制
func ValidateUnlockOpcodes(script []byte) error {
    for i := 0; i < len(script); {
        op := Opcode(script[i])
        if op > OP_OUTPUT {  // 超出值指令~交互指令范围
            return ErrInvalidUnlockOpcode
        }
        i += instructionSize(script, i)
    }
    return nil
}
```

---

## 八、测试策略

### 8.1 指令单元测试

```go
func TestOpPop(t *testing.T) {
    vm := NewVM(nil, nil)
    vm.stack.Push(int64(100))
    vm.stack.Push(int64(200))
    
    err := opPop(vm)
    require.NoError(t, err)
    require.Equal(t, 1, vm.stack.Size())
    
    // 验证返回值在默认情况下入栈
    top, _ := vm.stack.Top()
    require.Equal(t, int64(200), top)
}

func TestOpIf(t *testing.T) {
    // 编译脚本：TRUE IF { {100} } ELSE { {200} }
    code := compile("TRUE IF { {100} } ELSE { {200} }")
    
    vm := NewVM(code, nil)
    err := vm.Execute()
    
    require.NoError(t, err)
    top, _ := vm.stack.Top()
    require.Equal(t, int64(100), top)
}
```

### 8.2 集成测试

```go
func TestStandardPayment(t *testing.T) {
    // 模拟标准支付验证
    unlockScript := compile(`
        {1}
        DATA{...签名...}
        DATA{...公钥...}
    `)
    
    lockScript := compile(`SYS_CHKPASS`)
    
    env := &MockEnv{
        Receiver: testAddress,
        // ...
    }
    
    vm := NewVM(append(unlockScript, lockScript...), env)
    err := vm.Execute()
    
    require.NoError(t, err)
    require.True(t, vm.Passed())
}
```

---

这就是脚本虚拟机的详细实现策略。如有需要，我可以进一步细化：

1. 某个具体指令的完整实现？
2. 编译器的详细设计？
3. 错误处理机制？
4. 性能优化考虑？

---

