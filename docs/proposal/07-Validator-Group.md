# 07 — Validator Group & External Service Integration (校验组与外部服务集成)

> Derived from（来源于）: `conception/附.组队校验.md`, `conception/3.公共服务.md`,
> `conception/4.激励机制.md`.

本提案规定了校验组（validator groups，校验组）的架构与消息协议、
leader-check 快速路径、冗余 / 复核保障机制、铸币者协同流程（含 coinbase 交换）、
外部服务集成契约（Depots / Blockqs / STUN / basenet），以及将 Coinbase 外部服务
输出与延迟确认绑定的兑奖机制。

在协议边界上，外部服务被视为 **black boxes** —— 我们定义接口，不定义实现。


## 1. Architecture Overview (架构总览)

一个校验组是由三类角色通过组内紧密连接协作形成的逻辑单元；
而整个组在更广泛网络中表现为一个 P2P 节点。

```
                    ┌────────── inter-group ──────────┐
                    │                                 │
   external  →  Guardian  ───┐                ┌────────  Manager(其它组)
   (txs)        Guardian  ───┤                │
                          ↓ (leader-check pass)│
                       Manager (Broadcast +    │
                                Schedule)       │
                          ↑ (full-check)        │
                       Checker  ───────────────┘
                       Checker
                       …
                       │
                       └── shared services ──
                            UTXO/UTCO Cache
                            ExternalScript Cache
```

### 1.1 Roles (角色)

| Role            | Job                                                                                  |
|-----------------|--------------------------------------------------------------------------------------|
| **Manager** (管理层) | Broadcast + Schedule。打包区块、与其他组的 manager 交换信息、向 checker 派发交易、记录绩效、对接 minter。 |
| **Guardian** (守卫者) | 接收来自其他组的交易，执行 leader-check，将预校验通过的交易推送给 manager，并向对等 guardian 进行 gossip。 |
| **Checker**     | 从 manager 拉取任务，执行完整校验，将判定结果返回 manager，并向对等 guardian gossip 自身通过列表。 |

### 1.2 Manager sub-roles (Manager 子角色)

- **Broadcast** —— 组间 manager 链路；处理 minter 协同、区块发布、区块导入。
- **Schedule** —— 组内调度器；维护预校验交易池，向 checker 分派任务（含冗余），
  并记录每个成员的工作量用于结算分配。

### 1.3 Connectivity matrix (连接矩阵)

|              | Internal       | External                                          |
|--------------|----------------|---------------------------------------------------|
| Manager      | Guardian, Checker | Manager (peer groups)                          |
| Guardian     | Manager        | Guardian (peer groups), Checker (peer groups, inbound only) |
| Checker      | Manager        | Guardian (peer groups, outbound only)            |

### 1.4 Shared in-group services (组内共享服务)

- **UTXO Cache / UTCO Cache**（参见 `04` §4）— 快速查询可用输出。
- **External Script Cache**（参见 `05` §7）— 中介输出脚本的已解析正文缓存，
  以 `(year, txid_short, out_index)` 为键。

这些服务是独立进程（至少也是独立组件），通常与组内角色部署在同一台机器上。


## 2. Membership & Openness (成员关系与开放性)

- 加入组是**免费的**；节点 MAY 同时加入多个组（取决于硬件能力）。
- 组创建是**开放的**，但并非无成本：manager 必须承担带宽、调度、冗余与安全运维开销。
- 协议不授予特殊访问权限；加入无需许可，只需具备能力。
- 各组之间保持自由 P2P 关系；组内是 *micro-centralised*，组间是 *decentralised*。


## 3. Leader Check (首领校验) — Fast Path (快速路径)

当 guardian 收到一笔新交易时：

1. 仅校验**第一输入**：
   - 它是 Coin spend（协议要求；参见 `03` §2.2）；
   - 在该交易全部 Coin 输入中，它是 stakes burn 最大的输入；
   - 其签名能通过源输出 lock script 校验。
2. 若有效，将交易推入 manager 的预校验池，并向对等 guardian gossip。
3. 若无效，丢弃。将第一输入来源加入 **Black-list** 24 小时
   （`BlackListFreeze`），防止同一 lead-input 发起大量伪造交易洪泛。

### 3.1 Random-decision broadcast (随机决策广播)

为平衡广播时延与抗洪泛能力，每个 guardian 使用**50 % 随机开关**：
一半入站交易先做 leader-check 再转发；另一半直接转发。


## 4. Full Validation (Checker Side) (完整校验（Checker 侧）)

### 4.1 Task allocation (任务分配)

- Schedule 将每笔预校验交易派发给 **≥ 2** 个 checker（冗余度 ≥ 2）。
  checker 不知道自己拿到的是常规任务还是升级复核任务。
- checker MAY 以“负载过高”拒绝任务；manager 会将重复拒绝视作该交易的“冷处理”信号。

### 4.2 Verdict aggregation (判定聚合)

| Outcome from N ≥ 2 checkers           | Result                                |
|---------------------------------------|---------------------------------------|
| All PASS                              | Tx accepted                           |
| ≥ 1 FAIL                              | Escalate to **L1 re-verification**    |
| L1: 0 FAIL                            | PASS                                  |
| L1: > half FAIL                       | FAIL                                  |
| L1: < half FAIL                       | Escalate to **L2 re-verification**    |
| L2: ≥ 1 FAIL                          | FAIL                                  |

复核会派发给性能更高的 checker（仍保持冗余）。

### 4.3 Cross-group feedback (跨组反馈)

- checker MAY 直接把自己校验通过的交易转发给其他组 guardian（加快传播）。
- 若 guardian 拒绝入站交易，MUST 通知发送方（原始 checker）。
  发送方所属 manager 需对该交易重新执行 L1+L2 复核。
- 因此 manager MUST 记录每笔跨组交易的**投递来源**，确保错误反馈可正确路由回去。


## 5. Tx Admission Priority & Exclusions (交易准入优先级与排除规则)

### 5.1 Admission priority (when packing a block) (打包区块时的准入优先级)

```
高币权销毁  >  高交易费  >  携带凭信提前销毁
```

理由：优先提高攻击成本，其次才是纯手续费竞争。

### 5.2 Exclusions (do not include in block) (排除规则（不纳入区块）)

| Rule                                                                                 | Reason                                                  |
|--------------------------------------------------------------------------------------|---------------------------------------------------------|
| Inputs whose `TxIDPart[:20]` collides within the same tx                            | Defensive against (theoretical) 20-byte collisions      |
| New txs whose leaf-layer `TxID[:8]` collides with an existing UTXO/UTCO leaf        | Same                                                    |

这两类碰撞在实践中几乎不可能发生，但仍排除以确保 `TxID` 一致性的严密性。


## 6. Minter Coordination & Coinbase Exchange (铸币者协同与 Coinbase 交换)

### 6.1 Eligibility (资格)

区块签名者必须是当前目标区块**候选池**成员；
这与校验工作量无关——仅参与 PoH 铸币的 *观察者*（onlookers）也可被接受。

### 6.2 Per-minter low-revenue rule (单一 minter 最低收益规则)

若同一 minter 签署了多个竞争区块，则**收益最低的区块胜出**。
该规则消除多签激励，防止组为了追逐迟到高费交易而反复重打包（这会破坏准时出块）。

### 6.3 Coordination flow (information separation) (协同流程（信息分离）)

信息被刻意拆分在 minter 与 manager 之间，使任一方单独都无法完成区块最终定稿。流程如下：

```
Minter                                      Manager
   │                                              │
   │  (1) ApplyMint{MintCert.signData}            │
   ├─────────────────────────────────────────────►│
   │                                              │  校验 cert ∈ pool
   │                                              │  准备：
   │                                              │   - 区块手续费
   │                                              │   - 组奖励地址
   │                                              │   - 推荐公共服务地址
   │                                              │   - 铸币数量
   │                                              │   - AwardWithheld
   │  (2) MintParams{...}                          │
   │◄─────────────────────────────────────────────┤
   │                                              │
   │  按 §03.5 本地构建 Coinbase                  │
   │                                              │
   │  (3) SubmitCoinbase{coinbaseTx}              │
   ├─────────────────────────────────────────────►│
   │                                              │  校验 Coinbase
   │                                              │  打包区块
   │                                              │  计算路径证明
   │  (4) BlockProof{TreeRoot, UTXORoot,           │
   │                 UTCORoot, mt-path of Cb}     │
   │◄─────────────────────────────────────────────┤
   │                                              │
   │  校验 own Coinbase ∈ tree                    │
   │  签名 CheckRoot                              │
   │  (5) BlockSignature{sig over CheckRoot}      │
   ├─────────────────────────────────────────────►│
   │                                              │  校验 sig
   │                                              │  生成 BlockHeader
   │                                              │  发布区块
   │                                              │
```

### 6.4 Free choice of public-service receivers (公共服务接收地址可自由选择)

步骤 (2) 返回的推荐公共服务接收地址仅为**建议值**。
minter MAY 在 Coinbase 中替换为自己的选择。
若 manager 拒绝该替换，它只需不返回步骤 (4) / 不进入步骤 (5)；
此时 minter 可选择向**其他**校验组申请。

该设计在保留 minter 自主性的同时，也允许组执行质量控制。


## 7. External Service Integration (Black-Box Contracts) (外部服务集成（黑盒契约）)

以下四类外部网络均为链外进程。协议只定义它们对链代码暴露的**接口契约**；
内部实现位于各自独立项目。

| Service  | Project                              | Concern                                          |
|----------|--------------------------------------|--------------------------------------------------|
| basenet  | `github.com/cxio/p2p`                | Node discovery, gossip overlay                   |
| STUN     | `github.com/cxio/stun2p`             | NAT traversal                                    |
| Depots   | `github.com/cxio/depots`             | Attachment storage & big block bodies            |
| Blockqs  | `github.com/cxio/blockqs`            | Tx / output / UTXO / UTCO query                  |

### 7.1 Handshake contract (握手契约)

当 Peer 连接服务节点时，服务节点 MUST 提供其用于接收奖励的
**chain-account address**。握手消息如下：

```go
type ServiceHandshake struct {
    ProtocolID  string         // 例如 "Evidcoin@V1"
    ChainID     string
    ServiceKind string         // "blockqs" | "depots" | "stun"
    RewardAddr  [32]byte       // 公钥哈希
    Endpoints   []string
    Sig         []byte         // 由 RewardAddr 对应私钥对前述字段签名
}
```

Peer 会缓存 `RewardAddr`，并在构建 Coinbase 或参与奖励兑奖投票时使用它。

### 7.2 Blockqs query interface (subset) (Blockqs 查询接口（子集）)

```go
type Blockqs interface {
    GetTx(year uint32, txid [48]byte) (*tx.Tx, *Proof, error)
    GetOutputSet(year uint32, txid [48]byte) ([]tx.Output, *Proof, error)
    GetUTXOSet(year uint32) iter.Seq[*tx.Output]
    GetUTCOSet(year uint32) iter.Seq[*tx.Output]
    GetTxsByAccount(addr [32]byte) iter.Seq[[48]byte]
    GetAttachmentSmall(attID []byte) ([]byte, error)   // < 10 MB 或分片索引
    GetHeader(height uint64) (*block.BlockHeader, error)
}
```

返回的 `*Proof` 包含哈希树路径，使 Peer 能相对本地区块头链进行本地校验。

### 7.3 Depots interface (subset) (Depots 接口（子集）)

```go
type Depots interface {
    GetAttachmentChunk(attID []byte, chunkIdx uint16) ([]byte, *ChunkProof, error)
    GetFullBlock(blockID [48]byte) ([]byte, error)
    Probe(attID []byte) bool   // 数据心跳（可选公益接口）
}
```

### 7.4 STUN interface (subset) (STUN 接口（子集）)

该接口不在区块链主代码路径内；链侧仅需如下能力：

```go
type STUN interface {
    Probe(addr string) (publicAddr string, err error)
    Punch(local, remote string) error
}
```

### 7.5 basenet integration (basenet 集成)

`cxio/p2p` 提供节点发现；链层通过以下接口使用：

```go
type Discovery interface {
    FindKind(kind string, n int) ([]Peer, error)
    Announce(kind string, addr string) error
}
```

`kind` 示例：`"evidcoin/peer"`, `"evidcoin/manager"`, `"evidcoin/guardian"`,
`"blockqs"`, `"depots"`, `"stun"`。


## 8. Reward Redemption Mechanism (兑奖) (奖励兑奖机制)

### 8.1 Why deferred? (为何延迟确认？)

Coinbase 中支付给 Blockqs / Depots / STUN 的输出，在兑奖机制确认前**不可花费**。
这使随机地址或不提供服务的地址失去意义：其他 minter 看不到服务，也不会认可它们。

### 8.2 Award slots (奖励槽位)

每个区块 Coinbase 包含 `AwardSlots` 字段——一个 144-bit（18-byte）位图，
表示 48 个区块 × 3 类服务。每个新 minter 会检查**前序 48 个区块**，
并对其认定服务良好的目标（按区块与服务组合）翻转一位。

| Slot byte range | Service                          |
|-----------------|----------------------------------|
| `[0..6)`        | Blockqs slots (48 bits)          |
| `[6..12)`       | Depots slots (48 bits)           |
| `[12..18)`      | STUN slots (48 bits)             |

### 8.3 Confirmation thresholds (确认阈值)

```
confirmations  this-time-share  cumulative
1              50 %             50 %
2              50 %             100 %
```

即：第一位确认者释放一半奖励；第二位释放剩余一半。

### 8.4 Time windows (时间窗口)

| Window                                    | Length       |
|-------------------------------------------|--------------|
| `AwardConfirmWindow`                      | 48 blocks (≈ 4.8 h) |
| `AwardEarlyExitWindow` (after 2 confirms) | 29 blocks (no fork risk) |
| Uncollected residual recovery             | block 49     |

### 8.5 Recovery (回收机制)

在 `AwardConfirmWindow` 内未完全确认的部分会被**扣留**，
并在第 49 个区块作为 `Income.AwardWithheld` 回流到 Coinbase，
参与该区块标准的 5 方分配。

### 8.6 Cache for evaluators (评估者缓存)

每个 minter 必须记住哪些地址提供优质服务。若服务节点有 10 000 个，
且每个地址每类服务有 24 个确认槽位，缓存规模约为：

```
addr_cache_size ≈ 10 000 / 24 ≈ 416
```

即每种服务类型约 416 项。生产环境中可实现为基于直接交互观测地址的滚动 LRU。

### 8.7 Per-service evaluation hints (按服务类型的评估建议)

| Service | Evaluation method                                                                   |
|---------|-------------------------------------------------------------------------------------|
| STUN    | Probe / punch 往返成功率、时延                                                        |
| Depots  | 随机抽样：请求随机 block / tx / attachment chunk，并校验完整性                         |
| Blockqs | 已与链持续紧耦合；重点关注可达性 + 正确性                                              |

三类服务的槽位字节彼此**独立**，未来可分别调优各自窗口而无需引发协议级震荡。


## 9. Block Publication Optimisation (区块发布优化)

为提高网络效率，区块分三阶段发布：

1. **Proof broadcast** —— 最小数据集：Coinbase 交易、Coinbase 校验路径、
   minter 签名、区块头。
2. **Outline broadcast** —— 全量 TxID 列表（每个截断到前 16 字节；
   对 64 k-tx 区块约为 1 MB）。接收方本地比对后，请求缺失项或碰撞项的完整内容。
3. **Full sync** —— 拉取缺失交易正文。多数 peer 只存在少量缺口。


## 10. Tx-Volume Constraint (Anti-Selfish-Minting) (交易量约束（反自私铸币）)

按 `06` §5.4：若候选区块 `Stakes` ≥ 竞争主链区块 `Stakes` 的 3 倍，则候选区块胜出。
这实质上迫使“主”区块必须校验至少约 1/3 的可用交易，从而抑制仅靠空 Coinbase 的自铸尝试。


## 11. Public-API Surface (Layer 5) (公共 API 面（第 5 层）)

```go
package group

// Manager 是校验组 manager 进程的入口。
type Manager interface {
    // Start 启动 scheduler 与 broadcaster goroutine。
    Start(ctx context.Context) error

    // OnIncomingTx：guardian 推送一笔通过 leader-check 的交易。
    OnIncomingTx(t *tx.Tx, sourcePeer Peer)

    // OnCheckerVerdict：checker 上报某笔交易的判定结果。
    OnCheckerVerdict(txID [48]byte, ok bool, cost uint64, checker Peer)

    // ApplyMint：minter 通过进程间 RPC 发起铸币申请。
    ApplyMint(req *MintApply) (*MintParams, error)

    // SubmitCoinbase：minter 提交其构建的 Coinbase。
    SubmitCoinbase(cb *tx.Tx) (*BlockProof, error)

    // SubmitBlockSignature：minter 提交 CheckRoot 签名。
    SubmitBlockSignature(sig []byte) error

    // OnIncomingBlock：对等 manager 广播新区块。
    OnIncomingBlock(b *block.Block, peer Peer) error
}
```

`Guardian` 与 `Checker` 也有等价（更小）的接口；语义直接，形状与上述一致。


## 12. Optimal Group Size (最优组规模)

在 6 分钟出块、约 64 k tx/block（≈ 182 tx/s）条件下：

- 每组 **50–60 nodes** 已足够。
- checker 可自分配工作负载：能力强者多做，能力弱者少做。
- 组具备 **scale economy**；协议不保证均匀分布，但保证低门槛进入。
