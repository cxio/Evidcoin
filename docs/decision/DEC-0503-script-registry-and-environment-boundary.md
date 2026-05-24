# DEC-0503: Script Registry and Environment Boundary（脚本注册表与环境边界）

Status: Proposed

## Context（背景）

Conception 定义了环境指令、系统指令、函数指令、模块指令和扩展指令。但环境值编号、函数/模块注册表、公共验证与私有执行边界、外部引用目标缺失语义仍需冻结。

## Decision（决策）

建议注册表规则：

- opcode 是一级注册表，固定 1 字节。
- `ENV`、`IN`、`OUT`、`VALUE`、`FN`、`MO`、`EXT` 等子空间必须分别维护编号，不共享数值语义。
- 禁用指令仍保留编号，但公共验证静态检查拒绝执行。
- 新增注册项只能追加，不得复用已发布编号。

建议公共验证边界：

- 公共验证路径不得依赖本地时钟、外部输入、外部程序、私有扩展和网络查询的不确定结果。
- `SYS_TIME` 不得位于公共验证路径。
- `INPUT` 在公共验证节点视为隐式结束，不导入外部数据。
- `SHELL`、`EXT_PRIV` 在公共验证路径中非法。
- `GOTO`、`EMBED` 只能引用已确认且可验证的链上脚本。

建议环境命名：

- 区块推导时间使用 `BlockTime`。
- 交易时间戳使用 `TxTime`。
- 当前输入、来源输出、当前输出等环境必须显式区分。

## Rationale（理由）

脚本系统需要同时服务公共验证和私有中间件，必须明确哪些信息会影响共识。注册表分空间可避免环境值和函数编号混淆。

## Consequences（影响）

- 前期实现可只实现公共验证子集。
- 私有功能可以存在于源码和工具层，但不能影响交易合法性。
- `VALUE` 等禁用项解除前，不应进入任何链上标准脚本模板。

## Conception References（构想层依据）

- `docs/conception/6.脚本系统.md#缓存区和外部监听`
- `docs/conception/6.脚本系统.md#3个特例指令`
- `docs/conception/Instruction/13.环境指令.md`
- `docs/conception/Instruction/15.系统指令.md`
- `docs/conception/Instruction/16.函数指令.md`
- `docs/conception/Instruction/17.模块指令.md`
- `docs/conception/Instruction/18.扩展指令.md`

## Open Questions（开放问题）

- 各环境值、函数、模块和扩展的最终编号表。
- `GOTO` 目标缺失时是验证失败、脚本错误，还是可恢复 false。
- `INOUT` 禁用解除后是否允许公共验证路径触发网络查询。
