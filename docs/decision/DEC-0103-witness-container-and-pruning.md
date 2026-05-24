# DEC-0103: Witness Container and Pruning（见证容器与剪枝）

Status: Proposed

## Context（背景）

Conception 将见证信息从交易 ID 中排除，并允许后期剪枝。但见证容器格式、标准验证见证与定制解锁脚本的边界、Coinbase 签名保留规则尚未冻结。

## Decision（决策）

建议每个输入拥有一个见证容器：

```text
Witness = varint(item_count) || WitnessItem*
WitnessItem = type byte || bytes
```

建议标准 item 类型：

- `0x01`：验证类别。
- `0x02`：授权标记。
- `0x03`：签名。
- `0x04`：公钥。
- `0x05`：补全公钥哈希。
- `0x06`：解锁脚本外部数据。

剪枝建议：

- 普通交易输入见证可剪枝。
- 解锁脚本本身参与输入根，不属于可剪枝见证。
- Coinbase 交易签名可在常规归档中保留，但长期共识最小验证不依赖它。
- 创世区块铸造者对 `CheckRoot` 的签名必须保留，用于链根锚定。
- 择优凭证中对 `mintHash` 的签名不属于可剪枝见证，必须随 Coinbase 数据保留。

## Rationale（理由）

见证容器按输入分组，便于验证时定位，也便于剪枝。把择优凭证签名排除在剪枝范围外，符合 conception 对铸造资格证明的要求。

## Consequences（影响）

- TxID 不随见证变化。
- Blockqs 或归档服务需要声明是否返回完整见证。
- 定制验证若把签名数据放入解锁脚本，该数据计入交易体且不可剪枝。

## Conception References（构想层依据）

- `docs/conception/blockchain.md#关于见证信息`
- `docs/conception/附.交易.md#合法性验证`
- `docs/conception/1.共识-历史证明（PoH）.md#择优凭证`
- `docs/conception/6.脚本系统.md#解锁数据`

## Open Questions（开放问题）

- Coinbase 普通交易签名是否属于必须永久保存的历史数据。
- 多签见证中公钥集和补全集的排序是否由见证容器强制。
