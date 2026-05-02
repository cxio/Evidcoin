# ADR-0022: CheckRoot 承诺的 UTXO/UTCO 状态时间点

## Status（状态）

Accepted

## Context（背景）

区块头中的 `CheckRoot` 字段用于承诺该区块的完整性，其计算涉及交易树根与 UTXO/UTCO 指纹的组合。

在提案层（`docs/proposal/05.Blockchain-Core.md`）和实施计划（`docs/plan/02-Blockchain-Core.md`）中，对"UTXO/UTCO 指纹所反映的是哪个时刻的状态"存在描述不一致的问题：

- **构想层**（`docs/conception/blockchain.md`）明确指出：CheckRoot 中的 UTXO/UTCO 指纹应当是**上一个区块执行后的状态**（即当前区块高度 `H` 的 CheckRoot 承诺的是高度 `H-1` 之后的 UTXO/UTCO 集合抓取）。
- **提案层/实施计划**中部分描述暗示 CheckRoot 可能包含当前区块处理后的状态，与构想层直接冲突。

此问题被标记为阻塞性未决项（C-1），因为该设计决策影响区块链的链式约束机制，直接关系到长期安全性。

## Decision（决策）

**`CheckRoot` 中的 UTXO/UTCO 指纹承诺的是当前区块的前一个区块（高度 `H-1`）执行后的状态**，即"前置状态承诺"语义。

```
CheckRoot[H] = Commit(
    TransactionTreeRoot[H],      // 当前区块 H 的交易树根
    UTXO_Fingerprint[H-1],       // 高度 H-1 之后的 UTXO 状态
    UTCO_Fingerprint[H-1],       // 高度 H-1 之后的 UTCO 状态
)
```

### 创世区块的特殊情况

创世区块（高度 `#0`）没有前一区块，其 CheckRoot 中的 UTXO/UTCO 指纹为空集的哈希：

```
UTXO_Fingerprint[-1] = Hash(nil)   // 空状态
UTCO_Fingerprint[-1] = Hash(nil)   // 空状态
```

## Rationale（理由）

1. **链式约束的必要性**：若 `CheckRoot[H]` 承诺的是当前区块 `H` 处理后的状态，则验证者必须先完整执行区块 `H` 才能计算出 `CheckRoot[H]`，这与 CheckRoot 作为"区块打包前的质量承诺"的设计意图矛盾——铸凭者对 CheckRoot 进行签名时，理应能在执行前独立验证其合法性。
2. **多链耦合保护**：以前置状态承诺，使得 `CheckRoot[H]` 间接包含了 `CheckRoot[H-1]` 的状态影响（因 `H-1` 的 UTXO 变化来源于 `H-1` 的交易执行）。这构成了一条难以伪造的历史哈希链，提供了多链耦合的安全保护。
3. **与构想层设计一致**：构想层是三层文档体系中优先级最高（Tier 1）的规范来源，提案层的错误描述需要被纠正为与此一致。
4. **验证可并行化**：区块广播时包含 `CheckRoot`，验证节点可独立计算（使用自身持有的前置状态），在接收到新区块前即可预先准备，提升验证并发性。

## Consequences（影响）

- 需纠正 `docs/proposal/05.Blockchain-Core.md` 中对 CheckRoot 计算方式的描述，明确为"前置状态承诺"语义。
- 需在 `docs/plan/02-Blockchain-Core.md` 中对应更新 Task 描述，确保实现代码使用 `H-1` 时刻的状态快照。
- `internal/blockchain` 中区块验证函数在计算并校验 CheckRoot 时，须从状态管理层取出**上一高度**的 UTXO/UTCO 指纹，而非当前高度执行后的结果。
- 创世区块的实现需显式处理空状态指纹（`h == 0` 时使用空集哈希）。

## References（参考）

- `docs/conception/blockchain.md` — 区块链构想（CheckRoot 设计原始意图）
- `docs/proposal/05.Blockchain-Core.md` — 区块链核心提案
- `docs/plan/02-Blockchain-Core.md` — 区块链核心实施计划
- `docs/plan/08-Open-Questions-And-Acceptance.md` — C-1
