# DEC-0019: 签名消息规范化编码

## Status（状态）

Proposed

## Context（背景）

`conception/附.交易.md` 已定义签名授权 flag 的输入、输出约束含义，但未固定签名消息的字节级规范化编码。若不同实现对同一授权 flag 选择不同字段或序列化顺序，签名验证会产生跨实现不一致。

## Decision（决策）

本节为候选规则，状态转为 Accepted 前不得作为最终共识依据。

签名消息编码建议固定为：

```text
SignatureMessage =
    DomainTag("SignMessage") ||
    uint32_be(TxHeader.Version) ||
    byte(authFlag) ||
    uint32_be(inputIndex) ||
    TxID ||
    SelectedInputsHash ||
    SelectedOutputsHash
```

字段选择规则建议如下：

- `TxID` 为交易头规范编码后的 `SHA3-384` 摘要。
- `SelectedInputsHash` 按 `authFlag` 中 `SIGIN_ALL` 或 `SIGIN_SELF` 选择输入集合后计算。
- `SelectedOutputsHash` 按 `authFlag` 中 `SIGOUT_ALL` 或 `SIGOUT_SELF` 以及辅项 flag 选择输出字段后计算。
- 输入集合按交易内输入序位升序序列化。
- 输出集合按交易内输出序位升序序列化。
- 集合内部字段按交易规范结构顺序序列化，并使用 `DEC-0001` 的规范化无符号 Varint。
- 空集合不得省略，统一编码为空集合哈希。

非法组合建议如下：

- `SIGIN_ALL` 与 `SIGIN_SELF` 同时置位非法。
- 未置位任一输入主 flag 且未置位任一输出主 flag 非法。
- `SIGOUT_ALL` 与 `SIGOUT_SELF` 同时置位非法。
- 置位输出辅项但未置位 `SIGOUT_ALL` 或 `SIGOUT_SELF` 非法。
- 置位 `SIGOUT_SELF` 时，若不存在与 `inputIndex` 同序位的输出项非法。
- `SIGOUTPUT` 与 `SIGRECEIVER`、`SIGCONTENT`、`SIGSCRIPT` 同时置位非法，因为完整输出已包含这些子字段。

待冻结参数：

- `TxID` 是否足以替代完整 `TxHeader`，或是否必须显式包含完整交易头编码。
- 输入和输出子集合哈希的具体 domain tag 名称。
- 空集合哈希的固定编码。
- 输出 `SIGCONTENT` 对 Coin、Credit、Proof 各类型字段的精确序列化范围。

## Rationale（理由）

- domain tag 隔离签名消息与其它哈希用途。
- 固定版本、授权 flag、当前输入序位和 TxID，可防止签名在不同交易或不同输入位置之间被误用。
- 将 flag 映射到规范化集合哈希，便于实现和测试，同时避免把未授权字段纳入签名。

## Consequences（影响）

- 在待冻结参数确定前，实现只能将本文件作为候选规范，不能作为最终共识规则。
- 签名测试向量必须覆盖单签、多签、全部输入、自身输入、全部输出、自身输出和非法 flag 组合。
- 一旦 Accepted，钱包与验证节点必须共享完全相同的消息编码实现。

## Conception Relationship（与构想关系）

- 补充 `conception/附.交易.md` 中签名授权 flag 的字节级消息编码。
- 不改变单签、多签、内置验证或授权 flag 的构想层含义。
