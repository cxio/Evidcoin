# 02 — Block and Chain (区块与链)

> 来源：`conception/blockchain.md`，并吸收了
> `conception/2.共识-端点约定.md`（出块时序）与 `conception/附.组队校验.md`
>（区块发布阶段）中的少量交叉内容。

本提案定义区块头、CheckRoot、用于轻量链路导航的 year-block 机制、区块级交易
哈希树、初始链引导协议，以及分叉裁决相关元数据。它是首个依赖 `01`（密码学）
的提案。


## 1. Block Header (区块头)

### 1.1 Structure (结构)

```go
// 简化伪代码示意；序列化采用固定字段顺序的大端编码。
type BlockHeader struct {
    Version   uint32      // 协议版本号
    Height    uint64      // 区块高度（替代时间戳字段）
    PrevBlock [48]byte    // 前一区块 ID（SHA3-384）
    CheckRoot [48]byte    // 校验根，由交易树根 + UTXO/UTCO 指纹合并计算
    Stakes    uint64      // 币权销毁总值，单位：聪时（chx·hour）
    YearBlock [48]byte    // 仅当 Height % 87661 == 0 时有效，否则为 zero-bytes
}
```

**字段义释：**

- `Height` 充当时间表达：区块时间戳由
  `genesisTime + Height * 6min` 派生（毫秒精度）。
- `Stakes` 既是 PoH 评比因子（参见 `06`），也是分叉竞争中的“自私铸造”约束因子
  （参见 `06` §5.4）。
- `YearBlock` 仅在跨年点（`Height % 87 661 == 0`）携带前一年块的 BlockID，
  其余区块该字段全零；序列化时仍固定占用 48 字节以保持定长。

`Height = 0` 为创世块，其 `PrevBlock`、`YearBlock` 均为全零。

### 1.2 Block ID (区块 ID)

```text
BlockID = SHA3-384(Serialize(BlockHeader))
```

序列化规则为字段固定顺序、大端整数、定长字节序列。

### 1.3 Size budget (大小预算)

按 4-byte 整数估算（含 YearBlock 字段定长占位），区块头 = `4 + 8 + 48 + 48 +
8 + 48 = 164 字节`。
按 conception 的 112-byte 估算（约定 YearBlock 非固定存储）：
`112 × 87 661 ≈ 9.36 MB / year`。

> 实施时建议两种存储模式：**完整模式**（每块 164 B）与**省略模式**（仅保留年块衔
> 接，跨年区块完整，年内区块可丢弃）。详见 §5。


## 2. CheckRoot (校验根)

```text
CheckRoot = SHA3-384( TreeRoot ‖ UTXORoot ‖ UTCORoot )
```

- `TreeRoot` —— 区块内交易集合的哈希校验树根（见 §3）。
- `UTXORoot` / `UTCORoot` —— 当前 UTXO/UTCO 集合的指纹根（见 `04`）。

CheckRoot 是铸造者签名目标（`Sign(CheckRoot)` 出现在 Coinbase 中，参见 `03`）。
对区块内任意交易或 UTXO/UTCO 集的任何改动，都会导致 CheckRoot 不匹配。

注意：UTXO/UTCO 指纹基于 *当前* UTXO/UTCO 集计算（即应用本区块交易**之前**
的状态）。该定义为 §6 的逆推验证提供链式约束。


## 3. Block-Level Transaction Tree (区块级交易树)

### 3.1 Layout (布局)

区块内交易集合采用 **类 Merkle、含序、宽节点** 的二元哈希校验树。

- 叶子节点：`SHA3-384( seq[3] ‖ TxID )`，其中 `seq` 为 3 字节交易序号
  （从 0 起），用于支持快速定位与同步。
- 分支节点：`BLAKE3-256( left ‖ right )`。

> “叶 SHA3-384 / 枝 BLAKE3-256”是跨层级树的统一规则；与交易内部树（输入树、
> 输出树、附件片组树）统一使用 BLAKE3-256 的规则不同，参见 `03`。

### 3.2 Properties (性质)

- 节点序列以“链表 / 数组”形式组织，单笔交易可基于序号 + 路径独立验证。
- 序号区固定为 3 字节（≤ 16 777 215 笔交易/块），便于定长操作。

### 3.3 Coinbase 位置 (Coinbase Position)

按约定，Coinbase 交易必须位于序号 0（`seq = 0x000000`）。其叶子哈希同样按
`SHA3-384(0x000000 ‖ TxID_coinbase)` 计算。


## 4. Year-Block Mechanism (年块机制)

每 87 661 个区块（一个恒星年）形成一个 *年块*。年块的 `YearBlock` 字段保存
*前一年块的 BlockID*，从而构建一条 **稀疏年级链**（year-chain），其顶端为创世块。

这使节点可以：

1. 仅持有完整年块链（每年 1 块）即可声明对历史的承诺。
2. 在年区间内按需保留部分区块头（例如末端 29 块用于分叉判定，其余可丢弃）。
3. 在需要时通过 Blockqs 即时获取已丢弃区块头数据。

实施时，年块衔接连续性是 **强制要求**；客户端可丢弃年内区块头，但年块本身
不可缺失。


## 5. Storage Modes (存储模式)

下表列出官方支持的三种存储级别。任一级别都 MUST 保持年块衔接连续性。

| Mode      | Storage         | Adequacy                                               |
|-----------|-----------------|--------------------------------------------------------|
| Full      | 全量区块头链    | 校验组、Blockqs 等服务节点                             |
| Trim      | 末端 29 块 + 年块链 + 用户感兴趣年段 | 普通 Peer 默认值                       |
| Minimal   | 创始块 + 年块链 + 末端 29 块         | 极简轻客户端                              |

末端 29 块对应分叉竞争窗口（`06` §5）。任意 Mode 的节点都能完成新区块入链验证——
只要其 `PrevBlock` 与本地末端区块 ID 匹配。


## 6. Initial Chain Bootstrap (初始链引导)

新上线节点（本地无任何数据）按以下步骤初始化主链状态：

1. 读取硬编码的 *创世块*（区块头 + Coinbase 交易）。
2. 通过连接的 Blockqs / 校验组下载 *区块头链*，可按年块衔接方式压缩传输。
3. 下载 *末端部分区块*（默认 29 块）的 Coinbase 交易及其哈希校验树路径，包含
   UTXO/UTCO 指纹与铸造者对 CheckRoot 的签名。
4. 可选：下载当前 UTXO/UTCO 集 + 末端部分区块完整数据，用于 §7 所述逆推严格
   验证。

校验组通常缓存最近 240 个区块（约一天）的铸造者签名，步骤 4 的来源可补足该数据。

为对抗中间人，节点可向多个独立 Blockqs 节点 / 校验组并行获取上述数据，并逐项
比对以确认接入的是真实主链。


## 7. UTXO/UTCO Reverse-Derivation Verification (UTXO/UTCO 逆推验证)

详见 `04` §6。此处仅提示：当前 UTXO 集 + 上一区块交易 ⇒ 上一区块的“当前
UTXO 集”；其指纹应与该区块头的 UTXORoot 字段（即 CheckRoot 的输入项之一）匹
配。逐块逆推可一直追溯至创世块。

UTCO 同理（叠加凭信“转移计次”递减、可修改性等约束）。


## 8. Chain Identification Triple/Quadruple (链标识三元组/四元组)

| Field        | Type       | Purpose                                                       |
|--------------|------------|---------------------------------------------------------------|
| Protocol-ID  | string     | 区分本协议与其他链协议，例如 `Evidcoin@V1`                    |
| Chain-ID     | string     | 当前链运行态，例如 `mainnet`                                  |
| Genesis-ID   | `[48]byte` | 创世区块 ID                                                   |
| Bound-ID     | `[20]byte` | 主链绑定，可选；若设置则 `= BlockID[txTime.height-29][:20]` |

四元组用途：

1. 在 P2P 握手时声明节点所属链 / 模式。
2. 作为签名前缀（参见 `01` §5），杜绝跨链重放与分叉支链重放。

`Bound-ID` 为可选字段；不设置时，签名前缀中该字段以 20 字节零值占位。


## 9. Block Admission (Core Layer Behaviour) (区块接纳（核心层行为）)

`internal/blockchain` 的核心职责是接收外部组件提交的新区块，只执行**最小集**
合法性检查，*不*负责交易内部细节（由组队校验外部组件完成）。

最小集合法性检查清单：

1. `PrevBlock` 与本地末端区块 ID 一致；高度递增 1。
2. 区块头序列化字节长度合法。
3. 跨年点（`Height % 87 661 == 0`）必须携带正确 `YearBlock`；其余区块该字段
   必须为全零。
4. 时间戳由高度推导，无需校验时间戳字段（不存在该字段）。
5. CheckRoot 字段长度必须为 48 字节（本层不校验内容正确性）。
6. 同一高度发生二次提交：拒收并向调用者返回冲突错误，由提交者清理后再重提。

完整合法性（PoH 资格、交易合法性、UTXO/UTCO 指纹一致性等）由调用方（典型为
校验组管理层）保证。


## 10. Manual Fork Switching (手动分叉切换)

当全网出现 ≥ 2 小时分区（即分叉长度超过 20 区块）时，自动分叉竞争机制失效。
此时：

- 协议层将所有 ≤ 20 块分叉视为合法；超过 20 块的分叉不被算法接收。
- 客户端 SHOULD 提供 *手动选择主链* 入口：用户输入若干代表性节点地址，
  客户端从这些节点拉取分叉点之后的区块头链并作为本链。

这是社会性的“用脚投票”机制，而非协议级裁决。


## 11. Public-API Surface (Layer 1) (公共 API 面（第 1 层）)

`internal/blockchain` 对上层暴露的最小接口集：

```go
package blockchain

// Chain 管理本地区块头链。
type Chain interface {
    // Tip 返回末端区块头 ID 与高度。
    Tip() (id [48]byte, height uint64)

    // Submit 提交一个新区块（最小合法性检查）。
    Submit(b *Block) error

    // HeaderByHeight 按高度查询区块头；缺失时由 Blockqs 补足。
    HeaderByHeight(h uint64) (*BlockHeader, error)

    // HeaderByID 按 ID 查询区块头。
    HeaderByID(id [48]byte) (*BlockHeader, error)

    // YearChain 返回全部年块 ID 序列（从创世块开始）。
    YearChain() [][48]byte

    // SwitchTo 用于手动主链切换（§10）。
    SwitchTo(targetTip [48]byte, headers []BlockHeader) error
}
```

`Block` 仅是 `BlockHeader + 交易序列` 的聚合类型；交易类型定义见 `03`。
