# ADR-0019: SYS_NULL 在解锁段的使用例外

## Status（状态）

Accepted

## Context（背景）

脚本 VM 的解锁段（unlock script）对可使用的指令码有严格限制：仅允许 opcode 0-50（基础指令集的一部分），以防止解锁段执行任意复杂逻辑。

然而 `SYS_NULL`（opcode 169）位于系统指令区间（164-169），超出了此范围。提案层将其标记为"解锁段的唯一例外"，但未充分解释例外的理由和安全影响（评审报告 M-8）。

## Decision（决策）

**维持 `SYS_NULL`（opcode 169）作为解锁段唯一的超范围例外**，允许在解锁段中使用。

### SYS_NULL 的语义

`SYS_NULL` 用于在解锁段中标记 NULL 点（SOURCE 零点），表示"从此位置起，以下部分为某个特定来源的脚本输入"。这允许脚本验证器灵活地从解锁段的任意位置提取指令序列，用于综合性的脚本验证（如多签、条件分支解锁等场景）。

### 安全分析

`SYS_NULL` 本身不执行任何计算，不访问任何外部状态，也不影响栈的内容——它只是一个位置标记（marker）。因此允许其在解锁段出现不引入任何额外的安全风险。

## Rationale（理由）

1. **逻辑归属合理**：`SYS_NULL` 在语义上属于系统级基础设施（标记点），而非数据操作指令，因此划归系统指令区间（opcode 164-169）是合适的。若将其重分配到 0-50 区间，会破坏指令码按功能类型划分的整体结构，且"重分配"会占用基础指令区的编码空间。
2. **功能价值明确**：在解锁段标记 NULL 点是一个重要功能，特别是在构造复杂的多签解锁逻辑时，`SYS_NULL` 允许将解锁段的不同部分关联到不同的锁定脚本片段进行验证。
3. **唯一性保证**：基础指令集（opcode 0-50）已经相当完备，覆盖了解锁段所需的所有数据压栈和基础操作。`SYS_NULL` 是解锁段唯一需要的超范围指令，可以确信不会再有更多例外。

### 实现建议

验证器在检查解锁段 opcode 时，应采用白名单策略而非范围检查：

```go
func isAllowedInUnlock(op byte) bool {
    return op <= 50 || op == SYS_NULL  // SYS_NULL = 169
}
```

注释应明确说明 `SYS_NULL` 是唯一例外，防止未来实现者误解为"系统指令区间均可用"。

## Consequences（影响）

- 需在 `docs/proposal/09.Script-System.md` 中补充对 `SYS_NULL` 例外的详细说明，包括其语义和安全分析。
- `internal/script` 的解锁段 opcode 验证逻辑需显式处理此例外（建议白名单而非范围检查）。
- 需在测试套件中添加：解锁段中的 `SYS_NULL` 合法、其他系统指令（如 `SYS_TIME`）在解锁段非法。

## References（参考）

- `docs/proposal/09.Script-System.md` — 脚本系统
- `docs/proposal/Instruction/15.System-Instructions.md` — 系统指令
- `docs/plan/05-Script-System.md` — Task 2/5
