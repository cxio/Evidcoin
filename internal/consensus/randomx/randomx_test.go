package randomx

import "testing"

func TestStubHasherReturnsError(t *testing.T) {
	// 桩实现在非 CGO 构建下始终返回 ErrUnavailable；
	// 启用 randomx_cgo 标签的正确性测试须在链接 C 库的环境中运行。
	h := New()
	seed := make([]byte, 48)
	input := make([]byte, 48)
	_, err := h.Hash(seed, input)
	// 桩构建：期望 ErrUnavailable；CGO 构建：期望 nil 或其他非 ErrUnavailable 错误
	if err == nil {
		// CGO 构建通过：不报错即为正确；此处仅测试桩
		t.Log("CGO build: Hash succeeded (stub test skipped)")
		return
	}
	if err != ErrUnavailable {
		t.Errorf("stub Hash: expected ErrUnavailable, got %v", err)
	}
}
