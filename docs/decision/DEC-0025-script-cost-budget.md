# DEC-0025: Script Cost Budget（脚本成本预算）

Status: Proposed

## Context

conception 已建立脚本安全框架：参数限定、执行成本预算、区块总成本限制和组队校验冻结机制。Decision 只补充预算计量方向，不伪造具体数值。

## Decision

建议三层预算框架：

- 静态限制：脚本长度、栈高度、栈项大小、跳转次数和嵌入深度按 conception 执行。
- 运行成本：每条 opcode 有基础成本，按实际执行路径累计。
- 区块成本：区块内所有脚本运行成本总和不得超过当前区块成本上限。

建议边界：

- `SYS_TIME` 和 `ENV{Timestamp}` 不得用于公共验证共识路径。
- `INPUT` 对公共验证节点等同于终止公共验证路径。
- `GOTO`、`EMBED`、`SCRIPT`、`EVAL`、`CALL`、`INOUT`、`CREDIT`、`SOURCE`、`MODEL`、`MATCH`、正则、哈希、签名、模块调用、`PACK`、深拷贝、集合迭代、随机数、`SHELL` 和扩展指令应具有高于基础 opcode 的成本或显式禁用规则。
- `MODEL` 中的 `...` 匹配必须有最大回溯或线性化成本上限。
- `SCRIPT`、`VALUE`、`CALL`、`EVAL`、`SHELL` 当前属于构想层指令说明中的前期禁用项；正式实现需明确这是阶段性实现禁用还是协议禁用。
- 公共验证路径中，私有扩展不得影响 `PASS/FAIL` 结果。

## Rationale

成本表需要经验调参，不能在 conception 未明确时伪造精确数值。高成本清单先列出需要特殊计费或禁用的类别，避免实现只给外部脚本计费而漏掉正则、签名、哈希和动态执行。

## Consequences

实现可先保留成本计数接口，但正式区块验证需要等待成本表冻结。脚本测试需要覆盖静态限制、公共验证路径裁剪和高成本指令拒绝/计费边界。

## Conception references

- `docs/conception/6.脚本系统.md`
- `docs/conception/Instruction/12.模式指令.md`
- `docs/conception/Instruction/13.环境指令.md`
- `docs/conception/Instruction/14.工具指令.md`
- `docs/conception/Instruction/15.系统指令.md`
- `docs/conception/Instruction/16.函数指令.md`
- `docs/conception/Instruction/17.模块指令.md`
- `docs/conception/Instruction/18.扩展指令.md`

## Open questions

- 每个 opcode 的基础成本。
- 区块总成本随高度或区块大小增长的函数。
- 组队校验冻结机制如何对成本预算签名确认。
- 前期禁用指令是协议禁用、默认策略禁用还是实现阶段禁用。
