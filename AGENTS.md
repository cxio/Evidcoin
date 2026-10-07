# AGENTS.md - Evidcoin 仓库指令

## 命令

- Go 版本来自 `go.mod`：`go 1.26.2`，模块名 `github.com/cxio/evidcoin`。没有 Makefile 或 CI workflow，直接用 Go 命令验证。
- 全量验证：`go fmt ./... && go test ./... && go test -cover ./... && go build ./... && go mod tidy && go mod verify && golangci-lint run`。
- 聚焦包测试：`go test ./internal/blockchain -run TestName -v` 或 `go test ./pkg/types -run TestName -v`。
- 阶段验收要求核心逻辑覆盖率至少 80%，`golangci-lint run` 无 warning；先尝试执行，若本机未安装 lint，只能报告环境阻塞，不能写"lint 通过"。
- 运行 `go mod tidy` 后必须检查 `go.mod`/`go.sum` diff，只保留任务需要的依赖变化。

## 分层边界参考

- Layer 0：`pkg/types`、`pkg/crypto`、`pkg/hashtree`；不能依赖 `internal/*`。
- Layer 1：`internal/blockchain`、`internal/tx`；只依赖 Layer 0。
- Layer 2：`internal/utxo`、`internal/utco`；依赖 Layer 0-1。
- Layer 3：`internal/script`；封装对 https://github.com/cxio/Escript 的调用。
- Layer 4：`internal/consensus`；依赖 Layer 0-3。
- Layer 5：`internal/validation`、`internal/rewards`、`internal/services`、`cmd/evidcoin`、`test`；不能被 Layer 0-4 import。
- 检查方向可用：`go list -deps ./internal/blockchain | grep -E 'internal/(tx|utxo|utco|script|consensus|rewards|validation|services)'`，应无输出。

## 协议实现规则

- 所有协议字节序列必须由显式编码函数生成；不要用 JSON、反射、map 遍历顺序或平台字节序做共识前像。
- 固定宽度整数使用 `pkg/types` 的大端追加/读取工具。
- 哈希必须走 `pkg/crypto` 中按用途绑定的函数；不要在调用处拼自定义 domain tag。唯一免域标签例外是附件分片树 profile。
- `pkg/hashtree` 通用树混合 48B 叶哈希与 32B 分支哈希，`Root()` 返回 `[]byte`；空根由具体结构定义，单叶根会归一化为分支哈希，奇数层最后节点直接提升。
- `internal/blockchain` 只管理区块头链与最小衔接验证；不计算交易树、不执行交易/脚本/状态转移、不判断 PoH、不做自动长期分叉重组。
- 区块头规范编码：`Version||Height||PrevBlock||CheckRoot||Stakes`，仅年块追加 `YearBlock`；创世高度 0 是年块且 `YearBlock` 全零。
- `CheckRoot` 状态根取前一区块完成后的 UTXO/UTCO 指纹；创世使用空状态根。UTXO 与 UTCO 顺序不可交换。

## 代码与测试约定

- 导出符号写英文 Godoc；解释实现意图的源码注释用中文；`errors.New`、日志、程序输出使用英文文本。
- 测试使用 table-driven tests，并覆盖成功、失败和边界值；协议编码测试必须断言字节级输出。
- 生产存储、网络、长期数据保存尚不是低层包职责；测试替身放 `_test.go`，避免被误用为生产实现。
- 提交只在用户明确要求时执行；若按 plan 分任务提交，每个 Task 通过局部测试后单独提交，不混合层级，不提交 `.DS_Store`、临时日志、覆盖率文件或本地 IDE 配置。
