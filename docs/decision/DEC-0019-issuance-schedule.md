# DEC-0019: Issuance Schedule（发行计划）

Status: Accepted

## Context

conception 已给出原始铸币发行阶段、每年区块数和长期 3 币/块微通胀。Decision 只补充整数计算边界。

## Decision

- 每年按 87661 个区块计算。
- 前三年每块分别为 10、20、30 币。
- 正式发行期从第 4 年开始，每块 40 币，每 2 年递减 20%，按币为单位向下取整。
- 当递减结果到 3 币/块后，不再递减，长期保持 3 币/块。
- 币到 `chx` 的换算比例需由基础单位规范确认；在确认前，发行函数以“币”为展示单位。

## Rationale

以币为单位取整来自 conception，可避免后期微量持续递减。

## Consequences

Coinbase 金额校验需要先取得基础单位换算比例。旧 `Coin/chx` 比例不再单列为 Decision，因缺少 conception 明确依据。

## Conception references

- `docs/conception/4.激励机制.md`

## Open questions

- 1 币等于多少 `chx` 尚需 conception 或基础单位规范裁决。
- 创世块是否包含发行奖励需由创世规范定义。
