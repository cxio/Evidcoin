//go:build !equix_cgo

package equix

// stubSolver 是非 CGO 构建下的桩实现，始终返回 ErrUnavailable。
// 构建标签 equix_cgo 存在时由 cgo_impl.go 替换为真实 CGO 封装。
type stubSolver struct{}

// Solve 在非 CGO 构建下始终返回 ErrUnavailable。
func (stubSolver) Solve(_ []byte, _ uint64) ([][]byte, []byte, uint64, error) {
	return nil, nil, 0, ErrUnavailable
}

// Verify 在非 CGO 构建下始终返回 ErrUnavailable。
func (stubSolver) Verify(_ []byte, _ uint64, _ []byte) ([][]byte, bool, error) {
	return nil, false, ErrUnavailable
}

// New 返回桩实现实例（非 CGO 构建）。
// 生产环境需以 equix_cgo 构建标签链接对应 C 库。
func New() Solver {
	return stubSolver{}
}
