# ADR-0031: Coin 与 chx 换算比例

## Status（状态）

Accepted

## Context（背景）

Evidcoin 的基本计量单位在协议层面始终以最小不可分割单位进行整数运算。在人机界面和文档中，需要明确"1 个 Coin"对应多少个最小单位，以便用户理解和换算。

最小单位的名称在先前文档中已有提及（使用 `chx` 作为最小单位名称），但换算比例始终未正式确定，属于未解决的基础设计项。

## Decision（决策）

**1 Coin = 100,000,000 chx（一亿 chx）。**

| 单位 | 换算 | 类比 |
|------|------|------|
| 1 Coin | = 100,000,000 chx | 相当于 Bitcoin 中的 1 BTC |
| 1 chx | = 0.000,000,01 Coin | 相当于 Bitcoin 中的 1 聪（Satoshi） |

协议层所有涉及金额的字段（输出值、奖励额、费用等）均以 `chx` 为单位，采用无符号 64 位整数表示。

## Rationale（理由）

1. **与主流设计对齐**：100,000,000 的换算比例与 Bitcoin 采用的聪（Satoshi）设计相同。这是区块链领域中最广泛接受的精度设计，用户和开发者普遍熟悉。
2. **足够的精度**：`1 chx = 10^-8 Coin` 提供了八位小数精度，对于日常支付、微支付场景均已足够。
3. **uint64 可容纳**：理论总发行量以 `chx` 计的上限，在 uint64（最大约 1.84 × 10^19）范围内绰绰有余，不存在溢出风险（即便总量达到 1 亿 Coin，折合 10^16 chx，远低于 uint64 上限）。

## Consequences（影响）

- 需在 `docs/proposal/03.Identifiers-And-Constants.md` 中正式记录 `1 Coin = 100,000,000 chx` 的换算关系，以及 `chx` 是协议层的基础金额单位。
- 所有提案和计划文档中涉及金额的示例，应统一使用 `chx` 作为单位，不应出现小数 Coin 值（如 `0.001 Coin` 应写为 `100,000 chx`）。
- `pkg/types` 中金额类型应定义为 `type Amount uint64`，并提供 `CoinToChx(coin float64) uint64` 和 `ChxToCoin(chx uint64) string` 等辅助函数（仅用于显示层，协议层运算始终使用 `Amount` 整数类型）。

## References（参考）

- `docs/proposal/03.Identifiers-And-Constants.md` — 标识符与常量
- `docs/proposal/07.Coin-Credit-Proof-Units.md` — Coin 单位定义
- `docs/proposal/14.Incentives-And-Coinbase-Rewards.md` — 奖励金额计算
