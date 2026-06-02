// Package randomx 封装官方 RandomX 哈希器，用于分叉平局裁决（DEC-0303，第 12 章 §6）。
//
// 生产路径：使用官方 RandomX v2.0.1，commit aaafe71322df6602c21a5c72937ac284724ae561，
// 通过 CGO 封装 C/C++ 库，输出 32 字节（RANDOMX_HASH_SIZE）。禁止使用参数变体。
//
// 非 CGO 路径（构建标签 norandomx）：提供返回 ErrUnavailable 的桩实现，
// 使测试可在不安装 C 库的环境中编译，但运行时调用将报错。
// 向量测试（排序逻辑）在 randomx_tiebreak_test.go 中通过注入桩完成。
package randomx

import "errors"

// ErrUnavailable 表示 RandomX 哈希器在当前构建中不可用（未链接 C 库）。
var ErrUnavailable = errors.New("randomx: hasher unavailable in this build")

// Hasher 是 RandomX 哈希器接口。
// 实现必须遵循 DEC-0303 冻结的 profile：官方 v2.0.1、32B 输出、完整 VM 语义。
type Hasher interface {
	// Hash 计算 RandomX(seed, input)，返回 32 字节哈希值。
	// seed 为 ForkPointBlockID（48 字节），input 为 FirstForkBlockID（48 字节）。
	Hash(seed, input []byte) ([]byte, error)
}
