# AGENTS.md - Evidcoin 开发指南

本文件为 AI 编程代理提供开发指导，涵盖边界说明、文档分布、构建命令、代码风格和项目规范等内容。


## 项目概览

Evidcoin（证信链）是基于区块链技术的通用信用载体系统，包含三种基本信元：
- **币金**（代币）：价值计量单位
- **凭信**：可转移的信用凭证
- **存证**：静态的存在性证明

技术特性：
- 共识机制：历史证明（PoH）
- 出块时间：固定 6 分钟
- 脚本系统：类 Bitcoin 栈式脚本，具备基本图灵完备性

使用自签名证书进行安全连接（SPKI指纹验证），有一个合理的有效期（如30天）。


## 边界说明

- P2P的节点发现和连网分享，归由外部库实现（cxio/p2p...等），本项目仅需考虑合理的接口设计。
- 组队校验中，各个角色单独实现（独立的App），它们之间通过连接相互通讯。因为联系紧密，应考虑高效的通讯方式。
- 区块/交易及附件数据的长期存储，主要由外部第三方公共服务提供。

> **提示：**
> 交易数据的检索由区块查询服务（`Blockqs`）提供。
> 附件数据的获取由数据驿站服务（`Depots`）支持（以文件P2P分享的方式）。


## 文档内容

各个功能的构想由不同的文档描述，文件关联如下：

| 功能 | 文件（conception/*） |
|------|---------------------|
| 共识 | `1.共识-历史证明（PoH）.md`, `2.共识-端点约定.md` |
| 服务 | `3.公共服务.md`, `4.激励机制.md` |
| 信用 | `5.信用结构.md` |
| 脚本 | `5.信用结构.md`, `6.脚本系统.md`, `Instruction/*.md` |
| 交易 | `附.交易.md`, `5.信用结构.md`, `6.脚本系统.md` |
| 校验 | `附.组队校验.md`, `附.交易.md`。相关牵涉：`6.脚本系统.md`, `5.信用结构.md`, `4.激励机制.md`, `3.公共服务.md`, `2.共识-端点约定.md`, `1.共识-历史证明（PoH）.md`  |
| 核心总管 | `blockchain.md`, `README.md` |


### 输出提案

当由设计构想（conception/*）生成提案时，输出应存放在 `proposal/` 目录下，对应文件如下：

| 功能 | 输出文件（proposal/*） |
|------|-----------------|
| 共识 | `1.Consensus-PoH.md` |
| 服务 | `2.Services(Third-party).md` |
| 信用 | `3.Evidence-Design.md` |
| 脚本 | `4.Script-of-Stack.md` |
| 交易 | `5.Transaction.md` |
| 校验 | `6.Checks-by-Team.md` |
| 核心总管 | `blockchain-core.md` |

> **注：**
> 命名文件已经存在（为空），输出内容直接填充即可。


## 构建/测试/Lint 命令

```bash
go build ./...                                    # 构建
go test ./...                                     # 运行所有测试
go test -v ./path/to/package -run TestName        # 运行单个测试
go test -v ./path/to/package -run TestName/Sub    # 运行子测试
go test -cover ./...                              # 测试覆盖率
go fmt ./... && gofmt -s -w .                     # 格式化
golangci-lint run                                 # 静态检查
go mod tidy && go mod verify                      # 依赖管理
```


## 项目结构

```
├── proposal/       # 设计提案
├── docs/plan/      # 根据提案由AI创建的实现方案
├── cmd/evidcoin/   # 主程序入口
├── internal/       # 私有包：blockchain/, consensus/, script/, tx/, utxo/
├── pkg/            # 公共包：crypto/, types/
└── test/           # 集成测试
```

> **注**：以上结构可根据需要进行调整。


## 代码风格

### 导入分组（空行分隔）

1. 标准库
2. 第三方库
3. 本项目包

### 命名规范

| 类型 | 规范 | 示例 |
|------|------|------|
| 包名 | 小写单词 | `blockchain`, `utxo` |
| 接口 | 动词+er | `Validator`, `Reader` |
| 结构体/导出函数 | 大驼峰 | `TxHeader`, `Validate()` |
| 私有函数/变量 | 小驼峰 | `parseInput()`, `txCount` |
| 常量 | 大驼峰或全大写 | `MaxBlockSize` |

### 类型定义示例

```go
type Hash512 [64]byte  // 512 位哈希值

type TxHeader struct {
    Version   int      // 版本号
    Timestamp int64    // 交易时间戳（Unix 纳秒）
    HashBody  Hash512  // 数据体哈希
}
```

### 错误处理

```go
// 错误信息：英文、小写开头、无标点
var ErrInvalidTx = errors.New("invalid transaction")

// 包装错误保留链
if err := validate(tx); err != nil {
    return fmt.Errorf("validate transaction: %w", err)
}
```

### 注释规范

- 注释语言使用中文，便于作者理解
- 导出符号必须有 godoc 风格注释
- 日志和错误输出使用英文

```go
// Block 表示一个区块，包含区块头和交易列表。
type Block struct {
    Header BlockHeader // 区块头
    Txs    []Tx        // 交易列表
}
```

### 并发处理

- 使用 `context.Context` 控制生命周期
- 优先使用 channel 进行 goroutine 通信
- 避免裸 goroutine，确保正确的错误处理和清理

```go
func processBlocks(ctx context.Context, blocks <-chan *Block) error {
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case block, ok := <-blocks:
            if !ok { return nil }
            if err := block.Validate(); err != nil {
                return fmt.Errorf("block %d: %w", block.Height, err)
            }
        }
    }
}
```

## 区块链特定规范

### 哈希计算

- 默认使用 SHA-512（64 字节）提升安全性
- 附件使用 BLAKE3 算法

### 脚本系统

脚本指令分为四个块段：
- 基础指令段：`[0-169]`，170 个
- 函数指令段：`[170-209]`，40 个
- 模块指令段：`[210-249]`，40 个
- 扩展指令段：`[250-253]`，4 个

### 关键常量

```go
const (
    HashLength      = 64                  // 哈希长度（字节，SHA-512）
    BlockInterval   = 6 * time.Minute     // 出块间隔
    BlocksPerYear   = 87661               // 每年区块数
    MaxStackHeight  = 256                 // 脚本栈最大高度
    MaxStackItem    = 1024                // 栈数据项最大尺寸
    MaxLockScript   = 1024                // 锁定脚本最大长度
    MaxUnlockScript = 4096                // 解锁脚本最大长度
    MaxTxSize       = 8192                // 单笔交易最大尺寸
)
```


## 测试规范

### 测试文件命名

- 单元测试：`*_test.go`（与源文件同目录）
- 集成测试：`test/` 目录

### 测试函数命名

```go
func TestTxHeader_Validate(t *testing.T) { }
func TestBlockchain_AddBlock(t *testing.T) { }
func BenchmarkHash512(b *testing.B) { }
```

### 表驱动测试

```go
// 命名：Test<Type>_<Method> 或 Test<Function>
func TestTxHeader_Validate(t *testing.T) {}
func BenchmarkHash512(b *testing.B) {}

// 表驱动测试
func TestHash512(t *testing.T) {
    tests := []struct {
        name    string
        input   []byte
        wantErr bool
    }{
        {"empty", nil, true},
        {"valid", []byte("test"), false},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := Hash512Sum(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("Hash512Sum() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```
