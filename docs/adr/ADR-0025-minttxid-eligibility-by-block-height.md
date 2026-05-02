# ADR-0025: mintTxID 资格判定基准为区块高度

## Status（状态）

Accepted

## Context（背景）

PoH 共识中，铸凭交易（mint proof transaction）的资格窗口限定为 `[-28 ~ -80000]`，即铸凭交易必须出现在足够靠前（但不太新）的位置，以证明历史存在性。

评审报告（H-5）提出了一个安全性疑问：此资格窗口的判定基准是使用**铸凭交易自身的 Timestamp 字段**，还是**铸凭交易实际所在区块的高度**？若使用 Timestamp，则用户可以伪造 Timestamp 以绕过资格限制。

## Decision（决策）

**铸凭交易 ID（mintTxID）的资格窗口 `[-28 ~ -80000]` 基于铸凭交易实际打包所在区块的高度（`blockHeight`）来判定**，而非铸凭交易自身携带的 Timestamp 字段。

具体规则：

```
资格满足条件：
  当前区块高度 - blockHeight(mintTxID) ∈ [28, 80000]
  等价地：
  (currentHeight - 28) >= blockHeight(mintTxID) >= (currentHeight - 80000)
```

其中 `blockHeight(mintTxID)` 指的是包含该铸凭交易的区块的高度，这是链上可信的不可篡改数据。

## Rationale（理由）

1. **Timestamp 不可信**：交易 Timestamp 由交易创建者设置，在现有协议设计中用户具有一定的自由度。使用 Timestamp 作为资格判定依据会引入攻击面——攻击者可伪造 Timestamp 使一笔近期创建的交易伪装成"历史"铸凭交易，破坏 PoH 的核心安全假设。
2. **区块高度是链上确定性数据**：某笔交易所在区块的高度由矿工打包时确定，经全网共识后不可更改。以区块高度为判定基准，保证了资格判断的确定性和不可伪造性。
3. **与 PoH 设计意图一致**：PoH（历史证明）的核心思想是利用历史上已发生的事实。以"交易实际发生的区块高度"来衡量历史距离，直接对应了"证明事件确实发生在过去某个时间区间"这一语义。
4. **消除时间戳攻击面**：明确说明 Timestamp 对资格判定无影响，可指导实现者不要在资格校验逻辑中使用交易的 Timestamp 字段。

## Consequences（影响）

- 需在 `docs/proposal/10.PoH-Consensus.md` 中将所有描述资格窗口判定的语句修改为"基于交易所在区块高度"，若有任何使用"交易 Timestamp"的描述需一律纠正。
- 需在 `docs/plan/06-PoH-Consensus-And-Fork-Choice.md` 中对应更新相关 Task 描述。
- `internal/consensus` 中铸凭资格校验函数，须从链状态中查询铸凭交易所在的区块高度，而非读取交易的 Timestamp 字段。
- 测试套件需包含：Timestamp 可随意设置但不影响资格判定的测试用例。

## References（参考）

- `docs/proposal/10.PoH-Consensus.md` — PoH 共识
- `docs/plan/06-PoH-Consensus-And-Fork-Choice.md` — PoH 共识实施计划
- `docs/plan/08-Open-Questions-And-Acceptance.md` — H-5
