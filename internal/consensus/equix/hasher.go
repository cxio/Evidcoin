// Package equix 封装 Equi-X 工作量证明求解器，用于 PoH 铸凭哈希第一阶段（DEC-0301）。
//
// 默认构建使用桩实现，Solve/Verify 始终返回 ErrUnavailable。
// 生产环境需以 -tags equix_cgo 构建，并先链接对应 C 库（参见 cgo_impl.go 中的接入步骤）。
package equix

import "errors"

// ErrUnavailable 表示 Equi-X 求解器在当前构建中不可用（未链接 C 库）。
var ErrUnavailable = errors.New("equix: solver unavailable in this build")

// Solver 是 Equi-X 工作量证明接口。
// 实现必须遵循 DEC-0301 冻结的 profile：官方 Equi-X 算法、solution 索引严格升序。
type Solver interface {
	// Solve 对 seed 从 nonce >= startNonce 开始搜索满足 Equi-X 约束的解，
	// 返回哈希列表、solution 字节与实际使用的 nonce。
	// solution 的索引必须严格升序（DEC-0301）。
	Solve(seed []byte, startNonce uint64) (hashList [][]byte, solution []byte, nonce uint64, err error)

	// Verify 验证 (seed, nonce, solution) 三元组是否满足 Equi-X 约束，
	// 返回哈希列表（用于重算 MintHash）与验证结果。
	// 当求解器不可用（如未链接 C 库）时返回 ErrUnavailable。
	Verify(seed []byte, nonce uint64, solution []byte) (hashList [][]byte, valid bool, err error)
}
