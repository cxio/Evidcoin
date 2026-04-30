# 06 — Consensus: Proof of Historical (PoH) (共识：历史证明)

> 源自：`conception/1.共识-历史证明（PoH）.md`，
> `conception/2.共识-端点约定.md`、`conception/4.激励机制.md`（mint schedule）。

本提案规定 PoH 共识机制：包括 mint-pledge-hash 算法、selection-pool 的
生命周期与同步机制、区块生产时序、分叉决议规则、标准铸币奖励曲线，以及
初始阶段（early-block / 100-day expansion）的覆盖规则。


## 1. Concepts (概念)

| 术语                       | 定义                                                                              |
|---------------------------|-----------------------------------------------------------------------------------|
| **铸凭交易** (mint pledge tx) | 任何 `[-80 000, -28]` 区块区间内的合法交易                                  |
| **铸凭哈希** (mint pledge hash) | 候选者计算出的最终对比值；值小者优                                          |
| **评参区块** (reference block)  | 当前区块 `-9` 号区块，提供 ref-mint-hash                                  |
| **币权总值**               | 区块所收录交易"币量×币龄"的总和（聪时；见 `02` §1.1）                            |
| **铸造者**                 | 铸凭交易的*首领输入*接收者公钥地址                                                |
| **择优池**                 | 每个 *未来区块* 对应一个池，容量 20 名候选者                                       |
| **同步池 / 合并池**        | 择优池跨节点同步的中间结构                                                        |


## 2. Anti-Malleability Constraints (抗可塑性约束)

Hash-shaping 攻击（`通过填充无用数据塑造哈希值`）必须被约束。本设计采用以下对策：

- *交易 ID*：可塑（攻击者可向交易填充字节），因此仅用于末端动态参考之**前**的
  历史区域使用；攻击者在交易创建时无法预知未来的评参区块。
- *区块 ID*：对铸造者可塑（Coinbase 自由数据 + 交易序列重排），因此 **不**作为参与
  因子。
- *UTXO 指纹*：可塑但成本明确（需实际花费交易费）；本设计**未**使用，已被币权
  总值替代。
- *币权总值*：低可塑、成本高（需创建/移除真实交易）；作为新的因子。
- *评参区块时间戳*：完全无可塑（由高度严格推导）。


## 3. Mint Pledge Hash (铸凭哈希)

### 3.1 Eligibility (资格条件)

候选铸凭交易必须满足：

1. 区块高度 `txHeight` 落在 `[currentHeight - 80 000, currentHeight - 28]`；
2. 该交易的*首领输入*接收者地址（即铸造者）的币权销毁 ≥ 4 000 000 聪时
   （`MinterStakeFloor`）。

### 3.2 Algorithm (算法)

```go
// 常数与字段：
const Mix uint64 = 0x517cc1b727220a95   // 混合常数

Stakes    := chain.Header(currentHeight - 27).Stakes  // 聪时
timeStamp := blockTime(currentHeight)                  // 当前区块时间戳（毫秒）

X        := bytes(timeStamp * Stakes * Mix)            // 字节序列
hashData := SHA3_384(
    pledgeTxID                              // 铸凭交易 ID
    || refBlock.MintPledgeHash             // 评参区块的铸凭哈希（来自其 Coinbase）
    || X
)
signData := Sign(minterPrivKey, hashData)              // 私有
hashMint := BLAKE3_512(signData)                       // 公开
```

`hashMint` 的字节序列从小到大比较，值小者胜出。

### 3.3 Why sign first then hash again (为何先签名再二次哈希)

- `hashData` 是公开可推算的；攻击者可据此“猎取并收买”潜在高权重者。
- `signData` 含私钥，仅候选者本人可计算，攻击者无法预知 `hashMint` 的最终值。
- 二次哈希 (`BLAKE3-512(signData)`) 屏蔽签名内部结构带来的统计偏差，并提供
  足够熵宽。

### 3.4 Mint Certificate (铸凭证书)

候选者向网络广播的“择优凭证”包含：

```go
type MintCert struct {
    Year         uint32     // 铸凭交易年度（用于检索）
    PledgeTxID   [48]byte   // 铸凭交易 ID
    MinterPubKey []byte     // 首领输入接收者公钥（不是哈希）
    Sign         []byte     // signData
    HashMint     [64]byte   // 可选：BLAKE3_512(Sign) 缓存
}
```

### 3.5 Mint Cert Verification (legality of source) (铸凭证校验：来源合法性)

接收方验证流程：

0. 通过区块头链确认 `PledgeTxID` 所在区块的合法性。
1. 从 Blockqs 检索该交易的 TxHeader、首领输入、剩余输入哈希、输出根哈希。
2. 重算 TxID，验证一致。
3. 取首领输入的（年度 + TxID + 输出序位），检索其来源输出项 + 哈希校验路径。
4. 验证首领输入来源 TxID 的合法性。
5. 核验来源输出项里的接收者公钥哈希 = `SHA3-256(BLAKE2b-512(MinterPubKey))`。
6. 用 `MinterPubKey` 验证 `Sign`；可选：验证 `HashMint`。

> **优化建议：** 候选者也可主动附带“首领输入来源交易”的证明材料，免去验证
> 节点二次检索；管理层 SHOULD 接受这种优化路径。


## 4. Selection Pool (择优池)

### 4.1 Capacity & ordering (容量与排序)

- 容量：20 名。
- 排序：按 `HashMint` 字节序列升序（值小者列前）。
- 池集：每一个**未来区块**有一个择优池。

### 4.2 Insertion rule (插入规则)

新候选者尝试加入：

- 计算 `HashMint`，若优于池中末位，则有序插入并立即转播；
- 否则忽略（不转播）。

### 4.3 Authorised synchronisation tail (授权同步尾段)

为防止"迟到的优质者"无限触发同步震荡：

- 池中**位置 6..20** 共 15 名候选者享有同步发起权（`SyncAuthorisedTail`）。
- 任何一名授权节点仅有 **一次** 发起同步的权力。

### 4.4 Lifecycle (生命周期)

为方便描述，设区块 `B` 是 *待评参* 的目标区块。`B` 的择优池随 `B` 自身在主链
上的位置变化（"末端 → -7 → -9"）有三段：

| 阶段                     | `B` 在主链的位置 | 时长       | 行为                                                |
|--------------------------|------------------|------------|-----------------------------------------------------|
| 广播收集 (Broadcast)     | 末端 → -7        | 6 区块时段 | 计算 / 广播 / 优选；候选者持续涌入                   |
| 同步优化 (Sync)          | -7 → -9          | 2 区块时段 | 后段 15 名授权节点发起同步，合并优化                |
| 抵达终结 (Frozen)        | 至 -9            | —          | 该池冻结，作为新区块的评参依据                      |

### 4.5 Sync protocol (同步协议)

- 同步消息类型：`SyncPool`，包含发起者签名 + 择优池快照（最多 20 项）。
- 接收方：
  1. 校验签名者身份：在自己的*择优池*中（不论排名）或在*合并池*中。
  2. 校验签名者是否位于其发布的同步池后段 15 名内。
  3. 通过则将其条目并入本地的 *合并池*（独立结构）。
- 抵达终结时，**合并池替换**节点本地的目标区块择优池。

> **首次同步豁免：** 新上线节点首次同步时可放宽条件 1（任意签名者均接受）。


## 5. Block Production & Fork Resolution (出块与分叉决议)

### 5.1 Slot timing (时隙时序)

- 区块时间戳 = `genesisTime + Height × 6min`（毫秒精度）。
- 首个区块在标准时间戳后**延后 30 秒**发布（收集尾段交易）。
- 排名 1..20 的候选者按顺序发布候选区块，相邻 **15 秒** 间隔。

候选者若已收到排名靠前者的合法区块，应停止发布自身区块。

### 5.2 Block competition (合法多区块)

| 情况                              | 胜出规则                                              |
|-----------------------------------|-------------------------------------------------------|
| 同一铸造者签发多个区块             | 交易费收益**最低**的区块胜出（避免多签）              |
| 主区块与冗余区块                   | 若冗余区块的 Stakes ≥ 3 × 主区块 Stakes，则冗余胜出    |

第二条用于抑制自私铸造（仅含 Coinbase 的空块）。

### 5.3 New-coin spend delay (新币花费延迟)

Coinbase 中的"校验组 / 铸凭者"输出在被铸造的 29 个区块之后才允许作为输入花
费（`NewCoinSpendDelay = 29`）。这是为应对分叉竞争窗口与回收。

### 5.4 Fork competition (分叉竞争)

| 参数                             | 值                                                |
|----------------------------------|---------------------------------------------------|
| 评比窗口 (`ForkCompeteWindow`)   | 29 区块                                            |
| 接纳上限 (`ForkAdmissionMax`)    | ≤ 20 区块                                         |
| 早期决胜阈值                     | 一方先胜 15 个区块（过半）即提前胜出              |

规则：

- 节点逐块比较两条分叉的 `HashMint`，先到达 15 胜的一方接管为本链。
- 若分叉长度已 > 20（首次发现时即超），不予接收。
- 临界状况：分叉长度恰为 20 且发现时间临近本链当前块时间点，由 *本链当前区块
  的择优池前 5 名* 签名裁决。
  - 排名最靠前者裁决有效；
  - 5 人均未签名 ⇒ 默认否决。

### 5.5 Manual switch (手动切换)

> 网络分区超过 2 小时（即分叉长度 > 20 块）后，算法不再裁决。客户端 SHOULD
> 提供手动主链切换入口（详见 `02` §10）。

### 5.6 Transaction recovery (交易回收)

分叉竞争结束后，败方分叉上的交易 SHOULD 被回收并尝试合并到主链。新币 29 个
区块的延迟保证了 Coinbase 输入源在回收时不会出错。


## 6. Mint Reward Schedule (铸币奖励曲线)

### 6.1 Yearly reward (per-block coin amount) (年度奖励：每块币量)

按 *恒星年*（87 661 块）划分；单位为"币"（不是聪），整数除法 `x*80/100` 截
断，不补余。

| 年次       | 每块币量 | 备注                                              |
|------------|----------|---------------------------------------------------|
| 1          | 10       | 三年试运行期，逐年递增                            |
| 2          | 20       |                                                   |
| 3          | 30       |                                                   |
| 4–5        | 40       | 正式发行起点                                      |
| 6–7        | 32       | = 40 × 80 / 100                                  |
| 8–9        | 25       | = 32 × 80 / 100                                  |
| 10–11      | 20       |                                                   |
| 12–13      | 16       |                                                   |
| 14–15      | 12       |                                                   |
| 16–17      | 9        |                                                   |
| 18–19      | 7        |                                                   |
| 20–21      | 5        |                                                   |
| 22–23      | 4        |                                                   |
| 24+        | 3        | 永久维持，长期低通胀                              |

> **公式（年次 ≥ 4）：**
> `coin_per_block(year) = max(3, prev * 80 / 100)`，其中 `prev` 为前一阶段值，
> 阶段长度为 2 年。一旦达到 3，永久锁定。

### 6.2 Year boundary (年度边界)

"年次"按 *恒星年的区块计数* 划分：`year_n = (Height-1) / 87661 + 1`。该
"年次"与交易时间戳的"公历年" (见 `04` §3) **不**相同；切勿混淆。

### 6.3 Fee-burn rule (手续费销毁规则)

每个区块手续费总额的 **50%** 被销毁（在 Coinbase 中以 `Receiver = null`、
`config bit-5 = 1` 的"销毁输出"表达）；剩余 50% 才进入 `Income.Fees` 参与五
项分成。

### 6.4 Initial 100-day expansion (初始 100 天扩张)

| 区块号                | 称谓        | 处理                                              |
|-----------------------|-------------|---------------------------------------------------|
| 0                     | 创始区块    | 链启动                                            |
| 1 – 10                | 私钥扩张    | 创建初期收益地址                                  |
| 11 – 7 200 (30 日)    | 币金积累    | 观察期；保留终止运行的可能                        |
| 7 201 – 24 000 (百日)  | 抽奖扩张    | 社区收集地址、由积累币金发放奖励                  |
| 24 001 –              | 正常市场    | 接受公共服务，启动外部奖励，进入完整 5 项分成模式 |

百日期间约束（共约）：每笔交易的输出项数量 ≤ 输入项数量 × 2，以约束扩张速度。
百日之前 Coinbase 仅含 1 笔铸造者输出，无外部服务奖励。


## 7. Initial-Phase Algorithm Overrides (初始阶段算法覆盖)

链最初的 9 个区块没有 `-9` 号评参区块；最初的 28 个区块没有合法的铸凭交易
区间。需特别处理：

```go
// 评参区块取值
func RefBlockHeight(currentHeight uint64) uint64 {
    if currentHeight < 9 {
        return 0   // 取创始块
    }
    return currentHeight - 9
}

// 铸凭交易区块高度合法性
func IsPledgeBlockLegal(currentHeight, txHeight uint64) bool {
    if currentHeight < 28 {
        return true    // 全部交易皆可作为铸凭
    }
    h := currentHeight - txHeight
    return h > 27 && h <= 80000
}
```


## 8. Endpoint Convention (端点约定)

为完整性，记录共约（非协议级强制）规则如下：

| 名称        | 性质     | 内容                                                                                |
|-------------|----------|-------------------------------------------------------------------------------------|
| 最低交易费  | 共约     | `min_fee = avg_fee(last 6 000 blocks) / 4`                                           |
| 错时延迟    | 共约     | 出块时间到达后，时间戳早于该时间的"错时交易"应暂停转播至下一区块出块完成              |
| 全网通告    | 极简中心化 | 仅文本消息；签名者由官方/社区维护的公钥清单授权                                     |

> 协议级强制：`TxExpiryWindow = 240` 区块（约 24 h）。超过即作废，铸造者**不**能收录。

### 8.1 Zero-confirmation (零确认)

简化处理：节点发现双花交易时，本地 App SHOULD 弹出警告。系统不提供加固的零确认
保障；大额或重要交易应等待 ≥ 29 个确认（参见 `05` §3.3）。


## 9. Public-API Surface (Layer 4) (公开 API 面)

```go
package consensus

// Engine 是 PoH 共识引擎。
type Engine interface {
    // EvalCandidate 对一笔铸凭交易计算其在指定目标区块下的 hashMint。
    EvalCandidate(target uint64, cand *MintCert) ([]byte, error)

    // SubmitCandidate 把候选者加入本地择优池（如优于末位则有序插入并转播）。
    SubmitCandidate(target uint64, cand *MintCert) (inserted bool, err error)

    // Pool 返回目标区块的择优池快照。
    Pool(target uint64) []*MintCert

    // Sync 接受/发起择优池同步消息。
    Sync(msg *SyncPool) error

    // ResolveFork 评比两条分叉，返回胜出端点。
    ResolveFork(a, b *Branch) (winner *Branch, err error)
}
```

`MintCert`、`SyncPool`、`Branch` 等类型由 `consensus` 包私有定义；上层（验证
组管理层、cmd 层）通过该接口完成全部 PoH 相关操作。
