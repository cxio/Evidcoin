# ADR-0032: UTXO/UTCO 四层状态指纹树节点组合算法

## Status（状态）

Accepted

## Context（背景）

ADR-0008 已固定四层宽成员哈希树的中间层 TxID 字节索引（0-based `[8, 13, 18]`），但以下关键细节仍属开放问题（OQ-012 剩余部分）：

1. 年度分区的计算口径（按哪个时间点划分年度？）
2. 各层节点的排序规则
3. 各层节点的哈希组合方式（串接之后如何计算哈希？）
4. 空分支（某个槽位无条目）的处理策略
5. 末端叶子节点的排序规则

这些细节直接决定了 `UTXORoot` 和 `UTCORoot` 的最终字节值，是 `CheckRoot` 计算的前置依赖，也是跨节点共识一致性的关键保证。

## Decision（决策）

### 1. 年度划分口径

UTXO/UTCO 条目的年度（`Year`）按**创建该 UTXO/UTCO 的交易的时间戳**确定，取 UTC 年度。具体公式：

```
Year(entry) = UTC_Year(transaction.Timestamp)
```

计算说明：
- `transaction.Timestamp` 为交易头中的 int64 毫秒时间戳（按 ADR-H3 固定）
- UTC 年度以日历年为单位（2024, 2025, …），不使用区块高度折算

> **注：** 使用真实 UTC 年度而非高度折算年度，保证年度分区与实际时间锚定，避免因出块速度变化导致分区偏移。区块高度用于出块时间计算（见 `BlockTime`），而 UTXO/UTCO 年度用于状态树分区，两者语义不同，应独立处理。

### 2. 树结构概览

四层宽成员树从顶层到末端的层级路由键如下：

| 层级 | 路由键 | 说明 |
|------|--------|------|
| 顶层（Layer 0） | `Year`（UTC 年度整数） | 按年度分组，层数随时间无限增长 |
| Layer 1 | `TxID[8]`（0-based，uint8） | 256 个槽位 |
| Layer 2 | `TxID[13]`（0-based，uint8） | 256 个槽位 |
| Layer 3（末端） | `TxID[18]`（0-based，uint8） | 256 个槽位，每槽内含叶子节点列表 |

### 3. 各层节点排序规则

- **Layer 0（年度层）**：按年度整数升序排列（2024 < 2025 < …）。
- **Layer 1 / Layer 2 / Layer 3**：按路由键字节值自然升序排列（`0x00` → `0xFF`），共 256 个固定槽位。

### 4. 空分支处理

除顶层年度分区外，Layer 1、Layer 2、Layer 3 每层均为固定 256 个槽位。对于没有任何条目的槽位，以 **32 字节全零**（`[32]byte{}`）填充，代表空子树哈希。

```
EmptyNodeHash = [32]byte{0, 0, ..., 0}  // 32 字节全零
```

> **说明：** 固定 256 槽位使得同层节点数量恒定，简化了证明路径计算和批量更新逻辑。年度层不设上限，随链上活跃年度自然增长。

### 5. 层内哈希组合方式

对 Layer 1、Layer 2、Layer 3 中的每个父节点，其哈希值由**该层所有 256 个子节点哈希值串接后**，使用 `BLAKE3-256`（树分支算法，按 ADR-0004 `TreeBranch` domain tag）计算：

```
NodeHash[parent] = BLAKE3-256(domainTag("TreeBranch") || child[0] || child[1] || ... || child[255])
```

其中每个 `child[i]` 为 32 字节（非空子节点的哈希，或全零空节点哈希）。每个父节点的输入长度固定为 `32 × 256 = 8192` 字节，输出为 32 字节。

### 6. 末端叶子节点（Layer 3 内）

Layer 3 的每个槽位（`TxID[18]` 相同的条目）内，可能有多个 UTXO/UTCO 叶子。叶子按 **TxID 字典序升序排列**后，依次串接并计算哈希：

```
SlotHash = BLAKE3-256(domainTag("TreeBranch") || leaf[0] || leaf[1] || ... || leaf[n-1])
```

每个叶子哈希为 48 字节（`SHA3-384`，按 ADR-0004 `StateLeaf` domain tag，与 `proposal/08.UTXO-UTCO-State.md` 中的叶子定义一致）。

若槽位内只有一个叶子，则 `SlotHash = leaf[0]`（32 字节截断：叶子本身为 48 字节，存储时需截取前 32 字节放入父节点槽位）。

> **修正说明：** 叶子（`StateLeaf`）为 48 字节（SHA3-384），而树分支节点槽位为 32 字节（BLAKE3-256）。当槽位内叶子数量 ≥ 2 时，串接后经 BLAKE3-256 得 32 字节；当只有 1 个叶子时，取叶子哈希前 32 字节填入槽位。

### 7. 年度层（顶层）根的计算

`UTXORoot`（以及 `UTCORoot`，结构相同）为年度层的顶层根：

```
UTXORoot = BLAKE3-256(domainTag("TreeBranch") || yearNode[Y_min] || yearNode[Y_min+1] || ... || yearNode[Y_max])
```

其中：
- 年度范围取所有存在非空条目的年度，按年度升序排列
- 不存在任何条目的年度不参与串接（顶层不设固定槽位，以避免随时间无限膨胀）
- 若整个 UTXO/UTCO 集合为空，`UTXORoot = [48]byte{}`（48 字节全零，与空 `CheckRoot` 语义对齐）

> **注：** 顶层使用动态串接（只含非空年度），而非固定 256 槽位，因为年度数量随时间增长不可预期。这与 Layer 1-3 固定 256 槽位形成对比，是显式的设计权衡。

## Rationale（理由）

1. **年度按交易时间戳**：UTXO/UTCO 描述的是"交易产生的状态"，使用交易时间戳决定年度最自然，与其所在区块的高度无关，避免因创世时间或出块速度假设带来的分歧。
2. **固定 256 槽位**：以路由键 `uint8`（0-255）对应 256 个固定槽位，使得树结构确定、证明路径长度恒定，实现简单且无歧义。
3. **全零空节点**：统一的空节点哈希避免了"跳过空槽"带来的证明路径复杂性，同时保持了节点哈希计算的输入长度固定。
4. **BLAKE3-256 串接**：与哈希树分支节点一致（ADR-0004），利用 BLAKE3 的高性能处理批量输入。

## Consequences（影响）

- **关闭 OQ-012**：本 ADR 完整关闭 OQ-012 的剩余开放问题（状态树节点组合、年度分区编码）。
- `internal/utxo` 和 `internal/utco` 的 Task 7（四层根计算）可按本 ADR 实现最终版本，不再返回 `ErrSpecIncomplete`。
- `CheckRoot` 的完整端到端测试向量现在可以生成。
- 需在 `docs/proposal/08.UTXO-UTCO-State.md` 和 `docs/plan/04-UTXO-UTCO-State.md` 中补充 ADR-0032 追溯，移除对应的"未决"标注。
- 需在 `docs/plan/08-Open-Questions-And-Acceptance.md` 中将 OQ-012 标记为"已关闭（ADR-0008 + ADR-0032）"。

## References（参考）

- `docs/conception/附.组队校验.md` — UTXO/UTCO 指纹的四层宽成员哈希校验树结构
- `docs/proposal/08.UTXO-UTCO-State.md` — UTXO/UTCO 状态
- `docs/proposal/04.Hash-Trees.md` — 哈希树策略
- `docs/adr/ADR-0008-utxo-txid-byte-index.md` — TxID 字节索引（0-based）
- `docs/adr/ADR-0004-domain-tag-format.md` — Domain Tag 格式
- `docs/plan/04-UTXO-UTCO-State.md` — Task 7
- `docs/plan/08-Open-Questions-And-Acceptance.md` — OQ-012
