# 脚本指令集提案索引（Instruction Proposals Index）

## 来源追溯

- `docs/conception/6.脚本系统.md` —— 脚本 VM 模型、分段与命名、资源限制、公共/私有验证边界。
- `docs/conception/Instruction/*.md`（`0.基本约束` + `1~18` 各类 + `AGENTS`）—— 0~18 类指令逐条构想与 opcode 基线。
- `docs/proposal/10.Script-System.md` —— 脚本系统总提案（本子目录的总则）。
- `DEC-0501`~`DEC-0505` —— 字节码编码、浮点 profile、注册表与环境边界、成本模型、失败语义与禁用指令。

## 定位

本子目录将脚本指令集从 conception 转化为提案规格，是第 10 章《脚本系统》的**指令逐条规格附录**。每个文件对应一类指令，固定可从 conception/DEC 追溯的语义、opcode、附参、实参、返回、边界与待决，不臆造尚未冻结的最终成本数值。

> **冻结基线〔DEC-0503〕：** 当前 conception 指令文档列出的 opcode、附参子空间、值名称为冻结基线；新增只能追加，不得复用已发布编号。

## 文件映射与 opcode 区间

| conception 源 | proposal 文件 | opcode 区间 | 数量 |
|----------------|---------------|-------------|------|
| `0.基本约束.md` | `0.Base-Constraints.md` | —（总则） | — |
| `1.值指令.md` | `1.Value-Instructions.md` | `[0-18]` | 19 |
| `2.截取指令.md` | `2.Capture-Instructions.md` | `[19-23]` | 5 |
| `3.栈操作指令.md` | `3.Stack-Operations.md` | `[24-34]` | 11 |
| `4.集合指令.md` | `4.Collection-Operations.md` | `[35-45]` | 11 |
| `5.交互指令.md` | `5.Interaction-Instructions.md` | `[46-50]` | 5 |
| `6.结果指令.md` | `6.Result-Instructions.md` | `[51-57]` | 7 |
| `7.流程控制.md` | `7.Flow-Control.md` | `[58-66]` | 9 |
| `8.转换指令.md` | `8.Conversion-Instructions.md` | `[67-79]` | 13 |
| `9.运算指令.md` | `9.Arithmetic-Instructions.md` | `[80-103]` | 24 |
| `10.比较指令.md` | `10.Comparison-Instructions.md` | `[104-111]` | 8 |
| `11.逻辑指令.md` | `11.Logic-Instructions.md` | `[112-115]` | 4 |
| `12.模式指令.md` | `12.Pattern-Instructions.md` | `[116-127]` | 12 |
| `13.环境指令.md` | `13.Environment-Instructions.md` | `[128-137]` | 10 |
| `14.工具指令.md` | `14.Tool-Instructions.md` | `[138-163]` | 26 |
| `15.系统指令.md` | `15.System-Instructions.md` | `[164-169]` | 6 |
| `16.函数指令.md` | `16.Function-Instructions.md` | `[170-224]` | 55 |
| `17.模块指令.md` | `17.Module-Instructions.md` | `[225-250]` | 26 |
| `18.扩展指令.md` | `18.Extension-Instructions.md` | `[251-253]` | 3 |

**合计：** 基础段 `[0-169]` = 170；函数段 `[170-224]` = 55；模块段 `[225-250]` = 26；扩展段 `[251-253]` = 3；**总计 254**。`254/255` 为系统保留，不在指令集范畴〔18.扩展指令〕。

> **未用保留位（占位，不分配指令）：** 14-15（值）、49（交互）、78（转换）、132（环境）、148-151 / 156-163（工具，含 8 个量子安全保留位）、167-168（系统）、182-222（函数，41 位）、227-249（模块）、252（扩展）。

## 共享约束

- 所有指令规格必须引用 `0.Base-Constraints.md`（前置说明）〔conception AGENTS〕。
- 公共验证路径必须确定性〔DEC-0503〕。
- 私有逻辑必须位于 `END`、公共 `INPUT` 或等价公共结束边界之后〔DEC-0505〕。
- opcode、附参、关联数据、运行时实参必须区分〔0.基本约束〕。
- 文本源码不是链上规范表示；链上为编译后字节码〔DEC-0501〕。
- 前期禁用指令：`SCRIPT`(17)、`VALUE`(18)、`EVAL`(138)、`INOUT`(131)〔DEC-0505〕。
- 成本数值（base_cost、动态因子、三层上限）未冻结，见第 10 章待决 C-6〔DEC-0504〕。
