# Decision Rewrite Design（决策层重写设计）

## 目标

根据 `docs/conception/` 的最新修订，完全重写 `docs/decision/`。Decision 只记录构想层尚未冻结、但会影响实现一致性的补充决策；构想层已经明确的协议规则不再重复成独立 DEC。

## 设计原则

- `docs/conception/` 是唯一上游正式依据。
- `docs/decision/` 只补充字节编码、哈希前像、排序、边界、失败语义、注册表和服务验证格式。
- 已被构想层明确吸收的旧 DEC 不再保留正文，只在索引中列出来源。
- 构想层自身存在疑似冲突时，记录到 `CONCEPTION-CONFLICTS.md`，Decision 不替作者裁决。
- 新项目、个人项目，无需保留旧编号兼容；允许重新规划编号和文件名。

## 推荐结构

采用主题段编号：

- `DEC-000x`：基础编码、域标签、哈希树、字段编码。
- `DEC-010x`：交易、签名、见证、地址与密码学。
- `DEC-020x`：UTXO/UTCO 状态指纹。
- `DEC-030x`：PoH、创世初段、分叉选择。
- `DEC-040x`：Coinbase、奖励、发行、公共服务激活。
- `DEC-050x`：脚本字节码、Float、注册表、成本、失败语义。
- `DEC-060x`：区块证明、网络概要、Blockqs 验证格式。

## 文件清单

- `README.md`：当前索引、状态定义、维护规则、已吸收/移除清单、开放问题入口。
- `CONCEPTION-CONFLICTS.md`：构想层疑似矛盾和待作者裁决项。
- `DEC-0001-canonical-integer-and-bytes-encoding.md`
- `DEC-0002-domain-tags-and-hash-profiles.md`
- `DEC-0003-block-and-transaction-field-encoding.md`
- `DEC-0004-hash-tree-and-proof-edge-cases.md`
- `DEC-0101-transaction-body-and-output-payloads.md`
- `DEC-0102-signature-message-profile.md`
- `DEC-0103-witness-container-and-pruning.md`
- `DEC-0104-address-and-ml-dsa-profile.md`
- `DEC-0201-utxo-utco-state-fingerprint.md`
- `DEC-0301-poh-mint-hash-and-mint-proof.md`
- `DEC-0302-genesis-and-initial-window.md`
- `DEC-0303-fork-choice-and-randomx-tiebreaker.md`
- `DEC-0401-coinbase-serialization-rewards-and-award-slots.md`
- `DEC-0501-script-bytecode-encoding.md`
- `DEC-0502-script-float-profile.md`
- `DEC-0503-script-registry-and-environment-boundary.md`
- `DEC-0504-script-cost-budget.md`
- `DEC-0505-script-failure-and-disabled-opcodes.md`
- `DEC-0601-block-proof-package.md`
- `DEC-0602-network-summary-txid-profile.md`
- `DEC-0603-blockqs-verification-data-profile.md`

## 删除或吸收

- 删除旧 `DEC-0017-coinbase-hash-inputs.md`。
- 删除旧 `DEC-0021-announcement-trust-chain.md`。
- 删除其它旧编号 DEC，由新编号文件承接有效的补充决策。

## 构想层冲突处理

本次会记录以下类型的问题：

- Coinbase 奖励输出顺序不一致。
- 普通输出类型值与脚本验证类型值不一致。
- Coinbase 收益总额是否包含被销毁交易费表述不一致。
- 公共服务奖励范围与节点发现服务范围口径不一致。
- `MintPKHash` 与铸凭哈希 `pubKey` 来源说明不一致。
- 第一年区块容量边界存在 `87660/87661` 歧义。
- 附件 `10MB` 服务边界未覆盖等于值。
- 脚本禁用指令与正文示例/说明存在冲突。

这些问题不在 Decision 中强行裁决，只给出影响范围和需要作者修订的依据。
