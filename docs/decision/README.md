# Decision Index（架构决策索引）

`docs/decision/` 只记录 `docs/conception/` 尚未明确、但会影响跨实现一致性或实现路径的补充决策。`docs/conception/` 是唯一上游正式依据；若 Decision 与 conception 冲突，以 conception 为准并修订 Decision。

本目录已按 2026-05 的 conception 修订重新整理：取消 `DataID`、取消全网通告、Coinbase 省略 `HashInputs`、PoH `Stakes` 改取 `-32` 区块、地址校验采用双 `SHA2-256`、发行单位固定为 `1 币 = 10^8 chx`。

构想层自身疑似矛盾集中记录在 [CONCEPTION-CONFLICTS.md](CONCEPTION-CONFLICTS.md)。这些问题需要作者修正，Decision 不代替 conception 做最终裁决。

## Current Index（当前索引）

| DEC | Title | Status | 主题 |
|-----|-------|--------|------|
| [DEC-0001](DEC-0001-canonical-varint.md) | Canonical Varint | Accepted | 基础编码 |
| [DEC-0002](DEC-0002-domain-tags.md) | Domain Tags | Accepted | 基础编码 |
| [DEC-0003](DEC-0003-field-widths.md) | Field Widths | Proposed | 基础编码 |
| [DEC-0004](DEC-0004-hash-tree-edge-cases.md) | Hash Tree Edge Cases | Proposed | 哈希树 |
| [DEC-0005](DEC-0005-address-encoding.md) | Address Encoding | Accepted | 地址 |
| [DEC-0006](DEC-0006-ml-dsa-65-profile.md) | ML-DSA-65 Profile | Proposed | 密码学 |
| [DEC-0007](DEC-0007-transaction-body-canonical-encoding.md) | Transaction Body Canonical Encoding | Proposed | 交易 |
| [DEC-0008](DEC-0008-witness-encoding-and-pruning.md) | Witness Encoding and Pruning | Proposed | 见证 |
| [DEC-0009](DEC-0009-signature-message-encoding.md) | Signature Message Encoding | Proposed | 签名 |
| [DEC-0010](DEC-0010-utxo-utco-fingerprint-payload.md) | UTXO/UTCO Fingerprint Payload | Proposed | 状态 |
| [DEC-0011](DEC-0011-poh-parameters-and-collision.md) | PoH Mint Profile | Proposed | PoH |
| [DEC-0012](DEC-0012-poh-timestamp-and-stakes.md) | PoH Timestamp and Stakes | Accepted | PoH |
| [DEC-0013](DEC-0013-genesis-initial-boundary.md) | Genesis Initial Boundary | Proposed | 创世初段 |
| [DEC-0014](DEC-0014-block-competition-rules.md) | Block Competition Rules | Accepted | 区块竞争 |
| [DEC-0015](DEC-0015-fork-tiebreaker.md) | Fork Tiebreaker | Proposed | 分叉 |
| [DEC-0016](DEC-0016-reward-rounding.md) | Reward Rounding | Accepted | 激励 |
| [DEC-0017](DEC-0017-coinbase-hash-inputs.md) | Coinbase HashInputs Removed | Deprecated | Coinbase |
| [DEC-0018](DEC-0018-coinbase-serialization-and-award-slots.md) | Coinbase Serialization and Award Slots | Proposed | Coinbase |
| [DEC-0019](DEC-0019-issuance-schedule.md) | Issuance Schedule | Accepted | 发行 |
| [DEC-0020](DEC-0020-public-service-activation-boundary.md) | Public Service Activation Boundary | Accepted | 公共服务 |
| [DEC-0021](DEC-0021-announcement-trust-chain.md) | Announcement Trust Chain Removed | Deprecated | 通告 |
| [DEC-0022](DEC-0022-script-float-determinism.md) | Script Float Determinism | Proposed | 脚本 |
| [DEC-0023](DEC-0023-script-canonical-byte-encoding.md) | Script Canonical Byte Encoding | Proposed | 脚本 |
| [DEC-0024](DEC-0024-script-environment-registry.md) | Script Registry Spaces | Proposed | 脚本 |
| [DEC-0025](DEC-0025-script-cost-budget.md) | Script Cost Budget | Proposed | 脚本 |
| [DEC-0026](DEC-0026-script-float-derived-semantics.md) | Script Float Derived Semantics | Proposed | 脚本 |
| [DEC-0027](DEC-0027-block-proof-package.md) | Block Proof Package | Proposed | 区块证明 |
| [DEC-0028](DEC-0028-summary-txid-and-network-proof-formats.md) | Summary TxID and Network Proof Formats | Proposed | 网络证明 |
| [DEC-0029](DEC-0029-blockqs-verification-data-format.md) | Blockqs Verification Data Format | Proposed | Blockqs |

## Topic Index（主题索引）

| 主题 | 决策 |
|------|------|
| 基础编码 | [DEC-0001](DEC-0001-canonical-varint.md), [DEC-0002](DEC-0002-domain-tags.md), [DEC-0003](DEC-0003-field-widths.md) |
| 哈希树与状态 | [DEC-0004](DEC-0004-hash-tree-edge-cases.md), [DEC-0010](DEC-0010-utxo-utco-fingerprint-payload.md) |
| 密码学与地址 | [DEC-0005](DEC-0005-address-encoding.md), [DEC-0006](DEC-0006-ml-dsa-65-profile.md) |
| 交易与签名 | [DEC-0007](DEC-0007-transaction-body-canonical-encoding.md), [DEC-0008](DEC-0008-witness-encoding-and-pruning.md), [DEC-0009](DEC-0009-signature-message-encoding.md) |
| PoH 与分叉 | [DEC-0011](DEC-0011-poh-parameters-and-collision.md), [DEC-0012](DEC-0012-poh-timestamp-and-stakes.md), [DEC-0013](DEC-0013-genesis-initial-boundary.md), [DEC-0014](DEC-0014-block-competition-rules.md), [DEC-0015](DEC-0015-fork-tiebreaker.md) |
| 激励与 Coinbase | [DEC-0016](DEC-0016-reward-rounding.md), [DEC-0018](DEC-0018-coinbase-serialization-and-award-slots.md), [DEC-0019](DEC-0019-issuance-schedule.md), [DEC-0020](DEC-0020-public-service-activation-boundary.md) |
| 脚本 | [DEC-0022](DEC-0022-script-float-determinism.md), [DEC-0023](DEC-0023-script-canonical-byte-encoding.md), [DEC-0024](DEC-0024-script-environment-registry.md), [DEC-0025](DEC-0025-script-cost-budget.md), [DEC-0026](DEC-0026-script-float-derived-semantics.md) |
| 证明与服务 | [DEC-0027](DEC-0027-block-proof-package.md), [DEC-0028](DEC-0028-summary-txid-and-network-proof-formats.md), [DEC-0029](DEC-0029-blockqs-verification-data-format.md) |
| 已移除 | [DEC-0017](DEC-0017-coinbase-hash-inputs.md), [DEC-0021](DEC-0021-announcement-trust-chain.md) |

## Status Summary（状态统计）

| Status | Count |
|--------|-------|
| Accepted | 8 |
| Proposed | 19 |
| Deprecated | 2 |
| Superseded | 0 |
| Absorbed | 0 |

## Deprecated Decisions（已废止决策）

| DEC | 原因 | 当前依据 |
|-----|------|----------|
| [DEC-0017](DEC-0017-coinbase-hash-inputs.md) | conception 明确 Coinbase 省略 `HashInputs`，不再使用占位输入哈希。 | DEC-0018 |
| [DEC-0021](DEC-0021-announcement-trust-chain.md) | conception 已取消全网通告设计。 | 无；如未来恢复需先修订 conception |

## Reconciled Conception Revisions（已对齐的构想修订）

- UTXO/UTCO 指纹改为 `Hash384(TxID || FlagOutputs)`，删除 `DataID` 相关规则和开放问题。
- Coinbase 省略 `HashInputs`，删除 `coinbase.inputs` 域标签作为有效用途。
- PoH 铸凭中的 `Stakes` 取链末端 `-32` 区块，评参区块仍为 `-8`。
- `MintPKHash` 作为可选铸造身份进入交易头；未设置时回退 `LeadPKHash`。
- 分叉平局算法名称从旧 `HashX256` 改为 conception 中的 `RandomX` profile。
- 地址校验码采用 `SHA2-256(SHA2-256(prefix || pubKeyHash))` 末尾 4 字节。
- 发行单位固定为 `1 币 = 10^8 chx`，递减精度为 `chx`。
- 百日前只关闭公共服务奖励，不再把 Coinbase 简化为固定单输出。
- 创世块、创世 Coinbase 和 #1/#2 启动逻辑已吸收进 DEC-0013。
- 脚本异常浮点检测指令统一为 `ISEFV`，旧 `ISNAN` 名称废弃。

## Absorbed Or Not Separately Listed（已被 conception 吸收或不再单列）

- Coinbase 主体信息、输出目标和公共服务奖励比例已由 `docs/conception/附.交易.md` 与 `docs/conception/4.激励机制.md` 明确；Decision 只补充编码和兑奖槽边界。
- 公共服务兑奖窗口为后续 48 个区块，满足 31 个区块安全边界且达到确认数后可兑奖；不再保留旧 35 区块口径。
- 交易短引用碰撞规则已由 `docs/conception/附.交易.md` 明确为同年度末端集合按完整 TxID 排序后首个匹配。
- Credit 31 年过期和一笔交易最多创建 2 笔 Credit 输出已由 `docs/conception/5.信用结构.md` 明确。
- 交易 24 小时或 240 区块过期已由 `docs/conception/2.共识-端点约定.md` 明确。
- PoH 参数、择优池容量和同步授权成员已由 `docs/conception/1.共识-历史证明（PoH）.md` 明确；Decision 只补充碰撞排序和字节级边界。
- 区块竞争中 3 倍 Stakes 规则和同一铸造者低收益胜出规则已由 conception 明确；Decision 只补充比较边界。
- 交易费 50% 销毁属于 conception 经济规则；Coinbase 收益总额只包含未销毁部分。
- 公共服务、Blockqs、Depots 和组队校验的大部分网络流程属于服务/实现层边界，未进入当前共识编码 Decision。

## Open Questions（开放问题）

- `TxHeader.Timestamp` 在 conception 中存在 `int64` 与 `uint64` 冲突；冻结 TxID 前需作者修正。
- `MintProof` 的精确字段顺序、铸凭哈希前像中的 `pubKey` 编码、`X = Bytes(timeStamp * Stakes * Mix)` 编码仍需冻结。
- 第 240 块之前 Coinbase 交易能否参与铸凭竞争仍需裁决。
- `YearBlock` 的非年块省略规则已建议采用，但仍依赖区块头最终编码确认。
- 区块交易叶子的 3 字节序号与通用哈希树 `leafIndex` 的关系需冻结。
- UTXO/UTCO 指纹中的状态位顺序和空分组哈希需裁决。
- 公共服务槽位内第 0 位对应 `H-1` 还是 `H-48`，以及第 49 块截留回收编码位置需冻结。
- RandomX 的具体参数、输出长度、版本和库 profile 需裁决。
- 脚本 `BigInt` 字节编码、`Rune` 字节序、环境/函数/模块注册表数值、opcode 成本表和区块总成本函数需裁决。
- 脚本公共验证路径中 `SYS_TIME`、`ENV{Timestamp}`、`SHELL`、`EXT_PRIV`、`INPUT` 的统一失败/忽略/终止语义需裁决。
- Blockqs 与 Depots 对完整区块、小附件、大附件索引和分片数据的职责边界需在服务接口阶段细化。

## Maintenance Rules（维护规则）

- 新增 Decision 前必须先检查 `docs/conception/` 是否已经明确该规则。
- 若 conception 已明确，只在 README 吸收清单中记录，不新增 DEC。
- 若无法从 conception 直接裁定，状态必须为 `Proposed`，并清楚列出待裁决参数。
- 若后续 conception 吸收某 DEC，应删除该 DEC，或将其状态迁移为 `Deprecated` 并在本索引说明原因。
- 若发现 conception 自身冲突，记录到 [CONCEPTION-CONFLICTS.md](CONCEPTION-CONFLICTS.md)，不要在 Decision 中伪造最终规则。
