# DEC-0025: Script Cost Budget（脚本成本预算）

Status: Proposed

## Context

conception 已建立脚本安全框架：参数限定、执行成本预算、区块总成本限制和组队校验冻结机制。Decision 只补充预算计量方向，不重复主体规则。

## Decision

建议三层预算框架：

- 静态限制：脚本长度、栈高度、栈项大小、跳转次数和嵌入深度按 conception 执行。
- 运行成本：每条 opcode 有基础成本，按实际执行路径累计。
- 区块成本：区块内所有脚本运行成本总和不得超过当前区块成本上限。

建议边界：

- `SYS_TIME` 不得用于公共验证共识路径。
- `INPUT` 对公共验证节点等同于终止公共验证路径。
- `GOTO`、`EMBED`、`INOUT`、`CREDIT` 和模块调用应具有高于基础 opcode 的成本。

## Rationale

成本表需要经验调参，不能在 conception 未明确时伪造精确数值。

## Consequences

实现可先保留成本计数接口，但正式区块验证需要等待成本表冻结。

## Conception references

- `docs/conception/6.脚本系统.md`
- `docs/conception/Instruction/13.环境指令.md`
- `docs/conception/Instruction/15.系统指令.md`

## Open questions

- 每个 opcode 的基础成本。
- 区块总成本随高度或区块大小增长的函数。
- 组队校验冻结机制如何对成本预算签名确认。
