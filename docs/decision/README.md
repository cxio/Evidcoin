# Decision Index（架构决策索引）

`docs/decision/` 只记录 `docs/conception/` 尚未明确、但会影响跨实现一致性或实现路径的补充决策。`docs/conception/` 是唯一上游正式依据；若 Decision 与 conception 冲突，以 conception 为准并修订 Decision。

本目录在 2026-05-14 采用 C 方案全面重整。旧 DEC 编号不再具有语义，只在迁移索引中保留去向。

## Current Index（当前索引）

| DEC | Title | Status | 主题 |
|-----|-------|--------|------|
| [DEC-0001](DEC-0001-canonical-varint.md) | Canonical Varint | Accepted | 基础编码 |
| [DEC-0002](DEC-0002-domain-tags.md) | Domain Tags | Accepted | 基础编码 |
| [DEC-0003](DEC-0003-field-widths.md) | Field Widths | Proposed | 基础编码 |
| [DEC-0004](DEC-0004-hash-tree-edge-cases.md) | Hash Tree Edge Cases | Proposed | 哈希树 |
| [DEC-0005](DEC-0005-address-encoding.md) | Address Encoding | Proposed | 地址 |
| [DEC-0006](DEC-0006-ml-dsa-65-profile.md) | ML-DSA-65 Profile | Proposed | 密码学 |
| [DEC-0007](DEC-0007-transaction-body-canonical-encoding.md) | Transaction Body Canonical Encoding | Proposed | 交易 |
| [DEC-0008](DEC-0008-witness-encoding-and-pruning.md) | Witness Encoding and Pruning | Proposed | 见证 |
| [DEC-0009](DEC-0009-signature-message-encoding.md) | Signature Message Encoding | Proposed | 签名 |
| [DEC-0010](DEC-0010-utxo-utco-fingerprint-payload.md) | UTXO/UTCO Fingerprint Payload | Proposed | 状态 |
| [DEC-0011](DEC-0011-poh-parameters-and-collision.md) | PoH Parameters and Collision | Accepted | PoH |
| [DEC-0012](DEC-0012-poh-timestamp-and-stakes.md) | PoH Timestamp and Stakes | Accepted | PoH |
| [DEC-0013](DEC-0013-genesis-initial-boundary.md) | Genesis Initial Boundary | Proposed | PoH |
| [DEC-0014](DEC-0014-block-competition-rules.md) | Block Competition Rules | Accepted | 区块竞争 |
| [DEC-0015](DEC-0015-fork-tiebreaker.md) | Fork Tiebreaker | Proposed | 分叉 |
| [DEC-0016](DEC-0016-reward-rounding.md) | Reward Rounding | Accepted | 激励 |
| [DEC-0017](DEC-0017-coinbase-hash-inputs.md) | Coinbase Hash Inputs | Proposed | Coinbase |
| [DEC-0018](DEC-0018-coinbase-serialization-and-award-slots.md) | Coinbase Serialization and Award Slots | Proposed | Coinbase |
| [DEC-0019](DEC-0019-issuance-schedule.md) | Issuance Schedule | Accepted | 发行 |
| [DEC-0020](DEC-0020-public-service-activation-boundary.md) | Public Service Activation Boundary | Accepted | 公共服务 |
| [DEC-0021](DEC-0021-announcement-trust-chain.md) | Announcement Trust Chain | Proposed | 通告 |
| [DEC-0022](DEC-0022-script-float-determinism.md) | Script Float Determinism | Accepted | 脚本 |
| [DEC-0023](DEC-0023-script-canonical-byte-encoding.md) | Script Canonical Byte Encoding | Proposed | 脚本 |
| [DEC-0024](DEC-0024-script-environment-registry.md) | Script Environment Registry | Proposed | 脚本 |
| [DEC-0025](DEC-0025-script-cost-budget.md) | Script Cost Budget | Proposed | 脚本 |
| [DEC-0026](DEC-0026-script-float-derived-semantics.md) | Script Float Derived Semantics | Proposed | 脚本 |
| [DEC-0027](DEC-0027-block-proof-package.md) | Block Proof Package | Proposed | 区块证明 |
| [DEC-0028](DEC-0028-summary-txid-and-network-proof-formats.md) | Summary TxID and Network Proof Formats | Proposed | 网络证明 |
| [DEC-0029](DEC-0029-blockqs-verification-data-format.md) | Blockqs Verification Data Format | Proposed | Blockqs |

## Status Summary（状态统计）

| Status | Count |
|--------|-------|
| Accepted | 9 |
| Proposed | 20 |
| Deprecated | 0 |
| Superseded | 0 |
| Absorbed | 0 |

## Migration Index（迁移索引）

| 旧 DEC | 去向 | 说明 |
|--------|------|------|
| old DEC-0001 varint | [DEC-0001](DEC-0001-canonical-varint.md) | 重写为规范 varint。 |
| old DEC-0002 domain tag | [DEC-0002](DEC-0002-domain-tags.md) | 重写域标签格式和用途清单。 |
| old DEC-0003 field widths | [DEC-0003](DEC-0003-field-widths.md) | 保留为 Proposed，修正交易版本宽度冲突。 |
| old DEC-0004 PoH parameters | [DEC-0011](DEC-0011-poh-parameters-and-collision.md) | 主体由 conception 吸收，仅保留碰撞排序。 |
| old DEC-0005 timestamp | [DEC-0012](DEC-0012-poh-timestamp-and-stakes.md) | 与 Stakes 一起整理。 |
| old DEC-0006 Stakes | [DEC-0012](DEC-0012-poh-timestamp-and-stakes.md) | 保留实现边界。 |
| old DEC-0007 hash tree | [DEC-0004](DEC-0004-hash-tree-edge-cases.md) | 修正不能误套交易输入 `RestHash` 的边界。 |
| old DEC-0008 ML-DSA | [DEC-0006](DEC-0006-ml-dsa-65-profile.md) | 改为 Proposed 配置。 |
| old DEC-0009 address | [DEC-0005](DEC-0005-address-encoding.md) | 记录 `FN_ADDRESS` 两次哈希与地址附录差异。 |
| old DEC-0010 Coin/chx ratio | 吸收/开放问题 | conception 未固定比例，不再单列；见 DEC-0019 开放问题。 |
| old DEC-0011 reward rounding | [DEC-0016](DEC-0016-reward-rounding.md) | 按 Coinbase 输出编号重写。 |
| old DEC-0012 Coinbase HashInputs | [DEC-0017](DEC-0017-coinbase-hash-inputs.md) | 保留为 Proposed。 |
| old DEC-0013 leader blacklist | 吸收/删除 | conception 已由择优池同步和分叉竞争处理，不再单列。 |
| old DEC-0014 announcement | [DEC-0021](DEC-0021-announcement-trust-chain.md) | 删除无依据初始根，改为 Proposed。 |
| old DEC-0015 genesis year boundary | [DEC-0013](DEC-0013-genesis-initial-boundary.md) | 合并到创世初段边界。 |
| old DEC-0016 short reference collision | 吸收/删除 | conception 已明确短引用碰撞按排序首个匹配。 |
| old DEC-0017 script float | [DEC-0022](DEC-0022-script-float-determinism.md) | 降级为确定性边界补充。 |
| old DEC-0018 transaction expiry | 吸收/删除 | conception 已明确交易 24 小时、240 区块过期。 |
| old DEC-0019 signature message | [DEC-0009](DEC-0009-signature-message-encoding.md) | 重写为 Proposed，避免签 TxID 破坏选择性授权。 |
| old DEC-0020 Coinbase slots | [DEC-0018](DEC-0018-coinbase-serialization-and-award-slots.md) | 修正 31 区块安全边界，统一服务命名。 |
| old DEC-0021 issuance | [DEC-0019](DEC-0019-issuance-schedule.md) | 保留整数边界，比例问题开放。 |
| old DEC-0022 credit expiry | 吸收/删除 | conception 已明确 Credit 31 年过期。 |
| old DEC-0023 script budget | [DEC-0025](DEC-0025-script-cost-budget.md) | 降级为预算框架补充。 |
| old DEC-0024 float derived | [DEC-0026](DEC-0026-script-float-derived-semantics.md) | 修正与转换指令冲突。 |
| old DEC-0025 block proof | [DEC-0027](DEC-0027-block-proof-package.md), [DEC-0028](DEC-0028-summary-txid-and-network-proof-formats.md), [DEC-0029](DEC-0029-blockqs-verification-data-format.md) | 拆分为证明包、网络摘要和 Blockqs 数据格式。 |
| old DEC-0026 fork tiebreak | [DEC-0015](DEC-0015-fork-tiebreaker.md) | 修正为 31 区块口径，HashX 保持 Proposed。 |

## Absorbed Or Not Separately Listed（已被 conception 吸收或不再单列）

- Coinbase 主体信息、输出目标和公共服务奖励比例已由 `docs/conception/附.交易.md` 与 `docs/conception/4.激励机制.md` 明确；Decision 只补充编码和兑奖槽边界。
- 公共服务兑奖窗口为后续 48 个区块，满足 31 个区块安全边界且达到确认数后可兑奖；不再保留旧 35 区块口径。
- 交易短引用碰撞规则已由 `docs/conception/附.交易.md` 明确为同年度末端集合按完整 TxID 排序后首个匹配。
- Credit 31 年过期已由 `docs/conception/5.信用结构.md` 明确。
- 脚本 `Float=float64`、转换主体、运算主体和安全框架已由 `docs/conception/Instruction/*.md` 与 `docs/conception/6.脚本系统.md` 明确；Decision 只补充跨实现确定性。
- 交易 24 小时或 240 区块过期已由 `docs/conception/2.共识-端点约定.md` 明确。
- PoH 参数、择优池容量和同步授权成员已由 `docs/conception/1.共识-历史证明（PoH）.md` 明确；Decision 只补充碰撞排序。
- 区块竞争中 3 倍 Stakes 规则和同一铸造者低收益胜出规则已由 conception 明确；Decision 只补充比较边界。

## Open Questions（开放问题）

- 交易头 `Version` 最终使用 `uint16` 还是更宽字段；旧 `uint32` 签名消息写法已移除。
- 签名消息的选择性授权编码仍需作者裁决，尤其是是否包含时间戳、未授权字段是否省略、当前输入解锁脚本是否纳入。
- 地址校验码的一次哈希/两次哈希存在 conception 内部差异：`附.交易.md` 只说执行哈希运算，`FN_ADDRESS` 写明执行 2 次哈希。
- `FN_PUBHASH`、多签地址构造与 `FN_ADDRESS` 的简单哈希/双哈希输入边界需统一。
- 区块交易叶子的 3 字节序号与通用哈希树 `leafIndex` 的关系需冻结。
- UTXO/UTCO 指纹中的状态位顺序、空分组哈希和 `DataID` 是否包含已无效输出需裁决。
- 第 240 块之前 Coinbase 是否可作为铸凭交易参与竞争需裁决。
- 创世块时间戳、初始输出、创世/启动期 Coinbase 单输出归属和公告根公钥集合需由创世规范定义。
- 平局裁决中的 `HashX256` 具体算法未定义。
- 1 币等于多少 `chx` 尚无 conception 明确依据。
- Coinbase 中 `mint_proof` 字段顺序、兑奖槽位位序和未确认截留回收编码位置需冻结。
- 脚本 `BigInt` 字节编码、环境条目标识数值、opcode 成本表和区块总成本函数需裁决。
- conception 中 `BOOL` 对极小 Float 的判断使用 `x <= math.SmallestNonzeroFloat64`，可能把负数判为 false，建议作者复核。
- conception 中主区块定义在 `2.共识-端点约定.md` 与 `附.组队校验.md` 表述不同：前者指竞争中权重相对最高区块，后者指择优池最优候选者区块，建议统一。
- conception 中交易输入哈希公式在 `附.交易.md` 与 `blockchain.md` 的摘要表述不同，建议统一是否包含 `LeadPKHash`。
- conception 中脚本比较/工具指令命名存在 `CMPFLO`，若其他材料出现不同拼写，应统一为指令集中的正式名称。

## Maintenance Rules（维护规则）

- 新增 Decision 前必须先检查 `docs/conception/` 是否已经明确该规则。
- 若 conception 已明确，只在 README 吸收清单中记录，不新增 DEC。
- 若无法从 conception 直接裁定，状态必须为 `Proposed`，并清楚列出待裁决参数。
- 若后续 conception 吸收某 DEC，应删除该 DEC 或将其状态迁移记录到 README。
