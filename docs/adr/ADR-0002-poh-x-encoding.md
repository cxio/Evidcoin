# ADR-0002: 铸凭哈希 X 参数的计算与编码

## Status（状态）

Accepted

## Context（背景）

PoH 铸凭哈希演算中包含一个混合参数 X，其计算涉及时间戳、币权销毁量和混合常数的乘积：

```
Mix    = 0x517cc1b727220a95
Stakes = block(currentHeight - 27) 中的币权销毁总量
X      = Encode(timeStamp * Stakes * Mix)
```

以下三个具体问题此前未被规范：

1. **时间戳单位**：秒还是毫秒？
2. **乘法精度**：是否允许溢出截断？
3. **编码方式**：结果以何种字节序列化？

这些问题直接决定了 MintHash 的可复现性，属于共识级关键规范（OQ-017/018）。

## Decision（决策）

### 时间戳单位：毫秒

`timeStamp` 采用自 Unix 纪元以来的**毫秒**数（int64）。

计算公式：
```
timeStamp = (GenesisTimestampMs) + height * BlockIntervalMs
          = GenesisTimestampMs + height * 360000
```

其中 `BlockInterval = 6 min = 360,000 ms`。

### 乘法精度：大整数（无溢出）

三值相乘使用**大整数（BigInt）**运算，不限制宽度，不做截断。

```
X_BigInt = timeStamp * Stakes * Mix
```

三个因子均为非负整数（Stakes ≥ 0，timeStamp > 0，Mix > 0），乘积始终为非负整数。

### 编码方式：大端序最小字节表示

将 X_BigInt 编码为**大端序（big-endian）的最小字节序列**（无前导零字节）。若 X_BigInt = 0，则编码为单字节 `0x00`。

```
X = BigIntToMinimalBigEndianBytes(timeStamp * Stakes * Mix)
```

## Rationale（理由）

1. **时间戳选毫秒**：脚本系统（`SYS_TIME`）的时间戳默认单位为毫秒，统一使用毫秒可保持系统内的单位一致性，减少因单位换算引发的实现错误。粒度更细也对计算结果有更强的混淆效果。
2. **大整数乘法**：三个因子（毫秒时间戳约 53 位、Stakes 最大约 64 位、Mix 64 位）的乘积可达 ~181 位。使用固定宽度整数（如 uint64）会截断高位，丧失时间戳的混淆价值；BigInt 确保不损失精度，保持函数的单射性。
3. **大端序最小字节**：与项目其他固定宽度整数的大端序约定（见 `docs/proposal/01.Types-And-Encoding.md`）一致；最小字节表示（无前导零）确保编码的唯一性，满足规范化编码要求。

## Consequences（影响）

- 需在 `docs/proposal/10.PoH-Consensus.md` 中细化 `timeStamp` 的单位和 `Encode()` 的具体算法。
- `pkg/crypto` 中的铸凭哈希计算函数需引入 `math/big` 包。
- 需补充测试向量：给定 height、GenesisTimestampMs、Stakes，验证 X 的字节序列。
- OQ-017、OQ-018 部分关闭（Stakes 定义见 `docs/proposal/10.PoH-Consensus.md`，此 ADR 只处理计算语义）。

## References（参考）

- `docs/conception/1.共识-历史证明（PoH）.md` — 铸凭哈希原始定义
- `docs/proposal/10.PoH-Consensus.md` — 铸凭哈希演算
- `docs/proposal/01.Types-And-Encoding.md` — 字节序约定
- `docs/plan/08-Open-Questions-And-Acceptance.md` — OQ-017/018
