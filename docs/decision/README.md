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

| 编号 | 标题 | 状态 |
|------|------|------|
| [DECISION-0001](DECISION-0001-poh-inner-hash-algorithm.md) | PoH 铸凭哈希内层算法 | Accepted |
| [DECISION-0002](DECISION-0002-poh-x-encoding.md) | PoH 铸凭哈希 X 参数编码 | Accepted |
| [DECISION-0003](DECISION-0003-varint-encoding.md) | 规范化无符号 varint 编码 | Accepted |
| [DECISION-0004](DECISION-0004-domain-tag-format.md) | 哈希域分隔标签格式 | Accepted |
| [DECISION-0005](DECISION-0005-script-float-determinism.md) | 脚本 VM Float 确定性 | Accepted |
| [DECISION-0006](DECISION-0006-reward-rounding.md) | 奖励与交易费余数归属 | Accepted |
| [DECISION-0007](DECISION-0007-coinbase-hashinputs.md) | Coinbase HashInputs 计算 | Accepted |
| [DECISION-0008](DECISION-0008-minthash-collision.md) | 铸凭哈希碰撞处理 | Accepted |
| [DECISION-0009](DECISION-0009-hash-tree-edge-cases.md) | 哈希树边界情况 | Accepted |
| [DECISION-0010](DECISION-0010-ml-dsa-65-integration.md) | ML-DSA-65 集成路径 | Accepted |
| [DECISION-0011](DECISION-0011-address-text-encoding.md) | 地址文本编码 | Accepted |
| [DECISION-0012](DECISION-0012-leader-blacklist-convention.md) | 首领黑名单层级 | Accepted |
| [DECISION-0013](DECISION-0013-announcement-authority-keys.md) | 全网通告授权公钥 | Accepted |
| [DECISION-0014](DECISION-0014-genesis-year-boundary.md) | 创世高度年度边界 | Accepted |
| [DECISION-0015](DECISION-0015-coin-chx-ratio.md) | Coin 与 chx 换算 | Accepted |
| [DECISION-0016](DECISION-0016-stakes-definition.md) | Stakes 精确定义 | Accepted |
| [DECISION-0017](DECISION-0017-field-widths.md) | 区块头与交易头字段宽度 | Accepted |

## 不再单列的决策

已由 conception 明确的规则不在本目录重复建档。典型示例包括脚本初始通过状态、`CHECK` 覆盖语义、状态树 TxID 字节索引、`CheckRoot` 签名范围、资源边界、未确认输出不可作为输入、TxID 碰撞处理、公共服务局部评估、`SYS_NULL` 解锁段例外、Coinbase 独立格式、Witness 与解锁脚本分离、铸凭资格按区块高度判定、分叉平局处理、UTXO/UTCO 四层状态指纹树和区块限额增长规则。
