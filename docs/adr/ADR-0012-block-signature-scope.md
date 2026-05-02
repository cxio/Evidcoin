# ADR-0012: 区块铸造者签名消息范围

## Status（状态）

Accepted

## Context（背景）

在组队校验协作流程中（`docs/proposal/12.Team-Validation.md`），铸造者需要对区块进行签名以完成铸造。签名消息的范围影响铸造者与管理层之间的信任边界：

- **签名完整区块头**：铸造者需要验证完整的 CheckRoot（包含交易树根和状态根），对区块内容有更强的确认，但对普通铸造参与者（如围观者）要求更高的能力。
- **仅签名 CheckRoot 字段**：铸造者只需确认 CheckRoot 的值，不需要验证其来源。

评审报告（M-7）将此标记为未决，影响铸造者协作流程的安全性设计。

## Decision（决策）

**铸造者签名消息仅包含 `CheckRoot` 字段（48 字节）**，而非完整区块头。

签名消息构成：

```
SignMessage = domainTag || chainIdentityBytes || CheckRoot
```

其中：
- `domainTag` 遵循 ADR-0004（`Purpose = "BlockMintSign"` — 需在 ADR-0004 的对照表中补充此条目）。
- `chainIdentityBytes`：Protocol-ID、Chain-ID、Genesis-ID 的序列化，用于防重放。
- `CheckRoot`：48 字节，当前区块的 CheckRoot。

> **注意**：区块头的哈希计算（BlockID）包含 CheckRoot 字段，但不包含铸造者签名数据。签名仅用于铸造过程中的实时安全性，不进入区块头哈希。

## Rationale（理由）

1. **普通铸造参与者能力有限**：拥有铸凭交易的铸造者（包括围观者）通常是普通用户，他们实际上无法独立验证区块头内所有字段的正确性（如 Stakes 字段值、TransactionTreeRoot 的来源），因此要求其签名完整区块头没有实质意义。
2. **区块头字段自我约束**：区块头所有字段都有其他节点的独立验证——高度、PrevBlock、CheckRoot 的正确性由全网共识保证；管理层若构造错误的区块头，其他校验节点会拒绝该区块，管理层自身将蒙受损失。因此铸造者签名约束至 CheckRoot 已足够。
3. **签名的实时安全性定位**：区块链的长期安全性（防历史重构）不依赖签名，而依赖 PoH 共识的哈希链结构。签名只负责末端有限长度区块链的实时安全性（分叉选择窗口内）。超出此窗口的旧区块签名已无实质安全作用。
4. **CheckRoot 已充分代表区块内容**：CheckRoot 是交易树根、UTXO 状态根、UTCO 状态根的组合摘要。铸造者签名 CheckRoot，实际上间接确认了区块包含的所有交易和状态。

## Consequences（影响）

- 需在 `docs/proposal/12.Team-Validation.md` 中明确区块签名消息范围为 CheckRoot。
- 需在 ADR-0004 的 domain tag 对照表中补充 `BlockMintSign` 条目。
- `internal/validation` 和 `internal/consensus` 中，铸造者签名验证函数的消息构建使用此规则。
- 铸造者协作流程（10 步）中"铸造者验证 Coinbase"步骤后的签名步骤，签名对象为 CheckRoot。

## References（参考）

- `docs/proposal/12.Team-Validation.md` — 组队校验
- `docs/proposal/05.Blockchain-Core.md` — 区块头结构
- `docs/plan/07-Team-Validation-Services-Incentives.md` — Task 4
- ADR-0004（domain tag 格式）
