# DEC-0302: Genesis and Initial Window（创世与初段窗口）

Status: Proposed

## Context（背景）

Conception 已明确创世区块、创世 Coinbase、#1/#2 启动逻辑和百日扩张阶段。但创世工件的发布格式、初段评参和 `-240` 放宽规则的精确边界仍需冻结。

## Decision（决策）

建议创世工件包含：

- 创世区块头完整编码。
- 创世 Coinbase 完整交易体。
- 创世铸造者对 Coinbase 的签名。
- 创世铸造者对 `CheckRoot` 的签名。
- 创世声明 `FreeData`。

建议初段规则：

- `currentHeight < 8` 时评参区块高度为 `0`。
- `currentHeight >= 8` 时评参区块高度为 `currentHeight - 8`。
- `currentHeight < 240` 时铸凭交易高度检查放宽，但仍必须引用已确认交易。
- `currentHeight >= 240` 时使用 `h > 239 && h <= 80000`。

建议 #1/#2：

- #1 由创世 `MintPKHash` 铸造。
- #1 可以包含花费创世输出的普通交易。
- #2 起允许基于已确认初段交易形成竞争。

## Rationale（理由）

把创世工件作为客户端硬编码资料的一部分，可避免不同实现自行拼接创世块。初段规则直接沿用 conception，并补充“必须已确认”以保持交易输入规则一致。

## Consequences（影响）

- 创世工件一旦发布即不可变。
- 初段测试需要覆盖高度 0、1、2、7、8、239、240。
- #1 和 #2 的特殊处理不得泄漏到正常高度逻辑。

## Conception References（构想层依据）

- `docs/conception/blockchain.md#区块链启动`
- `docs/conception/1.共识-历史证明（PoH）.md#初段规则`

## Open Questions（开放问题）

- 创世 Coinbase 中 `Minter` 省略时，创世块的铸凭哈希记录如何表达。
- 创世工件是否应包含可读 JSON 描述和二进制 canonical 版本两套形式。
- 初段 Coinbase 是否可作为后续铸凭交易，见 `CONCEPTION-CONFLICTS.md` 相关项。
