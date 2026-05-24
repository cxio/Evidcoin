# DEC-0505: Script Failure and Disabled Opcodes（脚本失败语义与禁用指令）

Status: Proposed

## Context（背景）

Conception 中“失败”“正常失败”“正常退出”“公共验证隐式 END”等术语存在混用；同时 `Instruction/AGENTS.md` 声明若干指令前期禁用，覆盖其它地方的允许约定。Decision 只补充执行状态机和禁用规则的实现边界，不重新裁决禁用清单。

## Decision（决策）

建议执行状态分为：

- `Running`：继续执行。
- `PassStop`：正常结束，返回当前 PASS 状态。
- `VerifyFail`：验证失败，交易不合法。
- `ScriptError`：脚本错误，交易不合法。
- `PrivateStop`：私有路径停止，不影响公共验证结果。

建议语义：

- `PASS false` 产生 `VerifyFail`。
- 类型错误、栈下溢、成本超限、非法 opcode 产生 `ScriptError`。
- `INPUT` 在公共验证路径中产生 `PassStop`，保留既有 PASS 状态。
- `END` 在公共验证节点产生 `PassStop`；私有节点可忽略并继续私有路径。
- 公共验证路径遇到 `SYS_TIME`、`SHELL`、`EXT_PRIV` 产生 `ScriptError`。

前期禁用指令清单已由 `docs/conception/Instruction/AGENTS.md` 以最高优先级明确，Decision 不另行裁决。当前清单为：

- `SCRIPT`
- `VALUE`
- `CALL`
- `EVAL`
- `SHELL`
- `INOUT`

Decision 只补充执行边界建议：

- 禁用不是“未实现则忽略”；当前协议有效交易不得依赖禁用指令。
- 私有工具可解析和显示禁用指令，但必须标注当前禁用状态。
- 若未来解除禁用，应由 conception 或新 DEC 指定协议版本或激活高度。

## Rationale（理由）

失败语义直接决定交易是否有效，必须比指令说明更高层地统一。禁用指令若被不同实现忽略或执行，会导致共识分裂。

## Consequences（影响）

- 标准脚本模板必须避开禁用指令。
- 禁用解除需要新的协议版本或明确激活高度。
- 示例文档中使用禁用指令的内容只能作为未来能力说明，不能作为当前有效脚本。

## Conception References（构想层依据）

- `docs/conception/6.脚本系统.md#缓存区和外部监听`
- `docs/conception/6.脚本系统.md#3个特例指令`
- `docs/conception/Instruction/AGENTS.md`
- `docs/conception/Instruction/0.基本约束.md`
- `docs/conception/Instruction/5.交互指令.md`

## Open Questions（开放问题）

- 禁用解除是否逐项激活，还是同一脚本版本统一激活。
- `INPUT` 正常停止时 PASS 初始值和 `CHECK` 状态如何共同影响结果。
- 私有路径产生的 `OUTPUT` 是否应在公共验证日志中完全忽略。
