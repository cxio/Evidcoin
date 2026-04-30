# 05 — Script Engine (脚本引擎)

> 源自：`conception/6.脚本系统.md`，并结合
> `conception/Instruction/0.基本约束.md` 中的 opcode 集通用约束。
>
> 逐指令语义不在本文范围内；将于后续在 `docs/proposal/Instruction/`
> 下独立分批定义。

本提案定义 Evidcoin 脚本虚拟机的运行时模型与资源治理规则。本文不枚举
opcode——该部分后置处理。本文会对引擎做足够约束，使更高层提案（`03`
中的交易校验、`07` 中的校验组协同）可以依赖稳定的接口与限制语义。


## 1. Execution Model (执行模型)

引擎是一个 **stack VM**，包含以下区分明确的内存区域：

| Area              | Type               | Lifetime             | Purpose                                      |
|-------------------|--------------------|----------------------|----------------------------------------------|
| **Data stack**    | LIFO `[]any`       | Whole script         | opcode 默认输入/输出位置                     |
| **Argument area** | FIFO queue `[]any` | Whole script         | 预置参数区；每个 opcode 一次性消费           |
| **Local domain**  | per-block scope    | Lexical block        | 块私有命名存储（`VAR`）                      |
| **Global domain** | flat namespace     | Whole script         | 跨块命名存储                                 |
| **INPUT buffer**  | external in-channel | Per opcode call     | 阻塞式导入链外数据                           |
| **OUTPUT buffer** | external out-channel | Until `BUFDUMP`    | 非阻塞导出累积区                             |

### 1.1 Argument-resolution rule (参数解析规则)

 当执行一个 opcode 时：

1. 若 `argc` 为 `0`，则不访问 argument area / data stack。
2. 若 `argc` 为 `n`（`n > 0`）：
   - 尝试一次性从 **argument area** 精确取出 `n` 项。
   - 若 argument area 少于 `n` 项，则改为从 **data stack** 弹出 `n` 项。
   - 禁止混合取参：单个 opcode 只能完全从 argument area 取值，或完全从
     data stack 取值。
3. 若 `argc == -1`，该 opcode 为变参：清空 argument area 当前全部内容（可
   以为零项）；不会读取 data stack。
4. 若 `argc == -2`，参数数量由 opcode 自身附参在运行时决定。该规则仅适用
   于 `CALL`。

### 1.2 Auxiliary parameters (附参加载)

opcode 可在其字节后携带编译期固定长度参数。这些参数等同于 opcode 的属性
（例如 `IF{}` 的主体长度、`GOTO` 的目标位置）。它们在运行时不可变，也不会
经过 argument area 或 stack。

### 1.3 Associated data (关联数据)

部分 opcode 拥有内联数据载荷（如 `DATA{...}`、`IF{...}` 主体等）。这些载荷
在执行时只读，并计入 §3 定义的脚本长度预算。

### 1.4 Auto-spread (自动展开)

少数 opcode 会返回多个值。引擎 SHALL 自动将这些结果展开写入目标上下文
（data stack、argument area 或 local domain），无需额外包装 opcode。具体
哪些 opcode 具备该行为，由逐指令提案定义。


## 2. Opcode Block Layout (操作码分区布局)

8-bit opcode 空间按如下方式分区。分界固定，任何新增 opcode MUST 放入对应
分区的空闲槽位。

| Block                | Range      | Count | Purpose                                                     |
|----------------------|------------|-------|-------------------------------------------------------------|
| Base instructions    | `0..169`   | 170   | 流程控制、数学运算、栈操作、集合操作、模式匹配             |
| Function set         | `170..224` | 55    | 常用函数（PUBHASH、CHECKSIG、BASE32 等）+ 1 个自扩展槽位   |
| Module set           | `225..250` | 26    | 标准模块（`MO_RE`、`MO_TIME`、`MO_MATH` 等）+ 1 个自扩展槽位 |
| Extension references | `251..253` | 3     | `EXT_MO`、`EXT_PRIV` 等                                     |
| Reserved             | `254..255` | 2     | 预留给未来使用；启用需硬升级                                |

### 2.1 Naming convention (命名约定)

| Prefix    | Class                     | Example          |
|-----------|---------------------------|------------------|
| (none)    | 基础操作                  | `PASS`, `IF`     |
| `SYS_`    | 系统调用                  | `SYS_TIME`, `SYS_AWARD`, `SYS_CHKPASS`, `SYS_NULL` |
| `FN_`     | 函数                      | `FN_CHECKSIG`, `FN_PUBHASH` |
| `MO_`     | 模块                      | `MO_MATH`, `MO_TIME` |
| `EXT_`    | 扩展                      | `EXT_MO`, `EXT_PRIV` |

### 2.2 Three irregular opcodes (三类特殊 opcode，供引擎记录)

- `SYS_NULL` —— 可在 unlock script 中使用，作为 `[0..50]` opcode 范围限制
  的唯一例外（见 §4）。
- `SYS_TIME` —— 依赖 wall-clock；MUST NOT 出现在参与共识的
  *public-validation* 节点所执行的脚本中。
- `CALL` —— 参数数量为 `2 + 附参`，在运行时计算；这是唯一一个 `argc`
  无法静态确定的 opcode。


## 3. Resource Limits (Hard Cap, Protocol Level)（资源限制）

这些限制 MUST 在两个位置强制执行：

- **Static check** —— 交易准入阶段（parser）。
- **Runtime check** —— 脚本执行阶段（counter）。

| Limit                          | Value          |
|--------------------------------|----------------|
| Data-stack max height          | 256            |
| Stack-item max byte length     | 1 024          |
| Lock-script max length         | 1 024          |
| Unlock-script max length       | 4 096（不含通过 env 加载的标准签名数据） |
| Single transaction max size    | 65 535（不含 unlock data） |
| Output-item count              | < 1 024（1-byte varint × 8） |
| `EMBED` invocation count       | ≤ 4（运行时）  |
| `EMBED` nesting depth          | 0（嵌入脚本自身 MAY NOT 再 `EMBED` 或 `GOTO`） |
| `GOTO` invocation count (top-level script) | ≤ 2 |
| `GOTO` nesting depth (runtime) | ≤ 3            |

在校验流程中，lock-script 限制会检查两次：

1. **referenced source output** 的 lock script——已上链；verifier 仅做
   重新确认（数据完整性检查）。
2. **current transaction** 的 output lock scripts——仍由用户控制，用户有机会
   缩短脚本；超限交易 MUST 在准入阶段被拒绝。


## 4. Unlock-Script Restriction (解锁脚本限制)

unlock script MAY 仅使用 `[0..50]` 范围内的 opcode（value、capture、stack、
set、interaction 操作），外加唯一例外 `SYS_NULL`（opcode 169）。使用其他任意
opcode 都会导致交易被拒绝。

理由：放宽 unlock script 过于危险——`TRUE PASS EXIT` 本身就足以短路任意
lock script。


## 5. Cost Budget (Soft Cap, Protocol Level)（成本预算）

每个 opcode 都分配一个 **deterministic cost function**。该成本函数以
opcode 的静态参数和动态参数尺寸为输入，返回非负整数（"compute units"）。
内存占用单独跟踪并受独立限制（每栈项与栈高上限已足够）。

执行引擎维护三个累加器：

| Accumulator                | Cap symbol             |
|----------------------------|------------------------|
| Single-input script cost   | `MaxInputScriptCost`   |
| Single-tx total cost       | `MaxTxScriptCost`      |
| Single-block total cost    | `MaxBlockScriptCost`   |

超过任一上限都会以 FAIL 中止脚本；若发生在单输入场景，则该输入解锁失败，整笔
交易被拒绝。

### 5.1 Determinism rule (确定性规则)

> **A cost function returns a value that is purely a function of the opcode's
> static and dynamic parameters. It MUST NOT depend on local hardware.**

成本函数返回值必须仅由 opcode 的静态与动态参数决定，MUST NOT 依赖本地硬件。

这是强制要求；否则不同 validator 会产生分歧。

成本函数表 *not* 属于本提案范围；将在逐指令提案完成后派生。初始合理取值将参
考构想文档中引用的 GPT-5.5 成本分析。


## 6. Block Data Size Schedule (区块数据规模时间表)

每个区块的 payload 大小上限（不含标准签名数据，包含 unlock scripts）：

| Period                | Cap              | Notes                              |
|-----------------------|------------------|------------------------------------|
| Months 1–3 (≈21 915 blocks) | 1 MB        | 恒定                               |
| Months 4–12 (≈9 × 7 305) | +1 MB / month, up to 10 MB | 线性爬升                 |
| Year 2 onward         | +1 MB / sidereal year | Year 2 = 11 MB，Year 3 = 12 MB，… |

（`87 661` blocks = 1 sidereal year。）


## 7. External Script References (External Cache)（外部脚本引用）

`GOTO` 与 `EMBED` 会引用存放于其他交易中的脚本主体（即 *intermediary*
output，output config "介管"）。为避免每个 verifier 在热路径重复解析此类引
用，validator group MUST 维护 **External Script Cache**——见提案 `07`。

该缓存保存已解析脚本主体，以 `(year, txid_short, out_index)` 作为键，并在底层
intermediary output 创建或失效时更新。


## 8. Interaction Buffers (交互缓冲区)

| Op       | Direction | Buffer       | Semantics                                                                 |
|----------|-----------|--------------|---------------------------------------------------------------------------|
| `INPUT`  | inbound   | INPUT buffer | 阻塞式；若无值或数量不匹配则 FAIL（仅引擎本地生效；does NOT 影响调用者 PASS 状态）。在 public-validation 节点上，`INPUT` 等价于隐式 `END`。 |
| `OUTPUT` | outbound  | OUTPUT buffer | 非阻塞；将数据追加到缓冲区。                                              |
| `BUFDUMP`| outbound  | OUTPUT buffer | 刷新并重置。触发任意已注册监听器（节点私有）。                            |

两个缓冲区是独立通道；它们不共享数据。


## 9. Intermediary & Custom Output Classes (中介与自定义输出类别)

`03` 中定义的 output-config 低 4 位 class 字段决定脚本行为：

| Class id | Meaning      | Engine impact                                                       |
|----------|--------------|----------------------------------------------------------------------|
| 1        | Coin         | Lock script 决定 spend                                              |
| 2        | Credit       | Lock script 决定 transfer                                           |
| 3        | Proof        | 仅用于 identification script；永不进入 UTXO/UTCO                    |
| 4        | Intermediary (介管) | `GOTO` / `EMBED` 的目标；自身不能作为 input source          |

custom-class output（config bit 7 置位）携带不透明 private-class ID（≤ 127
bytes）。public verifier 忽略其私有语义；脚本主体仍 MUST 在标准引擎规则下可解析
并可执行。


## 10. Public-API Surface (Layer 3)（公共 API 面）

脚本引擎向上层暴露最小接口：

```go
package script

// Engine 在给定上下文下执行一对 {unlock || lock} 脚本。
// 引擎为只读、可并发实例化（每次执行新建一份运行态）。
type Engine interface {
    Execute(ctx ExecContext, unlock, lock []byte) Result
}

type ExecContext struct {
    Tx          TxView          // 只读交易视图
    InputIndex  int             // 当前输入序位
    SourceOut   OutputView      // 引用源输出
    SignatureEnv SignatureEnv   // 解锁数据携带的签名相关信息
    Caches      Caches          // UTXO/UTCO + external script cache
    CostBudget  CostBudget      // 三层成本上限
}

type Result struct {
    OK   bool
    Cost uint64    // 实际消耗的计算单位
    Err  error
}
```

`Caches` 与 `TxView` 类型分别归属其所在层（`02`、`04`、`07`）；`script`
仅依赖只读视图，从而保持严格分层规则。
