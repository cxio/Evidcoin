# DEC-0011: 奖励与交易费余数归属

## Status（状态）

Accepted

## Context（背景）

`conception/4.激励机制.md` 已明确奖励比例和交易费 50% 销毁规则，并说明奖励不能整除时余数留给最终分配者。但整数金额以最小单位计算，仍需固定计算顺序和交易费奇数余数归属。

## Decision（决策）

区块收益 `RewardTotal` 按以下顺序使用整数除法计算，最后一项通过减法吸收余数：

```text
validation = RewardTotal * 40 / 100
credentialProvider = RewardTotal * 10 / 100
depots     = RewardTotal * 20 / 100
blockqs    = RewardTotal * 20 / 100
stun2p     = RewardTotal - validation - credentialProvider - depots - blockqs
```

其中 `validation` 对应校验组 40% 奖励，`credentialProvider` 对应铸凭者或铸凭交易提供者 10% 奖励。

交易费销毁与回收按以下规则计算：

```text
recovered = TxFee * 50 / 100
destroyed = TxFee - recovered
```

所有计算均使用非负整数，不得使用浮点数。

## Rationale（理由）

- 固定顺序可保证跨实现一致。
- `stun2p` 是 conception 奖励列表中的最后分配者，吸收奖励余数符合“最终分配者保有最后余数”。
- 交易费奇数余数归销毁，避免凭空增加回收收益。

## Consequences（影响）

- Coinbase 金额校验必须使用上述顺序。
- 测试应覆盖不能整除的 `RewardTotal` 和奇数 `TxFee`。

## Conception Relationship（与构想关系）

- 细化 `conception/4.激励机制.md` 已给出的比例和余数原则。
- 不改变奖励比例和 50% 交易费销毁规则。
