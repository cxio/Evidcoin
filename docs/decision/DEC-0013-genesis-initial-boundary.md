# DEC-0013: Genesis Initial Boundary（创世初段边界）

Status: Proposed

## Context

最新 conception 已给出创世区块头、创世 Coinbase、启动期 #1/#2 逻辑和百日扩张边界。Decision 不再保留“创世块未定义”类旧问题，只记录仍需实现统一的初段边界。

## Decision

建议如下：

- 区块 0 为创世块，`Version=1`、`Height=0`、`PrevBlock=0`、`Stakes=0`。
- 创世块 `YearBlock` 字段存在且为全零，因为 `0 % 87661 == 0`。
- 创世块只包含一笔 Coinbase；关联 UTXO/UTCO 集指纹为全零，但 `CheckRoot` 仍按常规合并计算。
- 创世 Coinbase 总额为 `10_000_000_00 chx`（10 币），按校验组 8 币、铸凭者 2 币分配，不包含公共服务输出。
- 创世 Coinbase 省略 `HashInputs` 和 `Minter`，设置 `MintPKHash`，并保留铸造者对区块 `CheckRoot` 的签名。
- `currentHeight < 8` 时，评参区块高度固定为 0。
- `currentHeight < 240` 时，已确认交易均可作为铸凭交易；Coinbase 交易是否可作为铸凭交易暂不冻结。
- #1 号区块继续由创世 `MintPKHash` 铸造；#2 号区块开始可出现竞争者。
- 百日扩张期的公共服务奖励关闭规则来自 conception；公共服务激活边界见 DEC-0020。

## Rationale

创世与启动细节已由 conception 明确，Decision 只把影响实现和测试向量的边界集中列出。第 240 块前 Coinbase 是否可参与铸凭仍未被 conception 明确，应继续保持开放。

## Consequences

创世配置文件仍需提供具体公钥哈希、创世时间戳和 FreeData 文本。测试网可用不同创世配置，但必须显式声明。

## Conception references

- `docs/conception/blockchain.md`
- `docs/conception/1.共识-历史证明（PoH）.md`

## Open questions

- 第 240 块之前 Coinbase 交易能否参与铸凭竞争。
- 创世时间戳、创世 `MintPKHash`、校验组地址、铸凭者地址和 FreeData 由哪个发布工件承载。
