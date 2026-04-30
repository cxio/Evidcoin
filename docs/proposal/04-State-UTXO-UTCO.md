# 04 — State: UTXO / UTCO (状态集)

> 来源：`conception/blockchain.md` §哈希策略 / 哈希树结构；
> `conception/附.组队校验.md` §技术细节参考 / UTXO 集指纹的链式约束；
> `conception/5.信用结构.md`（UTCO 生命周期与凭信约束）。

本提案定义未花费 Coin 集（UTXO）、未转移 Credit 集（UTCO）、四层宽树指纹、
叶子载荷摘要 `DataID`、逆向推导的链式绑定机制，以及与信用单元语义绑定的生命周
期规则。


## 1. Set Definitions（集合定义）

### 1.1 UTXO（未花费交易输出）

`UTXO`（Unspent Transaction Output）包含满足以下条件的每个 Coin 输出：

- 属于已确认交易（即位于 canonical chain 某个区块中）；
- 未被后续任何已确认交易花费；
- 其配置位 bit-5（burn）未置位。

### 1.2 UTCO（未转移信用输出）

`UTCO`（UnTransferred Credit Output）包含满足以下条件的每个 Credit 输出：

- 属于已确认交易；
- 未被后续任何已确认交易转移；
- 其 `TransferRemaining` 计数器未耗尽（>= 1）；
- 未超过其 `HeightCutoff`；
- 在过去 11 个恒星年内被激活过（见 §5.3）。

> 定向给“validator group”/“mint pledger”的 Coinbase 输出属于 Coin 输出，
> 并会在满足 `06` §5.3 的可花费条件后进入 UTXO（29-confirmation 规则）。
> 三项 external-service Coinbase 输出在兑奖确认成功前 **DO NOT** 进入 UTXO；
> 见 §5.4 与 `07` §6。

### 1.3 Year of an entry（条目的年度）

每条 UTXO/UTCO 条目的“年度”等于它**所在交易**的年度（见 `03` §1.2，按交易时
间戳的 UTC 公历年）。该年度作为指纹树的顶层分级。


## 2. Fingerprint: 4-Layer Wide-Member Hash Tree（指纹：4 层宽成员哈希树）

### 2.1 Layer schema（分层结构）

```
Year  → [byte 8 of TxID]  → [byte 13 of TxID]  → [byte 18 of TxID]  → leaf
```

- 顶层按 `year(tx)` 分组；年是无界增长的稀疏维度，但粒度足够大。
- 中间三层各取交易 ID 的特定字节（`8`、`13`、`18`）作为分桶键，避开 ID 前缀
  以削弱“ID 塑造爱好者”的攻击影响。
- 末端为叶子节点（每条交易的可用输出聚合）。

> **跨树规则：** 中间分支节点哈希算法 = `BLAKE3-256`；叶子节点 = `SHA3-384`
> （见 `01` §1）。

### 2.2 Leaf payload（叶子载荷）

```text
Leaf = SHA3-384( TxID ‖ DataID ‖ FlagOutputs )
```

字段定义：

- `TxID` —— 该交易的 SHA3-384 ID。
- `DataID` —— 对该 TxID 下所有未花费/未转移输出项的“有效载荷数据”按
  `OutIndex` 升序序列化后的 SHA3-384 摘要。
- `FlagOutputs` —— 标记该交易各输出项是否仍处于可用状态：

  ```go
  type FlagOutputs struct {
      Count     uint8   // 标记位字节数，0..128
      FlagBytes []byte  // 每位对应一个输出项；1 = 可用，0 = 已无效
  }
  ```

  `Count == 0` 表示该叶子已无任何可用输出项，应当从树中移除。

### 2.3 Per-set roots（每个集合的根）

```text
UTXORoot = 基于 UTXO 条目构建的 4 层树根
UTCORoot = 基于 UTCO 条目构建的 4 层树根
```

二者作为 `CheckRoot` 的输入项之一（见 `02` §2）。

### 2.4 Why wide-member?（为何使用 wide-member）

每次“输出指引改变”（spend / transfer）只需重算一条从叶到根的路径；4 级分层
让每层的子节点数有限，重算量是 `O(1)`——这对每秒数十万级 IO 的校验组吞吐至关
重要。


## 3. Year as Top-Level Bucket（Year 作为顶层分桶）

`year(tx)` 取**交易时间戳的 UTC 公历年**（不是创世块年，也不是恒星年）。该定
义对所有 UTXO/UTCO 操作 **协议级强制**，否则不同节点会得到不同的指纹。

> **附：** 区块“年次”在 PoH 与铸币递减语境下采用恒星年（`87 661` 区块），
> 与本节“年度”语义不同；详见 `06` §6。请勿混淆。


## 4. UTXO/UTCO Cache (缓存集) for Output-Body Lookup（用于输出体查询）

指纹树的叶子节点仅记录“是否可用”。要从 UTXO/UTCO 检索输出项**数据本身**，
管理层会另维护一个并行结构——*UTXO/UTCO 缓存集*。

- 与指纹树同形（4 级分层）；
- 末端用 `map[uint64]Outputs` 存储该叶子分组下的可用输出项，键为
  `TxID[:8]` 的 `uint64` 值；
- 提供 O(1) 输出项数据读取；
- 不参与一致性哈希（仅本地缓存），可由其他公共服务（Blockqs）补足。

外部脚本引用缓存（参见 `05` §7、`07` §2.4）独立部署，但通常与 UTXO/UTCO 缓存
集同机运行。


## 5. Lifecycle Rules（生命周期规则）

### 5.1 Coin (UTXO)（Coin 生命周期）

| 事件                  | 处理                                                                                  |
|-----------------------|---------------------------------------------------------------------------------------|
| 交易确认 + 输出非销毁 | 加入 UTXO（按其年度分桶）                                                             |
| 输出被花费            | 从 UTXO 移除                                                                          |
| Coinbase 中“铸造者 / 铸凭者”输出 | 在第 29 个确认后才允许花费（`NewCoinSpendDelay = 29`）                       |
| 销毁标记输出          | 不进入 UTXO；接收地址 MAY 为 null                                                     |

### 5.2 Credit (UTCO)（Credit 生命周期）

| 事件                       | 处理                                                                       |
|----------------------------|----------------------------------------------------------------------------|
| 凭信新建 / 转移            | 输出按规则进入 UTCO（除非销毁、转移计次降至 0、或高度截止已过）              |
| 转移计次（`TransferRemaining`）从 N → N-1 | 每次转移递减；输入为 1 时输出为 0，则该输出**不**进入 UTCO         |
| 高度截止到达               | 高度变化时按“`高度=>[Index]`”索引集快速清理；维持 UTCO 指纹一致              |
| 11 年活跃保证              | 无截止凭信至少每 11 年激活一次（自转），否则失效；按高度差强制              |
| 可修改性                   | 单向退化：可改 → 不可改，反之不可                                            |
| 是否修改                   | 若上一态可改且本次确实修改，置位“是否修改”标志位                            |
| 创建者引用                 | 转移时 `CreatorOrRef` 不可变（指向原始凭信）                                |

### 5.3 Activation index（激活索引）

实现 SHOULD 维护一个 `高度 ⇒ [叶子索引]` 的辅助索引，用于：

- 高度截止到期的快速批量清理；
- 11 年活跃保证的滚动检查（每年的对应高度作为窗口）。

辅助索引可由 Blockqs 提供，也可由节点首次同步 UTCO 后做一次全量扫描自建；
其后每个新区块只做增量更新。

### 5.4 Coinbase external-service outputs（Coinbase 外部服务输出）

Blockqs / Depots / STUN 三项输出在被铸造当下 *不进入* UTXO；要等到 `07` §6
描述的兑奖确认机制满足“≥ 2 个后续铸造者认可”后才被纳入；超过 48 区块仍未达成
的部分会被回收（在第 49 号区块的 Coinbase 中作为 `Income.AwardWithheld` 重新计
入）。


## 6. Reverse-Derivation Chain Binding（逆向推导链式绑定）

设当前区块为 `H`，其当前 UTXO 集为 `U_H`（即“应用 `H` 区块交易**之前**”的状
态，也就是 `H-1` 的 UTXO 结果集）。

> 名词：“**当前 UTXO 集**” = 当前区块所依据的 UTXO 集；“**结果 UTXO 集**” =
> 应用本区块交易后的 UTXO 集，即下一个区块的当前 UTXO 集。

### 6.1 Forward update（正向更新）

```text
U_{H}.result = (U_H.current  ∖  spent_inputs(B_H))
                            ∪  new_outputs(B_H)
                            ∪  coinbase_unlocked_outputs(B_H)
```

`coinbase_unlocked_outputs` 是 29 个区块前的 Coinbase 中已可解锁的输出。

### 6.2 Reverse derivation（逆向推导）

```text
U_{H-1}.current = (U_H.current  ∖  new_outputs(B_{H-1})
                                ∖  coinbase_unlocked_outputs(B_{H-1}))
                                ∪  spent_inputs(B_{H-1})

assert  fingerprint(U_{H-1}.current) == B_{H-1}.UTXORoot   // CheckRoot 的输入项之一
```

逐步迭代可一直回溯到创世块。这种链式约束让 UTXO 指纹与 BlockID 链共同锁定历
史，形成三路（UTXO 花费 + UTCO 转移 + BlockID 链）耦合保护。

### 6.3 UTCO（UTCO 逆推要点）

UTCO 同理；增量更新时需额外考虑：

- 凭信转移计次的递减；
- “可修改 / 是否修改”位的合规性；
- 高度截止 / 11 年活跃保证带来的失效条目。


## 7. Public-API Surface (Layer 2)（第 2 层公共 API 接口）

```go
package utxo  // utco 同型

// Set 表示 UTXO/UTCO 的本地视图（含指纹树 + 缓存集）。
type Set interface {
    // Root 返回当前指纹根。
    Root() [48]byte

    // Lookup 按交易 ID 与输出序位查询输出项数据；缓存命中走本地，否则触发
    // Blockqs 检索（异步）。
    Lookup(txID [48]byte, out int) (*Output, error)

    // Apply 应用一个区块的状态变更（正向）。
    Apply(b *Block) error

    // Rewind 回滚一个区块（用于分叉切换 / 逆推验证）。
    Rewind(b *Block) error

    // YearStream 按年度分桶遍历，主要用于初始同步。
    YearStream(year uint32) iter.Seq[*Output]
}
```

`Block`、`Output` 类型来自 `02`、`03`。同一接口模板由 `internal/utxo` 与
`internal/utco` 两个包分别实现。
