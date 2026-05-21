# DEC-0019: Issuance Schedule（发行计划）

Status: Accepted

## Context

conception 已给出原始铸币发行阶段、每年区块数、基础单位换算和长期 3 币/块微通胀。Decision 只补充整数计算边界。

## Decision

- `1 币 = 10^8 chx`。
- 每年按 87661 个区块计算。
- 前三年每块分别为 10、20、30 币。
- 正式发行期从第 4 年开始，每块 40 币，每 2 年递减 20%。
- 第二阶段递减按 `chx` 精度进行整数计算，而不是按整币展示值取整。
- 当递减结果低于 `300_000_000 chx` 时，初期铸币结束。
- 长期微通胀阶段固定为 `300_000_000 chx/Block`，即 3 币/块。
- 创世块包含 10 币 Coinbase，分配见 DEC-0013。

## Rationale

`chx` 是链上最小金额单位，发行函数必须以 `chx` 计算。构想层表格按币展示只是简化示例，不能作为取整规则。

## Consequences

Coinbase 金额校验可以直接以 `chx` 实现。旧“1 币等于多少 chx 未定”和“按币为单位向下取整”的开放问题已关闭。

## Conception references

- `docs/conception/4.激励机制.md`
- `docs/conception/blockchain.md`
- `docs/conception/5.信用结构.md`

## Open questions

无。
