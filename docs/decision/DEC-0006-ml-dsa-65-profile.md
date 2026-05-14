# DEC-0006: ML-DSA-65 Profile（ML-DSA-65 使用配置）

Status: Proposed

## Context

项目目标使用 ML-DSA-65 作为后量子签名算法，但 conception 尚未固定签名编码、上下文字符串、密钥格式和库选择。

## Decision

建议配置如下：

- 算法为 ML-DSA-65。
- 签名输入必须是已编码的签名消息摘要或签名消息字节序列，不允许实现自行拼接字段。
- 上下文字符串为 `Evidcoin/ML-DSA-65/v1`。
- 公钥、私钥和签名均使用算法标准原始字节表示，包装容器负责长度校验。
- Go 实现优先使用标准库；若当前 Go 版本未提供，再使用 `github.com/cloudflare/circl`。

## Rationale

固定上下文字符串可降低跨协议签名复用风险。原始字节格式减少 DER 等外部编码差异。

## Consequences

签名消息 DEC 冻结前，本 DEC 不能独立形成完整签名测试向量。

## Conception references

- `docs/conception/附.交易.md`
- `docs/conception/blockchain.md`

## Open questions

- 标准库可用时的确切包路径和 API 仍待 Go 版本确认。
- 是否对多签公钥排序使用完整公钥或简单公钥哈希，需与 `FN_MPUBHASH` 一致。
