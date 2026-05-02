# ADR-0005: 脚本 VM 浮点数确定性策略

## Status（状态）

Accepted

## Context（背景）

Evidcoin 脚本 VM（`docs/proposal/09.Script-System.md`）支持 `Float` 作为运行时值类型，算术指令集（Instruction `9.Arithmetic-Instructions`）也允许浮点运算。

然而 IEEE 754 浮点运算存在平台相关性风险：
- 不同硬件的 NaN payload 可能不同
- 部分平台可能使用扩展精度（80 位 x87）
- 舍入模式在某些编译器优化下可能变化

如果浮点计算结果参与公共验证逻辑，不同节点可能得出不同的 pass/fail 结论，破坏共识确定性。

## Decision（决策）

**脚本 VM 中的 Float 类型遵循以下规范**：

1. **精度**：采用 IEEE 754 **binary64**（双精度浮点），即 Go 的 `float64`。
2. **舍入模式**：**round-to-nearest-even**（银行家舍入，IEEE 754 默认模式）。
3. **NaN 规范化**：所有 NaN 值统一规范化为 **quiet NaN**，且 NaN payload 统一为 `0`（即 `0x7FF8000000000000`）。禁止 signaling NaN 出现在栈或实参区中。
4. **Infinity**：正负无穷（`+Inf`/`-Inf`）保留其语义，不做特殊处理。

### 对公共验证的额外约束

在公共验证路径（`EndedForPublicValidation` 状态前）：
- 浮点运算结果**可以存在于栈中**，但不得作为 `CHECK`/`PASS` 的直接参数（须先转换为整数或布尔值）。
- 实现时建议：若 Float 值直接参与 `CHECK`，验证器应将其视为脚本错误而非通过。

## Rationale（理由）

IEEE 754 binary64 在现代主流平台（x86-64、ARM64）上的实现已高度一致，在 round-to-nearest-even 模式下，同一算术表达式在不同平台产生相同结果的概率极高。NaN 规范化消除了唯一已知的平台差异来源（NaN payload）。

相比于完全禁止浮点运算（会限制脚本的数值计算能力），此策略在实用性和确定性之间取得了合理平衡。

## Consequences（影响）

- 需在 `docs/proposal/09.Script-System.md` 中补充 Float 类型的精度和 NaN 规范说明。
- `internal/script` 的值系统实现中，在 Float 值入栈前需规范化 NaN。
- 需在测试套件中添加 NaN 规范化的单元测试（如 `math.NaN()` 入栈后取出应得到 quiet NaN）。
- OQ（浮点确定性）关闭。

## References（参考）

- `docs/proposal/09.Script-System.md` — 脚本系统
- `docs/proposal/Instruction/9.Arithmetic-Instructions.md` — 算术指令
- `docs/plan/05-Script-System.md` — Task 3（值系统）
