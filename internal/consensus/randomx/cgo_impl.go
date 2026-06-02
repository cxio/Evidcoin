//go:build ignore

// cgo_impl.go 是 RandomX CGO 封装的实现模板（DEC-0303）。
// 默认不编译（build ignore）；接入官方 C 库后移除 ignore 标签，
// 并使用构建标签 randomx_cgo 替换 stub.go 的桩实现：
//
//	go build -tags randomx_cgo ./...
//
// 接入步骤：
//  1. 克隆官方仓库：git clone https://github.com/tevador/RandomX
//     checkout commit aaafe71322df6602c21a5c72937ac284724ae561（tag v2.0.1）
//  2. 将 src/*.cpp 与 include/randomx.h 置于本目录，或通过 CGO_LDFLAGS
//     指向已编译的 librandomx.a。
//  3. 将本文件首行改为：//go:build randomx_cgo
//     并将 stub.go 首行改为：//go:build !randomx_cgo
//  4. 重新构建：go build -tags randomx_cgo ./...
//
// CGO 封装骨架（接入 C 库后填充 rx_hash 调用）：
//
//	/*
//	#cgo LDFLAGS: -lrandomx
//	#include "randomx.h"
//	static int rx_hash(const void* seed, size_t seed_len,
//	                   const void* input, size_t input_len,
//	                   unsigned char* out) {
//	    randomx_flags flags = randomx_get_flags();
//	    randomx_cache* cache = randomx_alloc_cache(flags);
//	    if (!cache) return -1;
//	    randomx_init_cache(cache, seed, seed_len);
//	    randomx_vm* vm = randomx_create_vm(flags, cache, NULL);
//	    if (!vm) { randomx_release_cache(cache); return -2; }
//	    randomx_calculate_hash(vm, input, input_len, out);
//	    randomx_destroy_vm(vm);
//	    randomx_release_cache(cache);
//	    return 0;
//	}
//	*/
//	import "C"

package randomx
