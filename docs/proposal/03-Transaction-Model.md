# 03 — Transaction Model (交易模型)

> 来源：`conception/附.交易.md`，并交叉吸收
> `conception/5.信用结构.md`（按信用单元类型定义输出载荷）与
> `conception/4.激励机制.md`（Coinbase 奖励分成）的内容。

本提案规定交易头、输入项与输出项结构、交易内哈希树、签名语义，以及
Coinbase 交易的特殊结构与验证规则。


## 1. Transaction Header (TxHeader)（交易头）

### 1.1 Structure (proposed; subject to user review)（结构；提案版，待用户审阅）

```go
type TxHeader struct {
    Version     uint32      // 协议版本号
    Timestamp   int64       // 交易时间戳（Unix 毫秒）
    HashInputs  [32]byte    // 输入项根哈希（BLAKE3-256）
    HashOutputs [32]byte    // 输出项根哈希（BLAKE3-256）
}

// 交易ID：
// 算法：SHA3-384
TxID = SHA3-384(Serialize(TxHeader))
```

序列化为定长字段顺序、大端整数。`Timestamp` 取毫秒精度；其 UTC 公历年决定交
易的"年度"（参见 §1.2 与 `04` §3）。

> **设计注记：** 交易头中不包含手续费字段，手续费通过输入金额合计 - 输出金额合计
> 隐式表达；不包含输入/输出列表本身（仅含其哈希），具体载荷在交易体中。

### 1.2 Year of a transaction（交易所属年度）

```text
year(tx) = UTC_calendar_year(tx.Timestamp)
```

该年度用于：

- 输入项中"交易所在年度"字段（§2.1）；
- UTXO/UTCO 指纹的顶层分级（参见 `04` §3）；
- 铸币递减公式中"年次"的判定（参见 `06` §6 与 conception/4）。


## 2. Input Items (输入项)

### 2.1 Per-item layout（单项布局）

每个输入项包含 3 ~ 4 个字段：

| 字段                  | 长度       | 说明                                                                |
|-----------------------|------------|---------------------------------------------------------------------|
| `Year`                | 变长整数   | 来源交易所在年度                                                    |
| `TxID` / `TxIDPart`   | 48B 或 20B | 首领输入用全 48B；其余用前 20B 片段                                 |
| `OutIndex`            | 变长整数   | 来源输出项序位（多数 < 128，1 字节）                                |
| `TransferIndex`       | 变长整数   | 仅凭信转移类输入项使用：当前凭信转出到的目标输出位置（用于合规检查）|

### 2.2 Lead Input Constraint（首领输入约束）

- 索引 0 的输入项称为 **首领输入** (Lead Input)，必须使用 48B 完整 TxID。
- 首领输入必须是币金（Coin）花费，且为全部币金输入中**币权销毁最大**者；这是
  首领校验（参见 `07` §3）的基本前提。
- 其余输入仅以 20B 前缀引用；UTXO/UTCO 集的指纹机制保证了引用唯一性，故
  20B 缩短引用仅理论上有"重名"风险（参见 `07` 不收录规则）。

### 2.3 Input Tree (2-layer)（输入树，2 层）

```text
LeadHash = BLAKE3-256( Serialize(LeadInput) )
RestHash = BLAKE3-256( Serialize([RestInput₁, …, RestInputₙ]) )
HashInputs = BLAKE3-256( LeadHash ‖ RestHash )
```

> **跨树规则一致性：** 交易内部所有树（输入树、输出树、附件片组树）的叶与枝**统**
> **一**采用 BLAKE3-256，此为用户授权的明确决定（详见 `01` §1）。

### 2.4 Constraint: no chained-unconfirmed spends（约束：禁止未确认链式花费）

UTXO/UTCO 仅来源于 **已确认** 区块。一笔未确认交易的输出 *不能* 作为另一笔
未确认交易的输入。上链顺序按交易费、币权销毁等优先级由管理层裁定（参见 `07`
§4）。这同时是组队校验对交易独立性的硬性要求。


## 3. Output Items (输出项)

### 3.1 Common config byte（通用配置字节）

每输出项前置 1 字节 `Config`：

```text
位 7  自定义类标志：置位时表示后续含 1 字节"类 ID 长度"+ 类 ID 数据
位 6  附件标志：声明本输出携带附件（自定义类置位时此位含义变更，由私有协议解释）
位 5  销毁标志：用于币金 / 凭信，置位则该输出不进入 UTXO/UTCO
位 4  保留
位 0..3  类型值（低 4 位）：
       0 reserved
       1 Coin            可作输入源
       2 Credit          可作输入源
       3 Proof           不可作输入源
       4 Intermediary    介管脚本，仅供脚本跳转 / 嵌入引用
```

销毁配置（位 5）的输出项接收地址 MAY 为 `null`，锁定脚本任意。

非标准接收地址通常意味着自定义验证（脚本不含 `SYS_CHKPASS`）。

### 3.2 Coin output payload（Coin 输出载荷）

| 字段          | 说明                                                       |
|---------------|------------------------------------------------------------|
| `Receiver`    | 接收者地址（公钥哈希 32B；自定义验证时 ≤ 256 B 任意字节）   |
| `Amount`      | 变长整数；最小单位 `chx`（同 Satoshi）                      |
| `Note`        | 附言；≤ 255 B；可选                                        |
| `LockScript`  | 锁定脚本                                                    |

### 3.3 Credit output payload（Credit 输出载荷）

| 字段             | 说明                                                                                  |
|------------------|---------------------------------------------------------------------------------------|
| `Receiver`       | 接收者地址                                                                            |
| `CreatorOrRef`   | ≤ 256 B；初创时为创建者，转移时为对原始凭信的引用                                     |
| `Config2`        | 2 字节凭信配置（参见下表）                                                            |
| `Title`          | ≤ 256 B；通常人类可读                                                                 |
| `Description`    | ≤ 1 024 B；按 `Config2` 低 10 位编码长度                                              |
| `AttachmentID`   | 可选，详见 `5.信用结构` §附；本提案在 §7 给出固化结构                                  |
| `LockScript`     | 锁定脚本                                                                              |

`Config2` 16 位标志位：

```text
位 15  新建标记
位 14  可否修改：允许下家修订描述（创建者、标题、附件ID 不可变）
位 13  是否修改：若上一态可修改且本次已修改，置位
位 12  保留
位 11  转移计次：表示后续 2 字节为剩余转移次数
位 10  高度截止：表示后续变长整数为有效期截止区块高度
位 0..9  低 10 位：描述内容长度（< 1024）
```

约束：

- 转移计次与高度截止可同时存在，以先到为准。
- 转移计次每次输出递减，降至 0 时该输出**不再**进入 UTCO。
- 高度截止不可超过 100 年（相对高度差，约 8 766 100 块）。
- 无截止凭信至少每 11 年激活一次（可自转），否则失效；此为协议强制约束。
- 可修改性单向退化：可修改 → 不可修改是允许的，反之不可。

### 3.4 Proof output payload（Proof 输出载荷）

| 字段           | 说明                                              |
|----------------|---------------------------------------------------|
| `Creator`      | ≤ 256 B；可为空                                   |
| `Title`        | ≤ 256 B                                            |
| `Config2`      | 2 字节；低 12 位编码内容长度（≤ 4 KB）            |
| `Content`      | 内容主体                                          |
| `AttachmentID` | 可选                                              |
| `RecogScript`  | 识别脚本，用于第三方应用程序化识别（不可作为输入源）|

### 3.5 Intermediary output payload（Intermediary 输出载荷）

仅含锁定脚本（即"介管脚本"），不进入 UTXO/UTCO，不可作输入项；只能在脚本中
通过 `GOTO` / `EMBED` 引用。其输出 `Receiver` 字段允许为空。

### 3.6 Output Tree（输出树）

输出项作为叶子节点，按二元含序结构组织：

```text
OutputLeaf_i = BLAKE3-256( Serialize(Output_i) )
HashOutputs  = BLAKE3-256( binary-tree-root over { OutputLeaf_0, OutputLeaf_1, … } )
```

序号信息已隐含于树中位置（`Output_i.Serial = i`）。


## 4. Signature Semantics（签名语义）

### 4.1 Authorisation mask（授权掩码）

参见 `01` §6（独项 / 主项 / 辅项的位定义）。该掩码字节是解锁数据的一部分，
作为验证 pre-image 的强制因子。

### 4.2 Pre-image structure（预映像结构）

```text
MixData  = Protocol-ID ‖ Chain-ID ‖ Genesis-ID ‖ Bound-ID ‖ TxMSG
SignData = Sign(MixData)
```

`TxMSG` 由"授权种类"决定：

- `SIGIN_*` 选定输入项集合；
- `SIGOUT_*` 选定输出项集合；
- `SIGRECEIVER` / `SIGCONTENT` / `SIGSCRIPT` / `SIGOUTPUT` 决定每个被选输出项
  的字段范围。

具体序列化顺序为：`AuthMask ‖ TxHeader ‖ SelectedInputsCanonical ‖
SelectedOutputsCanonical`，其中 *Canonical* 表示按输出/输入项的自然序号顺序逐
项序列化对应字段。

### 4.3 Multi-signature（多重签名）

参见 `01` §3。多签验证完全由解锁数据驱动；锁定脚本一般为 `SYS_CHKPASS` 调用，
亦可由用户自行编写。


## 5. Coinbase Transaction (铸币交易)

### 5.1 Position & uniqueness（位置与唯一性）

每个区块 **必须** 在序号 0 位置包含且仅包含一笔 Coinbase 交易。无输入项，
其"输入树"以约定方式哈希（`HashInputs = BLAKE3-256(empty)` 或固定哨兵，由实现
固定为 `BLAKE3-256("coinbase")`，待确认时统一）。

### 5.2 Header & body（头部与体部）

| 区块 | 字段                                                        |
|------|-------------------------------------------------------------|
| 头部 | 标准 `TxHeader`（同其它交易）                              |
| 体部 | 无 `Inputs`；含 *特殊数据段* + *输出项序列*                |

特殊数据段（按字段顺序）：

| 字段                  | 说明                                                          |
|-----------------------|---------------------------------------------------------------|
| `BlockHeight`         | 当前区块高度（明确位置）                                      |
| `MintCert`            | 择优凭证：交易定位 + 铸造者公钥 + 签名数据 + (可选) 铸凭哈希 |
| `Income`              | 收益分配项：`{Mint, Fees, AwardWithheld, AwardSlots}`         |
| `FreeData`            | 自由数据，≤ 256 B                                            |

`MintCert` 形式（与 `06` §3.4 一致）：

```go
type MintCert struct {
    Year       uint32     // 铸凭交易年度
    PledgeTxID [48]byte   // 铸凭交易 ID
    MinterPubKey []byte   // 首领输入接收者公钥（不是哈希）
    Sign       []byte     // 铸造者对铸凭演算数据的签名
    HashMint   [64]byte   // BLAKE3-512(Sign) 可选缓存
}
```

`AwardSlots` 是兑奖槽位标记，参见 §5.4。

### 5.3 Output Items（输出项；固定五分成）

按 conception/4 与用户最终决定的比例：

| 序位 | Beneficiary       | 配置低 4 位 | 比例 |
|------|-------------------|-------------|------|
| 0    | Validator Group   | 2           | 40 % |
| 1    | Mint Pledger      | 1           | 10 % |
| 2    | Blockqs           | 3           | 20 % |
| 3    | Depots            | 4           | 20 % |
| 4    | STUN              | 5           | 10 % |

整数除法 `Total * pct / 100`，余数累加给 *最后一位* 输出项接收者（即 STUN）。

> **配置位标志：** 该 5 项输出的 1-byte Config 仅启用低 4 位作"奖励/分成目标"
> 编码，高 4 位（自定义类、附件、销毁、保留）一律为 0。这是固定模式，便于
> O(1) 校验。

如需销毁交易费的 50% 部分，则在 `Income.Fees` 计入前先做 `floor / 2`，余下
50% 进入 `Total`，被销毁的 50% 表达为一个特殊的"销毁输出"（位 5 置位、接收者
为 null）；该销毁输出**追加**在 5 项分成之后（序位 5）。

### 5.4 Award Redemption Slots (兑奖槽)（兑奖槽位）

`AwardSlots` 是 18 字节标记位（48 区块 × 3 服务 = 144 位）。每个新铸造者按其
所在区块的位置在槽中翻转对应位，对前 48 个区块内的三类受奖目标进行确认。

- 兑奖完整规则与回收规则参见 `04` 与 `07` §6。
- 三种服务的兑奖槽 SHOULD 在实现时分开为 `AwardSlotsBlockqs[6B]`、
  `AwardSlotsDepots[6B]`、`AwardSlotsSTUN[6B]`，便于后期分别调整。

### 5.5 Coinbase Validation Checklist（Coinbase 验证清单）

校验节点接收 Coinbase 时，按顺序执行以下检查；任何一项失败即拒绝：

1. 位置：交易序号 0；当且仅当一笔。
2. 无输入项。
3. `BlockHeight` 与所属区块的高度字段一致。
4. `MintCert` 合法：铸凭交易在 `[-80 000, -28]` 区间；铸造者公钥可哈希得到
   该铸凭交易首领输入的接收者地址（账户地址匹配）；签名验签通过；铸造者位于
   评参区块的择优池中（详见 `06` §3）。
5. `Income.Mint` 与该高度的官方铸币公式吻合（参见 `06` §6）。
6. `Income.Fees` ≤ 区块内所有非 Coinbase 交易的实际手续费总和；销毁了
   等额的另 50%。
7. 5 项分成输出严格按 §5.3 顺序、比例与配置位排列；金额合计 ≤ Total，余数
   归 STUN 项。
8. `AwardSlots` 长度 = 18 B；内部三段 6 B 各自合法。
9. Coinbase 也参与"输出树"计算，与其它交易一同贡献于 `HashOutputs`、进而
   `TxID`、进而区块的交易树根。

### 5.6 100-Day Initial Phase（前 100 天初始阶段）

对于 `Height ≤ 24 000` 的早期区块，Coinbase 输出仅含 1 个"铸造者"输出
（无公共服务奖励），具体阶段划分见 `06` §6.4。期间每笔交易的输出项数量 SHOULD
≤ 输入项数量 × 2，以约束扩张速度（共约）。


## 6. Public Key Hash & Address Encoding（公钥哈希与地址编码）

参见 `01` §4。任何在交易中作为接收者出现的标准账户地址（公钥哈希）都遵循该编
码规则。


## 7. Attachment ID Layout（附件 ID 布局）

凭信、存证（以及未来扩展到币金时）可携带附件 ID。固定结构：

```text
[1B]  TotalLen   附件 ID 总长度（≤ 255）
[1B]  TypeMajor  附件大类
[1B]  TypeMinor  附件小类
[64B] Fingerprint  附件指纹 SHA3-512（不分片，整体哈希）
[2B]  ChunkCount 分片数量（< 65 536）
[32B] ChunkRoot  片组哈希；分片 ≤ 1 时该字段缺省 / 直接为单片哈希
[~]   FileSize   附件大小，变长整数
```

规则：

- `ChunkCount == 0`：无分片；`ChunkRoot` 字段省略。
- `ChunkCount == 1`：无分片；`ChunkRoot` 即为该单片的哈希值。
- `ChunkCount > 1`：哈希校验树（含序），叶 = `BLAKE3-256(seq[2] ‖
  chunkData)`，枝 = `BLAKE3-256(left ‖ right)`。
- 单分片上限 1 MB；64 k 分片可表达 64 GB。
- `FileSize` 字段无法被链上验证，用户填写不实可能影响数据被存储的概率。


## 8. Public-API Surface (Layer 1)（公共 API 形态，第 1 层）

```go
package tx

// Tx 是交易的内存表达。
type Tx struct {
    Header  TxHeader
    Inputs  []Input         // 首项必为首领输入；Coinbase 时为空
    Outputs []Output
    // Coinbase 专用字段（仅 IsCoinbase() 时非零）：
    Cb *CoinbaseExtra
}

// ID 计算并返回 TxID（缓存于结构体内）。
func (t *Tx) ID() [48]byte

// IsCoinbase 判断本交易是否为 Coinbase。
func (t *Tx) IsCoinbase() bool

// Validator 验证交易合法性（依赖 UTXO/UTCO + Script Engine）。
type Validator interface {
    Validate(t *Tx, ctx ValidateContext) error
}
```

`ValidateContext` 包括 UTXO/UTCO 视图、当前区块上下文、脚本引擎实例、缓存集
合等，由 `04`、`05`、`07` 共同提供。
