# DEC-0024: 脚本 Float 派生语义

## Status（状态）

Proposed

## Context（背景）

`DEC-0017` 已固定脚本 VM `Float` 的基础确定性要求，并禁止 Float 直接作为 `PASS` 或 `CHECK` 的实参。`conception/Instruction/` 中仍有 `FLOAT`、`CMPFLO`、`WITHIN`、`RANGE` 和 `MO_MATH` 等可能产生或消费 Float 的指令，需要补充公共验证路径中的派生约束。

## Decision（决策）

本节为候选规则，状态转为 Accepted 前不得作为最终共识依据。

建议采用以下派生语义：

- `FLOAT` 转换必须只接受可确定转换的 `Int`、规范十进制 `String` 或已规范化 `Float`。
- `FLOAT` 对非法字符串、溢出、signaling NaN 或实现相关格式必须失败。
- 所有 Float 运算结果必须继续应用 `DEC-0017` 的 NaN 规范化规则。
- `CMPFLO` 输出必须是 `Int` 或 `Bool`，不得输出 Float。
- `CMPFLO` 遇到 NaN 时必须失败，不得给出排序结果。
- `WITHIN` 若任一比较实参为 Float，则必须使用 `CMPFLO` 同等比较语义；存在 NaN 时失败。
- `RANGE` 若起点或步长为 Float，则生成序列只可作为中间数据；该序列不得直接进入 `PASS` 或 `CHECK`。
- `MO_MATH` 在公共验证中只允许确定性、平台无关的函数；超越函数、随机函数、依赖本地环境或可能因数学库差异产生不同结果的函数不得用于公共验证。
- Float 比较产生的 `Bool` 或 `Int` 可以进入 `PASS` 或 `CHECK`，但 Float 原值不得直接进入。

待冻结参数：

- 规范十进制 `String` 到 Float 的语法集合。
- `MO_MATH` 在公共验证中允许的函数白名单。
- Float 到 `Int`、`Bool`、`String` 的转换规则和溢出行为。
- `RANGE` 对 Float 步长累计误差的规范化方式。

## Rationale（理由）

- 禁止 Float 直接通关不足以覆盖由 Float 派生出的比较和范围逻辑。
- 允许比较结果进入 `PASS` 或 `CHECK` 保留 Float 的实用性，同时把最终通关值限制为离散类型。
- 对 `MO_MATH` 保守处理可避免不同平台数学库造成共识分歧。

## Consequences（影响）

- 在待冻结参数确定前，公共验证脚本应避免依赖 Float 派生结果作为关键授权条件。
- VM 需要区分 Float 原值、Float 派生离散结果和普通离散值的验证边界。
- 后续测试应覆盖 NaN、Inf、`CMPFLO`、`WITHIN`、Float `RANGE` 和 `MO_MATH` 禁用函数。

## Conception Relationship（与构想关系）

- 补充 `conception/6.脚本系统.md` 和 `conception/Instruction/` 中 Float 相关指令的公共验证边界。
- 不改变脚本支持 Float 类型，也不改变 `DEC-0017` 的基础确定性规则。
