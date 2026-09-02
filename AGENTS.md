# AGENTS.md - Evidcoin 仓库指令

## 先读哪里

- 实现功能前先读对应 `docs/plan/NN-*.md`，再追溯 `docs/proposal/NN.*.md`；若冲突，权威顺序为 `docs/conception/` > `docs/decision/` > `docs/proposal/` > `docs/plan/`。
- `docs/plan/00-Implementation-Roadmap.md` 是阶段与分层索引；`docs/plan/12-Open-Questions-And-Acceptance.md` 是待决项、验收和依赖检查索引。
- `docs/AGENTS.md` 说明了 `docs/` 目录的四层文档结构与维护规则，实现前应阅读。
- `working/` 与 `docs/plans/`（注意：是复数 `plans`，是临时区）不是规范来源，也不提交；`opencode.json` 配置为忽略它们。`docs/plan/`（单数）才是正式计划。

## 当前代码状态

所有生产包均已实现：`pkg/types`、`pkg/crypto`、`pkg/hashtree`（Layer 0），`internal/blockchain`、`internal/tx`（Layer 1），`internal/utxo`、`internal/utco`（Layer 2），`internal/script`（Layer 3），`internal/consensus`（Layer 4），`internal/validation`、`internal/rewards`、`internal/services`（Layer 5）。`test/` 集成目录尚未创建（Plan 12 阶段）。

- `internal/script` 当前覆盖率约 64.7%，低于 80% 目标，是覆盖率缺口包。
- `pkg/types` 当前覆盖率约 78.3%，低于 90% 目标。

## 命令

- Go 版本来自 `go.mod`：`go 1.26.2`，模块名 `github.com/cxio/evidcoin`。没有 Makefile 或 CI workflow，直接用 Go 命令验证。
- 全量验证：`go fmt ./... && go test ./... && go test -cover ./... && go build ./... && go mod tidy && go mod verify && golangci-lint run`。
- 聚焦包测试：`go test ./internal/blockchain -run TestName -v` 或 `go test ./pkg/types -run TestName -v`。
- 阶段验收要求核心逻辑覆盖率至少 80%，`golangci-lint run` 无 warning；先尝试执行，若本机未安装 lint，只能报告环境阻塞，不能写"lint 通过"。
- 运行 `go mod tidy` 后必须检查 `go.mod`/`go.sum` diff，只保留任务需要的依赖变化。

## 分层边界

- Layer 0：`pkg/types`、`pkg/crypto`、`pkg/hashtree`；不能依赖 `internal/*`。
- Layer 1：`internal/blockchain`、`internal/tx`；只依赖 Layer 0。
- Layer 2：`internal/utxo`、`internal/utco`；依赖 Layer 0-1。
- Layer 3：`internal/script`；依赖低层接口，不反向依赖。
- Layer 4：`internal/consensus`；依赖 Layer 0-3。
- Layer 5：`internal/validation`、`internal/rewards`、`internal/services`、`cmd/evidcoin`、`test`；不能被 Layer 0-4 import。
- 检查方向可用：`go list -deps ./internal/blockchain | grep -E 'internal/(tx|utxo|utco|script|consensus|rewards|validation|services)'`，应无输出。

## 协议实现规则

- 所有协议字节序列必须由显式编码函数生成；不要用 JSON、反射、map 遍历顺序或平台字节序做共识前像。
- 固定宽度整数使用 `pkg/types` 的大端追加/读取工具；BigInt 遵循 DEC-0001 的 `slen||magnitude` 最短编码，拒绝前导零与负零。
- 哈希必须走 `pkg/crypto` 中按用途绑定的函数；不要在调用处拼自定义 domain tag。唯一免域标签例外是附件分片树 profile。
- `pkg/hashtree` 通用树混合 48B 叶哈希与 32B 分支哈希，`Root()` 返回 `[]byte`；空根由具体结构定义，单叶根会归一化为分支哈希，奇数层最后节点直接提升。
- `internal/blockchain` 只管理区块头链与最小衔接验证；不计算交易树、不执行交易/脚本/状态转移、不判断 PoH、不做自动长期分叉重组。
- 区块头规范编码：`Version||Height||PrevBlock||CheckRoot||Stakes`，仅年块追加 `YearBlock`；创世高度 0 是年块且 `YearBlock` 全零。
- `CheckRoot` 状态根取前一区块完成后的 UTXO/UTCO 指纹；创世使用空状态根。UTXO 与 UTCO 顺序不可交换。

## 待决项不能硬编码

- 当前全局待决项只限 C-6、C-7、C-9、C-10；编码时用策略参数、接口注入、明确错误或阻塞标注隔离。
- C-6：脚本成本数值未定，不能固化 opcode 成本向量。
- C-7：禁用指令解除方式未定，不能预设激活路径。
- C-9：创世时间戳与 mainnet `Genesis-ID` 未冻结；只能固定创世工件结构与验证规则，不能伪造具体值。
- C-10：P2P 线格式、版本分叉治理、通用子链派生协议未抽象；只声明接口边界，外包给相关服务/库。

## 代码与测试约定

- 导出符号写英文 Godoc；解释实现意图的源码注释用中文；`errors.New`、日志、程序输出使用英文文本。
- 测试使用 table-driven tests，并覆盖成功、失败和边界值；协议编码测试必须断言字节级输出。
- 生产存储、网络、长期数据保存尚不是低层包职责；测试替身放 `_test.go`，避免被误用为生产实现。
- 提交只在用户明确要求时执行；若按 plan 分任务提交，每个 Task 通过局部测试后单独提交，不混合层级，不提交 `.DS_Store`、临时日志、覆盖率文件或本地 IDE 配置。
