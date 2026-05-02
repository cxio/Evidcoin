# ADR-0010: Coinbase 交易 HashInputs 计算规则

## Status（状态）

Accepted

## Context（背景）

普通交易的 `HashInputs` 字段由 LeadInput 和 RestInputs 的哈希计算得出，用于构成 `TxHeader`，进而计算 `TxID`。

Coinbase 交易没有输入，因此 `HashInputs` 的计算方式无法沿用普通交易的逻辑。此问题被标记为阻塞性未决项（OQ-009），因为 Coinbase TxID 是交易树的第 0 项，其正确性直接影响 `TransactionTreeRoot` 和 `CheckRoot` 的验证。

## Decision（决策）

**Coinbase 的 `HashInputs` 计算公式**：

```
HashInputs = BLAKE3-256("Evidcoin:CoinbaseInputs:v1\x00" || CanonicalBytes(blockHeight))
```

其中：
- 前缀字符串 `"Evidcoin:CoinbaseInputs:v1\x00"` 为 domain tag，遵循 ADR-0004 规范（`Purpose = "CoinbaseInputs"`）。
- `blockHeight` 为当前区块高度，以规范化大端序 uint64（8 字节）编码。
- 输出为 32 字节 `Hash32`（TreeHash 类型）。

### 完整 Coinbase TxHeader 构建

```
TxHeader {
    Version:      <Coinbase 专用版本号>
    HashInputs:   BLAKE3-256(domainTag || bigEndian(blockHeight))   // 32B
    HashOutputs:  <正常计算输出树根>                                  // 32B
    Timestamp:    <区块确定性时间戳，单位毫秒>
}
TxID = SHA3-384(TxHeader.CanonicalBytes())
```

## Rationale（理由）

1. **与区块高度绑定**：Coinbase 在语义上是对特定高度区块的铸造奖励，以高度作为 HashInputs 的核心输入，确保不同高度的 Coinbase 有不同的 TxID，防止跨区块重放。
2. **domain tag 保护**：前缀 domain tag（遵循 ADR-0004）确保此计算结果不会与普通交易的任何 Hash 值发生语义混淆。
3. **BLAKE3-256 一致性**：与普通交易的 `LeadHash`/`RestHash` 使用同一算法（BLAKE3-256），保持编码层的一致性。
4. **确定性**：区块高度为确定性值，计算结果在所有节点完全一致。

## Consequences（影响）

- 需在 `docs/proposal/06.Transaction-Model.md` 中为 Coinbase 补充 HashInputs 的特殊计算规则。
- `internal/tx` 的 Coinbase 构建函数需实现此计算逻辑。
- 需在测试套件中提供测试向量：给定高度 1、高度 0 和 uint64 最大高度，验证 HashInputs 的输出值。
- OQ-009 关闭。

## References（参考）

- `docs/proposal/06.Transaction-Model.md` — 交易模型
- `docs/plan/03-Transaction-And-Units.md` — Task 9
- `docs/plan/08-Open-Questions-And-Acceptance.md` — OQ-009
- ADR-0004（domain tag 格式）
