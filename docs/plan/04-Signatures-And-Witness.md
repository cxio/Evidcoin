# Signatures And Witness Implementation Plan

**Goal:** 实现签名消息字节布局与覆盖范围、授权种类与辅项互斥、Coinbase 签名消息、见证容器与剪枝边界、单签与多签 M-of-N 验证（含三套排序规则）。

**Architecture:** 签名相关结构位于 `internal/tx`（签名消息构造、见证容器），ML-DSA 验证经 `pkg/crypto` 的 `Signer`/`Verifier` 接口；`FN_CHECKSIG`/`FN_MCHECKSIG` 的脚本调用在第 06 章脚本层注入当前 `input_index`。本层只产出「待签字节序列」与「见证容器解析/剪枝」，不实现脚本执行。

**Tech Stack:** Go 1.26.2、`pkg/types`、`pkg/crypto`（ML-DSA-65、地址/复合公钥哈希）、`internal/tx`（交易头/输入输出编码）、`internal/blockchain`（链身份）、表驱动测试。

---

## 来源提案

- `docs/proposal/08.Signatures-And-Witness.md`
- 依赖 `docs/proposal/02.Cryptography-And-Hashing.md`（公钥哈希、复合公钥哈希、ML-DSA、`signature.message` 域）
- 依赖 `docs/proposal/03.Identifiers-And-Constants.md`（链标识 MixData）
- 依赖 `docs/proposal/06.Transaction-Model.md`、`07.Coin-Credit-Proof-Units.md`（交易头、输入输出、payload 字段）
- DEC-0102：签名消息布局（ChainScope/SigScope/TxHeaderCore/Covered*）、覆盖范围、辅项冲突、Coinbase 签名消息、多签签名集顺序。
- DEC-0103：见证容器格式、item 类型、剪枝规则、多签见证排序。
- DEC-0104：多签复合公钥哈希派生、ML-DSA-65 profile（circl）。

## 非目标

- 不执行锁定/解锁脚本（脚本层在第 06 章）。
- 不实现 PoH 铸凭资格（第 07 章）。
- 不在本层注入运行时 `input_index`（由脚本引擎注入）。
- 不实现完整 Coinbase 奖励结算（第 10 章）。

## 建议文件

| 文件 | 内容 |
|------|------|
| `internal/tx/signature_message.go` | 签名消息布局、ChainScope/SigScope/TxHeaderCore/Covered* 构造 |
| `internal/tx/auth_flag.go` | 授权种类 8 位标记、辅项互斥静态检查 |
| `internal/tx/coinbase_sig.go` | Coinbase 签名消息与 CheckRoot 签名消息构造 |
| `internal/tx/witness.go` | 见证容器格式、item 类型、剪枝规则、TxID 排除 |
| `internal/tx/multisig.go` | 单签/多签验证、三套排序规则 |
| `internal/tx/sig_errors.go` | 签名/见证错误定义 |

## Task 1: 签名消息布局

**Files:**
- Create: `internal/tx/signature_message.go`
- Create: `internal/tx/signature_message_test.go`

**Step 1: 写失败测试**

测试（DEC-0102 §1）：

- 整体布局为 `DomainTag("signature.message") || ChainScope || SigScope || TxHeaderCore || CoveredInputs || CoveredOutputs`。
- `ChainScope = ProtoLen || ProtocolID || ChainLen || ChainID || GenesisID(48) || BoundLen || BoundID`；`BoundID` 空时编码 `varint(0)` **占位不省略**。
- `SigScope = chk_type(1B) || auth_flag(1B) || input_index(varint)`；`chk_type ∈ {1=币金花费, 2=凭信转移}`。
- `TxHeaderCore` 固定含 `Version(uint16,BE) || Timestamp(int64,BE)`；存在铸凭公钥哈希时追加 `MintPKHash(varint(len)||bytes)`，否则 `varint(0)`（三处编码对照第 3 处）。
- 修改链身份、输入范围或输出范围会改变签名消息字节。

**Step 2: 实现并提交**

```bash
go test ./internal/tx -run 'TestSignatureMessage' -v
git add internal/tx/signature_message.go internal/tx/signature_message_test.go
git commit -m "feat: add signature message layout"
```

## Task 2: 授权种类与覆盖范围

**Files:**
- Create: `internal/tx/auth_flag.go`
- Create: `internal/tx/auth_flag_test.go`

**Step 1: 写失败测试**

测试（DEC-0102 §2-§4）：

- 8 位标记：`bit7 SIGIN_ALL`、`bit6 SIGIN_SELF`（独项）；`bit5 SIGOUT_ALL`、`bit4 SIGOUT_SELF`（主项）；`bit3 SCRIPT`、`bit2 CONTENT`、`bit1 RECEIVER`、`bit0 OUTPUT`（辅项）。
- CoveredInputs：`SIGIN_ALL` 全部输入含解锁脚本前置 `varint(count)`；`SIGIN_SELF` 仅当前输入前置 `varint(input_index)`；均未设置为空。
- CoveredOutputs：`SIGOUT_ALL` 全部输出每项前置序位；`SIGOUT_SELF` 仅同序位输出，`input_index >= len(outputs)` 时验证**必须失败**。
- 辅项内嵌字段：`SCRIPT` 仅 LockScript；`CONTENT` 除 LockScript 与接收者外字段；`RECEIVER` 仅接收者；`OUTPUT` 等价三者并集；每段 `varint(len)||bytes`。
- **辅项互斥**：`OUTPUT` 与 `SCRIPT|CONTENT|RECEIVER` 任一并存 → 构造失败、交易拒绝（静态可检）。
- 主项缺失但辅项存在、或辅项缺失但主项存在 → 构造失败。

**Step 2: 实现并提交**

```bash
go test ./internal/tx -run 'TestAuthFlag' -v
git add internal/tx/auth_flag.go internal/tx/auth_flag_test.go
git commit -m "feat: add authorization flags and coverage"
```

## Task 3: Coinbase 与 CheckRoot 签名消息

**Files:**
- Create: `internal/tx/coinbase_sig.go`
- Create: `internal/tx/coinbase_sig_test.go`

**Step 1: 写失败测试**

测试（DEC-0102 §5）：

- Coinbase 签名消息：`DomainTag("signature.message") || ChainScope || 0x00 || CoinbaseTxID(48)`；`chk_type=0` 标记 Coinbase 域，**不使用**授权种类。
- Coinbase 直接对整笔交易签名，不走 auth_flag 覆盖范围路径。
- 铸造者对区块 `CheckRoot` 的签名**独立于** Coinbase 交易签名，使用各自域标签/消息构造。
- 创世 CheckRoot 签名构造可复现（保留规则见 Task 4）。

**Step 2: 实现并提交**

```bash
go test ./internal/tx -run 'TestCoinbaseSig' -v
git add internal/tx/coinbase_sig.go internal/tx/coinbase_sig_test.go
git commit -m "feat: add coinbase signature messages"
```

## Task 4: 见证容器与剪枝

**Files:**
- Create: `internal/tx/witness.go`
- Create: `internal/tx/witness_test.go`

**Step 1: 写失败测试**

测试（DEC-0103 §6-§7）：

- 见证容器 `Witness = varint(item_count) || WitnessItem*`，`WitnessItem = type(byte) || bytes`；按输入分组。
- item 类型固定 6 类：`0x01` 验证类别、`0x02` 授权标记、`0x03` 签名、`0x04` 公钥、`0x05` 补全公钥哈希、`0x06` 解锁脚本外部数据。
- 见证**不计入 TxID**；修改见证字节不改变 `TxID`，但改变完整交易编码。
- 解锁脚本参与输入根（第 03 章），**不属于**可剪枝见证；定制验证放入解锁脚本的签名数据计入交易体、不可剪枝。
- 必须保留项：择优凭证对 `mintHash` 的签名、创世 CheckRoot 签名；Coinbase 普通签名分层保存（长期共识最小验证不依赖）。
- Witness 可为空且交易结构合法；缺失见证是否失败由脚本执行 `SYS_CHKPASS` 时判定（第 06 章）。

**Step 2: 实现并提交**

```bash
go test ./internal/tx -run 'TestWitness' -v
git add internal/tx/witness.go internal/tx/witness_test.go
git commit -m "feat: add witness container and pruning"
```

## Task 5: 单签与多签验证

**Files:**
- Create: `internal/tx/multisig.go`
- Create: `internal/tx/multisig_test.go`

**Step 1: 写失败测试**

测试（DEC-0102/0103/0104 §8-§9）：

- 单签：输入来源为单公钥哈希 `SHA3-256(BLAKE2b-512(pubKey))`，单私钥签名；多笔单签输入各自全部签名。
- 多签 M-of-N：`M ≤ N`、`M,N` 均非 0、无重复公钥；越界配置拒绝。
- 见证三集合：签名集（M 个签名）、公钥集（与签名集**一一对应顺序**）、补全集（未签名公钥初级哈希 `BLAKE3-256(pubKey)`，`N-M` 个，**字典序升序**）。
- 验证过程：由集合大小算 `m/N` → 计算公钥集初级哈希 → 与补全集混合排序串联前置 `m||n` → 计算复合公钥哈希（第 01 章地址 `address.multi`）→ 比对接收者 → 逐一验证签名集（按见证内顺序）。
- **三套排序严格区分**：①地址派生（N 个 `BaseH` 字典序排序串联前置 `m||n`）；②补全集字典序升序 + 签名/公钥配对顺序；③`FN_MCHECKSIG` 实参按见证提供顺序，不强制哈希排序。
- 同一 M-of-N 允许多种合法签名见证组合；规范编码唯一性由 `UnlockScript` 整体哈希保证，不强加签名集排序。

**Step 2: 实现并提交**

```bash
go test ./internal/tx -run 'TestMultisig' -v
git add internal/tx/multisig.go internal/tx/multisig_test.go
git commit -m "feat: add single and multisig verification"
```

## Task 6: ML-DSA 验证接入

**Files:**
- Modify: `internal/tx/multisig.go`
- Create: `internal/tx/sig_errors.go`

**Step 1: 写失败测试**

测试：

- 签名验证输入即 Task 1/Task 5 定义的签名消息字节序列。
- 验证仅通过 `pkg/crypto` 的 `Signer`/`Verifier` 接口（ML-DSA-65 profile = circl，DEC-0104；A-2 待决见第 01 章）。
- 错误签名、错误公钥、`SIGOUT_SELF` 越界、辅项互斥违例返回各自明确错误。

**Step 2: 实现并提交**

```bash
go test ./internal/tx -run 'Test(Multisig|SigError)' -v
git add internal/tx/multisig.go internal/tx/sig_errors.go
git commit -m "feat: wire ml-dsa verification"
```

## 阶段验收

运行：

```bash
go fmt ./...
go test ./internal/tx ./pkg/types ./pkg/crypto
go test ./...
go build ./...
go mod tidy
go mod verify
golangci-lint run
```

通过标准：

- 签名消息字节序列唯一对应一组 `auth_flag`；辅项互斥与主辅配合静态检查生效。
- `BoundID` 空时仍占位 `varint(0)`，主链/分叉链签名消息结构一致。
- Coinbase 走 `chk_type=0` + 完整 TxID 路径；CheckRoot 签名独立域标签。
- 见证不计入 TxID；解锁脚本不可剪枝；择优凭证签名、创世 CheckRoot 签名必须保留。
- 多签三套排序分别实现，对照表与 DEC-0102/0103/0104 一致。
- ML-DSA 验证仅经 `pkg/crypto` 接口，profile 固定 circl。
