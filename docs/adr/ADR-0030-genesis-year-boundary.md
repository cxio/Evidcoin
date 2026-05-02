# ADR-0030: 创世高度 #0 为年度边界

## Status（状态）

Accepted

## Context（背景）

Evidcoin 的 TxID 引用系统使用"年度（Year）"分区来限制 TxIDPart 的歧义碰撞搜索空间（见 ADR-0016）。"年度"的计算依赖于区块链自身的起点（而非现实日历年）。

评审报告（L-4）提出：创世区块（高度 `#0`）是否为第一个年度边界？这影响年度分区的计算方式。

## Decision（决策）

**创世区块（高度 `#0`）为第一年度边界，即区块链的年度从高度 `#0` 开始计算。**

具体计算方式：

```
Year(blockHeight) = blockHeight / BlocksPerYear
```

其中 `BlocksPerYear = 87661`（每年区块数，基于 6 分钟出块间隔，参见 AGENTS.md）。

- 年度 0（Year 0）：区块高度 `[0, 87660]`
- 年度 1（Year 1）：区块高度 `[87661, 175321]`
- 以此类推

创世区块 `#0` 是年度 0 的第一个区块，也是整个年度分区体系的起点。

## Rationale（理由）

1. **自然起点**：创世区块是区块链存在的逻辑起点，以其作为年度起点，年度编号与区块高度的对应关系简洁直观（整数除法即可获得年度编号），无需额外的偏移量。
2. **消除歧义**：明确声明 `#0` 为年度边界，避免实现者对"年度从 `#1` 开始"或"使用现实日历年"等错误假设产生混淆。
3. **确定性计算**：纯基于区块高度的年度计算完全确定，不依赖节点本地时间，保证了全网一致性。

## Consequences（影响）

- 需在 `docs/proposal/03.Identifiers-And-Constants.md` 中明确指出创世高度 `#0` 为年度 0 的起点，以及 `Year = blockHeight / BlocksPerYear` 的计算公式。
- `internal/utxo`/`internal/utco` 中引用年度的地方，须使用上述统一公式。
- 测试套件需包含年度边界值的验证：高度 `0`→Year 0，高度 `87660`→Year 0，高度 `87661`→Year 1。

## References（参考）

- `docs/proposal/03.Identifiers-And-Constants.md` — 标识符与常量
- `docs/proposal/08.UTXO-UTCO-State.md` — UTXO/UTCO 状态（TxIDPart 年度分区）
- ADR-0016（TxIDPart 碰撞处理，年度分区概念来源）
- AGENTS.md — `BlocksPerYear = 87661`
