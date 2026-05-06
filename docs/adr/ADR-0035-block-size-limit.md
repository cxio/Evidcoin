# ADR-0035: 区块大小上限与增长函数

## Status（状态）

Accepted

## Context（背景）

构想层（`docs/conception/6.脚本系统.md`）提到"区块限额从 1MB 起逐步增长"，但 Proposal 层从未正式定义区块大小上限常量或增长函数。缺乏这一规则会导致：

1. 验证节点无法确定性地拒绝超大区块，DoS 防护不完整
2. 跨实现节点可能对同一区块的合法性产生分歧
3. 网络吞吐量缺乏可预期的增长路径

## Decision（决策）

### 1. 区块大小的统计范围

区块大小（`BlockSize`）统计范围为该区块内**所有交易的规范化编码字节总和**，采用与 `MaxTxSize` 相同的统计口径（按 ADR-0024：不含 Witness，包含 UnlockScript）。

Coinbase 交易也计入区块大小。区块头本身（160 字节，按 ADR-0034）不计入区块大小统计（区块头独立传输）。

### 2. 区块大小上限函数

区块大小上限为高度的函数，按如下公式计算：

```
BlockSizeLimit(height) = BaseBlockSize + GrowthPerYear * YearsElapsed(height)
```

其中：

| 参数 | 值 | 说明 |
|------|----|------|
| `BaseBlockSize` | 1,048,576 字节（1 MB） | 创世时的初始区块大小上限 |
| `GrowthPerYear` | 1,048,576 字节（1 MB） | 每年增加 1 MB |
| `YearsElapsed(height)` | `height / BlocksPerYear`（整除） | 已过年数，即高度对应的年度索引（从 0 开始，按 ADR-0030） |

展开公式：

```
BlockSizeLimit(height) = (1 + floor(height / 87661)) * 1,048,576  字节
```

示例：

| 高度 | 年度 | 区块大小上限 |
|------|------|-------------|
| 0 – 87,660 | Year 0 | 1 MB |
| 87,661 – 175,321 | Year 1 | 2 MB |
| 175,322 – 262,982 | Year 2 | 3 MB |
| 876,610 – … | Year 10 | 11 MB |
| 8,766,100 – … | Year 100 | 101 MB |

### 3. 上限不设封顶

区块大小上限随时间线性增长，**不设封顶值**。百年后约为 101 MB，千年后约为 1001 MB。此增长速度远低于存储和带宽技术的演进速度，不构成实际瓶颈。如未来需要调整增长率，应通过协议升级（新版本 ADR）实现。

### 4. 验证规则

区块验证时，若 `BlockSize > BlockSizeLimit(height)`，则拒绝该区块，返回 `ErrBlockTooLarge`。

## Rationale（理由）

1. **与构想层一致**：构想层明确提出"每年递增 1MB"的线性增长，本 ADR 将此转化为精确公式。
2. **使用高度折算年度**：与 ADR-0030 一致，保证跨节点计算结果相同（高度确定，年度确定，无需依赖本地时钟）。
3. **线性增长**：相比指数增长（如按比例递增），线性增长更易预测、更保守，有利于低配置节点持续参与。
4. **无封顶**：设置封顶会在达到上限时形成硬性约束，引发网络拥堵和手续费竞价，不符合项目设计目标。

## Consequences（影响）

- **关闭 M-6 未决项**：本 ADR 正式关闭区块大小上限的未决问题。
- `internal/blockchain`（入块验证）中需新增 `BlockSizeLimit(height)` 函数和对应的区块大小检查逻辑。
- 需在 `docs/proposal/05.Blockchain-Core.md` 或 `docs/proposal/09.Script-System.md` 中补充 ADR-0035 追溯，并移除"区块限额待定"相关描述。
- 需在 `docs/plan/08-Open-Questions-And-Acceptance.md` 中新增 OQ-028（如适用）或直接关闭对应的未决描述。

## References（参考）

- `docs/conception/6.脚本系统.md` — 区块限额增长原始描述
- `docs/adr/ADR-0030-genesis-year-boundary.md` — 年度 = height / BlocksPerYear
- `docs/adr/ADR-0024-signature-witness-separation.md` — MaxTxSize 统计口径
- `docs/adr/ADR-0034-field-widths.md` — 区块头不计入区块大小
- `docs/proposal/05.Blockchain-Core.md` — 区块链核心（入块验证）
- `docs/proposal/09.Script-System.md` — 脚本系统（区块限额原始提及）
