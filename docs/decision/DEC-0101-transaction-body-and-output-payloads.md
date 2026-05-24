# DEC-0101: Transaction Body and Output Payloads（交易体与输出载荷）

Status: Proposed

## Context（背景）

Conception 明确交易体由输入项和输出项构成，输出项可承载 Coin、Credit、Proof 或自定义类。但三类输出 payload 的字段顺序、空值编码、附件字段、销毁输出和输入集合规范序列化尚未冻结。

## Decision（决策）

建议交易体编码为：

- `Inputs = varint(count) || Input*`
- `Outputs = varint(count) || Output*`
- 输入和输出按交易创建者给定顺序编码，不自动排序。
- `HashInputs` 与 `HashOutputs` 只由规范编码计算，不包含见证信息。

建议输入项编码为：

1. `Year varint`
2. `TxIDPart bytes`，长度必须 `>=16`
3. `OutIndex varint`
4. `UnlockScript bytes`

建议输出项公共头为：

1. `Config byte`
2. `Payload bytes`
3. `LockScript bytes`

建议三类 payload：

- Coin：`Amount varint || Receiver bytes<256> || Memo optional bytes<256>`。
- Credit：`Receiver bytes<256> || Creator bytes<256> || Title bytes<256> || Description bytes || AttachmentID optional bytes`。
- Proof：`Creator bytes<256> || Title bytes<256> || Content bytes || AttachmentID optional bytes`。

销毁输出建议：

- 仅 Coin 输出可设置销毁位。
- 销毁输出的 `Receiver` 必须为空字节序列。
- 销毁金额不进入 UTXO。

## Rationale（理由）

保留输入输出创建顺序可支持授权语义中的 `SIGOUT_SELF` 和脚本引用。公共头拆分可让输出类型扩展不影响基础解析。销毁输出显式编码便于审计交易费销毁。

## Consequences（影响）

- 同一输入在一笔交易中重复引用应视为非法，避免双花歧义。
- 输出 payload 的空值必须规范编码，不能同时允许省略和零长度两种形式。
- Credit 第二笔输出的交易费加倍是共约，不应进入 TxID 编码规则。

## Conception References（构想层依据）

- `docs/conception/附.交易.md#交易体`
- `docs/conception/附.交易.md#输入项`
- `docs/conception/附.交易.md#输出项`
- `docs/conception/5.信用结构.md`

## Open Questions（开放问题）

- `Memo` 和 `AttachmentID` 的 optional 标记放在 payload 位图还是由长度零表达。
- 自定义类输出是否允许进入 UTXO/UTCO；conception 当前倾向不可作为输入源。
- Credit 31 年过期边界是 `age >= 31*87661` 还是 `age > 31*87661`。
