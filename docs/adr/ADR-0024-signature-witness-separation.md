# ADR-0024: 签名数据隔离（签名 Witness 与脚本分离）

## Status（状态）

Accepted

## Context（背景）

在标准内置签名验证场景下（通过 `SYS_CHKPASS` 指令），解锁脚本（UnlockScript）提供控制逻辑、公钥、哈希和授权辅助数据；真正用于证明私钥持有的 ML-DSA 签名字节由 Witness 提供。定制验证场景不使用 `SYS_CHKPASS` 时，可将签名字节作为普通脚本数据放入 UnlockScript，并由 `FN_CHECKSIG` / `FN_MCHECKSIG` 等指令验证。

一种直观的设计是将签名字节直接压入 UnlockScript 的脚本字节流中（类似于早期 Bitcoin 的设计）。然而这带来几个问题：
1. ML-DSA-65 签名的尺寸约为 3293 字节，大幅压缩 UnlockScript 可用于实际执行逻辑的空间（设计上限约 4095 字节）。
2. 签名字节混入脚本字节流时，签名本身会参与 `TxID` 的计算，形成循环依赖（签名依赖于 TxID，而 TxID 又依赖于签名）。

评审报告（C-4）指出，标准内置验证路径实际上已通过**签名数据不进入脚本字节流**的方式解决了这两个问题，但该设计决策从未在任何提案或 ADR 中正式记载。后续构想层修订进一步明确：UnlockScript 本身应作为输入规范字段参与 `HashInputs` 和 `TxID`，只有 Witness 签名附件排除在 `TxID` 之外。若定制验证把签名字节放入 UnlockScript，该签名字节也随 UnlockScript 一起参与 `HashInputs`、`TxID` 和 `MaxTxSize`。

## Decision（决策）

**`SYS_CHKPASS` 所需签名数据通过系统环境传入，而非从 UnlockScript 字节流读取；定制验证可自行从 UnlockScript/栈数据读取签名。**具体规则如下：

1. **标准内置验证使用 Witness**：当锁定脚本使用 `SYS_CHKPASS` 时，`SYS_CHKPASS` 所需签名字节必须来自系统环境中的 Witness 签名附件。
2. **定制验证可使用 UnlockScript**：当锁定脚本不使用 `SYS_CHKPASS`，而使用 `FN_CHECKSIG` / `FN_MCHECKSIG` 等函数指令进行定制验证时，签名字节可以作为普通脚本数据放入 UnlockScript，经数据栈或实参区传递给函数指令。
3. **`SYS_CHKPASS` 从系统环境读取签名**：当脚本 VM 执行到 `SYS_CHKPASS` 时，该指令从系统环境（而非操作数栈）中读取对应的签名数据，完成签名验证。
4. **UnlockScript 参与 TxID 计算**：UnlockScript 是输入项规范字段，进入 `LeadInputCanonicalBytes` 或 `RestInputCanonicalBytes`，并通过 `HashInputs` 被 `TxHeader` 承诺；修改 UnlockScript 必须改变 `TxID`。
5. **Witness 可为空**：交易不含 `SYS_CHKPASS` 验证需求时，Witness 可以为空且仍然合法；若执行到 `SYS_CHKPASS` 但系统环境中没有对应 Witness 实参，脚本验证失败退出。
6. **Witness 签名数据不参与 TxID 计算**：`TxID = SHA3-384(TxHeader.CanonicalBytes())`，而 `TxHeader` 不包含 Witness 签名附件字段。Witness 签名数据是在 `TxID` 已确定之后，以 `TxID` 为签名对象（之一）生成的，因此不存在循环依赖。
7. **Witness 不计入单笔交易大小上限**：`MaxTxSize = 65535 bytes` 按不含 Witness 的交易规范编码计算；该编码包含 Header、Inputs、Outputs，其中 Inputs 包含 UnlockScript。网络传输 framing、消息 envelope 或外部存储元数据不计入协议层交易大小。

### 签名对象

`SYS_CHKPASS` 标准 ML-DSA-65 签名的消息输入包含：

```
SignMessage = domainTag || chainIdentityBytes || TxID || [其他上下文数据]
```

这确保了签名与特定链、特定交易绑定，防止重放攻击。

定制验证若把签名字节放入 UnlockScript，则该签名字节会参与 `TxID`。因此 `FN_CHECKSIG` / `FN_MCHECKSIG` 的签名消息构造必须避免把"正在验证的签名字节本身"作为签名消息的一部分；具体自排除、占位或传统脚本签名 preimage 规则由交易签名消息 Proposal 固定。

### 链识别信息入签名

链识别信息（`Protocol-ID / Chain-ID / Genesis-ID / Bound-ID`）**进入签名消息**，但**不进入普通区块头哈希输入**。这两点不相互冲突：签名的防重放保护与区块头的哈希链式安全性各司其职。

## Rationale（理由）

1. **消除标准验证循环依赖，同时承诺解锁逻辑**：UnlockScript 进入输入 Hash 和 `TxID`，并计入交易大小，防止解锁逻辑被事后替换或绕开尺寸约束；标准 `SYS_CHKPASS` 的 Witness 签名数据不进入 `TxHeader`，避免"签名依赖 TxID、TxID 又包含签名"的逻辑悖论。
2. **空间效率显著提升但保留自由合约能力**：ML-DSA-65 签名约 3293 字节，标准验证若内嵌于 UnlockScript（上限约 4095 字节），脚本本身几乎无剩余空间。Witness 分离让标准验证更紧凑；定制验证仍可选择承担空间成本，把签名放入 UnlockScript。
3. **类比已验证设计**：Witness 签名字节隔离与 Bitcoin SegWit（Segregated Witness，隔离见证）的消除签名循环依赖思路一致；本设计额外要求 UnlockScript 自身仍由 `TxID` 承诺。
4. **系统环境机制已存在**：`SYS_CHKPASS` 从系统环境读取数据的机制与其他系统指令（如 `SYS_TIME`）的环境注入方式一致，无需引入新的架构组件。

## Consequences（影响）

- 需在 `docs/proposal/06.Transaction-Model.md` 中正式定义签名附件（Witness）字段，明确其与脚本字节流的关系、UnlockScript 参与输入 Hash 和 `TxID`、签名附件不参与 `TxID` 计算且不计入 `MaxTxSize`、Witness 可为空的规则。
- 需在 `docs/proposal/09.Script-System.md` 中明确 `SYS_CHKPASS` 从系统环境读取签名数据；`FN_CHECKSIG` / `FN_MCHECKSIG` 等定制验证指令可从普通脚本数据获取签名。
- `internal/tx` 的交易序列化需区分"用于 TxID 哈希的标准字段"和"签名附件"两部分；UnlockScript 属于前者，Witness 属于后者。
- `internal/script` 的 VM 在执行前需将对应的签名数据注入系统环境，供 `SYS_CHKPASS` 调用。
- 需更新 `docs/proposal/02.Cryptography-And-Hashing.md` 中签名消息范围的描述（链识别信息进入签名、不进入区块头哈希）。

## References（参考）

- `docs/proposal/06.Transaction-Model.md` — 交易模型
- `docs/proposal/09.Script-System.md` — 脚本系统（SYS_CHKPASS）
- `docs/proposal/02.Cryptography-And-Hashing.md` — 密码学与哈希
- `docs/plan/03-Transaction-And-Units.md` — Task 2/8
- `docs/plan/05-Script-System.md` — SYS_CHKPASS 实现
