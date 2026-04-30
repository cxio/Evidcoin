# 00 — Overview (总论)

> 状态：Proposal（Tier 2）。来源于 `docs/conception/*` 与经授权用户澄清。
> 粒度：中等粒度的架构描述。

本文档是 Evidcoin（证信链）开发提案集合的入口。它定义术语、总体架构、分层、
提案间依赖关系，以及构想文档到提案文档的映射。所有技术细节均下放到下文列出的
专门提案中；本总览仅在词汇定义与跨文档边界方面具有规范性。


## 1. Project Identity（项目标识）

- **Protocol-ID**：`Evidcoin@V1`
- **Chain-ID**：`mainnet`（规范的生产链；测试网 MUST 使用不同标识符）
- **Genesis-ID**：创世区块头的哈希（在客户端中硬编码）
- **Bound-ID**：可选，取值为相对于交易时间戳的 `BlockID[height-29][:20]`；用于
  在分叉竞争结束后，将交易绑定到某一条分叉上。

以上四个标识会按顺序拼接，并前置到每一条已签名交易消息之前：

```
MixData  = Protocol-ID || Chain-ID || Genesis-ID || Bound-ID || TxMSG
SignData = Sign(MixData)
```

同一组标识也用于 P2P 握手，用于声明节点所属链。


## 2. Core Goals（核心目标）

1. **General-purpose credit substrate.** 链承载三种信元（credit units）：可分割的
   *Coin*、不可分割且可转让的 *Credit*，以及不可分割且不可转让的 *Proof*。
2. **Proof of Historical (PoH).** 出块铸币权由历史上低可篡改性的交易 ID 哈希与
   动态参考区块共同决定，而不是由算力或财富决定。
3. **Post-quantum readiness.** 通过审慎设计的哈希算法矩阵（SHA3-384 / BLAKE3 /
   SHA3-512 / BLAKE2b-512 / ML-DSA-65）保障长期完整性，使链在后量子时代仍然可靠。
4. **Light client friendliness.** 区块头为 112 bytes，且链可通过年度检查点区块进行
   导航，因此任意节点都可在有界存储下从 genesis 验证整条链。
5. **Cooperative validation.** 重型校验工作委托给 *validator groups*（校验组），
   使低性能节点无需运行完整验证器也能参与铸币。


## 3. Architectural Layering（架构分层）

实现 MUST 遵守严格的单向依赖顺序（高层可 import 低层；反向依赖被禁止）。

| Layer | Concern                          | Go package(s)                                          |
|-------|----------------------------------|--------------------------------------------------------|
| 0     | 基础类型与密码学                 | `pkg/types/`, `pkg/crypto/`                            |
| 1     | 区块与交易核心                   | `internal/blockchain/`, `internal/tx/`                 |
| 2     | 状态集合                         | `internal/utxo/`, `internal/utco/`                     |
| 3     | 脚本引擎                         | `internal/script/`                                     |
| 4     | 共识（PoH）                      | `internal/consensus/`                                  |
| 5     | 集成 / 校验组                    | `cmd/evidcoin/`, `internal/group/`, `test/`           |

外部服务（Depots、Blockqs、STUN、basenet）均为进程外组件，仅建模为
**interface contracts** —— 详见提案 `07`。


## 4. Proposal Set & Dependencies（提案集合与依赖）

```
00-Overview                  (this file)
 │
 ├── 01-Cryptography-Primitives
 │     │
 │     ├── 02-Block-and-Chain
 │     │     │
 │     │     ├── 03-Transaction-Model
 │     │     │     │
 │     │     │     ├── 04-State-UTXO-UTCO
 │     │     │     │     │
 │     │     │     │     └── 05-Script-Engine
 │     │     │     │           │
 │     │     │     │           └── 06-Consensus-PoH
 │     │     │     │                 │
 │     │     │     │                 └── 07-Validator-Group
```

每份提案都会引用其来源的 conception 文档；权威映射表维护在 `docs/AGENTS.md`。

脚本 *instruction set* 提案位于 `docs/proposal/Instruction/`，属于独立的一批工作，
本轮不产出。


## 5. Glossary (Authoritative)（术语表，权威）

以下术语在所有提案中必须保持一致，不允许偏离。

| Term            | Meaning                                                                                  |
|-----------------|------------------------------------------------------------------------------------------|
| *信元* (credit unit) | {Coin, Credit, Proof} 三者之一                                                      |
| *Coin*          | 可分割的货币价值，花费前保存在 UTXO 中                                                   |
| *Credit*        | 不可分割、可转让的凭证，转让前保存在 UTCO 中                                             |
| *Proof*         | 不可分割、不可转让的存在记录，永不进入 UTCO/UTXO                                         |
| *UTXO*          | Unspent Transaction Output 集合（Coins）                                                 |
| *UTCO*          | UnTransferred Credit Output 集合（Credits）                                              |
| *TxID*          | `SHA3-384(TxHeader)`                                                                     |
| *BlockID*       | `SHA3-384(BlockHeader)`                                                                  |
| *CheckRoot*     | `SHA3-384(TreeRoot ‖ UTXORoot ‖ UTCORoot)`                                              |
| *YearBlock*     | 高度 H 且满足 `H % 87661 == 0` 的区块；引用上一年块                                       |
| *Stakes*        | 已消耗 coin-age 的 "Coin × Hour"，以 聪时（sat-hours）计量                               |
| *Mint Pledge Tx* (铸凭交易) | 使其首输入接收者获得铸币资格的历史交易                                        |
| *Mint Pledge Hash* (铸凭哈希) | `BLAKE3-512(Sign(Hash(TxID ‖ refMintHash ‖ X)))` —— 竞争值                  |
| *Reference Block* (评参区块) | 高度为 `current-9` 的区块，为 PoH 提供 ref-mint-hash                        |
| *Selection Pool* (择优池) | 面向未来区块的最佳 20 个候选集合                                                   |
| *Validator Group* (校验组) | 由节点集合（manager / guardian / checker）组成，共同完成区块校验              |
| *Lead Input* (首领输入) | 一笔交易的第一输入；MUST 为 stakes 销毁量最高的 Coin 支出                            |
| *Coinbase*      | 每个区块索引 0 处的唯一交易，且无输入来源                                                 |


## 6. Cardinal Constants（核心常量）

这些常量在协议层固定。任何变更都需要 hard upgrade。

| Constant                  | Value                       | Source                       |
|---------------------------|-----------------------------|------------------------------|
| `BlockInterval`           | 6 分钟                      | conception/2                 |
| `BlocksPerYear`           | 87 661（恒星年）            | conception/4                 |
| `BlocksPerMonth`          | 7 305（≈ 87661/12）         | conception/6                 |
| `MintPledgeRange`         | `[-80 000, -28]` blocks     | conception/1                 |
| `RefBlockOffset`          | `-9`                        | conception/1                 |
| `StakesRefOffset`         | `-27`                       | conception/1                 |
| `MintPledgePoolSize`      | 20 个候选                   | conception/1                 |
| `SyncAuthorisedTail`      | 15（位置 6–20）             | conception/1                 |
| `MinterStakeFloor`        | 4 000 000 sat-hours         | conception/1                 |
| `ForkCompeteWindow`       | 29 blocks                   | conception/2                 |
| `ForkAdmissionMax`        | 20 blocks                   | conception/2                 |
| `NewCoinSpendDelay`       | 29 次确认                   | conception/2                 |
| `BlockBroadcastDelay`     | 30 s（slot 开始后）         | conception/2                 |
| `RedundantBroadcastGap`   | 15 s                        | conception/2                 |
| `TxExpiryWindow`          | 240 blocks（24 h）          | conception/2                 |
| `MinFeeWindow`            | 6 000 blocks                | conception/2                 |
| `MinFeeFraction`          | trailing avg 的 1/4         | conception/2                 |
| `AwardConfirmWindow`      | 48 blocks（≈4.8 h）         | conception/4                 |
| `AwardEarlyExitWindow`    | 29 blocks（2 次确认后）     | conception/4                 |
| `MaxStackHeight`          | 256                         | conception/6                 |
| `MaxStackItem`            | 1024 bytes                  | conception/6                 |
| `MaxLockScript`           | 1024 bytes                  | conception/6                 |
| `MaxUnlockScript`         | 4096 bytes                  | conception/6                 |
| `MaxTxSize`               | 65 535 bytes（不含 unlock） | conception/6                 |
| `MaxOutputs`              | 1024（1-byte varint × 8）   | conception/6                 |
| `EmbedMaxCount`           | 4                           | conception/6                 |
| `EmbedMaxDepth`           | 0                           | conception/6                 |
| `GotoMaxCount`            | 2（top script）             | conception/6                 |
| `GotoMaxDepth`            | 3                           | conception/6                 |
| `BlackListFreeze`         | 24 h                        | conception/校验               |


## 7. Coinbase Reward Split (Authoritative)（Coinbase 奖励分配，权威）

每笔 Coinbase 交易都包含 5 类受益方的奖励份额。验证器 MUST 通过检查
output-config bytes 来强制执行各受益方分配比例：

| Beneficiary  | Share | Output-config low-4-bit value |
|--------------|-------|-------------------------------|
| Mint Pledger (铸凭者) | 10 % | 1 |
| Validator Group (校验组) | 40 % | 2 |
| Blockqs      | 20 % | 3 |
| Depots       | 20 % | 4 |
| STUN service | 10 % | 5 |

整数除法采用截断（`x*pct/100`）；余数归属于迭代顺序中的最后一个受益方。
三项外部服务份额（Blockqs、Depots、STUN）受提案 `07` 所述
**redemption mechanism** 约束。


## 8. Out-of-Scope（范围外）

以下事项被明确设定为*不*在本提案集覆盖范围内，交由外部项目或后续开发周期处理：

- P2P 节点发现与覆盖网络路由（`cxio/p2p`）。
- NAT 穿透（`cxio/stun2p`）。
- 交易体与附件的长期归档（`cxio/depots`、`cxio/blockqs`）。
- 钱包 UX、移动客户端、浏览器扩展。
- 区块浏览器 GUI。
- 按指令粒度的脚本 opcode 语义——在 `docs/proposal/Instruction/` 中单独覆盖。

## 9. Conformance（一致性）

声称符合 Evidcoin@V1 的实现 MUST：

1. 按 `02` 的规定精确编码区块头并计算 BlockID。
2. 按 `03` 的规定精确验证交易，包括签名语义与 Coinbase 奖励分配。
3. 按 `04` 的规定精确维护 UTXO/UTCO 指纹与 CheckRoot。
4. 按 `05` 的规定强制执行全部脚本资源限制，以及 unlock script 的
   `[0–50]+SYS_NULL` opcode 子集。
5. 拒绝任何违反 `06` 中 PoH 择优池或分叉决议规则的区块。
6. 在节点参与校验组时，识别 `07` 中定义的校验组协同消息。

凡未明确规定之处，均属于 implementation-defined；实现者 SHOULD 提交 issue，
而非静默产生分歧。
