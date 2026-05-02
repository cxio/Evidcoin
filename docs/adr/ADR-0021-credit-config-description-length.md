# ADR-0021: Credit Config 低 10 位描述长度语义

## Status（状态）

Accepted

## Context（背景）

凭信（Credit）输出项的 `Config` 字段为 uint16（16 位），其中低 10 位用于描述内容的长度定义。同时，实际的描述字段在序列化时使用 `Bytes` 类型（varint length prefix + 原始字节）进行编码。

评审报告（N-2）指出：Config 中的 10 位长度值与 Bytes 编码中的 varint length prefix 形成了双重长度表达，其协调关系不明确。

## Decision（决策）

**Credit Config 低 10 位的描述长度是凭信输出项的固定配置，存在于交易数据序列化之后，对描述字段实际长度的约束定义。**

具体语义：
- `Config[9:0]`（低 10 位）= 允许的描述最大字节长度（0-1023）
- 实际描述字段仍使用标准 `Bytes` 编码（varint length prefix + 内容）
- 验证时：`len(description) <= Config[9:0]`，超出则拒绝

两者不冲突——varint length prefix 是序列化格式的一部分（描述字段有多长），Config 低 10 位是业务约束（这个凭信允许的最大描述长度）。

### 特殊情况

- `Config[9:0] = 0`：不允许任何描述（描述字段必须为空 Bytes）。
- `Config[9:0] = 1023`：允许最长 1023 字节的描述。

## Rationale（理由）

区分**序列化格式的长度**（varint length prefix，描述"实际有多少字节"）和**业务约束的长度**（Config 低 10 位，描述"允许最多多少字节"）是常见的设计模式。Config 中的长度限制使得凭信创建者可以在创建时约束未来描述的最大规模，而不需要修改字段本身的编码格式。这两个字段服务于不同的目的，不存在冗余。

## Consequences（影响）

- 需在 `docs/proposal/07.Coin-Credit-Proof-Units.md` 中明确 Credit Config 低 10 位的语义定义，以及与 Bytes 编码的关系。
- `internal/tx` 的 Credit 输出项验证函数需在解析描述字段后，校验其实际长度是否不超过 `Config[9:0]`。
- 需在测试套件中添加：描述超出 Config 限制时应拒绝；描述恰好等于 Config 限制时应接受。

## References（参考）

- `docs/proposal/07.Coin-Credit-Proof-Units.md` — Credit payload 定义
- `docs/proposal/01.Types-And-Encoding.md` — Bytes 类型编码
- `docs/plan/03-Transaction-And-Units.md` — Task 5
