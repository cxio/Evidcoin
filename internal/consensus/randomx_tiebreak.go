package consensus

import (
	"bytes"
	"errors"

	rxpkg "github.com/cxio/evidcoin/internal/consensus/randomx"
	"github.com/cxio/evidcoin/pkg/types"
)

// RandomX 平局裁决（第 12 章 §6，DEC-0303）。
//
// 31 高度比较完成仍平局时计算：
//   score = RandomX(seed=ForkPointBlockID, input=FirstForkBlockID)
//
// 排序：score 字典序升序较小者胜；score 相同则比较分叉首块 ID 较小者胜。
// RandomX 版本：官方 v2.0.1，commit aaafe71322df6602c21a5c72937ac284724ae561，
// 32B 输出，完整 VM 语义；CGO 封装见 internal/consensus/randomx/。
// 禁止在常规出块路径触发本函数。

// RandomXTiebreak 使用给定哈希器对两条分叉的首块 ID 计算 RandomX 得分，
// 返回胜出侧（DEC-0303，第 12 章 §6）。
//
// 参数：
//   - hasher：RandomX 哈希器（生产用 randomx.New()，测试可注入桩）。
//   - forkPoint：分叉点区块 ID（48 字节 seed）。
//   - forkA：A 链分叉首块 ID（48 字节 input for A）。
//   - forkB：B 链分叉首块 ID（48 字节 input for B）。
//
// 若 hasher 不可用（返回 ErrUnavailable），则直接转为比较 forkA / forkB 字节序。
func RandomXTiebreak(
	hasher rxpkg.Hasher,
	forkPoint, forkA, forkB types.BlockID,
) (ForkSide, error) {
	seed := forkPoint[:]

	scoreA, errA := hasher.Hash(seed, forkA[:])
	scoreB, errB := hasher.Hash(seed, forkB[:])

	if errors.Is(errA, rxpkg.ErrUnavailable) || errors.Is(errB, rxpkg.ErrUnavailable) {
		return ForkSideNone, ErrRandomXUnavailable
	}
	if errA != nil {
		return ForkSideNone, errA
	}
	if errB != nil {
		return ForkSideNone, errB
	}

	// score 字典序升序，较小者胜（DEC-0303）
	cmp := bytes.Compare(scoreA, scoreB)
	if cmp < 0 {
		return ForkSideA, nil
	}
	if cmp > 0 {
		return ForkSideB, nil
	}

	// score 相同 → 比较分叉首块 ID（DEC-0303）
	cmp = bytes.Compare(forkA[:], forkB[:])
	if cmp < 0 {
		return ForkSideA, nil
	}
	if cmp > 0 {
		return ForkSideB, nil
	}
	// 完全相同（极低概率）→ 返回平局
	return ForkSideNone, nil
}
