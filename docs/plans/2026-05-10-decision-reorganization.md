# Decision Reorganization Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 按最新构想层合并、删除、重编号并重整 `docs/decision`，再新增高优先级补充决策。

**Architecture:** 先把现有 20 篇 DEC 重整为主题清晰、编号连续的新集合，并在 README 中保留旧编号迁移索引。完成后从新编号之后新增影响跨实现一致性的高优先级 DEC。`docs/conception` 是最高依据；被构想层明确吸收的规则不再单独保留 Decision。

**Tech Stack:** Markdown 文档、Git、shell 校验命令（`rg`、`find`）。

---

## 前置说明

当前会话已创建设计文档：

- `docs/plans/2026-05-10-decision-reorganization-design.md`

执行本计划时应先确认该文件是否位于执行工作区；如果使用新 worktree，需要把该文件同步过去或重新创建。

## Task 1: 建立隔离工作区并确认基线

**Files:**
- Read: `.gitignore`
- Read: `docs/decision/*.md`
- Read: `docs/conception/**/*.md`

**Step 1: 创建 worktree**

按 `using-git-worktrees` 技能执行：优先使用已有 `.worktrees/` 或 `worktrees/`，否则询问目录。分支名建议：

```bash
docs/decision-reorganization
```

**Step 2: 同步设计文档**

若新 worktree 中缺少：

```text
docs/plans/2026-05-10-decision-reorganization-design.md
```

则从主工作区复制过去，或按设计文档内容重建。

**Step 3: 检查当前 DEC 文件**

Run:

```bash
find docs/decision -maxdepth 1 -type f -name 'DEC-*.md' | sort
```

Expected: 显示旧 DEC-0001 到 DEC-0020。

**Step 4: 检查过时口径**

Run:

```bash
rg --line-number -- '-27|内层|外层|64 字节|64字节' docs/decision docs/conception
```

Expected: 记录命中项；后续任务应消除 `docs/decision` 中与最新构想冲突的命中。

**Step 5: Commit 基线准备**

```bash
git add docs/plans/2026-05-10-decision-reorganization-design.md docs/plans/2026-05-10-decision-reorganization.md
git commit -m "docs: plan decision reorganization"
```

## Task 2: 设计新编号与旧 DEC 迁移表

**Files:**
- Modify: `docs/decision/README.md`

**Step 1: 确定第一阶段新 DEC 清单**

建议将现有内容重整为以下第一阶段清单：

| 新编号 | 标题 | 来源 | 处理 |
|---|---|---|---|
| DEC-0001 | 规范化无符号 Varint 编码 | old DEC-0003 | 保留并重编号 |
| DEC-0002 | 哈希域分隔标签格式 | old DEC-0004 | 保留，删除 `MintInner`/`MintHash` 双层口径，改为单层 `PoHMintHash` |
| DEC-0003 | 区块头与交易头字段宽度 | old DEC-0017 | 保留并重编号 |
| DEC-0004 | PoH 参数编码与碰撞处理 | old DEC-0002 + old DEC-0008 | 合并；删除 old DEC-0001 |
| DEC-0005 | PoH 时间戳的推导与隔离 | old DEC-0019 | 保留并重编号 |
| DEC-0006 | Stakes 精确定义 | old DEC-0016 | 保留并修正为评参区块 Stakes，不再引用 `-27` |
| DEC-0007 | 哈希树边界情况 | old DEC-0009 | 保留并消除空树规则歧义 |
| DEC-0008 | ML-DSA-65 集成路径 | old DEC-0010 | 保留并重编号 |
| DEC-0009 | 地址文本编码 | old DEC-0011 | 保留并重编号 |
| DEC-0010 | Coin 与 chx 换算 | old DEC-0015 | 保留并重编号 |
| DEC-0011 | 奖励与交易费余数归属 | old DEC-0006 | 保留并重编号 |
| DEC-0012 | Coinbase HashInputs 计算 | old DEC-0007 | 保留并重编号 |
| DEC-0013 | 首领黑名单层级 | old DEC-0012 | 保留并重编号 |
| DEC-0014 | 全网通告授权信任链 | old DEC-0013 + old DEC-0018 | 合并 |
| DEC-0015 | 创世高度年度边界 | old DEC-0014 | 保留并重编号 |
| DEC-0016 | 短引用歧义的协议级处理 | old DEC-0020 | 保留但按 conception 修正碰撞规则 |
| DEC-0017 | 脚本 VM Float 确定性 | old DEC-0005 | 保留并重编号 |

**Step 2: 在 README 草拟迁移表**

添加“旧编号迁移索引”，至少包含：

```markdown
| 旧编号 | 处理 | 新位置 |
|---|---|---|
| DEC-0001 | 删除/吸收 | `docs/conception/1.共识-历史证明（PoH）.md` 已固定单层铸凭哈希；剩余 X 编码与碰撞规则合并进新 DEC-0004 |
```

**Step 3: 暂不提交**

本任务只确定结构，和 Task 3 的文件变更一起提交。

## Task 3: 重命名、合并和删除第一阶段 DEC 文件

**Files:**
- Delete: `docs/decision/DEC-0001-poh-inner-hash-algorithm.md`
- Delete: old duplicate files after merging
- Create/Modify: `docs/decision/DEC-0001-*.md` through `DEC-0017-*.md`
- Modify: `docs/decision/README.md`

**Step 1: 使用临时目录避免重命名覆盖**

Run:

```bash
mkdir -p /tmp/evidcoin-decision-reorg
cp docs/decision/DEC-*.md /tmp/evidcoin-decision-reorg/
```

**Step 2: 清空旧 DEC 文件**

Run:

```bash
rm docs/decision/DEC-*.md
```

**Step 3: 重新创建新 DEC 文件**

按 Task 2 清单从旧文件内容迁移。关键修正：

- 新 DEC-0004 不再包含 old DEC-0001 的哈希算法决策主体，只说明：构想层已固定单层 `mintHash = BLAKE3-256(...)`；本 DEC 只补充 `X` 的编码与完全相同 mintHash 的碰撞处理。
- 新 DEC-0004 中 `Stakes` 来源写为“评参区块自身的 `Stakes` 字段”。
- 新 DEC-0004 中碰撞理由写“铸凭哈希为 32 字节”，不得再写 64 字节。
- 新 DEC-0002 的用途表删除 `MintInner` 和 `MintHash`，改为单层 `PoHMintHash` 或与构想一致的唯一命名。
- 新 DEC-0007 的空树规则只能保留一种：建议“空树根为对应根类型长度的全零字节，不再额外哈希 EMPTY”。
- 新 DEC-0014 合并授权公钥和授权根轮换，明确其只影响客户端展示层，不影响链上共识。
- 新 DEC-0016 按最新 conception 修正短引用碰撞：碰撞时按末端集合交易 ID 排序，首个匹配即引用；不得保留“≥2 项交易非法”的旧规则。

**Step 4: 更新 README 当前索引**

索引表包含列：编号、标题、主题域、状态、说明。

**Step 5: 验证编号连续**

Run:

```bash
find docs/decision -maxdepth 1 -type f -name 'DEC-*.md' | sort
```

Expected: 只显示 `DEC-0001` 到 `DEC-0017`，无旧文件残留。

**Step 6: 验证过时口径消失**

Run:

```bash
rg --line-number -- '-27|内层|外层|64 字节|64字节|MintInner|MintHash' docs/decision
```

Expected: 无与旧 PoH 双层哈希或 `-27` Stakes 相关命中。若 `MintHash` 作为迁移表旧标题出现，应改写为“旧 PoH 哈希决策”。

**Step 7: Commit**

```bash
git add docs/decision docs/plans
git commit -m "docs: reorganize existing decisions"
```

## Task 4: 新增高优先级 DEC 第二阶段文档

**Files:**
- Create: `docs/decision/DEC-0018-transaction-expiry-boundary.md`
- Create: `docs/decision/DEC-0019-signature-message-encoding.md`
- Create: `docs/decision/DEC-0020-coinbase-reward-slots.md`
- Create: `docs/decision/DEC-0021-issuance-schedule-rounding.md`
- Create: `docs/decision/DEC-0022-credit-expiry-boundary.md`
- Create: `docs/decision/DEC-0023-script-cost-budget.md`
- Create: `docs/decision/DEC-0024-script-float-derived-semantics.md`
- Create: `docs/decision/DEC-0025-block-proof-and-summary.md`
- Create: `docs/decision/DEC-0026-fork-tiebreak-hash.md`
- Modify: `docs/decision/README.md`

**Step 1: 创建 DEC-0018 交易过期边界**

内容应固定 `240` 个区块边界、比较字段、未来交易处理和区块验证行为。若 conception 已明确部分规则，只补边界。

**Step 2: 创建 DEC-0019 签名消息规范化编码**

内容应固定授权 flag 对应的输入/输出集合、序列化顺序、domain tag、非法组合。

**Step 3: 创建 DEC-0020 Coinbase 完整编码与公共服务兑奖槽**

内容应固定收益分配输出顺序、48 区块兑奖窗口、三类服务各 6 字节槽位、位序和未确认回收规则。

**Step 4: 创建 DEC-0021 发行曲线边界与取整**

内容应固定三年试运行、正式发行期、高度边界、每两年递减 20% 的整数取整方式。

**Step 5: 创建 DEC-0022 Credit 过期边界**

内容应固定 31 年按 `31 * 87661` 块计算，边界块验证语义和从 UTCO 移除时间。

**Step 6: 创建 DEC-0023 脚本成本预算**

内容应先固定成本预算框架和必须存在的三层限制；若缺少具体数值，状态可用 `Proposed`，并列出待冻结参数。

**Step 7: 创建 DEC-0024 脚本 Float 派生语义**

内容应补充 Float 比较、转换、`CMPFLO`、`WITHIN`、`RANGE`、`MO_MATH` 对公共验证结果的约束。

**Step 8: 创建 DEC-0025 区块证明包与概要短 TxID**

内容应固定区块证明包最小字段和概要中 16 字节 TxID 前缀碰撞处理。

**Step 9: 创建 DEC-0026 分叉平局 Hash 算法**

内容应在 `Hash256` 与 `HashX` 之间收敛为一个明确算法、domain tag 和输入编码。

**Step 10: 更新 README 索引**

把 DEC-0018 到 DEC-0026 加入当前索引。若某些新增 DEC 因信息不足为 `Proposed`，README 状态也必须一致。

**Step 11: Commit**

```bash
git add docs/decision
git commit -m "docs: add high priority decisions"
```

## Task 5: 全局一致性校验

**Files:**
- Read/Modify as needed: `docs/decision/*.md`
- Read/Modify as needed: `docs/decision/README.md`

**Step 1: 链接与编号检查**

Run:

```bash
find docs/decision -maxdepth 1 -type f -name 'DEC-*.md' | sort
rg --line-number 'DEC-[0-9]{4}' docs/decision/README.md docs/decision/DEC-*.md
```

Expected: README 中所有链接指向存在文件；编号连续。

**Step 2: 旧口径检查**

Run:

```bash
rg --line-number -- '-27|内层|外层|64 字节|64字节|MintInner|MintHash|>=2 项.*非法|拒绝.*歧义' docs/decision
```

Expected: 无未解释的旧口径残留；短引用 DEC 不得保留旧“歧义即非法”规则。

**Step 3: 文档格式检查**

Run:

```bash
for f in docs/decision/DEC-*.md; do
  grep -q '## Status（状态）' "$f" || echo "missing status: $f"
  grep -q '## Context（背景）' "$f" || echo "missing context: $f"
  grep -q '## Decision（决策）' "$f" || echo "missing decision: $f"
done
```

Expected: 无输出。

**Step 4: Commit 修正**

如有修正：

```bash
git add docs/decision
git commit -m "docs: fix decision consistency"
```

## Task 6: 请求代码/计划审查

**Files:**
- Review: `docs/decision/*.md`
- Review: `docs/decision/README.md`
- Review: `docs/plans/2026-05-10-decision-reorganization*.md`

**Step 1: 使用 code-plan-reviewer 审查**

请求审查重点：

- 是否仍有与 conception 冲突的 DEC。
- 新旧编号迁移是否可追踪。
- 删除 old DEC-0001 是否合理。
- 新增 DEC 是否越界重复 conception 已明确内容。
- 是否有高风险实现歧义未覆盖。

**Step 2: 根据审查结果修正**

只修正有证据的问题，不为风格意见大改结构。

**Step 3: 最终提交**

```bash
git add docs/decision docs/plans
git commit -m "docs: address decision review feedback"
```

## Task 7: 完成前验证

**Files:**
- Read: `docs/decision/README.md`
- Read: `docs/decision/DEC-*.md`

**Step 1: 最终 grep 验证**

Run:

```bash
find docs/decision -maxdepth 1 -type f -name 'DEC-*.md' | sort
rg --line-number -- '-27|内层|外层|64 字节|64字节|MintInner|MintHash' docs/decision || true
```

Expected: 文件编号连续；grep 无冲突性旧口径。

**Step 2: Git 状态**

Run:

```bash
git status --short
```

Expected: 工作区干净，或只剩用户明确要求不提交的文件。

**Step 3: 汇报结果**

汇报：

- 新 DEC 数量。
- 删除/合并的旧 DEC。
- 新增高优先级 DEC。
- 仍需用户裁决的 `Proposed` 项。
