# DEC-0021: 原始铸币发行曲线高度边界与取整

## Status（状态）

Accepted

## Context（背景）

`conception/4.激励机制.md` 已定义初期三年预发布、正式发行期每两年递减 20%、降至 3 Coin 后长期维持低通胀，并明确每年 `87661` 个区块。实现层需要固定高度边界和 Coin 到 chx 的整数换算取整。

## Decision（决策）

发行曲线按 `BlocksPerYear = 87661` 计算，区块高度从 `0` 开始。

```text
yearIndex = floor(height / 87661)
```

每块原始铸币金额：

- `yearIndex == 0`：10 Coin。
- `yearIndex == 1`：20 Coin。
- `yearIndex == 2`：30 Coin。
- `yearIndex >= 3`：进入正式期，起始为 40 Coin，每 2 年递减 20%。
- 当递减结果低于或等于 3 Coin 阶段后，长期固定为 3 Coin。

正式期阶段计算：

```text
stage = floor((yearIndex - 3) / 2)
rewardCoin[0] = 40
rewardCoin[n] = floor(rewardCoin[n-1] * 4 / 5)
rewardCoin = max(rewardCoin[stage], 3)
```

阶段表：

| stage | rewardCoin |
|-------|------------|
| 0 | 40 |
| 1 | 32 |
| 2 | 25 |
| 3 | 20 |
| 4 | 16 |
| 5 | 12 |
| 6 | 9 |
| 7 | 7 |
| 8 | 5 |
| 9 | 4 |
| 10 | 3 |

当递推结果降至 3 Coin 后，后续所有阶段长期固定为 3 Coin。实现不得使用浮点计算，必须使用整数乘法、整数除法和向下取整。

所有金额必须转换为 chx 整数：

```text
rewardChx = floor(rewardCoin * ChxPerCoin)
```

其中 `ChxPerCoin` 使用 `DEC-0010` 固定的 Coin 与 chx 换算关系。

## Rationale（理由）

- conception 的发行参考表按完整年和两年阶段表达，使用高度除法可直接得到确定性年度边界。
- 正式期“以币为单位而不是聪”要求先按 Coin 计算，再转换为 chx。
- 递推整数规则与 conception 参考表一致，避免一次性幂计算造成阶段 2 等位置偏离。
- 向下取整避免产生超发。

## Consequences（影响）

- 创世块是否包含 Coinbase 奖励若另有规则，应由创世块规范单独定义；本决策只定义给定高度的发行金额函数。
- 实现必须避免使用浮点计算，并用整数分子分母表达 `0.8 = 4/5` 后逐阶段向下取整。
- 测试应覆盖每个年度边界、正式期两年阶段边界和长期 3 Coin 下限。

## Conception Relationship（与构想关系）

- 补充 `conception/4.激励机制.md` 中原始铸币曲线的高度边界与取整语义。
- 不改变前三年 `10/20/30`、正式期 `40`、每两年递减 `20%` 和长期 `3` Coin 的构想层规则。
