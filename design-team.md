# 组队校验实现方案

基于 `docs/附.组队校验.md` 设计文档的实现方案。

## 核心概念

**组队校验** 将多个节点组成一个逻辑上的"校验组"，通过分工协作完成交易验证：

| 角色 | 职责 |
|------|------|
| **管理层**（广播+调度） | 交易分发、冗余控制、区块打包/发布 |
| **守卫者** | 接收交易、执行首领校验、快速广播 |
| **校验员** | 执行完整验证、向外传递合法交易 |
| **UTXO缓存器** | 缓存 UTXO 集，提供查询和双花检测 |


## 阶段一：基础设施层

### 目录结构

```
internal/
├── checkteam/              # 组队校验核心包
│   ├── types.go            # 基础类型定义
│   ├── message.go          # 组内通信消息协议
│   ├── role.go             # 角色接口定义
│   └── team.go             # 校验组管理
```

### 核心接口设计

```go
// 校验组成员的基础角色接口
type TeamMember interface {
    ID() NodeID
    Role() RoleType
    Start(ctx context.Context) error
    Stop() error
}

// 角色类型
type RoleType uint8

const (
    RoleManager   RoleType = iota + 1  // 管理层
    RoleGuard                          // 守卫者
    RoleValidator                      // 校验员
)

// 交易验证结果
type ValidationResult struct {
    TxID     Hash512
    Valid    bool
    Error    error
    NodeID   NodeID
    Duration time.Duration
}
```


## 阶段二：各角色实现

### 目录结构

```
internal/checkteam/
├── manager/
│   ├── dispatcher.go       # 调度员：交易分发、冗余控制
│   ├── broadcaster.go      # 广播者：区块发布、组间通信
│   ├── txpool.go           # 待验证交易池
│   └── performance.go      # 节点业绩记录
├── guard/
│   ├── guard.go            # 守卫者主逻辑
│   ├── leader_check.go     # 首领校验实现
│   └── blacklist.go        # 黑名单管理（24h冻结）
├── validator/
│   ├── validator.go        # 校验员主逻辑
│   └── workload.go         # 负载评估
└── cache/
    └── utxo_cache.go       # UTXO缓存服务
```

### 首领校验实现（关键算法）

```go
// 首领校验：仅验证首笔输入（币金且币权最大）
func (g *Guard) LeaderCheck(tx *Tx) (bool, error) {
    // 1. 找到币权最大的币金输入
    leaderInput := findLeaderInput(tx.Inputs)
    if leaderInput == nil {
        return false, ErrNoLeaderInput
    }
    
    // 2. 确认它是首笔输入
    if tx.Inputs[0].ID() != leaderInput.ID() {
        return false, ErrLeaderNotFirst
    }
    
    // 3. 检查黑名单
    if g.blacklist.Contains(leaderInput.OutPoint) {
        return false, ErrBlacklisted
    }
    
    // 4. 执行该输入的脚本验证
    return g.scriptVM.Verify(leaderInput)
}
```


## 阶段三：冗余与复核机制

```go
// 冗余校验配置
type RedundancyConfig struct {
    MinRedundancy   int     // 最小冗余度，默认 2
    ReviewThreshold float64 // 触发复核的异议比例
}

// 扩展复核逻辑
type ReviewLevel int

const (
    ReviewLevel1 ReviewLevel = iota + 1  // 一级复核：零报错合法，>50%报错非法
    ReviewLevel2                          // 二级复核：任何报错即非法
)

// 调度员的交易派发逻辑
func (d *Dispatcher) DispatchTx(tx *Tx) error {
    // 选择 N 个校验员（冗余）
    validators := d.selectValidators(d.config.MinRedundancy)
    
    // 并行派发，收集结果
    results := d.dispatchAndCollect(tx, validators)
    
    // 判断是否需要复核
    if hasDisagreement(results) {
        return d.extendedReview(tx, ReviewLevel1)
    }
    
    return d.confirmTx(tx, results)
}
```


## 阶段四：铸造者流程

### 目录结构

```
internal/checkteam/
└── minting/
    ├── candidate.go        # 铸造候选者逻辑
    ├── coinbase.go         # Coinbase 交易构建
    ├── signing.go          # 区块签署流程
    └── lowcost.go          # 低收益原则实现
```

### 铸造流程状态机

```
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│  提交择优证明  │ -> │  构建Coinbase │ -> │  签署区块    │
└──────────────┘    └──────────────┘    └──────────────┘
       │                   │                   │
       ▼                   ▼                   ▼
   验证资格            验证交易结构         验证哈希路径
   返回区块信息        打包区块             发布区块
```


## 阶段五：网络层集成

利用现有 P2P 框架（`github.com/cxio/p2p`）：

```go
// 组内通信（半信任）
type IntraTeamNetwork interface {
    RegisterMember(member TeamMember) error
    Broadcast(msg Message) error
    SendTo(nodeID NodeID, msg Message) error
}

// 组间通信（P2P）
type InterTeamNetwork interface {
    ConnectTeam(teamID TeamID) error
    BroadcastBlock(block *Block) error
    SyncTx(txID Hash512) (*Tx, error)
}
```


## 优先级建议

| 优先级 | 模块 | 理由 |
|--------|------|------|
| P0 | `types.go`, `role.go` | 基础类型，其他模块依赖 |
| P0 | `leader_check.go` | 首领校验是交易入口的关键 |
| P1 | `dispatcher.go`, `validator.go` | 核心验证流程 |
| P1 | `utxo_cache.go` | 验证依赖 UTXO 查询 |
| P2 | 冗余/复核机制 | 安全保障 |
| P3 | 铸造流程 | 依赖前置模块完成 |
