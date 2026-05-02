# ADR-0006: 脚本初始 pass 状态

## Status（状态）

Accepted

## Context（背景）

Evidcoin 脚本 VM 使用 pass 状态机来决定脚本验证的最终结果。`PASS` 和 `CHECK` 指令可以设置此状态，`END` 指令以当前状态作为最终结果返回。

初始 pass 状态应为 `true` 还是 `false`，在至少 4 个文档中被标注为未决，且倾向于 `false`，但始终未被最终确认（OQ-013）。

两种选择的安全模型截然不同：
- **初始 false（fail-closed）**：空脚本默认拒绝，必须显式 PASS 才能通过。
- **初始 true（pass-through）**：空脚本默认通过，只有显式 CHECK(false) 才能阻止。

## Decision（决策）

**脚本初始 pass 状态为 `true`（通过）。**

即：若脚本中没有任何 `CHECK` 或 `PASS` 指令，默认执行结果为通过。

设计哲学：**只有设置了关卡，才可能阻止**。

## Rationale（理由）

1. **与锁定脚本语义匹配**：在 UTXO 模型中，锁定脚本是资产持有者**主动设置**的花费条件。持有者通过编写检查逻辑来保护资产，而非依赖"什么都不写就安全"的假设。初始状态为 true 更符合"我写的条件，你来满足"的直觉模型。
2. **简化常见模式**：标准签名验证脚本（P2PK/P2PKH）只需验证签名后 `CHECK(result)` 即可，无需额外的 `PASS` 初始化。
3. **空脚本语义明确**：一个没有锁定脚本的输出（如销毁输出，burn flag 已设），其空脚本自然通过验证，符合"无锁"的语义。

## Consequences（影响）

- 需在 `docs/proposal/09.Script-System.md` 中明确初始 pass 状态为 `true`。
- 需在 `docs/proposal/Instruction/0.Base-Constraints.md` 和 `6.Result-Instructions.md` 中更新相关描述。
- `internal/script` 的执行器初始化时将 passState 设为 `true`。
- 测试套件需覆盖：空脚本通过、仅 `CHECK(false)` 拒绝、`CHECK(true)` 后再 `CHECK(false)` 最终为拒绝（见 ADR-0007）。
- OQ-013 关闭。

## References（参考）

- `docs/proposal/09.Script-System.md` — 脚本系统
- `docs/proposal/Instruction/0.Base-Constraints.md` — 基础约束
- `docs/proposal/Instruction/6.Result-Instructions.md` — 结果指令
- `docs/plan/05-Script-System.md` — Task 4（执行状态机）
- `docs/plan/08-Open-Questions-And-Acceptance.md` — OQ-013
- ADR-0007（CHECK 覆盖语义，与本决策联动）
