# DEC-0006: Stakes 精确定义

## Status（状态）

Accepted

## Context（背景）

`conception/blockchain.md` 将区块头 `Stakes` 描述为币权销毁总值，单位为聪时。`conception/1.共识-历史证明（PoH）.md` 使用评参区块自身的 `Stakes` 字段参与铸凭哈希计算。构想没有固定币龄取整和字段溢出规则。

## Decision（决策）

`Stakes[H]` 表示区块 `H` 内所有非 Coinbase 交易输入所销毁的币权总和。

单个输入引用 UTXO 的币权：

```text
CoinAge = amount_chx * floor((BlockTime(H) - CreatedTime(utxo)) / 1 hour)
```

区块 Stakes：

```text
Stakes[H] = sum(CoinAge(input_utxo)) mod 2^64
```

> **注：**
> 累计的币权销毁总值指目标区块内所有交易的币权销毁总和。

规则：

- `amount_chx` 使用最小单位 `chx`。
- 不足 1 小时的币龄舍去。
- Coinbase 不计入 Stakes。
- 字段类型为 `uint64`，编码见 `DEC-0003`。
- 累加超过 `uint64` 时取低 64 位，不报错、不饱和。
- PoH 计算使用评参区块自身的 `Stakes` 字段。

## Rationale（理由）

- 输入引用历史 UTXO，才能表达被销毁的币权；新输出币龄为零。
- 小时取整可滤除过高频交易对币权的影响。
- 截断规则简单确定，避免不同语言或库对溢出处理不一致。
- 明确 PoH 参数来源可避免实现者读取错误高度或本地派生值。

## Consequences（影响）

- 区块竞争和 PoH X 参数计算必须使用同一 Stakes 口径。
- 测试应覆盖不足 1 小时、正好 1 小时、多输入累加和溢出截断。

## Conception Relationship（与构想关系）

- 精确化 conception 中“币权销毁总值（聪时）”的计算。
- 不改变 Stakes 参与 PoH 和区块竞争的 conception 用途。
