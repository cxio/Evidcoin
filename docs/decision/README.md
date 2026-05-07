# Architecture Decisions（架构决策）

`docs/decision` 记录 Evidcoin 在 `docs/conception` 中尚未直接固定、但会影响跨实现一致性或后续实现路径的补充决策。

项目文档的正式结构收敛为两层：

| 层级 | 目录 | 作用 |
|------|------|------|
| Conception | `docs/conception/` | 作者构想与协议设计来源，优先级最高。 |
| Decision | `docs/decision/` | 仅补充 conception 未明确的架构与规范化决策。 |

## 维护原则

- 若某项规则已在 `docs/conception` 中明确，不再单独建立 Decision。
- Decision 不重复描述 conception 已固定的协议主体，只记录缺口、边界语义、编码细节或实现路径。
- 若 Decision 与 conception 冲突，以 conception 为准，并应修订或删除对应 Decision。
- `docs/proposal` 与 `docs/plan` 当前视为待重构材料，不作为本目录整理依据。

## 命名规范

```text
DECISION-NNNN-<short-description>.md
```

- `NNNN`：四位数字编号，从 `0001` 开始。
- `<short-description>`：英文小写短描述，单词间使用连字符。

## 当前决策索引

| 编号 | 标题 | 来源旧 ADR | 状态 |
|------|------|------------|------|
| [DECISION-0001](DECISION-0001-poh-inner-hash-algorithm.md) | PoH 铸凭哈希内层算法 | ADR-0001 | Accepted |
| [DECISION-0002](DECISION-0002-poh-x-encoding.md) | PoH 铸凭哈希 X 参数编码 | ADR-0002 | Accepted |
| [DECISION-0003](DECISION-0003-varint-encoding.md) | 规范化无符号 varint 编码 | ADR-0003 | Accepted |
| [DECISION-0004](DECISION-0004-domain-tag-format.md) | 哈希域分隔标签格式 | ADR-0004 | Accepted |
| [DECISION-0005](DECISION-0005-script-float-determinism.md) | 脚本 VM Float 确定性 | ADR-0005 | Accepted |
| [DECISION-0006](DECISION-0006-reward-rounding.md) | 奖励与交易费余数归属 | ADR-0009 | Accepted |
| [DECISION-0007](DECISION-0007-coinbase-hashinputs.md) | Coinbase HashInputs 计算 | ADR-0010 | Accepted |
| [DECISION-0008](DECISION-0008-minthash-collision.md) | 铸凭哈希碰撞处理 | ADR-0011 | Accepted |
| [DECISION-0009](DECISION-0009-hash-tree-edge-cases.md) | 哈希树边界情况 | ADR-0013 | Accepted |
| [DECISION-0010](DECISION-0010-ml-dsa-65-integration.md) | ML-DSA-65 集成路径 | ADR-0018 | Accepted |
| [DECISION-0011](DECISION-0011-address-text-encoding.md) | 地址文本编码 | ADR-0020 | Accepted |
| [DECISION-0012](DECISION-0012-leader-blacklist-convention.md) | 首领黑名单层级 | ADR-0026 | Accepted |
| [DECISION-0013](DECISION-0013-announcement-authority-keys.md) | 全网通告授权公钥 | ADR-0028 | Accepted |
| [DECISION-0014](DECISION-0014-genesis-year-boundary.md) | 创世高度年度边界 | ADR-0030 | Accepted |
| [DECISION-0015](DECISION-0015-coin-chx-ratio.md) | Coin 与 chx 换算 | ADR-0031 | Accepted |
| [DECISION-0016](DECISION-0016-stakes-definition.md) | Stakes 精确定义 | ADR-0033 | Accepted |
| [DECISION-0017](DECISION-0017-field-widths.md) | 区块头与交易头字段宽度 | ADR-0034 | Accepted |

## 已由 Conception 吸收的旧 ADR

以下旧 ADR 的核心决策已在 `docs/conception` 中明确，不再迁移为 Decision：

| 旧 ADR | 处理理由 |
|--------|----------|
| ADR-0006 | `Instruction/6.结果指令.md` 已明确初始通过状态为 `true`。 |
| ADR-0007 | `Instruction/6.结果指令.md` 已明确 `CHECK` 会被后续 `CHECK` 覆盖。 |
| ADR-0008 | `附.组队校验.md` 已明确状态树 TxID 字节索引 `[8,13,18]`。 |
| ADR-0012 | `附.组队校验.md` 已明确铸造者只签署 `CheckRoot`。 |
| ADR-0014 | `6.脚本系统.md` 已用 `< N` 形式明确资源边界。 |
| ADR-0015 | `附.交易.md` 已明确未确认输出不能作为输入。 |
| ADR-0016 | `附.组队校验.md` 已明确 TxID 碰撞交易不会被收录。 |
| ADR-0017 | `4.激励机制.md` 已明确公共服务采用局部评估与后续确认。 |
| ADR-0019 | `6.脚本系统.md` 已明确 `SYS_NULL` 是解锁脚本例外。 |
| ADR-0022 | `附.组队校验.md` 已明确当前区块使用上一区块执行后的状态集合。 |
| ADR-0023 | `附.交易.md` 已明确 Coinbase 是特殊交易且输出配置独立。 |
| ADR-0024 | `附.交易.md` 与 `6.脚本系统.md` 已明确 Witness 与解锁脚本分离。 |
| ADR-0025 | `1.共识-历史证明（PoH）.md` 已明确资格按交易所在区块高度判定。 |
| ADR-0027 | `2.共识-端点约定.md` 已明确同铸造者多签与分叉平局处理。 |
| ADR-0029 | 旧决策依赖 plan 中的客户端策略，当前不作为正式 Decision 保留。 |
| ADR-0032 | `附.组队校验.md` 已明确 UTXO/UTCO 四层状态指纹树结构。 |
| ADR-0035 | `6.脚本系统.md` 已明确区块限额增长规则，旧 ADR 公式不再保留。 |
