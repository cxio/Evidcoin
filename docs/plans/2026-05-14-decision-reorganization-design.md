# docs/decision 重整设计 2026-05-14

## 背景

`docs/conception/` 在近期发生重要修订，多个旧 Decision 的主体内容已经被 conception 吸收，部分旧 Decision 与最新 conception 出现冲突。`docs/decision/` 需要改为只记录 conception 尚未明确、但会影响跨实现一致性或实现路径的补充决策。

本次整理采用 C 方案：全面重整 `docs/decision/`，允许重新编号、合并、拆分、删除和重写。旧 DEC 只作为可复用材料，不保留旧编号语义。

## 目标

- 建立一组与最新 `docs/conception/` 对齐的 Decision 文件。
- 删除或降级已被 conception 明确吸收的旧决策主体。
- 将无法从 conception 直接裁定的字节级格式、边界条件和实现路径缺口标为 `Proposed`。
- 在 `docs/decision/README.md` 中维护当前索引、迁移索引、已吸收清单和开放问题。
- 明确记录 conception 内部仍不一致或需要作者裁决的事项，但不直接修改 conception。

## 边界

- 唯一正式上游依据是 `docs/conception/`。
- `docs/proposal/`、`docs/plan/` 不作为本次依据。
- `docs/plans/` 是临时执行资料，不作为正式文档依据。
- `working/` 不读取、不参考。
- 本次只重整文档，不修改协议源码或 conception 正文。

## 整理策略

- 对 conception 已明确的内容：不作为 Decision 主体重复定义，仅在 README 的“已吸收/不再单列”清单中说明。
- 对 conception 已明确但仍缺少字节级编码的内容：创建边界补充类 DEC，状态按确定性使用 `Accepted` 或 `Proposed`。
- 对旧 Decision 与 conception 冲突的内容：以 conception 为准，旧内容删除或改写为开放问题。
- 对 conception 内部不一致的内容：不擅自裁定，记录为 README 开放问题或 Proposed DEC 的待裁决参数。
- 新 DEC 统一结构：Title、Status、Context、Decision、Rationale、Consequences、Conception references、Open questions。

## 主题分组

- 基础编码与密码学：varint、domain tag、字段宽度、哈希树边界、地址、ML-DSA。
- 交易与见证：交易体规范编码、见证编码与剪枝边界、签名消息编码。
- 状态指纹：UTXO/UTCO payload、状态位顺序、宽成员树边界。
- PoH 与分叉：参数补充、碰撞规则、时间戳、Stakes、创世初段、同铸造者低收益、候选区块阈值、平局裁决。
- 激励与 Coinbase：奖励取整、Coinbase 输入哈希、序列化、兑奖槽、发行计划、公共服务激活边界。
- 通告与脚本：公告信任链、Float 确定性、脚本字节编码、环境注册表、成本预算、Float 衍生语义。
- 证明与服务格式：区块证明包、概要 TxID、网络证明格式、Blockqs 验证数据格式。

## 新 DEC 目录草案

- `DEC-0001-canonical-varint.md`
- `DEC-0002-domain-tags.md`
- `DEC-0003-field-widths.md`
- `DEC-0004-hash-tree-edge-cases.md`
- `DEC-0005-address-encoding.md`
- `DEC-0006-ml-dsa-65-profile.md`
- `DEC-0007-transaction-body-canonical-encoding.md`
- `DEC-0008-witness-encoding-and-pruning.md`
- `DEC-0009-signature-message-encoding.md`
- `DEC-0010-utxo-utco-fingerprint-payload.md`
- `DEC-0011-poh-parameters-and-collision.md`
- `DEC-0012-poh-timestamp-and-stakes.md`
- `DEC-0013-genesis-initial-boundary.md`
- `DEC-0014-block-competition-rules.md`
- `DEC-0015-fork-tiebreaker.md`
- `DEC-0016-reward-rounding.md`
- `DEC-0017-coinbase-hash-inputs.md`
- `DEC-0018-coinbase-serialization-and-award-slots.md`
- `DEC-0019-issuance-schedule.md`
- `DEC-0020-public-service-activation-boundary.md`
- `DEC-0021-announcement-trust-chain.md`
- `DEC-0022-script-float-determinism.md`
- `DEC-0023-script-canonical-byte-encoding.md`
- `DEC-0024-script-environment-registry.md`
- `DEC-0025-script-cost-budget.md`
- `DEC-0026-script-float-derived-semantics.md`
- `DEC-0027-block-proof-package.md`
- `DEC-0028-summary-txid-and-network-proof-formats.md`
- `DEC-0029-blockqs-verification-data-format.md`

## 迁移策略

- 旧 `DEC-0001` 至 `DEC-0009` 的基础编码、字段宽度、哈希树、地址、ML-DSA 内容按主题重写到新 `DEC-0001` 至 `DEC-0006`，冲突项转入开放问题。
- 旧 PoH、Stakes、创世边界、分叉平局与区块竞争内容合并到新 `DEC-0011` 至 `DEC-0015`。
- 旧奖励、Coinbase、发行计划与公共服务兑奖内容合并到新 `DEC-0016` 至 `DEC-0020`，35 区块口径改为 conception 的 31 区块确认安全边界。
- 旧公告信任链重写为 Proposed，删除“创世 #0/#1 签名公钥作为初始公告根”的无依据设定。
- 旧脚本 Float、预算和派生语义重写为边界补充，避免与 conversion 指令和 Float=float64 主体冲突。
- 旧区块证明和短 TxID 内容分拆为证明包、网络摘要格式、Blockqs 验证数据格式。

## 验收标准

- `docs/decision/` 只保留新编号 DEC 和 README。
- 每个 DEC 使用统一结构，并含 `Status: Accepted|Proposed|Deprecated|Superseded|Absorbed`。
- README 包含当前索引、迁移索引、已被 conception 吸收/不再单列清单、开放问题。
- 不再出现过时的 `35 个区块`、`35区块`、`uint32_be(TxHeader.Version)`、`MintInner`、`MintHash`、内外层旧术语或 `-27` 分叉口径。
- `COMFLO` 如出现，只能作为 conception 内部命名不一致的开放问题引用。
- README 链接均指向存在文件。
- 完成至少一次自审并记录校验命令结果。
