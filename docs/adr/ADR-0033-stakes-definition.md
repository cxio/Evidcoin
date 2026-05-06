# ADR-0033: Stakes 字段精确定义与溢出处理

## Status（状态）

Accepted

## Context（背景）

区块头中的 `Stakes` 字段在构想层（`docs/conception/blockchain.md`）中描述为"币权销毁，单位：聪时（聪*小时）"，但其精确计算口径从未在 Proposal 或 ADR 中正式确定，属于 OQ-018 剩余的未决问题。

`Stakes` 参与以下两处协议计算：
1. **铸凭哈希 X 参数**：`X = timestamp * Stakes * Mix`（ADR-0002）
2. **区块竞争"3 倍币权"规则**：候选区块 Stakes 需满足一定条件

此外，uint64 的最大值约为 1.84 × 10^19，在极端情况下（高价值交易聚集）可能溢出，需要明确处理方式。

## Decision（决策）

### 1. Stakes 的含义

`Stakes` 表示**该区块内所有交易的币权销毁总和**。

"币权销毁"定义为：交易中每笔**输入**所对应的 UTXO 的币权值之和。

单笔 UTXO 的币权计算公式：

```
CoinAge(utxo) = utxo.Amount(chx) * floor((BlockTime(currentHeight) - utxo.CreatedTime) / 1 hour)
```

说明：
- `Amount` 单位为 `chx`（即"聪"），因此 `CoinAge` 单位为 `chx·小时`（即"聪时"）
- 时间差不足 1 小时的部分舍去（`floor`），滤除高频交易影响
- `BlockTime(currentHeight)` 为当前区块的标准时间（由高度和创世时间折算）
- 交易输出是新产生的，其币龄从零开始，因此**输出不贡献任何币权**

区块 `Stakes` 为该区块内所有非 Coinbase 交易的所有输入的 `CoinAge` 之和：

```
Stakes[H] = Σ CoinAge(input_utxo) for all inputs in all non-Coinbase transactions in block H
```

### 2. 字段类型与溢出处理

`Stakes` 字段为 **uint64 类型（8 字节，big-endian）**。

当累加结果超过 `0xFFFF_FFFF_FFFF_FFFF` 时，**直接截断（取低 64 位）**，溢出部分丢弃。不报错，不饱和。

```
Stakes[H] = (Σ CoinAge) & 0xFFFF_FFFF_FFFF_FFFF
```

> **理由：** `Stakes` 是一个竞争因子和统计值，不影响资金安全性。溢出在实际网络中极难发生（需要海量大额长期未花费输出同时花费），即使偶发，对铸凭哈希计算和区块竞争判断的影响可以忽略。饱和处理反而会制造边界条件的共识分歧风险，简单截断最为安全。

### 3. Coinbase 交易的处理

Coinbase 交易没有引用历史 UTXO 的输入，**不计入 Stakes**。

### 4. Stakes 与 chx 单位

Stakes 的计量单位为 `chx·小时`（聪时）。由于 `chx` 是协议层最小计量单位（1 Coin = 10^8 chx，ADR-0031），Stakes 的数值量级远大于 Coin 量级，uint64 在正常使用范围内足够。

## Rationale（理由）

1. **与构想层一致**：构想层明确"币权销毁"来自输入，输出为新产生无币龄，Stakes = 所有输入 UTXO 的币权合计，本 ADR 对此做精确化。
2. **截断而非饱和**：截断保证了溢出行为确定且无歧义——所有节点得到相同结果。饱和在不同语言实现中容易出现不一致（有的库默认截断，有的库溢出是 UB）。
3. **排除 Coinbase**：Coinbase 不引用历史 UTXO，无法计算币龄，排除是自然的。

## Consequences（影响）

- **关闭 OQ-018**：本 ADR 完整关闭 OQ-018（Stakes 精确定义）。
- `internal/blockchain`（区块头编码）中 Stakes 字段类型确认为 `uint64`。
- `internal/consensus`（铸凭哈希、区块竞争）实现中使用本 ADR 计算 Stakes 值。
- 需在 `docs/proposal/05.Blockchain-Core.md` 补充 ADR-0033 追溯，明确 Stakes 的计算规则。
- 需在 `docs/plan/08-Open-Questions-And-Acceptance.md` 中将 OQ-018 标记为"已关闭（ADR-0033）"。

## References（参考）

- `docs/conception/blockchain.md` — 区块头 Stakes 字段原始定义
- `docs/adr/ADR-0002-poh-x-encoding.md` — X 参数计算（使用 Stakes）
- `docs/adr/ADR-0031-coin-chx-ratio.md` — chx 单位定义
- `docs/proposal/05.Blockchain-Core.md` — 区块链核心（区块头结构）
- `docs/proposal/10.PoH-Consensus.md` — PoH 铸凭哈希计算
- `docs/plan/08-Open-Questions-And-Acceptance.md` — OQ-018
