# Architecture Decision Records (架构决策记录)

本目录记录 Evidcoin 项目中所有正式的架构决策。每份 ADR 对应一个具体的技术问题，包含问题背景、决策内容及其理由。

## 命名规范

```
ADR-NNNN-<简短描述>.md
```

- `NNNN`：四位数字编号，从 `0001` 开始
- 描述：全小写英文，单词间用连字符（`-`）分隔

## 状态说明

| 状态 | 含义 |
|------|------|
| `Accepted` | 已确认并生效 |
| `Superseded` | 已被更新的 ADR 取代 |
| `Deprecated` | 已废弃，保留作历史参考 |

## ADR 索引

### 编码与哈希基础（Critical）

| 编号 | 标题 | 状态 |
|------|------|------|
| [ADR-0001](ADR-0001-poh-inner-hash-algorithm.md) | PoH 铸凭哈希内层算法 | Accepted |
| [ADR-0002](ADR-0002-poh-x-encoding.md) | 铸凭哈希 X 参数的计算与编码 | Accepted |
| [ADR-0003](ADR-0003-varint-encoding.md) | 规范化可变长度整数编码（varint） | Accepted |
| [ADR-0004](ADR-0004-domain-tag-format.md) | 哈希域分隔标签（Domain Tag）格式 | Accepted |

### 脚本系统（High）

| 编号 | 标题 | 状态 |
|------|------|------|
| [ADR-0005](ADR-0005-float-determinism.md) | 脚本 VM 浮点数确定性策略 | Accepted |
| [ADR-0006](ADR-0006-script-initial-pass-state.md) | 脚本初始 pass 状态 | Accepted |
| [ADR-0007](ADR-0007-check-override-semantics.md) | CHECK 指令覆盖语义 | Accepted |

### 状态与共识（High）

| 编号 | 标题 | 状态 |
|------|------|------|
| [ADR-0008](ADR-0008-utxo-txid-byte-index.md) | UTXO/UTCO 指纹树中间层 TxID 字节索引 | Accepted |
| [ADR-0009](ADR-0009-reward-rounding.md) | 区块奖励分配的取整与余数归属 | Accepted |
| [ADR-0010](ADR-0010-coinbase-hashinputs.md) | Coinbase 交易 HashInputs 计算规则 | Accepted |
| [ADR-0011](ADR-0011-minthash-tie-breaker.md) | 铸凭哈希碰撞的决胜规则 | Accepted |
| [ADR-0012](ADR-0012-block-signature-scope.md) | 区块铸造者签名消息范围 | Accepted |

### 哈希树策略（Medium）

| 编号 | 标题 | 状态 |
|------|------|------|
| [ADR-0013](ADR-0013-hash-tree-edge-cases.md) | 哈希树边界情况处理策略 | Accepted |
| [ADR-0014](ADR-0014-maxstackheight-boundary.md) | MaxStackHeight 边界检查语义 | Accepted |
| [ADR-0015](ADR-0015-intra-block-chained-spending.md) | 同区块内链式消费规则 | Accepted |
| [ADR-0016](ADR-0016-txidpart-collision.md) | 输入项 TxIDPart 碰撞处理 | Accepted |
| [ADR-0017](ADR-0017-public-service-evaluation.md) | 公共服务质量评估策略 | Accepted |
| [ADR-0018](ADR-0018-ml-dsa-65-integration.md) | ML-DSA-65 签名库集成路径 | Accepted |
| [ADR-0019](ADR-0019-sys-null-unlock-exception.md) | SYS_NULL 在解锁段的使用例外 | Accepted |

### 其他（Low/Nit）

| 编号 | 标题 | 状态 |
|------|------|------|
| [ADR-0020](ADR-0020-address-text-encoding.md) | 地址文本编码前缀 | Accepted |
| [ADR-0021](ADR-0021-credit-config-description-length.md) | Credit Config 低 10 位描述长度语义 | Accepted |

### 第二批决策（user-answer47 补充）

#### 区块链核心与共识（Critical/High）

| 编号 | 标题 | 状态 | 来源 |
|------|------|------|------|
| [ADR-0022](ADR-0022-checkroot-utxo-state-timing.md) | CheckRoot 承诺的 UTXO/UTCO 状态时间点 | Accepted | C-1 |
| [ADR-0023](ADR-0023-coinbase-independent-processing.md) | Coinbase 交易作为特殊交易独立处理 | Accepted | C-2 |
| [ADR-0024](ADR-0024-signature-witness-separation.md) | 签名数据隔离（签名 Witness 与脚本分离） | Accepted | C-4 |
| [ADR-0025](ADR-0025-minttxid-eligibility-by-block-height.md) | mintTxID 资格判定基准为区块高度 | Accepted | H-5 |
| [ADR-0027](ADR-0027-same-height-minter-tiebreaker.md) | 同高度同铸造者竞争区块的决胜规则 | Accepted | H-2/M-8 |

#### 节点行为与协议层级（Medium）

| 编号 | 标题 | 状态 | 来源 |
|------|------|------|------|
| [ADR-0026](ADR-0026-leader-blacklist-convention.md) | 首领黑名单冻结机制的层级（共约） | Accepted | M-5 |
| [ADR-0028](ADR-0028-announcement-authority-pubkey.md) | 全网通告与授权公钥更新机制 | Accepted | M-9 |
| [ADR-0029](ADR-0029-expansion-client-convention.md) | 百日扩张为客户端运行策略 | Accepted | L-3 |

#### 基础常量（Low）

| 编号 | 标题 | 状态 | 来源 |
|------|------|------|------|
| [ADR-0030](ADR-0030-genesis-year-boundary.md) | 创世高度 #0 为年度边界 | Accepted | L-4 |
| [ADR-0031](ADR-0031-coin-chx-ratio.md) | Coin 与 chx 换算比例 | Accepted | L-5 |
