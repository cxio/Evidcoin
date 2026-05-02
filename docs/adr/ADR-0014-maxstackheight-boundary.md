# ADR-0014: MaxStackHeight 边界检查语义

## Status（状态）

Accepted

## Context（背景）

构想层（`docs/conception/6.脚本系统.md`）对脚本栈高度的约束使用"小于"符号（`< 256`），而提案层（`docs/proposal/09.Script-System.md`）和 `AGENTS.md` 则定义常量 `MaxStackHeight = 256`。

两者存在潜在的 off-by-one 差异：
- 若实现为 `stackHeight > MaxStackHeight`（即 `> 256`）则允许 256 个元素，与构想层的 `< 256`（最多 255 个）相差 1。
- 若实现为 `stackHeight >= MaxStackHeight`（即 `>= 256`）则允许 255 个元素，与构想层一致。

## Decision（决策）

**遵循构想层的原始限制，`MaxStackHeight` 系列常量均按"小于（`<`）"语义解读**：

即常量值定义时，如构想层用 `< N` 表达上限，则提案层和实现层的对应常量取值为 `N - 1`，以保持语义一致。

具体修正（所有受此规则影响的常量）：

| 常量 | 构想层约束 | 修正后常量值 | 拒绝条件 |
|------|-----------|-------------|---------|
| `MaxStackHeight` | `< 256` | `255` | `stackHeight > 255` |
| `MaxStackItem` | `< 1024` | `1023` bytes | `itemSize > 1023` |
| `MaxLockScript` | `< 1024` | `1023` bytes | `scriptLen > 1023` |
| `MaxUnlockScript` | `< 4096` | `4095` bytes | `scriptLen > 4095` |

> **注意**：`AGENTS.md` 中列出的常量值（256、1024 等）需同步更新。

## Rationale（理由）

构想层是最权威的设计来源（三层文档体系中 Tier 1），当构想层使用明确的 `< N` 约束时，不应在提案层转化时引入歧义。将常量定义为 `N - 1` 并使用 `>` 作为拒绝条件，保持了与构想层语义的严格一致性，避免了实现者对常量含义的误解。

## Consequences（影响）

- 需在 `docs/proposal/09.Script-System.md` 和 `docs/proposal/03.Identifiers-And-Constants.md` 中将相关常量值更新为修正值（255、1023 等）。
- 需同步更新 `AGENTS.md` 中的常量列表。
- `internal/script` 和相关包在实现资源检查时，拒绝条件为 `> 常量值`（而非 `>= 常量值`）。
- 需在测试套件中明确测试边界值：255 个栈元素合法，256 个被拒绝。

## References（参考）

- `docs/conception/6.脚本系统.md` — 脚本系统构想
- `docs/proposal/09.Script-System.md` — 脚本系统提案
- `docs/proposal/03.Identifiers-And-Constants.md` — 常量定义
- `AGENTS.md` — 项目常量列表
