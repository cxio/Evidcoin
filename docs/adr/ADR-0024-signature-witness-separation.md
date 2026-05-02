# ADR-0024: 签名数据隔离（签名 Witness 与脚本分离）

## Status（状态）

Accepted

## Context（背景）

在标准签名验证场景下（通过 `SYS_CHKPASS` 指令），解锁脚本（UnlockScript）需要提供签名以证明对某个公钥对应私钥的持有。

一种直观的设计是将签名字节直接压入 UnlockScript 的脚本字节流中（类似于早期 Bitcoin 的设计）。然而这带来几个问题：
1. ML-DSA-65 签名的尺寸约为 3293 字节，大幅压缩 UnlockScript 可用于实际执行逻辑的空间（设计上限约 4095 字节）。
2. 签名字节混入脚本字节流时，签名本身会参与 `TxID` 的计算，形成循环依赖（签名依赖于 TxID，而 TxID 又依赖于签名）。

评审报告（C-4）指出，当前设计实际上已通过**签名数据不进入脚本字节流**的方式解决了这两个问题，但该设计决策从未在任何提案或 ADR 中正式记载。

## Decision（决策）

**签名数据通过系统环境传入 `SYS_CHKPASS`，而非内嵌于 UnlockScript 字节流中**。具体规则如下：

1. **UnlockScript 不含签名数据**：脚本字节流中不包含签名字节，只包含控制逻辑和辅助数据（如公钥、哈希等）。
2. **签名数据经独立通道传递**：交易发起者将签名数据附加到交易的签名附件（Witness）中，通过与交易序列化格式平行的独立字段传递给节点。
3. **`SYS_CHKPASS` 从系统环境读取签名**：当脚本 VM 执行到 `SYS_CHKPASS` 时，该指令从系统环境（而非操作数栈）中读取对应的签名数据，完成签名验证。
4. **签名数据不参与 TxID 计算**：`TxID = SHA3-384(TxHeader.CanonicalBytes())`，而 `TxHeader` 不包含签名附件字段。签名数据是在 TxID 已确定之后，以 TxID 为签名对象（之一）生成的，因此不存在循环依赖。

### 签名对象

标准 ML-DSA-65 签名的消息输入包含：

```
SignMessage = domainTag || chainIdentityBytes || TxID || [其他上下文数据]
```

这确保了签名与特定链、特定交易绑定，防止重放攻击。

### 链识别信息入签名

链识别信息（`Protocol-ID / Chain-ID / Genesis-ID / Bound-ID`）**进入签名消息**，但**不进入普通区块头哈希输入**。这两点不相互冲突：签名的防重放保护与区块头的哈希链式安全性各司其职。

## Rationale（理由）

1. **消除循环依赖**：签名数据不进入脚本/TxHeader，彻底消除了"签名依赖 TxID、TxID 又包含签名"的逻辑悖论。
2. **空间效率显著提升**：ML-DSA-65 签名约 3293 字节，若内嵌于 UnlockScript（上限约 4095 字节），脚本本身几乎无剩余空间。分离后，UnlockScript 空间完全用于逻辑表达。
3. **类比已验证设计**：此方案与 Bitcoin SegWit（Segregated Witness，隔离见证）设计思路一致，是经过大规模生产验证的技术路线。
4. **系统环境机制已存在**：`SYS_CHKPASS` 从系统环境读取数据的机制与其他系统指令（如 `SYS_TIME`）的环境注入方式一致，无需引入新的架构组件。

## Consequences（影响）

- 需在 `docs/proposal/06.Transaction-Model.md` 中正式定义签名附件（Witness）字段，明确其与脚本字节流的关系，以及签名附件不参与 TxID 计算的规则。
- 需在 `docs/proposal/09.Script-System.md` 中明确 `SYS_CHKPASS` 从系统环境读取签名数据的行为，以及解锁段中不应包含签名字节的规范。
- `internal/tx` 的交易序列化需区分"用于 TxID 哈希的标准字段"和"签名附件"两部分。
- `internal/script` 的 VM 在执行前需将对应的签名数据注入系统环境，供 `SYS_CHKPASS` 调用。
- 需更新 `docs/proposal/02.Cryptography-And-Hashing.md` 中签名消息范围的描述（链识别信息进入签名、不进入区块头哈希）。

## References（参考）

- `docs/proposal/06.Transaction-Model.md` — 交易模型
- `docs/proposal/09.Script-System.md` — 脚本系统（SYS_CHKPASS）
- `docs/proposal/02.Cryptography-And-Hashing.md` — 密码学与哈希
- `docs/plan/03-Transaction-And-Units.md` — Task 3/4
- `docs/plan/05-Script-System.md` — SYS_CHKPASS 实现
