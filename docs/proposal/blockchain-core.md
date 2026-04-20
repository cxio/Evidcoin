# Blockchain Core（区块链核心）

## 1. Design Philosophy（设计哲学）

Blockchain Core 采用**极致简化**的设计原则，仅专注于区块头链的管理与维护。

核心不包含共识逻辑、交易校验、脚本执行等功能——这些由外部组件负责。Core 仅作为区块链数据的可信管理者，接受经外部校验后提交的区块，维护区块头链的连续性。

这种设计使 Core 可以被自由集成到各种场景中：
- **轻客户端**：普通用户的钱包应用
- **围观者**：参与铸造竞争的轻节点
- **校验组成员**：正式参与交易校验的节点（外挂共识模块）
- **公共服务器**：Blockqs、Depots 等第三方服务节点
- **浏览器插件 / 移动应用**：嵌入式集成


## 2. Block Header Structure（区块头结构）

### 2.1 Header Fields（字段定义）

| Field | Type | Size | Description |
|-------|------|------|-------------|
| `Version` | int32 | 4 bytes | 协议版本号 |
| `Height` | int32 | 4 bytes | 区块高度（从 0 开始） |
| `PrevBlock` | [48]byte | 48 bytes | 前一区块的 SHA3-384 哈希 |
| `CheckRoot` | [48]byte | 48 bytes | 校验根：由交易哈希树根 + UTXO/UTCO 双指纹合并计算的哈希 |
| `Stakes` | uint64 | 8 bytes | 币权销毁量（聪时），反映交易活跃度 |

**常规区块头大小：112 bytes**

### 2.2 Year-Block Field（年块字段）

| Field | Type | Size | Condition |
|------|------|------|-----------|
| `YearBlock` | [48]byte | 48 bytes | 仅当 `Height % 87661 == 0` 时存在 |

当区块高度是 87661 的整数倍时（即年度边界），区块头额外包含 `YearBlock` 字段，引用前一个年块的哈希值。

**年块机制的意义：**
- 提供年度粒度的链锚点，使节点无需存储完整区块头即可验证链的连续性
- 大幅降低存储需求：仅需存储年块哈希即可跨越整年数据
- 年数据量约 112 × 87661 ≈ 9.36 MB（不含年块字段本身）

### 2.3 Block ID（区块 ID）

区块 ID 由区块头的 SHA3-384 哈希计算得出。连续的区块 ID 通过 `PrevBlock` 字段相互链接，形成区块头链。

### 2.4 Timestamps（时间戳）

区块头**不包含时间戳字段**。区块的时间通过以下公式确定性计算：

```
BlockTime = GenesisTimestamp + Height × 6min
```

创世区块的时间戳作为硬编码常量存在于系统中。

### 2.5 Stakes（币权销毁）

- 币权 = 未花费输出金额（聪） × 持有时间（小时，不足 1 小时记为零）
- 币权总值 = 区块内全部交易的币权之和，单位：聪时（聪*小时）

一旦某输出被花费，币权即归零。新的输出重新开始累计币权。不足 1 小时的持有时间计为零，可滤除高频交易的影响。

区块头中的 `Stakes` 字段记录该区块所有交易消耗的币权总量，反映区块的交易活跃度。在新区块的候选竞争中，`Stakes` 可作为辅助判断因子（详见 Consensus-PoH 提案 §4.5）。此外，`Stakes` 也是铸凭哈希的计算因子之一。

> **单位说明：**
> - 1 币 = 100,000,000 聪。
> - 币权单位为「聪时」（聪 × 小时）。
> - 1 币持有 1 天 = 100,000,000 × 24 = 2,400,000,000 聪时。

### 2.6 CheckRoot（校验根）

CheckRoot 是交易哈希树根与 UTXO/UTCO 双指纹的合并哈希，由铸造者签名确认。具体计算方式参见 Transaction 提案和 Consensus-PoH 提案中的相关章节。


## 3. Block Admission（入块验证）

### 3.1 Validation Scope（验证范围）

Core 仅执行**结构性验证**，不负责交易内容或脚本的合法性校验：

1. **PrevBlock 一致性**：新区块的 `PrevBlock` 必须等于当前链顶区块的哈希
2. **Height 连续性**：新区块的 `Height` 必须等于当前链顶高度 + 1
3. **Version 兼容性**：版本号在可接受范围内
4. **YearBlock 正确性**：如果处于年度边界，验证 `YearBlock` 引用正确
5. **基本字段合法性**：`Stakes` 非负，`CheckRoot` 非零等

### 3.2 Conflict Handling（冲突处理）

当同一高度的区块被二次提交时：
- Core 返回冲突错误，拒绝入块
- 外部组件必须先删除已有区块，再重新提交替代区块
- 这确保了链的唯一性，分叉选择的决策权交由外部共识模块

### 3.3 Submission Interface（提交接口）

```go
// SubmitBlock 提交一个新区块到链上。
// 区块的内容合法性由外部调用者保证。
// 返回 ErrConflict 如果该高度已有区块。
func (bc *Blockchain) SubmitBlock(header *BlockHeader) error

// ReplaceBlock 替换指定高度的区块（用于分叉切换）。
// 要求目标高度必须是当前链顶。
func (bc *Blockchain) ReplaceBlock(height int, header *BlockHeader) error
```


## 4. Data Storage（数据存储）

### 4.1 Storage Strategy（存储策略）

Core 采用**年块衔接机制**下的灵活存储策略：

- **完整存储**：存储从创世块至今的所有区块头（约 9.36 MB/年）
- **年块骨架存储**：仅存储年度边界区块头，中间区块头按需从 Blockqs 获取
- **混合存储**：近期（如最近 1-2 年）完整存储，更早的年份仅保留年块骨架
- **自由存储**：在保持链的连续性前提下，自由选择忽略某些年度的区块头

用户可通过配置选择存储策略，在存储空间与查询性能之间取得平衡。

### 4.2 Chain Continuity（链连续性保证）

无论采用何种存储策略，链的连续性通过以下机制保证：

1. 完整存储的区间内，每个区块的 `PrevBlock` 指向前一区块
2. 省略的区间内，年块的 `YearBlock` 字段链接前一年块
3. 创世块作为锚点，整条链可验证至源头

### 4.3 Storage Backend（存储后端）

Core 对存储后端不做强制要求，提供存储接口抽象：

```go
// HeaderStore 区块头存储接口。
type HeaderStore interface {
    // Get 按高度获取区块头。
    Get(height int) (*BlockHeader, error)

    // GetByHash 按区块哈希获取区块头。
    GetByHash(hash Hash384) (*BlockHeader, error)

    // Put 存储一个区块头。
    Put(header *BlockHeader) error

    // Has 检查指定高度的区块头是否存在。
    Has(height int) bool

    // Tip 返回当前链顶区块头。
    Tip() (*BlockHeader, error)

    // YearBlock 获取指定年份的年块区块头。
    YearBlock(year int) (*BlockHeader, error)
}
```


## 5. Data Completeness（数据完整性）

### 5.1 On-Demand Fetching（按需获取）

当查询的区块头不在本地存储中时，Core 自动从第三方公共服务（Blockqs）获取：

1. 接收查询请求（按高度或哈希）
2. 检查本地存储
3. 若本地缺失，向 Blockqs 节点发起请求
4. 验证获取的区块头与年块骨架的一致性
5. 可选择性缓存获取的数据

### 5.2 Multi-Source Verification（多源验证）

为确保从 Blockqs 获取的数据可靠，Core 可向多个 Blockqs 节点请求同一数据，比较结果一致性。不一致时标记异常并报告。

### 5.3 Blockqs Connector（Blockqs 连接器）

```go
// BlockqsClient 区块查询服务客户端接口。
type BlockqsClient interface {
    // FetchHeader 从远程服务获取指定高度的区块头。
    FetchHeader(height int) (*BlockHeader, error)

    // FetchHeaders 批量获取区块头（用于同步）。
    FetchHeaders(from, to int) ([]*BlockHeader, error)

    // FetchHeaderByHash 按哈希获取区块头。
    FetchHeaderByHash(hash Hash384) (*BlockHeader, error)
}
```


## 6. Data Access API（数据访问接口）

### 6.1 Query Interface（查询接口）

```go
// Blockchain 区块链核心，提供区块头链的管理与查询。
type Blockchain struct { /* ... */ }

// HeaderByHeight 按高度查询区块头。
// 如果本地缺失，自动从 Blockqs 获取。
func (bc *Blockchain) HeaderByHeight(height int) (*BlockHeader, error)

// HeaderByHash 按哈希查询区块头。
func (bc *Blockchain) HeaderByHash(hash Hash384) (*BlockHeader, error)

// HeadersByYear 获取指定年份的所有区块头。
func (bc *Blockchain) HeadersByYear(year int) ([]*BlockHeader, error)

// ChainTip 返回当前链顶信息。
func (bc *Blockchain) ChainTip() (*BlockHeader, error)

// ChainHeight 返回当前链高度。
func (bc *Blockchain) ChainHeight() int
```

### 6.2 Sync Interface（同步接口）

供外部组件同步区块头链数据：

```go
// SyncHeaders 同步指定范围的区块头。
// 用于节点启动时的初始同步或缺失数据的补全。
func (bc *Blockchain) SyncHeaders(from, to int) error

// Subscribe 订阅新区块事件。
// 当新区块被成功提交时，通知订阅者。
func (bc *Blockchain) Subscribe(ch chan<- *BlockHeader)
```


## 7. Manual Chain Switching（手动切换主链）

### 7.1 Background（背景）

当全球网络发生长时间分区（超过 2 小时 / 20 个区块），分叉竞争机制将终结（被拒绝参与），两条链均会被视为合法。此时系统无法自动决策，需要用户人工介入。

### 7.2 Switching Process（切换流程）

1. **发现分叉**：用户获知存在替代链（通过社区、公告或其他渠道）
2. **指定节点**：用户手动指定目标分叉链上的若干可信节点
3. **获取数据**：Core 从这些节点获取分叉点之后的区块头链数据
4. **验证连续性**：验证替代链的区块头连续性（PrevBlock 链接）
5. **执行切换**：用户确认后，Core 将本地链切换至替代链

### 7.3 Interface（接口）

```go
// ForkInfo 分叉信息。
type ForkInfo struct {
    ForkHeight int          // 分叉高度
    LocalTip   *BlockHeader // 本地链顶
    RemoteTip  *BlockHeader // 远程链顶
    Length     int          // 远程链长度（分叉点之后）
}

// DetectFork 检测与指定节点之间的分叉。
func (bc *Blockchain) DetectFork(peers []string) (*ForkInfo, error)

// SwitchChain 切换到替代链。
// 这是一个需要用户明确确认的危险操作。
func (bc *Blockchain) SwitchChain(forkHeight int, headers []*BlockHeader) error
```

### 7.4 Social Consensus（社会共识）

手动切换主链本质上是一种"用脚投票"的社会性选择机制，不属于算法逻辑的范畴。在未来天基互联网（卫星互联网）的强连接环境下，全球网络长时间分区的情况应极为罕见。


## 8. Security（安全性）

### 8.1 Trust Model（信任模型）

Core 自身不执行深度校验，它信任外部提交者的合法性判断。不同场景下的信任模型不同：
- **校验节点**：外挂完整的共识和校验模块，自行验证后再提交给 Core
- **轻客户端**：默认信任从 Blockqs 获取的区块头链数据
- **公共服务节点**：可能运行部分校验逻辑

### 8.2 Year-Block Anchoring（年块锚定）

年块机制提供了一种高效的完整性验证手段：即使中间区块头缺失，只要年块哈希链完整，就能确认链的宏观连续性。这类似于区块链的"骨架"。

### 8.3 Connection Security（连接安全）

与 Blockqs 等外部服务的连接采用自签名证书（SPKI 指纹验证），证书有效期建议为 30 天，兼顾安全性与运维便捷性。


## 9. Initial Chain Verification（初始主链验证）

没有任何数据的初始节点上线时，需要获取主链信息并验证其合法性。主链信息由 Blockqs 或某个校验组提供。

### 9.1 Verify the required data（验证所需数据）

| 序号 | 数据 | 说明 |
|------|------|------|
| 1 | **创世区块及区块头** | 客户端硬编码内置，作为链的信任锚点 |
| 2 | **区块头链** | 从当前区块高度到创世块的完整区块头链，可局部用年块衔接以减轻负载 |
| 3 | **末端部分区块的 Coinbase 交易及其哈希验证路径** | 包含 UTXO/UTCO 指纹以及铸造者对 `CheckRoot` 的签名，用于验证末端区块的真实性 |
| 4 | **当前 UTXO/UTCO 集与末端部分区块数据**（可选） | 用于逆推式严格验证（参见 Checks-by-Team 提案 §8.5 Chain Constraint） |

> **注：**
> - 末端部分区块长度可能为 **29 个区块**，以涵盖分叉竞争区间。
> - 校验组通常至少保存最近一天的铸造者签名数据，即 **240 个区块**长度。

### 9.2 Verification process（验证流程）

初始节点可向不同的 Blockqs 节点或校验组请求上述数据，相互比对以确保找到真实的主链。

```go
// BootstrapVerify 初始主链验证。
// 从多个数据源获取主链信息并交叉验证，确认目标主链的合法性。
func (bc *Blockchain) BootstrapVerify(sources []string) (*BootstrapResult, error)

// BootstrapResult 初始验证结果。
type BootstrapResult struct {
    ChainTip    *BlockHeader // 验证通过的链顶区块头
    Height      int          // 当前链高度
    Confidence  float64      // 置信度（基于多源一致性）
    Sources     int          // 参与验证的数据源数量
}
```


## 10. Chain Identity（主链和分叉标识）

### 10.1 Identity Fields（标识字段）

标识一条区块链需要以下几项信息：

| 字段 | 说明 | 示例 |
|------|------|------|
| `Protocol-ID` | 区分本区块链与其它区块链 | `Evidcoin@V1` |
| `Chain-ID` | 当前区块链的运行态 | `mainnet`（正式主链） |
| `Genesis-ID` | 创始块信息，即创世区块的区块 ID | — |
| `Bound-ID` | 主链绑定（可选），为 -29 号区块 ID 的前 20 字节 | — |

**Bound-ID 说明：**
- 主要用于分叉后绑定主链，避免新交易在支链上被重放。
- **可选**：仅在分叉竞争完成后有用，随着时间推移可能无需此标识。
- 分叉后依然为可选（关乎未来交易），用户需自行承担不绑定主链的风险。

### 10.2 Usage in Transaction Signing（参与交易签名）

标识信息参与交易的签名，作为链区分前置于交易消息之前：

```
// 识别信息前置
// TxMSG 为交易相关的签名消息（有内在结构）
MixData = ( Protocol-ID || Chain-ID || Genesis-ID || Bound-ID ) || TxMSG
signData = Sign( MixData )
```

> **设计意图：**
> 放在签名中可方便后期签名数据剪枝，同时维持交易信息纯粹。

### 10.3 Usage in P2P Handshake（参与 P2P 握手）

标识信息同样用于节点在 P2P 连接握手时声明身份，以便节点之间相互识别所属链网络，拒绝来自不同链的连接请求。

```go
// ChainIdentity 链标识信息。
type ChainIdentity struct {
    ProtocolID string // 协议标识，如 "Evidcoin@V1"
    ChainID    string // 运行态标识，如 "mainnet"
    GenesisID  Hash384 // 创世区块 ID
    BoundID    []byte  // 主链绑定（可选，取 -29 号区块 ID 前 20 字节）
}

// Identity 返回当前节点的链标识信息。
func (bc *Blockchain) Identity() *ChainIdentity
```
