# DECISION-0015: Coin 与 chx 换算

## Status（状态）

Accepted

## Context（背景）

`docs/conception/5.信用结构.md` 使用 `chx` 作为最小币金单位，并说明其类似 Satoshi。构想未直接固定 `Coin` 与 `chx` 的换算比例。

## Decision（决策）

```text
1 Coin = 100,000,000 chx
1 chx  = 0.00000001 Coin
```

协议层所有金额字段使用 `chx` 作为整数单位。

## Rationale（理由）

- 与 Bitcoin 的 BTC/Satoshi 精度一致，用户和开发者容易理解。
- 八位小数足以覆盖常规支付和微支付。
- 使用整数金额避免浮点金额误差。

## Consequences（影响）

- 奖励、交易费、输出金额和币权计算均以 `chx` 为基础。
- 展示层可显示 Coin，小数换算不得进入共识计算。

## Conception Relationship（与构想关系）

- 补充 `chx` 与人类展示单位 `Coin` 的比例。
- 不改变 conception 中 `chx` 是最小单位的设定。

## Source（来源）

- 迁移重写自旧 `ADR-0031`。
