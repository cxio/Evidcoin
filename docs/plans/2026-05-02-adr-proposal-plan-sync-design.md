# ADR 到 Proposal/Plan 的全量精准同步设计

## 背景

`docs/adr/` 已记录 ADR-0001 至 ADR-0031，覆盖编码、哈希、脚本、交易、状态、共识、激励和客户端策略等关键决策。现有 `docs/proposal/` 与 `docs/plan/` 中仍存在若干“未决问题”表述，部分内容与已接受的 ADR 冲突，可能导致后续实现依据不一致。

## 目标

- 以 ADR-0001 至 ADR-0031 为最高优先级决策源，同步修订相关 Proposal 和 Plan。
- 保持现有文档结构，不做大规模重写。
- 删除或改写与 ADR 冲突的未决表述。
- 在 Proposal 中固化协议/技术规格。
- 在 Plan 中补充实现任务、边界测试和验收要求。
- 在 `docs/plan/08-Open-Questions-And-Acceptance.md` 中标记已由 ADR 关闭的开放问题。

## 非目标

- 不修改构想层 `docs/conception/`。
- 不重排 Proposal/Plan 的整体章节体系。
- 不编写 Go 生产代码。
- 不新增新的 ADR。

## 推荐方案：全量精准同步

采用定点编辑方式同步 ADR 结论：对每个受影响文档，只修改与 ADR 直接相关的段落、表格、未决项和测试要求。必要时新增短小章节，但避免重写整篇文档。

相比最小修补，该方案能让实现者直接从 Proposal/Plan 获得完整规则；相比重构式同步，该方案改动风险更低，也更容易审阅。

## 修订原则

1. ADR 优先：Proposal/Plan 与 ADR 冲突时，以 ADR 为准。
2. Proposal 写规格：算法、字段、状态语义、协议合法性规则应落在 Proposal。
3. Plan 写执行：实现任务、测试向量、验收边界应落在 Plan。
4. 保留追溯：新增或修订内容尽量标注对应 ADR 编号。
5. 关闭未决：被 ADR 覆盖的 OQ 不再作为阻塞项保留。

## 主要同步领域

- 基础编码与密码学：LEB128 varint、Domain Tag、MintHash 内层哈希、地址文本编码、ML-DSA-65 集成策略。
- 哈希树与状态：空树、单叶、奇数叶、UTXO/UTCO TxID 字节索引、CheckRoot 前置状态承诺。
- 脚本系统：Float 确定性、初始 pass 状态、CHECK 双向覆盖、资源边界、SYS_NULL 解锁段例外。
- 交易模型：Coinbase 独立解析、Coinbase HashInputs、Witness 与 UnlockScript 分离、TxIDPart 碰撞、同区块链式消费禁止。
- 共识与分叉：PoH X 编码、MintHash 碰撞、mintTxID 高度窗口、同铸造者同高区块与分叉平局决胜。
- 团队校验与服务：铸造者签名范围、首领黑名单本地策略、公共服务质量评估、全网通告与授权公钥更新。
- 激励与单位：奖励/费用取整、chx 单位、百日扩张客户端策略、创世年度边界。

## 验收标准

- `docs/proposal/` 与 `docs/plan/` 不再包含与 ADR-0001 至 ADR-0031 冲突的表述。
- 被 ADR 关闭的开放问题在 `08-Open-Questions-And-Acceptance.md` 中有明确关闭说明。
- 关键实现 Plan 均包含对应边界测试或测试向量要求。
- 文档仍保持原有 Proposal/Plan 分层和追溯关系。
