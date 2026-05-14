# DEC-0013: Genesis Initial Boundary（创世初段边界）

Status: Proposed

## Context

conception 明确最初 8 个区块评参区块取创世块，第 240 块之前铸凭交易高度合法性可灵活处理，并描述百日扩张期。仍需固定实现边界，避免初段链出现多种合法解释。

## Decision

建议如下：

- `currentHeight < 8` 时，评参区块高度固定为 0。
- `currentHeight < 240` 时，已确认交易均可作为铸凭交易，但 Coinbase 交易是否可作为铸凭交易暂不冻结。
- 百日扩张期的公共服务奖励关闭规则来自 conception；公共服务激活边界见 DEC-0020。
- 创世块和初段私钥扩张交易的具体内容不由 Decision 伪造，应由创世规范单独给出。

## Rationale

初段需要比正常 PoH 更少历史依赖，但 Decision 不能替作者创建创世分配和初始身份规则。

## Consequences

创世配置文件或创世规范仍是实现前置条件。测试网可用不同创世配置，但必须显式声明。

## Conception references

- `docs/conception/1.共识-历史证明（PoH）.md`

## Open questions

- 第 240 块之前 Coinbase 交易能否参与铸凭竞争。
- 百日扩张期的单输出 Coinbase 是否允许包含自由数据。
- 创世块时间戳、初始输出和内置公告公钥均未定义。
