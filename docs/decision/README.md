# Architecture Decisions（架构决策）

`decision/` 记录 Evidcoin 在 `conception/` 中尚未直接固定、但会影响跨实现一致性或后续实现路径的补充决策。

项目文档的正式结构收敛为两层：

| 层级 | 目录 | 作用 |
|------|------|------|
| Conception | `conception/` | 作者构想与协议设计来源，优先级最高。 |
| Decision | `decision/` | 仅补充 conception 未明确的架构与规范化决策。 |

## 维护原则

- 若某项规则已在 `conception/` 中明确，不再单独建立 Decision。
- Decision 不重复描述 conception 已固定的协议主体，只记录缺口、边界语义、编码细节或实现路径。
- 若 Decision 与 conception 冲突，以 conception 为准，并应修订、合并、吸收或删除对应 Decision。
- 本次重整已重新编号、合并和删除既有 DEC，使编号顺序与当前协议结构一致。
- `proposal/` 与 `plan/` 当前视为待重构材料，不作为本目录整理依据。

## 状态枚举

| 状态 | 含义 |
|------|------|
| Accepted | 已接受，作为当前补充决策有效。 |
| Proposed | 草案状态，尚未最终接受。 |
| Deprecated | 已废弃，不再作为当前实现依据。 |
| Superseded | 已被后续 Decision 替代。 |
| Absorbed | 已被 `conception/` 或其他 Decision 吸收，不再单列。 |

## 命名规范

```text
DEC-NNNN-<short-description>.md
```

- `NNNN`：四位数字编号，从 `0001` 开始。
- `<short-description>`：英文小写短描述，单词间使用连字符。

## 当前决策索引

本表列出当前保留的 Decision 文件，编号连续为 `DEC-0001` 到 `DEC-0026`。

| 新编号 | 标题 | 状态 | 来源 |
|--------|------|------|------|
| [DEC-0001](DEC-0001-varint-encoding.md) | 规范化无符号 Varint 编码 | Accepted | old DEC-0003 |
| [DEC-0002](DEC-0002-domain-tag-format.md) | 哈希域分隔标签格式 | Accepted | old DEC-0004 |
| [DEC-0003](DEC-0003-field-widths.md) | 区块头与交易头字段宽度 | Accepted | old DEC-0017 |
| [DEC-0004](DEC-0004-poh-parameters-and-collision.md) | PoH 参数编码与碰撞处理 | Accepted | old DEC-0002 + old DEC-0008；旧 PoH 算法主体已被 conception/ 吸收 |
| [DEC-0005](DEC-0005-poh-timestamp-derivation.md) | PoH 时间戳的推导与隔离 | Accepted | old DEC-0019 |
| [DEC-0006](DEC-0006-stakes-definition.md) | Stakes 精确定义 | Accepted | old DEC-0016；PoH 使用评参区块自身 Stakes |
| [DEC-0007](DEC-0007-hash-tree-edge-cases.md) | 哈希树边界情况 | Accepted | old DEC-0009 |
| [DEC-0008](DEC-0008-ml-dsa-65-integration.md) | ML-DSA-65 集成路径 | Accepted | old DEC-0010 |
| [DEC-0009](DEC-0009-address-text-encoding.md) | 地址文本编码 | Accepted | old DEC-0011 |
| [DEC-0010](DEC-0010-coin-chx-ratio.md) | Coin 与 chx 换算 | Accepted | old DEC-0015 |
| [DEC-0011](DEC-0011-reward-rounding.md) | 奖励与交易费余数归属 | Accepted | old DEC-0006 |
| [DEC-0012](DEC-0012-coinbase-hashinputs.md) | Coinbase HashInputs 计算 | Accepted | old DEC-0007 |
| [DEC-0013](DEC-0013-leader-blacklist-convention.md) | 首领黑名单层级 | Accepted | old DEC-0012 |
| [DEC-0014](DEC-0014-announcement-trust-chain.md) | 全网通告授权信任链 | Accepted | old DEC-0013 + old DEC-0018 |
| [DEC-0015](DEC-0015-genesis-year-boundary.md) | 创世高度年度边界 | Accepted | old DEC-0014 |
| [DEC-0016](DEC-0016-short-reference-collision-rule.md) | 短引用歧义的协议级处理 | Accepted | old DEC-0020；按最新 conception 修正 |
| [DEC-0017](DEC-0017-script-float-determinism.md) | 脚本 VM Float 确定性 | Accepted | old DEC-0005 |
| [DEC-0018](DEC-0018-transaction-expiry-boundary.md) | 交易过期边界语义 | Accepted | 高优先级第二阶段；补充交易过期高度边界 |
| [DEC-0019](DEC-0019-signature-message-encoding.md) | 签名消息规范化编码 | Proposed | 高优先级第二阶段；待冻结签名消息字节级编码 |
| [DEC-0020](DEC-0020-coinbase-reward-slots.md) | Coinbase 完整编码与公共服务兑奖槽 | Proposed | 高优先级第二阶段；待冻结 Coinbase 字段结构 |
| [DEC-0021](DEC-0021-issuance-schedule-rounding.md) | 原始铸币发行曲线高度边界与取整 | Accepted | 高优先级第二阶段；补充发行曲线高度与取整 |
| [DEC-0022](DEC-0022-credit-expiry-boundary.md) | Credit 31 年过期边界 | Accepted | 高优先级第二阶段；补充 UTCO 过期边界 |
| [DEC-0023](DEC-0023-script-cost-budget.md) | 脚本成本预算 | Proposed | 高优先级第二阶段；待冻结预算数值和成本表 |
| [DEC-0024](DEC-0024-script-float-derived-semantics.md) | 脚本 Float 派生语义 | Proposed | 高优先级第二阶段；待冻结 Float 派生规则 |
| [DEC-0025](DEC-0025-block-proof-and-summary.md) | 区块证明包与概要短 TxID | Proposed | 高优先级第二阶段；待冻结证明包精确字段 |
| [DEC-0026](DEC-0026-fork-tiebreak-hash.md) | 分叉平局 Hash 算法收敛 | Proposed | 高优先级第二阶段；待裁决 HashX/Hash256 口径冲突 |

## 旧编号迁移索引

| 旧编号 | 处理 | 目标 | 说明 |
|--------|------|------|------|
| old DEC-0001 | 主体吸收，细节合并 | conception/ + 新 DEC-0004 | PoH 铸凭算法主体已由 conception/ 固定；X 编码和碰撞边界进入新 DEC-0004。 |
| old DEC-0002 | Merged | 新 DEC-0004 | 并入 PoH 参数编码与碰撞处理。 |
| old DEC-0003 | Renumbered | 新 DEC-0001 | 规范化无符号 Varint 编码。 |
| old DEC-0004 | Renumbered/Updated | 新 DEC-0002 | 哈希域分隔标签格式。 |
| old DEC-0005 | Renumbered | 新 DEC-0017 | 脚本 VM Float 确定性。 |
| old DEC-0006 | Renumbered | 新 DEC-0011 | 奖励与交易费余数归属。 |
| old DEC-0007 | Renumbered | 新 DEC-0012 | Coinbase HashInputs 计算。 |
| old DEC-0008 | Merged | 新 DEC-0004 | 并入 PoH 参数编码与碰撞处理。 |
| old DEC-0009 | Renumbered/Updated | 新 DEC-0007 | 哈希树边界情况。 |
| old DEC-0010 | Renumbered | 新 DEC-0008 | ML-DSA-65 集成路径。 |
| old DEC-0011 | Renumbered | 新 DEC-0009 | 地址文本编码。 |
| old DEC-0012 | Renumbered | 新 DEC-0013 | 首领黑名单层级。 |
| old DEC-0013 | Merged | 新 DEC-0014 | 并入全网通告授权信任链。 |
| old DEC-0014 | Renumbered | 新 DEC-0015 | 创世高度年度边界。 |
| old DEC-0015 | Renumbered | 新 DEC-0010 | Coin 与 chx 换算。 |
| old DEC-0016 | Renumbered/Updated | 新 DEC-0006 | Stakes 精确定义；PoH 使用评参区块自身 Stakes。 |
| old DEC-0017 | Renumbered | 新 DEC-0003 | 区块头与交易头字段宽度。 |
| old DEC-0018 | Merged | 新 DEC-0014 | 并入全网通告授权信任链。 |
| old DEC-0019 | Renumbered | 新 DEC-0005 | PoH 时间戳的推导与隔离。 |
| old DEC-0020 | Renumbered/Updated | 新 DEC-0016 | 短引用歧义的协议级处理；按最新 conception 修正。 |

## 不再单列的决策

已由 conception 明确的规则不在本目录重复建档。旧 PoH 铸凭算法主体已由 `conception/1.共识-历史证明（PoH）.md` 固定；仅 `X` 编码和碰撞边界进入新 DEC-0004。典型示例还包括脚本初始通过状态、`CHECK` 覆盖语义、状态树 TxID 字节索引、`CheckRoot` 签名范围、未确认输出不可作为输入、TxID 碰撞处理、公共服务局部评估、`SYS_NULL` 解锁段例外、Witness 与解锁脚本分离、铸凭资格按区块高度判定、UTXO/UTCO 四层状态指纹树和区块限额增长规则。Coinbase 主体规则已明确，字段编码和兑奖槽细节见 DEC-0020；脚本资源边界主体已明确，成本预算细节见 DEC-0023。分叉竞争主体规则已在 conception 中存在，但平局哈希算法仍由 DEC-0026 Proposed 待裁决。
