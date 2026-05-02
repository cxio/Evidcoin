# ADR-0009: 区块奖励分配的取整与余数归属

## Status（状态）

Accepted

## Context（背景）

区块奖励（`RewardTotal`）按以下比例分配给五方（`docs/proposal/14.Incentives-And-Coinbase-Rewards.md`）：

| 接收方 | 比例 | 概念归属 |
|--------|------|----------|
| 校验组（Validation team） | 40% | 铸造者（50%）的组成部分 |
| 铸凭者（Mint proof provider） | 10% | 铸造者（50%）的组成部分 |
| Depots | 20% | — |
| Blockqs | 20% | — |
| stun2p | 10% | — |

> **概念说明**：构想层（`docs/conception/4.激励机制.md`）将铸造奖励描述为"铸造者 50%"，其中铸造者包含两类参与者：校验组（负责打包区块、参与组队校验）和铸凭者（负责提供铸凭交易 ID 并签名 CheckRoot）。提案层将其细分为校验组 40% 与铸凭者 10%，两者合计恰为 50%，与构想层一致，不存在冲突。

百分比层面总和为 100%，但奖励以最小单位整数（chx）计算，整数除法必然产生余数。各方如何取整、余数归属何方，属于共识级规则（OQ-021/022/023）。

不同实现若对余数处理方式不同，将导致 Coinbase 金额不匹配→CheckRoot 不一致→共识分歧。

## Decision（决策）

**按顺序依次计算，前四项向下取整，最后一项为减法**：

计算顺序：`校验组(40%) → 铸凭者(10%) → Depots(20%) → Blockqs(20%) → stun2p(10%)`

```
R = RewardTotal   // 整数，最小单位

validation  = R * 40 / 100        // floor 除法
minter      = R * 10 / 100        // floor 除法
depots      = R * 20 / 100        // floor 除法
blockqs     = R * 20 / 100        // floor 除法
stun2p      = R - validation - minter - depots - blockqs   // 减法，自动吸收所有余数
```

### 整数除法规则

- Go 的整数除法 `/` 即为 floor 除法（正整数范围内等同于截断取整）。
- 禁止使用浮点数进行中间计算再取整。

### 交易费的 50%/50% 分配

交易费分配（50% 回收 / 50% 销毁）同理：

```
recovered = TxFee * 50 / 100   // floor 除法
destroyed = TxFee - recovered  // 减法，余数归销毁
```

## Rationale（理由）

1. **确定性**：顺序固定、规则明确，任意两个独立实现必然产生相同的分配结果。
2. **简洁性**：不需要复杂的舍入模式（如四舍五入、银行家舍入），纯整数运算即可完成。
3. **余数归末项**：末项（stun2p）通过减法隐式吸收所有余数（最多 4 个 chx），在比例上误差极小，不影响激励效果。交易费的余数归销毁，不影响流通量计算。

## Consequences（影响）

- 需在 `docs/proposal/14.Incentives-And-Coinbase-Rewards.md` 中补充取整规则和计算示例。
- `internal/rewards` 的奖励计算函数必须按此顺序使用整数除法，禁止浮点中间值。
- 需提供测试向量：给定 RewardTotal，验证五项分配值及其总和等于 RewardTotal。
- OQ-021/022/023 关闭。

## References（参考）

- `docs/proposal/14.Incentives-And-Coinbase-Rewards.md` — 激励机制
- `docs/plan/07-Team-Validation-Services-Incentives.md` — Task 7
- `docs/plan/08-Open-Questions-And-Acceptance.md` — OQ-021/022/023
