//go:build !randomx_cgo

package randomx

// stubHasher 是 no-CGO 构建下的桩实现，始终返回 ErrUnavailable。
// 构建标签 randomx_cgo 存在时由 cgo_impl.go 替换为真实 CGO 封装。
type stubHasher struct{}

// Hash 在非 CGO 构建下始终返回 ErrUnavailable。
func (stubHasher) Hash(_, _ []byte) ([]byte, error) {
	return nil, ErrUnavailable
}

// New 返回桩实现实例（非 CGO 构建）。
// 生产环境需以 randomx_cgo 构建标签链接官方 C 库。
func New() Hasher {
	return stubHasher{}
}
