# Instruction Proposals（脚本指令提案索引）

## 来源构想

- `docs/conception/6.脚本系统.md`：脚本 VM、资源限制、公共验证和私有监听边界。
- `docs/conception/Instruction/*.md`：0-18 类指令构想。
- `docs/proposal/09.Script-System.md`：脚本系统总 Proposal。

## 目标

本目录将脚本指令集从构想层转化为 Proposal 规格。每个文件对应一类指令，只固定可从构想层追溯的语义、边界、风险和未决问题，不伪造尚未确认的最终 bytecode 细节。

## 文件映射

| 构想文件 | Proposal 文件 | 范围 |
|----------|---------------|------|
| `0.基本约束.md` | `0.Base-Constraints.md` | 指令结构、实参、返回、错误、公共验证边界。 |
| `1.值指令.md` | `1.Value-Instructions.md` | `0-18`，运行时值和字面量。 |
| `2.截取指令.md` | `2.Capture-Instructions.md` | `19-23`，返回值截取与取参来源。 |
| `3.栈操作指令.md` | `3.Stack-Operations.md` | `24-34`，数据栈操作。 |
| `4.集合指令.md` | `4.Collection-Operations.md` | `35-45`，Slice、Dict、Module/Object 集合操作。 |
| `5.交互指令.md` | `5.Interaction-Instructions.md` | `46-50`，INPUT/OUTPUT/BUFDUMP/PRINT。 |
| `6.结果指令.md` | `6.Result-Instructions.md` | `51-57`，PASS、CHECK、GOTO、EMBED、EXIT、RETURN、END。 |
| `7.流程控制.md` | `7.Flow-Control.md` | `58-66`，IF、SWITCH、EACH、BLOCK 等。 |
| `8.转换指令.md` | `8.Conversion-Instructions.md` | `67-79`，类型转换。 |
| `9.运算指令.md` | `9.Arithmetic-Instructions.md` | `80-103`，数学、位运算、复制和删除。 |
| `10.比较指令.md` | `10.Comparison-Instructions.md` | `104-111`，比较和范围。 |
| `11.逻辑指令.md` | `11.Logic-Instructions.md` | `112-115`，布尔聚合。 |
| `12.模式指令.md` | `12.Pattern-Instructions.md` | `116-127`，脚本/字节模式匹配。 |
| `13.环境指令.md` | `13.Environment-Instructions.md` | `128-137`，环境取值和脚本变量。 |
| `14.工具指令.md` | `14.Tool-Instructions.md` | `138-163`，EVAL、复制、正则、随机、MAP/FILTER 等。 |
| `15.系统指令.md` | `15.System-Instructions.md` | `164-169`，SYS_TIME、SYS_AWARD、SYS_CHKPASS、SYS_NULL。 |
| `16.函数指令.md` | `16.Function-Instructions.md` | `170-224`，编码、Hash、地址、签名等函数。 |
| `17.模块指令.md` | `17.Module-Instructions.md` | `225-250`，标准模块引用。 |
| `18.扩展指令.md` | `18.Extension-Instructions.md` | `251-253`，公共和私有扩展。 |

## 共享约束

- 所有指令 Proposal 必须引用 `0.Base-Constraints.md`。
- 公共验证路径必须确定性。
- 私有监听逻辑必须位于 `END`、隐式 `INPUT` 或等价公共结束边界之后。
- opcode、附参、关联数据和运行时实参必须区分。
- 文本源码不是链上规范表示。
- 未确认 varint、signed integer、domain tag、地址文本编码和成本公式前，不得生成最终协议测试向量。
