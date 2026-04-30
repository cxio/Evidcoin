# 01 — Cryptography Primitives (密码学原语)

> 来源：`conception/blockchain.md`（哈希策略）、`conception/附.交易.md`
>（签名、多签、地址编码）。

本提案固定全链使用的密码学原语、哈希算法分配、地址编码，以及签名/验签接口。
所有更高层提案都应导入此处定义的符号。


## 1. Hash Algorithm Allocation (哈希算法分配)

每个调用点的算法选择都属于规范性要求。实现 **MUST NOT** 以“等效”强度的其他算法替代。

| Site                                  | Algorithm           | Output | Rationale                                                                 |
|---------------------------------------|---------------------|--------|---------------------------------------------------------------------------|
| Block header                          | SHA3-384            | 48 B   | 在量子安全余量与载荷大小之间平衡                                          |
| `CheckRoot`                           | SHA3-384            | 48 B   | 一次性组合，计算成本可忽略                                                 |
| Transaction header (TxID)             | SHA3-384            | 48 B   | 与 Block header 保持对称                                                   |
| Inner tree branches & leaves of block-level trees (Tx tree, UTXO/UTCO trees) | SHA3-384 leaf, BLAKE3-256 branch | 48/32 B | 跨层树结构遵循“leaf ≠ branch”规则 |
| Inner tree branches & leaves of intra-tx trees (Inputs, Outputs, Attachment chunks) | BLAKE3-256 (uniform) | 32 B | 优化性能与内存；允许用户显式授权覆盖 |
| Attachment fingerprint (file-level)   | SHA3-512            | 64 B   | 长期安全；与 chunk-tree 配合形成双重哈希保护                               |
| Public-key hash (account address)     | SHA3-256(BLAKE2b-512(pk)) | 32 B | 在确认 128-bit Grover 量子下限前提下节省空间（32 vs 48 B）               |
| Mint-pledge hash (`hashMint`)         | BLAKE3-512          | 64 B   | 面向大规模并发计算；兼顾速度与熵                                           |
| UTXO/UTCO leaf payload digest (`DataID`) | SHA3-384         | 48 B   | 与状态树叶子保持同一算法族                                                 |
| Address checksum                      | SHA3-256 trailing 4 B | 4 B  | 见 §4                                                                      |

### 1.1 BLAKE3-512 implementation note (BLAKE3-512 实现说明)

多数 BLAKE3 库默认输出 256-bit。mint-pledge hash **MUST** 使用可扩展输出（XOF）模式计算，且输出长度为 64 bytes，例如
`lukechampine.com/blake3` 的 `XOF.Read(buf[:64])`。调用默认 API 后再截断/扩展属于不符合规范的实现。

### 1.2 TxID transitive dependence on BLAKE3 (TxID 对 BLAKE3 的传递性依赖)

`TxID = SHA3-384(TxHeader)`，而 `TxHeader` 包含 `HashInputs` / `HashOutputs`（BLAKE3-256）。因此，TxID 的抗碰撞性除依赖 SHA3 外，还会传递性依赖 BLAKE3 的密码学假设。这是可接受的权衡；实现方 **SHOULD** 在面向用户的文档中说明该点。

### 1.3 Public-key hash quantum margin (公钥哈希的量子安全余量)

`SHA3-256(BLAKE2b-512(pk))` 可提供约 128-bit 的 Grover 量子安全，是系统中最弱的哈希安全环境。UTXO 地址会在链上暴露至未花费输出生命周期结束，因此在后量子攻击场景下，地址哈希将成为最先承压点。该选择是有意的（空间 vs 量子安全），并已为归档记录明确注明。


## 2. Signature Suite (签名套件)

本链优先采用后量子方案。默认且当前强制算法为 **ML-DSA-65**（FIPS 204）。实现 **MUST** 通过一个小而稳定的 Go 接口暴露该能力，以便未来 hard-fork 升级算法套件时，不会在代码库中引发大范围改动。

```go
package crypto

// Signer 表示可生成签名的实体。
// 抽象签名能力，便于后续替换或追加算法。
type Signer interface {
    PublicKey() PublicKey         // 关联公钥
    Sign(message []byte) ([]byte, error)
    Algorithm() AlgID             // 例如：AlgMLDSA65
}

// Verifier 表示可验证签名的实体。
type Verifier interface {
    Verify(pk PublicKey, message, sig []byte) bool
    Algorithm() AlgID
}

// AlgID 枚举受支持的签名算法。
type AlgID uint8

const (
    AlgUnknown AlgID = 0
    AlgMLDSA65 AlgID = 1   // 默认：FIPS 204
    // AlgMLDSA87 AlgID = 2 // 预留，未来可用
)
```

### 2.1 ML-DSA-65 sizes (FIPS 204)（ML-DSA-65 尺寸）

| Object       | Size (bytes) |
|--------------|--------------|
| Public key   | 1 952        |
| Secret key   | 4 032        |
| Signature    | 3 309        |

这些数值决定了 §05 中 unlock-script 与 signature-data 预算的上限；尤其是，signature data *not* 计入 4 KB 的 unlock-script 上限 —— 见提案 `05` §3。


## 3. Multi-Signature (M-of-N) (多重签名)

组合 public-key hash 地址在外观上与单签地址不可区分。签名语义如下：

- **N** = 参与的 public key 总数。
- **M** = 最小所需签名数（`M ≤ N`，且二者都 ≤ 255）。

### 3.1 Address derivation (地址推导)

```text
PKH_i  = SHA3-256(BLAKE2b-512(pk_i))     for i = 1..N
PKHs   = SHA3-256(BLAKE2b-512(
                       byte(M) ‖ byte(N) ‖
                       PKH_1  ‖ PKH_2 ‖ … ‖ PKH_N))
```

拼接前需先将 `PKH_i` 按升序排序（字典序 byte 顺序）；地址创建方与验证方都 **MUST** 执行该排序，以确保同一组 public key 只会产出唯一规范地址。

### 3.2 Multi-sig unlock data (多签解锁数据)

unlock data 包含三组集合：

1. *Signature set* —— `M` 个签名。
2. *Public-key set* —— 与 (1) 对应的 `M` 个 public key，顺序一致。
3. *Completion set* —— 未参与签名的 `N − M` 个 public-key *hashes*。

`M` 与 `N` 分别由 `len(sig set)` 以及 `len(sig set) + len(completion set)` 推导。

### 3.3 Multi-sig verification flow (多签验证流程)

1. 对 (2) 中每个 public key 重新哈希，得到 `PKH'_i`。
2. 将 `{PKH'_i}` 与 (3) 合并为一个集合，升序排序并拼接，前置 `M ‖ N`，计算 `PKHs'`。
3. 将 `PKHs'` 与收款地址比较（需先去除链上 checksum/prefix）。
4. 使用 (2) 中对应 `pk` 逐一验证 `M` 个签名。

任一步骤失败即中止 unlock，并返回 FAIL。


## 4. Address Encoding (地址编码)

地址是 `PKH || checksum` 的文本编码，并带有网络 prefix。

### 4.1 Encoding (编码)

```
0. data       = prefix || PKH                       （二进制）
1. checksum   = SHA3-256(data)[28:32]               （末尾 4 bytes）
2. payload    = PKH || checksum                     （不含 prefix）
3. address    = prefix-text || Base58(payload)      （最终可读文本）
```

`Base58` 使用标准字母表（Bitcoin 约定）—— 参考 `mr-tron/base58`。`prefix-text` 是短 ASCII 标签（如 mainnet 使用 `evd1`）；二进制 `prefix` byte(s) 来自 `pkg/types/address` 维护的注册表。

### 4.2 Decoding & validation (解码与校验)

1. 拆分 `prefix-text`，映射得到 `prefix` bytes。
2. 对主体做 Base58 decode，拆为 `PKH || checksum`（最后 4 bytes）。
3. 重新计算 `SHA3-256(prefix || PKH)[28:32]`，并与解析出的 checksum 比较。
4. 不一致则拒绝；一致则返回 `(prefix, PKH)`。


## 5. Signing Domain (Chain-binding Prefix) (签名域)

每条已签名交易消息都以前置链标识元组的方式绑定链上下文，以抵御跨链重放：

```
MixData  = Protocol-ID ‖ Chain-ID ‖ Genesis-ID ‖ Bound-ID ‖ TxMSG
SignData = Sign(MixData)
```

`Bound-ID` 是可选项；但若存在，**MUST** 等于 `BlockID[txTimestamp.height - 29][:20]`。验证者 **MUST** 将缺失视为用户的显式选择并接受签名；但对于已提供且与规范链不匹配的 `Bound-ID`，**MUST** 无条件拒绝。


## 6. Authorisation Modes (授权模式)

签名消息可通过 1-byte authorisation mask 收紧或放宽。以下 bit 分配为规范性要求（NORMATIVE）。

```
// 独项（可独立成立；也可按需组合）
bit 7  SIGIN_ALL      所有 inputs
bit 6  SIGIN_SELF     仅当前 input
// 主项（至少需要一个辅项）
bit 5  SIGOUT_ALL     所有 outputs
bit 4  SIGOUT_SELF    与当前 input 同序位的 output
// 辅项（选择被签 outputs 的哪些字段）
bit 0  SIGRECEIVER    仅 receiver address
bit 1  SIGCONTENT     value/note（Coin）；creator/config/title/desc/attID（Credit）
bit 2  SIGSCRIPT      lock script
bit 3  SIGOUTPUT      完整 output 条目（receiver + content + script）
```

最常见配置 `SIGIN_ALL | SIGOUT_ALL | SIGOUTPUT` 作为 wallet 默认值，除非用户明确选择其他模式。

mask byte 是 unlock data 的组成部分，并会进入 verification pre-image，因此任何 bit 翻转都会导致签名失败。


## 7. Internal API Surface (Layer 0) (内部 API 边界)

`pkg/crypto` 仅允许包含以下 packages；其子包 **MUST** 保持对所有 `internal/*` 代码的无依赖。

```
pkg/crypto/
├── hash/        # sha3、blake3 封装；blake3-512 的 xof 辅助
├── address/     # base58、prefix 注册表、encode/decode/checksum
├── sig/         # Signer/Verifier 接口 + ML-DSA-65 实现
└── multisig/    # M-of-N 组合与验证
```

该层预期依赖的外部 Go modules：

```
golang.org/x/crypto              // sha3
lukechampine.com/blake3          // blake3 + XOF
github.com/cloudflare/circl      // ML-DSA-65（在 std-lib 覆盖前使用）
github.com/mr-tron/base58        // base58 字母表
```

若 `crypto/mldsa` 进入标准库，则 **MUST** 优先使用它，而非 `circl` 导入。
