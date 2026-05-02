# ADR-0013: 哈希树边界情况处理策略

## Status（状态）

Accepted

## Context（背景）

Evidcoin 的哈希树（`docs/proposal/04.Hash-Trees.md`）用于交易集合树、附件片组树和状态指纹树。提案层对以下三种边界情况的处理策略未定义（OQ-004/005/006）：

1. **空树**：没有叶子节点时，树根是什么值？
2. **单叶树**：只有一个叶子节点时，根等于什么？
3. **奇数叶树**：叶子节点数为奇数时，最后一个叶子如何处理？

这些边界情况直接影响所有哈希树的根计算，进而影响 CheckRoot 的验证。

## Decision（决策）

### 空树根

**空树根 = 全零 Hash**，长度与当前树类型的根 Hash 类型一致：
- 交易树（TreeHash，BLAKE3-256）→ 32 字节全零
- UTXO/UTCO 状态指纹树（Hash48，SHA3-384）→ 48 字节全零

### 单叶树根

**单叶树根 = 叶子 Hash 本身**，不再经过分支节点哈希。

即：若树只有一个叶子，`Root = LeafHash`，跳过任何包装计算。

### 奇数叶处理

**复制最后一个叶子**：若叶子数为奇数，将最后一个叶子复制，配对后正常计算父节点。

```
叶子: [A, B, C]
处理: [A, B, C, C]   // 复制 C
树结构:
    Root
   /    \
 AB      CC
 /\      /\
A  B    C  C
```

## Rationale（理由）

1. **空树全零**：全零值在任何哈希算法中都有独特语义，容易识别和调试，且不会与任何合法 Hash 值产生语义混淆（合法 Hash 全零的概率约为 `2^-256`）。
2. **单叶等于自身**：最简洁的处理方式，避免引入无意义的包装计算。与比特币、以太坊等主流区块链的惯例一致。
3. **复制最后叶子**：这是 Bitcoin Merkle 树的经典处理方式，广为人知，已有充分验证。相比于"奇数叶单独成路径"或"填充 null 哈希"，复制方案实现最简单，也不引入额外的安全假设。

## Consequences（影响）

- 需在 `docs/proposal/04.Hash-Trees.md` 中补充三种边界情况的规范定义和示意图。
- `pkg/hashtree` 的树构建函数需处理 0 个、1 个和奇数个叶子的场景，并通过相应的单元测试。
- Plan 01 中的 `OddLeafPolicy` 和 `EmptyTreePolicy` 策略参数可以移除（改为硬编码），或保留作测试灵活性（建议保留接口但默认值固定为本 ADR 规定的行为）。
- OQ-004/005/006 关闭。

## References（参考）

- `docs/proposal/04.Hash-Trees.md` — 哈希树
- `docs/plan/01-Foundation-Types-Crypto.md` — Task 6
- `docs/plan/08-Open-Questions-And-Acceptance.md` — OQ-004/005/006
