# DEC-0015: 创世高度年度边界

## Status（状态）

Accepted

## Context（背景）

`conception/` 使用年度组织区块、交易引用和状态指纹。`conception/blockchain.md` 已明确每年约 `87661` 个区块和年块机制，但没有直接给出年度编号公式。

## Decision（决策）

创世高度 `#0` 是年度 0 的起点。

```text
Year(blockHeight) = floor(blockHeight / 87661)
```

年度范围：

| 年度 | 高度范围 |
|------|----------|
| 0 | `[0, 87660]` |
| 1 | `[87661, 175321]` |
| 2 | `[175322, 262982]` |

## Rationale（理由）

- 以创世块为起点最简洁，不依赖现实日历年。
- 纯高度计算可保证跨节点一致。
- 与年块边界和 TxID 年度分区自然对应。

## Consequences（影响）

- 所有按年度分组的协议数据应使用同一公式。
- 测试应覆盖 `0`、`87660`、`87661` 等边界高度。

## Conception Relationship（与构想关系）

- 补充 conception 年度概念的编号公式。
- 不改变 `BlocksPerYear = 87661` 和年块机制。
