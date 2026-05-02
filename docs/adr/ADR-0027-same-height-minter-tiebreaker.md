# ADR-0027: 同高度同铸造者竞争区块的决胜规则

## Status（状态）

Accepted

## Context（背景）

在 PoH 分叉竞争中，如果同一铸造者（铸凭者）在某一特定区块高度对两个不同的区块分别进行了签名（即"双签"或"多签"），两个区块的 MintHash 将完全相同（因为使用相同的 mintTxID 和相同的评参区块）。

此时在 PoH 权重比较中，这两个区块在该高度"打平"，fork choice 算法需要一个确定性的决胜规则来避免无限平局导致网络分裂。

同时，这条规则本身应当对铸造者的双签行为形成经济威慑。

## Decision（决策）

**同高度同铸造者下，多个竞争区块的决胜规则分为两层：**

### 第一层：分叉竞争平局时比较第一区块 ID

当两条竞争的分叉链在累积 PoH 权重上完全相等（即从分叉点开始，双方各高度的最优 MintHash 均相等）时，**比较分叉点之后第一个区块的 BlockID，字典序更小的一方胜出**。

```
ForkTieBreaker(forkA, forkB):
    firstBlockA = forkA.blocks[forkPoint+1].BlockID
    firstBlockB = forkB.blocks[forkPoint+1].BlockID
    if bytes.Compare(firstBlockA, firstBlockB) < 0:
        return forkA  // forkA 胜出
    else:
        return forkB  // forkB 胜出
```

### 第二层：同铸造者多签时，交易费更低的区块优先

当同一高度出现多个合法区块且铸造者相同（MintHash 相等），需要判断哪个区块进入本地择优视图时，**优先选择交易费总和更低的区块**。

这条规则的目的是：若铸造者同时签署了两个区块，则费用较低的区块胜出，这使得铸造者无法通过多签获得额外的高费用收益，从而形成对多签行为的直接经济威慑。

## Rationale（理由）

**关于第一层（BlockID 字典序）：**
1. **确定性**：BlockID 由区块内容的哈希计算得出，两个不同区块的 BlockID 以 2^256 级别的概率不会相等，字典序比较产生确定性结果。
2. **抗操控**：铸造者无法预先控制某个区块的 BlockID，也无法通过双签来获得字典序优势——双签反而暴露了身份，有损声誉。

**关于第二层（交易费低者优先）：**
1. **威慑多签**：铸造者多签的动机，一般是希望从更高交易费的区块中获取更多奖励。"费用低者胜出"直接反转了这种激励——多签不但无法获益，反而让自己的签名所附着的高费用区块失效。
2. **不影响正常激励**：正常情况下（一个铸造者只签一个区块），该规则根本不触发，不会干扰校验组选择高费用交易的通常激励。
3. **极罕见场景**：多签在协议经济模型下极为非理性，实践中几乎不会发生。此规则仅作为安全兜底，不会影响绝大多数情况下的区块生产行为。

## Consequences（影响）

- 需在 `docs/proposal/11.Endpoint-Conventions-And-Fork-Choice.md` 中补充分叉竞争平局的决胜规则（BlockID 字典序）。
- 需在 `docs/proposal/10.PoH-Consensus.md` 中补充同高度同铸造者竞争时"交易费低者优先"的规则，并说明其激励设计意图。
- `internal/consensus` 中 fork choice 函数需实现上述两层决胜逻辑。
- 需在 ADR-0011 中交叉引用本 ADR，说明两个 ADR 分别处理不同场景（ADR-0011 处理择优池内哈希碰撞，本 ADR 处理分叉竞争层的同值 MintHash）。

## References（参考）

- `docs/proposal/10.PoH-Consensus.md` — PoH 共识
- `docs/proposal/11.Endpoint-Conventions-And-Fork-Choice.md` — 端点共约与分叉选择
- `docs/plan/06-PoH-Consensus-And-Fork-Choice.md` — Task 3
- ADR-0011（铸凭哈希碰撞的择优池规则）
