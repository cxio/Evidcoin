# ADR-0007: CHECK 指令覆盖语义

## Status（状态）

Accepted

## Context（背景）

脚本 VM 的 `CHECK` 指令用于设置 pass 状态。在多个文档中，对于 `CHECK` 是否可以双向覆盖 pass 状态的描述不一致：

- 是否允许 `true → false`？（即已通过后能否被后续 CHECK 推翻）
- 是否允许 `false → true`？（即已拒绝后能否被后续 CHECK 挽救）

此语义直接影响多条件验证脚本的编写方式，例如多签脚本、条件分支验证等场景。

## Decision（决策）

**`CHECK` 可按检查结果任意覆盖 pass 状态**，即：

- `false → true`：已拒绝的状态可被 `CHECK(true)` 改为通过。
- `true → false`：已通过的状态可被 `CHECK(false)` 改为拒绝。

**最终态决定最终结果**：脚本执行到 `END`（或等效终止点）时，当前 pass 状态就是最终验证结论。

## Rationale（理由）

1. **灵活支持复杂验证逻辑**：双向覆盖允许脚本编写者使用多个 `CHECK` 指令构建复杂的条件组合，而无需依赖变通手段（如将所有结果 AND 后一次性 CHECK）。
2. **编程模型清晰**：`CHECK(expr)` 语义等价于"**此处的验证结果是 expr**"，直觉上就是覆盖式的。开发者不需要思考"当前状态是什么，我的 CHECK 有没有生效"。
3. **与初始状态（ADR-0006）联动一致**：初始状态为 `true`（见 ADR-0006），任意 `CHECK(false)` 都能阻止通过；反之 `CHECK(true)` 也能在任何时候重置为通过。这使得"卫兵模式"（先 CHECK 前置条件，后 CHECK 主条件）自然可行。

## Consequences（影响）

- 需在 `docs/proposal/Instruction/6.Result-Instructions.md` 中明确 CHECK 的覆盖语义。
- 需在 `docs/proposal/09.Script-System.md` 的执行状态机描述中更新相关说明。
- `internal/script` 的 CHECK 指令实现中，直接将 passState 赋值为操作数，不做任何"防覆盖"保护。
- 需在测试套件中覆盖：连续 CHECK(true) → CHECK(false)、连续 CHECK(false) → CHECK(true) 两种翻转场景。

## References（参考）

- `docs/proposal/09.Script-System.md` — 脚本系统
- `docs/proposal/Instruction/6.Result-Instructions.md` — 结果指令
- `docs/plan/05-Script-System.md` — Task 4/5
- ADR-0006（脚本初始 pass 状态，与本决策联动）
